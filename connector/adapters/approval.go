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
	"net/http"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	approvalRequestType        = "approval.request"
	approvalRequestDisplayName = "Request Approval"
)

// ApprovalRequestAdapter sends a human approval request via an external notification service.
// Single purpose: dispatch an approval request. Pair with a signal_wait step to await the response.
type ApprovalRequestAdapter struct {
	client *http.Client
}

// NewApprovalRequestAdapter creates a new ApprovalRequestAdapter.
func NewApprovalRequestAdapter(client *http.Client) *ApprovalRequestAdapter {
	return &ApprovalRequestAdapter{client: client}
}

func (a *ApprovalRequestAdapter) Type() string        { return approvalRequestType }
func (a *ApprovalRequestAdapter) DisplayName() string { return approvalRequestDisplayName }

func (a *ApprovalRequestAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["approver", "title"],
		"properties": {
			"approver": {"type": "string", "description": "Approver identifier (email, user ID, phone)"},
			"title": {"type": "string", "description": "Approval request title"},
			"description": {"type": "string", "description": "Detailed description of what needs approval"},
			"options": {
				"type": "array",
				"items": {"type": "string"},
				"default": ["approve", "reject"],
				"description": "Available response options"
			},
			"callback_url": {"type": "string", "format": "uri", "description": "URL for the approver to respond to"},
			"expires_in": {"type": "string", "description": "Expiration duration (e.g. 24h, 7d)"}
		}
	}`)
}

func (a *ApprovalRequestAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["api_url"],
		"properties": {
			"api_url": {"type": "string", "format": "uri", "description": "Approval/notification service API URL"}
		}
	}`)
}

func (a *ApprovalRequestAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"request_id": {"type": "string"},
			"status": {"type": "string", "enum": ["pending", "sent", "failed"]},
			"approver": {"type": "string"}
		}
	}`)
}

func (a *ApprovalRequestAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["approver"] == nil {
		return errors.New("approver is required")
	}

	if req.Input["title"] == nil {
		return errors.New("title is required")
	}

	return nil
}

func (a *ApprovalRequestAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	parsed, execErr := executeAPIPost(ctx, a.client, req, a.buildPayload(req))
	if execErr != nil {
		return nil, execErr
	}

	output := map[string]any{
		"status":   "pending",
		"approver": req.Input["approver"],
	}

	if id, ok := parsed["id"]; ok {
		output["request_id"] = id
	}

	if output["request_id"] == nil {
		if id, ok := parsed["request_id"]; ok {
			output["request_id"] = id
		}
	}

	if status, ok := parsed["status"]; ok {
		output["status"] = status
	}

	return &connector.ExecuteResponse{Output: output}, nil
}

// buildPayload constructs the approval request payload from the execute request.
func (a *ApprovalRequestAdapter) buildPayload(req *connector.ExecuteRequest) map[string]any {
	payload := map[string]any{
		"type":     "approval",
		"approver": req.Input["approver"],
		"title":    req.Input["title"],
	}

	if description, ok := req.Input["description"]; ok {
		payload["description"] = description
	}

	options, _ := req.Input["options"].([]any)
	if len(options) == 0 {
		payload["options"] = []string{"approve", "reject"}
	} else {
		payload["options"] = options
	}

	if callbackURL, ok := req.Input["callback_url"]; ok {
		payload["callback_url"] = callbackURL
	}

	if expiresIn, ok := req.Input["expires_in"]; ok {
		payload["expires_in"] = expiresIn
	}

	// Include execution metadata so the approval response can signal back.
	if execID, ok := req.Metadata["execution_id"]; ok {
		payload["execution_id"] = execID
	}

	if instanceID, ok := req.Metadata["instance_id"]; ok {
		payload["instance_id"] = instanceID
	}

	return payload
}
