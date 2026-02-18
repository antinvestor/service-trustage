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
	notificationStatusType        = "notification.status"
	notificationStatusDisplayName = "Check Notification Status"
)

// NotificationStatusAdapter checks the delivery status of a previously sent notification.
// Single purpose: poll the notification service for delivery state.
type NotificationStatusAdapter struct {
	client *http.Client
}

// NewNotificationStatusAdapter creates a new NotificationStatusAdapter.
func NewNotificationStatusAdapter(client *http.Client) *NotificationStatusAdapter {
	return &NotificationStatusAdapter{client: client}
}

func (a *NotificationStatusAdapter) Type() string        { return notificationStatusType }
func (a *NotificationStatusAdapter) DisplayName() string { return notificationStatusDisplayName }

func (a *NotificationStatusAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["notification_id"],
		"properties": {
			"notification_id": {"type": "string", "description": "ID of the notification to check"}
		}
	}`)
}

func (a *NotificationStatusAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["api_url"],
		"properties": {
			"api_url": {"type": "string", "format": "uri", "description": "Notification service API URL"}
		}
	}`)
}

func (a *NotificationStatusAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"notification_id": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "sent", "delivered", "failed", "bounced"]},
			"delivered_at": {"type": "string", "format": "date-time"},
			"error": {"type": "string"}
		}
	}`)
}

func (a *NotificationStatusAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["notification_id"] == nil {
		return errors.New("notification_id is required")
	}

	return nil
}

func (a *NotificationStatusAdapter) Execute(
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

	notificationID, _ := req.Input["notification_id"].(string)
	statusURL := apiURL + "/" + notificationID

	if err := validateExternalURL(ctx, statusURL); err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "SSRF_BLOCKED",
			Message: fmt.Sprintf("URL not allowed: %v", err),
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
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
		"notification_id": notificationID,
		"status":          "unknown",
	}

	if status, ok := parsed["status"]; ok {
		output["status"] = status
	}

	if deliveredAt, ok := parsed["delivered_at"]; ok {
		output["delivered_at"] = deliveredAt
	}

	if errMsg, ok := parsed["error"]; ok {
		output["error"] = errMsg
	}

	return &connector.ExecuteResponse{Output: output}, nil
}
