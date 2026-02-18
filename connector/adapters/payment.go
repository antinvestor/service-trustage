package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	paymentInitiateType        = "payment.initiate"
	paymentInitiateDisplayName = "Initiate Payment"
)

// PaymentInitiateAdapter initiates a payment via an external payment service API.
// Single purpose: start one payment transaction.
type PaymentInitiateAdapter struct {
	client *http.Client
}

// NewPaymentInitiateAdapter creates a new PaymentInitiateAdapter.
func NewPaymentInitiateAdapter(client *http.Client) *PaymentInitiateAdapter {
	return &PaymentInitiateAdapter{client: client}
}

func (a *PaymentInitiateAdapter) Type() string        { return paymentInitiateType }
func (a *PaymentInitiateAdapter) DisplayName() string { return paymentInitiateDisplayName }

func (a *PaymentInitiateAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["amount", "currency", "recipient", "reference"],
		"properties": {
			"amount": {"type": "number", "minimum": 0, "description": "Payment amount"},
			"currency": {"type": "string", "minLength": 3, "maxLength": 3, "description": "ISO 4217 currency code"},
			"recipient": {"type": "string", "description": "Recipient identifier (phone, account number, etc.)"},
			"description": {"type": "string", "description": "Payment description"},
			"reference": {"type": "string", "description": "Unique payment reference"},
			"method": {"type": "string", "enum": ["mobile_money", "bank_transfer", "card"], "description": "Payment method"}
		}
	}`)
}

func (a *PaymentInitiateAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["api_url"],
		"properties": {
			"api_url": {"type": "string", "format": "uri", "description": "Payment service API URL"}
		}
	}`)
}

func (a *PaymentInitiateAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"payment_id": {"type": "string"},
			"status": {"type": "string"},
			"reference": {"type": "string"}
		}
	}`)
}

func (a *PaymentInitiateAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["amount"] == nil {
		return errors.New("amount is required")
	}

	if req.Input["currency"] == nil {
		return errors.New("currency is required")
	}

	if req.Input["recipient"] == nil {
		return errors.New("recipient is required")
	}

	if req.Input["reference"] == nil {
		return errors.New("reference is required")
	}

	return nil
}

func (a *PaymentInitiateAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	parsed, execErr := executeAPIPost(ctx, a.client, req, a.buildPayload(req))
	if execErr != nil {
		return nil, execErr
	}

	output := map[string]any{
		"status":    "initiated",
		"reference": req.Input["reference"],
	}

	if id, ok := parsed["id"]; ok {
		output["payment_id"] = id
	}

	if output["payment_id"] == nil {
		if id, ok := parsed["payment_id"]; ok {
			output["payment_id"] = id
		}
	}

	if status, ok := parsed["status"]; ok {
		output["status"] = status
	}

	return &connector.ExecuteResponse{Output: output}, nil
}

// buildPayload constructs the payment payload from the execute request.
func (a *PaymentInitiateAdapter) buildPayload(req *connector.ExecuteRequest) map[string]any {
	payload := map[string]any{
		"amount":    req.Input["amount"],
		"currency":  req.Input["currency"],
		"recipient": req.Input["recipient"],
		"reference": req.Input["reference"],
	}

	if description, ok := req.Input["description"]; ok {
		payload["description"] = description
	}

	if method, ok := req.Input["method"]; ok {
		payload["method"] = method
	}

	return payload
}
