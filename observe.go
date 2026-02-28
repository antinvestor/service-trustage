//go:build ignore
// +build ignore

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
)

// ObserveHandler exposes observability and control endpoints for UI.
type ObserveHandler struct {
	obsBiz business.ObservabilityBusiness
	authz  authz.Middleware
}

// NewObserveHandler creates a new ObserveHandler.
func NewObserveHandler(obsBiz business.ObservabilityBusiness, authzMiddleware authz.Middleware) *ObserveHandler {
	return &ObserveHandler{obsBiz: obsBiz, authz: authzMiddleware}
}

// ListInstances handles GET /api/v1/instances.
func (h *ObserveHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanInstanceView(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	query := r.URL.Query()
	status := query.Get("status")
	workflowName := query.Get("workflow_name")
	limit := parseLimit(query.Get("limit"), 50, 200)
	cursor, err := parseTimeCursor(query.Get("cursor"))
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	items, listErr := h.obsBiz.ListInstances(ctx, tenantID, business.InstancesQuery{
		Status:        status,
		WorkflowName:  workflowName,
		Limit:         limit,
		CreatedBefore: cursor,
	})
	if listErr != nil {
		statusCode, msg := httpStatusForError(listErr)
		http.Error(w, msg, statusCode)
		return
	}

	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339)
	}

	type item struct {
		ID              string `json:"id"`
		WorkflowName    string `json:"workflow_name"`
		WorkflowVersion int    `json:"workflow_version"`
		CurrentState    string `json:"current_state"`
		Status          string `json:"status"`
		TriggerEventID  string `json:"trigger_event_id"`
		StartedAt       string `json:"started_at,omitempty"`
		FinishedAt      string `json:"finished_at,omitempty"`
		CreatedAt       string `json:"created_at"`
	}

	respItems := make([]item, 0, len(items))
	for _, inst := range items {
		respItems = append(respItems, item{
			ID:              inst.ID,
			WorkflowName:    inst.WorkflowName,
			WorkflowVersion: inst.WorkflowVersion,
			CurrentState:    inst.CurrentState,
			Status:          string(inst.Status),
			TriggerEventID:  inst.TriggerEventID,
			StartedAt:       formatTime(inst.StartedAt),
			FinishedAt:      formatTime(inst.FinishedAt),
			CreatedAt:       inst.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       respItems,
		"next_cursor": nextCursor,
	})
}

// GetInstance handles GET /api/v1/instances/{id}.
func (h *ObserveHandler) GetInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanInstanceView(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	instanceID := r.PathValue("id")
	inst, err := h.obsBiz.GetInstance(ctx, tenantID, instanceID)
	if err != nil {
		statusCode, msg := httpStatusForError(err)
		http.Error(w, msg, statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":               inst.ID,
		"workflow_name":    inst.WorkflowName,
		"workflow_version": inst.WorkflowVersion,
		"current_state":    inst.CurrentState,
		"status":           string(inst.Status),
		"trigger_event_id": inst.TriggerEventID,
		"started_at":       formatTime(inst.StartedAt),
		"finished_at":      formatTime(inst.FinishedAt),
		"created_at":       inst.CreatedAt.UTC().Format(time.RFC3339),
	})
}

