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

package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/antinvestor/service-trustage/connector"
)

// defaultAdapterHTTPTimeout is the per-call outbound HTTP timeout applied by
// every adapter. Workers that process up to 500 concurrent messages (NATS
// consumer_max_ack_pending) must not be pinned indefinitely by a slow
// third-party endpoint. The value is intentionally conservative; operators
// can tune per-adapter timeouts in a future release.
//
// TODO(v1.4): expose per-adapter timeout via config.AdapterHTTPTimeoutSeconds
// threaded through the registry constructor.
const defaultAdapterHTTPTimeout = 30 * time.Second

// withAdapterTimeout derives a child context bounded by defaultAdapterHTTPTimeout,
// unless the parent context already has a shorter deadline.
func withAdapterTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= defaultAdapterHTTPTimeout {
		// Parent context is already tighter; don't extend.
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, defaultAdapterHTTPTimeout)
}

// executeAPIPost validates the URL, POSTs the payload, reads the response,
// and returns the parsed JSON body. It is the shared execution path for all
// adapters that call an external JSON API with a Bearer key.
func executeAPIPost(
	ctx context.Context,
	client *http.Client,
	req *connector.ExecuteRequest,
	payload map[string]any,
) (map[string]any, *connector.ExecutionError) {
	apiURL, _ := req.Config["api_url"].(string)
	if apiURL == "" {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "MISSING_CONFIG",
			Message: "api_url is required in config",
		}
	}

	if err := validateExternalURL(ctx, apiURL); err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "SSRF_BLOCKED",
			Message: fmt.Sprintf("URL not allowed: %v", err),
		}
	}

	respBody, execErr := doAPIPost(ctx, client, apiURL, req, payload)
	if execErr != nil {
		return nil, execErr
	}

	var parsed map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &parsed)
	}

	return parsed, nil
}

// doAPIPost marshals payload, POSTs it to apiURL using the given client,
// and returns the raw response body. It handles idempotency and Bearer auth
// from req. Callers own error classification via classifyHTTPStatus.
func doAPIPost(
	ctx context.Context,
	client *http.Client,
	apiURL string,
	req *connector.ExecuteRequest,
	payload map[string]any,
) ([]byte, *connector.ExecutionError) {
	// Apply per-adapter timeout so a slow target can't pin a worker goroutine.
	ctx, cancel := withAdapterTimeout(ctx)
	defer cancel()

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "MARSHAL_ERROR",
			Message: fmt.Sprintf("failed to marshal payload: %v", err),
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	if apiKey, ok := req.Credentials["api_key"]; ok {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    "HTTP_ERROR",
			Message: fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("failed to read response: %v", err),
		}
	}

	if execErr := classifyHTTPStatus(resp.StatusCode, string(respBody)); execErr != nil {
		return nil, execErr
	}

	return respBody, nil
}
