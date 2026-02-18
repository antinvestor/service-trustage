package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	httpType        = "http.request"
	httpDisplayName = "HTTP Request"
)

// HTTPAdapter makes generic HTTP requests with support for query params,
// auth headers, and multiple body encoding types.
type HTTPAdapter struct {
	client *http.Client
}

// NewHTTPAdapter creates a new HTTPAdapter with the given HTTP client.
func NewHTTPAdapter(client *http.Client) *HTTPAdapter {
	return &HTTPAdapter{client: client}
}

func (a *HTTPAdapter) Type() string        { return httpType }
func (a *HTTPAdapter) DisplayName() string { return httpDisplayName }
func (a *HTTPAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["url", "method"],
		"properties": {
			"url": {"type": "string", "format": "uri"},
			"method": {"type": "string", "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"]},
			"headers": {"type": "object", "additionalProperties": {"type": "string"}},
			"query": {"type": "object", "additionalProperties": {"type": "string"}},
			"body": {"type": "object"},
			"auth_header": {"type": "string"}
		}
	}`)
}

func (a *HTTPAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *HTTPAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"status_code": {"type": "integer"},
			"headers": {"type": "object"},
			"body": {"type": "object"}
		}
	}`)
}

func (a *HTTPAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["url"] == nil {
		return errors.New("url is required")
	}

	if req.Input["method"] == nil {
		return errors.New("method is required")
	}

	return nil
}

func (a *HTTPAdapter) Execute( //nolint:funlen,gocognit // HTTP request building is inherently verbose
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	rawURL, _ := req.Input["url"].(string)
	method, _ := req.Input["method"].(string)

	// SSRF protection: validate URL is not targeting internal networks.
	if ssrfErr := validateExternalURL(ctx, rawURL); ssrfErr != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "SSRF_BLOCKED",
			Message: fmt.Sprintf("URL not allowed: %v", ssrfErr),
		}
	}

	// Build URL with query params.
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "INVALID_URL",
			Message: fmt.Sprintf("invalid URL: %v", err),
		}
	}

	if queryParams, ok := req.Input["query"].(map[string]any); ok {
		q := parsedURL.Query()

		for k, v := range queryParams {
			if s, sOk := v.(string); sOk {
				q.Set(k, s)
			}
		}

		parsedURL.RawQuery = q.Encode()
	}

	// Build body.
	var bodyReader io.Reader

	if body, ok := req.Input["body"]; ok {
		bodyBytes, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "MARSHAL_ERROR",
				Message: fmt.Sprintf("failed to marshal body: %v", marshalErr),
			}
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers.
	if headers, ok := req.Input["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, sOk := v.(string); sOk {
				httpReq.Header.Set(k, s)
			}
		}
	}

	// Set auth header.
	if authHeader, ok := req.Input["auth_header"].(string); ok && authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
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

	respHeaders := make(map[string]any, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) == 1 {
			respHeaders[k] = v[0]
		} else {
			respHeaders[k] = v
		}
	}

	return &connector.ExecuteResponse{
		Output: map[string]any{
			"status_code": resp.StatusCode,
			"headers":     respHeaders,
			"body":        parsedBody,
		},
		RawBody: respBody,
	}, nil
}
