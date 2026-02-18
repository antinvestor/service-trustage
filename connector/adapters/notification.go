package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	notificationSendType        = "notification.send"
	notificationSendDisplayName = "Send Notification"
)

// NotificationSendAdapter sends a notification via an external notification service API.
// Single purpose: dispatch one notification (SMS, email, or push).
type NotificationSendAdapter struct {
	client *http.Client
}

// NewNotificationSendAdapter creates a new NotificationSendAdapter.
func NewNotificationSendAdapter(client *http.Client) *NotificationSendAdapter {
	return &NotificationSendAdapter{client: client}
}

func (a *NotificationSendAdapter) Type() string        { return notificationSendType }
func (a *NotificationSendAdapter) DisplayName() string { return notificationSendDisplayName }

func (a *NotificationSendAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["recipient", "channel", "body"],
		"properties": {
			"recipient": {"type": "string", "description": "Recipient address (phone, email, device token)"},
			"channel": {"type": "string", "enum": ["sms", "email", "push"], "description": "Delivery channel"},
			"subject": {"type": "string", "description": "Notification subject (required for email)"},
			"body": {"type": "string", "description": "Notification body text"},
			"template_id": {"type": "string", "description": "Optional template identifier"},
			"template_vars": {"type": "object", "description": "Variables for template rendering"}
		}
	}`)
}

func (a *NotificationSendAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["api_url"],
		"properties": {
			"api_url": {"type": "string", "format": "uri", "description": "Notification service API URL"}
		}
	}`)
}

func (a *NotificationSendAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"notification_id": {"type": "string"},
			"status": {"type": "string"},
			"channel": {"type": "string"}
		}
	}`)
}

func (a *NotificationSendAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["recipient"] == nil {
		return errors.New("recipient is required")
	}

	channel, _ := req.Input["channel"].(string)
	if channel == "" {
		return errors.New("channel is required")
	}

	if channel != "sms" && channel != "email" && channel != "push" {
		return fmt.Errorf("unsupported channel %q: must be sms, email, or push", channel)
	}

	if req.Input["body"] == nil {
		return errors.New("body is required")
	}

	if channel == "email" {
		if req.Input["subject"] == nil {
			return errors.New("subject is required for email channel")
		}
	}

	return nil
}

func (a *NotificationSendAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	parsed, execErr := executeAPIPost(ctx, a.client, req, a.buildPayload(req))
	if execErr != nil {
		return nil, execErr
	}

	output := map[string]any{
		"channel": req.Input["channel"],
		"status":  "sent",
	}

	if id, ok := parsed["id"]; ok {
		output["notification_id"] = id
	}

	if output["notification_id"] == nil {
		if id, ok := parsed["notification_id"]; ok {
			output["notification_id"] = id
		}
	}

	if status, ok := parsed["status"]; ok {
		output["status"] = status
	}

	return &connector.ExecuteResponse{Output: output}, nil
}

// buildPayload constructs the notification payload from the execute request.
func (a *NotificationSendAdapter) buildPayload(req *connector.ExecuteRequest) map[string]any {
	payload := map[string]any{
		"recipient": req.Input["recipient"],
		"channel":   req.Input["channel"],
		"body":      req.Input["body"],
	}

	if subject, ok := req.Input["subject"]; ok {
		payload["subject"] = subject
	}

	if templateID, ok := req.Input["template_id"]; ok {
		payload["template_id"] = templateID
	}

	if templateVars, ok := req.Input["template_vars"]; ok {
		payload["template_vars"] = templateVars
	}

	return payload
}
