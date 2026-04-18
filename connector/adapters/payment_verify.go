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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	paymentVerifyType        = "payment.verify"
	paymentVerifyDisplayName = "Verify Payment"
)

// PaymentVerifyAdapter checks the status of a previously initiated payment.
// Single purpose: verify whether a payment completed, failed, or is still pending.
type PaymentVerifyAdapter struct {
	client *http.Client
}

// NewPaymentVerifyAdapter creates a new PaymentVerifyAdapter.
func NewPaymentVerifyAdapter(client *http.Client) *PaymentVerifyAdapter {
	return &PaymentVerifyAdapter{client: client}
}

func (a *PaymentVerifyAdapter) Type() string        { return paymentVerifyType }
func (a *PaymentVerifyAdapter) DisplayName() string { return paymentVerifyDisplayName }

func (a *PaymentVerifyAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["payment_id"],
		"properties": {
			"payment_id": {"type": "string", "description": "ID of the payment to verify"}
		}
	}`)
}

func (a *PaymentVerifyAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["api_url"],
		"properties": {
			"api_url": {"type": "string", "format": "uri", "description": "Payment service API URL"}
		}
	}`)
}

func (a *PaymentVerifyAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"payment_id": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "processing", "completed", "failed", "reversed"]},
			"amount": {"type": "number"},
			"currency": {"type": "string"},
			"completed_at": {"type": "string", "format": "date-time"},
			"error": {"type": "string"}
		}
	}`)
}

func (a *PaymentVerifyAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["payment_id"] == nil {
		return errors.New("payment_id is required")
	}

	return nil
}

func (a *PaymentVerifyAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	apiURL, _ := req.Config["api_url"].(string)
	if apiURL == "" {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "MISSING_CONFIG",
			Message: "api_url is required in config",
		}
	}

	paymentID, _ := req.Input["payment_id"].(string)
	verifyURL := apiURL + "/" + paymentID

	if err := validateExternalURL(ctx, verifyURL); err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "SSRF_BLOCKED",
			Message: fmt.Sprintf("URL not allowed: %v", err),
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, verifyURL, nil)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	httpReq.Header.Set("Accept", "application/json")

	if apiKey, ok := req.Credentials["api_key"]; ok {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
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

	if execErr := classifyHTTPStatus(resp.StatusCode, string(respBody)); execErr != nil {
		return nil, execErr
	}

	var parsed map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &parsed)
	}

	output := map[string]any{
		"payment_id": paymentID,
		"status":     "unknown",
	}

	for _, key := range []string{"status", "amount", "currency", "completed_at", "error"} {
		if v, ok := parsed[key]; ok {
			output[key] = v
		}
	}

	return &connector.ExecuteResponse{Output: output}, nil
}
