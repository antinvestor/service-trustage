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
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	webhookType        = "webhook.call"
	webhookDisplayName = "Webhook"
	maxResponseBody    = 1 << 20 // 1MB
)

// WebhookAdapter sends HTTP requests to webhook URLs.
type WebhookAdapter struct {
	client *http.Client
}

// NewWebhookAdapter creates a new WebhookAdapter with the given HTTP client.
func NewWebhookAdapter(client *http.Client) *WebhookAdapter {
	return &WebhookAdapter{client: client}
}

func (a *WebhookAdapter) Type() string        { return webhookType }
func (a *WebhookAdapter) DisplayName() string { return webhookDisplayName }
func (a *WebhookAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["url"],
		"properties": {
			"url": {"type": "string", "format": "uri"},
			"method": {"type": "string", "enum": ["POST", "PUT", "PATCH"], "default": "POST"},
			"headers": {"type": "object", "additionalProperties": {"type": "string"}},
			"body": {"type": "object"}
		}
	}`)
}

func (a *WebhookAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *WebhookAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"status_code": {"type": "integer"},
			"body": {"type": "object"}
		}
	}`)
}

func (a *WebhookAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["url"] == nil {
		return errors.New("url is required")
	}

	return nil
}

func (a *WebhookAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	// Apply per-adapter timeout so a slow target can't pin a worker goroutine.
	ctx, cancel := withAdapterTimeout(ctx)
	defer cancel()

	url, _ := req.Input["url"].(string)
	method, _ := req.Input["method"].(string)

	if method == "" {
		method = http.MethodPost
	}

	// SSRF protection: validate URL is not targeting internal networks.
	if err := validateExternalURL(ctx, url); err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "SSRF_BLOCKED",
			Message: fmt.Sprintf("URL not allowed: %v", err),
		}
	}

	var bodyReader io.Reader

	if body, ok := req.Input["body"]; ok {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "MARSHAL_ERROR",
				Message: fmt.Sprintf("failed to marshal body: %v", err),
			}
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if headers, ok := req.Input["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, sOk := v.(string); sOk {
				httpReq.Header.Set(k, s)
			}
		}
	}

	// Set idempotency key header if provided.
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := a.client.Do(httpReq)
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

	execErr := classifyHTTPStatus(resp.StatusCode, string(respBody))
	if execErr != nil {
		return nil, execErr
	}

	var parsedBody map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &parsedBody)
	}

	return &connector.ExecuteResponse{
		Output: map[string]any{
			"status_code": resp.StatusCode,
			"body":        parsedBody,
		},
		RawBody: respBody,
	}, nil
}

const maxErrorBodyLen = 512

// truncateBody limits response body included in error messages to prevent data leakage.
func truncateBody(body string) string {
	if len(body) > maxErrorBodyLen {
		return body[:maxErrorBodyLen] + "...(truncated)"
	}

	return body
}

// classifyHTTPStatus classifies HTTP response codes per ADR-016.
func classifyHTTPStatus(status int, body string) *connector.ExecutionError {
	safeBody := truncateBody(body)

	switch {
	case status >= http.StatusOK && status < http.StatusMultipleChoices:
		return nil
	case status == http.StatusBadRequest,
		status == http.StatusUnauthorized,
		status == http.StatusForbidden,
		status == http.StatusNotFound,
		status == http.StatusMethodNotAllowed,
		status == http.StatusConflict,
		status == http.StatusUnprocessableEntity:
		return &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    fmt.Sprintf("HTTP_%d", status),
			Message: safeBody,
		}
	case status == http.StatusTooManyRequests:
		return &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "HTTP_429",
			Message: "rate limited",
		}
	case status >= http.StatusInternalServerError:
		return &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    fmt.Sprintf("HTTP_%d", status),
			Message: safeBody,
		}
	default:
		return &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    fmt.Sprintf("HTTP_%d", status),
			Message: safeBody,
		}
	}
}