// ListExecutions handles GET /api/v1/executions.
func (h *ObserveHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanExecutionView(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	query := r.URL.Query()
	status := query.Get("status")
	instanceID := query.Get("instance_id")
	limit := parseLimit(query.Get("limit"), 50, 200)
	cursor, err := parseTimeCursor(query.Get("cursor"))
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	items, listErr := h.obsBiz.ListExecutions(ctx, tenantID, business.ExecutionsQuery{
		InstanceID:    instanceID,
		Status:        status,
		Limit:         limit,
		CreatedBefore: cursor,
	})
	if listErr != nil {
		statusCode, msg := httpStatusForError(listErr)
		http.Error(w, msg, statusCode)
		return
	}

	var nextCursor string
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339)
	}

	type item struct {
		ExecutionID  string `json:"execution_id"`
		InstanceID   string `json:"instance_id"`
		State        string `json:"state"`
		Attempt      int    `json:"attempt"`
		Status       string `json:"status"`
		ErrorClass   string `json:"error_class,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		NextRetryAt  string `json:"next_retry_at,omitempty"`
		StartedAt    string `json:"started_at,omitempty"`
		FinishedAt   string `json:"finished_at,omitempty"`
		CreatedAt    string `json:"created_at"`
		TraceID      string `json:"trace_id,omitempty"`
	}

	respItems := make([]item, 0, len(items))
	for _, exec := range items {
		respItems = append(respItems, item{
			ExecutionID:  exec.ExecutionID,
			InstanceID:   exec.InstanceID,
			State:        exec.State,
			Attempt:      exec.Attempt,
			Status:       string(exec.Status),
			ErrorClass:   exec.ErrorClass,
			ErrorMessage: exec.ErrorMessage,
			NextRetryAt:  formatTime(exec.NextRetryAt),
			StartedAt:    formatTime(exec.StartedAt),
			FinishedAt:   formatTime(exec.FinishedAt),
			CreatedAt:    exec.CreatedAt.UTC().Format(time.RFC3339),
			TraceID:      exec.TraceID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":       respItems,
		"next_cursor": nextCursor,
	})
}

// ListInstanceExecutions handles GET /api/v1/instances/{id}/executions.
func (h *ObserveHandler) ListInstanceExecutions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := query
	q.Set("instance_id", r.PathValue("id"))
	r.URL.RawQuery = q.Encode()
	h.ListExecutions(w, r)
}

// GetExecution handles GET /api/v1/executions/{id}.
func (h *ObserveHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanExecutionView(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	executionID := r.PathValue("id")
	includeOutput := strings.EqualFold(r.URL.Query().Get("include_output"), "true")

	detail, err := h.obsBiz.GetExecution(ctx, tenantID, executionID, includeOutput)
	if err != nil {
		statusCode, msg := httpStatusForError(err)
		http.Error(w, msg, statusCode)
		return
	}

	exec := detail.Execution
	response := map[string]any{
		"execution_id":  exec.ExecutionID,
		"instance_id":   exec.InstanceID,
		"state":         exec.State,
		"attempt":       exec.Attempt,
		"status":        string(exec.Status),
		"error_class":   exec.ErrorClass,
		"error_message": exec.ErrorMessage,
		"next_retry_at": formatTime(exec.NextRetryAt),
		"started_at":    formatTime(exec.StartedAt),
		"finished_at":   formatTime(exec.FinishedAt),
		"created_at":    exec.CreatedAt.UTC().Format(time.RFC3339),
		"trace_id":      exec.TraceID,
	}

	if exec.InputPayload != "" {
		response["input_payload"] = json.RawMessage(exec.InputPayload)
	}

	if detail.Output != nil {
		response["output"] = json.RawMessage(detail.Output.Payload)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// RetryExecution handles POST /api/v1/executions/{id}/retry.
func (h *ObserveHandler) RetryExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanExecutionRetry(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	executionID := r.PathValue("id")
	newExec, err := h.obsBiz.RetryExecution(ctx, tenantID, executionID)
	if err != nil {
		statusCode, msg := httpStatusForError(err)
		http.Error(w, msg, statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"execution_id": newExec.ExecutionID,
		"status":       string(newExec.Status),
	})
}

// RetryInstance handles POST /api/v1/instances/{id}/retry.
func (h *ObserveHandler) RetryInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, _, ok := requireTenant(ctx, w)
	if !ok {
		return
	}

	subject := GetSubject(ctx)
	if subject == "" {
		http.Error(w, "missing subject", http.StatusUnauthorized)
		return
	}

	if h.authz != nil {
		if err := h.authz.CanInstanceRetry(ctx, subject, tenantID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	instanceID := r.PathValue("id")
	newExec, err := h.obsBiz.RetryInstanceLastFailure(ctx, tenantID, instanceID)
	if err != nil {
		statusCode, msg := httpStatusForError(err)
		http.Error(w, msg, statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"execution_id": newExec.ExecutionID,
		"status":       string(newExec.Status),
	})
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
