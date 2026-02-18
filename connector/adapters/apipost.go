package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

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
