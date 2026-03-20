package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

// InstanceHandler handles workflow instance endpoints.
type InstanceHandler struct {
	instanceRepo repository.WorkflowInstanceRepository
	execRepo     repository.WorkflowExecutionRepository
	runtimeRepo  repository.WorkflowRuntimeRepository
	auditRepo    repository.AuditEventRepository
	authz        authz.Middleware
}

// NewInstanceHandler creates a new InstanceHandler.
func NewInstanceHandler(
	instanceRepo repository.WorkflowInstanceRepository,
	execRepo repository.WorkflowExecutionRepository,
	auditRepo repository.AuditEventRepository,
	authzMiddleware authz.Middleware,
) *InstanceHandler {
	return &InstanceHandler{
		instanceRepo: instanceRepo,
		execRepo:     execRepo,
		runtimeRepo:  repository.NewWorkflowRuntimeRepository(execRepo.Pool()),
		auditRepo:    auditRepo,
		authz:        authzMiddleware,
	}
}

// List handles GET /api/v1/instances.
func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	if h.authz != nil {
		if err := h.authz.CanInstanceView(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	status := r.URL.Query().Get("status")
	workflowName := r.URL.Query().Get("workflow_name")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, err := h.instanceRepo.List(ctx, status, workflowName, limit)
	if err != nil {
		http.Error(w, "failed to list instances", http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(items))
	for _, inst := range items {
		resp = append(resp, map[string]any{
			"id":               inst.ID,
			"workflow_name":    inst.WorkflowName,
			"workflow_version": inst.WorkflowVersion,
			"current_state":    inst.CurrentState,
			"status":           inst.Status,
			"trigger_event_id": inst.TriggerEventID,
			"started_at":       inst.StartedAt,
			"finished_at":      inst.FinishedAt,
			"created_at":       inst.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": resp,
	})
}

// Retry handles POST /api/v1/instances/{id}/retry.
func (h *InstanceHandler) Retry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	if h.authz != nil {
		if err := h.authz.CanInstanceRetry(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	instanceID := r.PathValue("id")
	if instanceID == "" {
		http.Error(w, "instance id is required", http.StatusBadRequest)
		return
	}

	instance, err := h.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	exec, err := h.execRepo.GetLatestByInstance(ctx, instanceID)
	if err != nil {
		http.Error(w, "latest execution not found", http.StatusNotFound)
		return
	}

	newExec, retryErr := createRetryExecution(ctx, h.runtimeRepo, h.auditRepo, exec, instance)
	if retryErr != nil {
		log.WithError(retryErr).Error("retry instance failed")
		http.Error(w, retryErr.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"execution_id": newExec.ID,
		"status":       newExec.Status,
	})
}
