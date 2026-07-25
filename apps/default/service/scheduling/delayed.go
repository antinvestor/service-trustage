// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduling

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pitabwire/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const defaultCloudTasksHorizonDays = 30

// DelayedPublisher schedules work to fire at a future time.
// Cloud Tasks path reuses Frame's cloudtasks URL shape; Noop relies on reconcile.
type DelayedPublisher interface {
	PublishAt(ctx context.Context, ref string, at time.Time, payload any) error
}

// NoopDelayedPublisher always succeeds without scheduling (reconcile is SoT).
type NoopDelayedPublisher struct{}

// PublishAt implements DelayedPublisher.
func (NoopDelayedPublisher) PublishAt(context.Context, string, time.Time, any) error {
	return nil
}

// CloudTasksDelayedPublisher creates tasks with per-message scheduleTime.
// URL template: cloudtasks:///projects/{p}/locations/{l}/queues/{q}?url=https://host/_frame/queue/{ref}&oidc_sa=...
// The {ref} placeholder in the url query is replaced with the registration reference.
type CloudTasksDelayedPublisher struct {
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	baseURL     string // CreateTask API URL without trailing /tasks
	targetURL   string // template with {ref}
	oidcSA      string
	oidcAud     string
	maxHorizon  time.Duration
}

// NewCloudTasksDelayedPublisherFromTemplate parses a Frame-style cloudtasks URL template.
// Returns Noop when template is empty.
func NewCloudTasksDelayedPublisherFromTemplate(
	ctx context.Context,
	template string,
	httpClient *http.Client,
	maxHorizonHours int,
) (DelayedPublisher, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return NoopDelayedPublisher{}, nil
	}

	u, err := url.Parse(template)
	if err != nil {
		return nil, fmt.Errorf("parse cloudtasks template: %w", err)
	}
	if strings.ToLower(u.Scheme) != "cloudtasks" {
		return nil, fmt.Errorf("cloudtasks template: unsupported scheme %q", u.Scheme)
	}

	// Path form: /projects/p/locations/l/queues/q
	path := strings.Trim(u.Path, "/")
	if u.Host != "" && path == "" {
		path = strings.Trim(u.Host+"/"+strings.Trim(u.Opaque, "/"), "/")
	}
	parts := strings.Split(path, "/")
	// projects/{p}/locations/{l}/queues/{q}
	var project, location, queueName string
	for i := 0; i+1 < len(parts); i++ {
		switch parts[i] {
		case "projects":
			project = parts[i+1]
		case "locations":
			location = parts[i+1]
		case "queues":
			queueName = parts[i+1]
		}
	}
	if project == "" || location == "" || queueName == "" {
		return nil, errors.New("cloudtasks template: need projects/locations/queues in path")
	}

	target := u.Query().Get("url")
	if target == "" {
		return nil, errors.New("cloudtasks template: missing url query param")
	}

	if httpClient == nil {
		return nil, errors.New("cloudtasks delayed publisher: http client is required")
	}

	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("cloudtasks token source: %w", err)
	}

	const hoursPerDay = 24
	horizon := time.Duration(maxHorizonHours) * time.Hour
	if horizon <= 0 {
		horizon = defaultCloudTasksHorizonDays * hoursPerDay * time.Hour
	}

	apiURL := fmt.Sprintf(
		"https://cloudtasks.googleapis.com/v2/projects/%s/locations/%s/queues/%s/tasks",
		project, location, queueName,
	)

	return &CloudTasksDelayedPublisher{
		httpClient:  httpClient,
		tokenSource: ts,
		baseURL:     apiURL,
		targetURL:   target,
		oidcSA:      u.Query().Get("oidc_sa"),
		oidcAud:     u.Query().Get("oidc_audience"),
		maxHorizon:  horizon,
	}, nil
}

// PublishAt creates a Cloud Task scheduled at `at`. Beyond max horizon, no-ops (reconcile covers).
func (p *CloudTasksDelayedPublisher) PublishAt(ctx context.Context, ref string, at time.Time, payload any) error {
	if p == nil {
		return nil
	}
	now := time.Now().UTC()
	if at.After(now.Add(p.maxHorizon)) {
		util.Log(ctx).Debug("delayed publisher: beyond horizon, relying on reconcile",
			"ref", ref, "due_at", at,
		)
		return nil
	}
	if at.Before(now) {
		at = now
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("delayed publish marshal: %w", err)
	}

	target := strings.ReplaceAll(p.targetURL, "{ref}", ref)
	// Also support .../_frame/queue/REF_PLACEHOLDER style: append ref if no placeholder.
	if !strings.Contains(p.targetURL, "{ref}") && !strings.HasSuffix(target, ref) {
		target = strings.TrimRight(target, "/") + "/" + ref
	}

	task := map[string]any{
		"scheduleTime": at.UTC().Format(time.RFC3339),
		"httpRequest": map[string]any{
			"httpMethod": "POST",
			"url":        target,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"body": base64.StdEncoding.EncodeToString(body),
		},
	}
	if p.oidcSA != "" {
		httpReq, ok := task["httpRequest"].(map[string]any)
		if !ok {
			return errors.New("delayed publish: httpRequest missing")
		}
		oidc := map[string]string{"serviceAccountEmail": p.oidcSA}
		if p.oidcAud != "" {
			oidc["audience"] = p.oidcAud
		}
		httpReq["oidcToken"] = oidc
	}

	reqBody, err := json.Marshal(map[string]any{"task": task})
	if err != nil {
		return fmt.Errorf("delayed publish task marshal: %w", err)
	}

	tok, err := p.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("delayed publish token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("delayed publish request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delayed publish do: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delayed publish: unexpected status %d", resp.StatusCode)
	}
	return nil
}
