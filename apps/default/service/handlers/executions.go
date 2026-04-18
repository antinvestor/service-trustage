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

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

// ExecutionHandler handles execution endpoints.
type ExecutionHandler struct {
	execRepo     repository.WorkflowExecutionRepository
	runtimeRepo  repository.WorkflowRuntimeRepository
	instanceRepo repository.WorkflowInstanceRepository
	outputRepo   repository.WorkflowOutputRepository
	auditRepo    repository.AuditEventRepository
}

// NewExecutionHandler creates a new ExecutionHandler.
func NewExecutionHandler(
	execRepo repository.WorkflowExecutionRepository,
	instanceRepo repository.WorkflowInstanceRepository,
	outputRepo repository.WorkflowOutputRepository,
	auditRepo repository.AuditEventRepository,
) *ExecutionHandler {
	return &ExecutionHandler{
		execRepo:     execRepo,
		runtimeRepo:  repository.NewWorkflowRuntimeRepository(execRepo.Pool()),
		instanceRepo: instanceRepo,
		outputRepo:   outputRepo,
		auditRepo:    auditRepo,
	}
}

// List handles GET /api/v1/executions.
func (h *ExecutionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	status := r.URL.Query().Get("status")
	instanceID := r.URL.Query().Get("instance_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, err := h.execRepo.List(ctx, status, instanceID, limit)
	if err != nil {
		http.Error(w, "failed to list executions", http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]any, 0, len(items))
	for _, exec := range items {
		resp = append(resp, map[string]any{
			"execution_id":      exec.ID,
			"instance_id":       exec.InstanceID,
			"state":             exec.State,
			"attempt":           exec.Attempt,
			"status":            exec.Status,
			"error_class":       exec.ErrorClass,
			"error_message":     exec.ErrorMessage,
			"next_retry_at":     exec.NextRetryAt,
			"started_at":        exec.StartedAt,
			"finished_at":       exec.FinishedAt,
			"created_at":        exec.CreatedAt,
			"trace_id":          exec.TraceID,
			"input_schema_hash": exec.InputSchemaHash,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": resp,
	})
}

// Get handles GET /api/v1/executions/{id}.
func (h *ExecutionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	executionID := r.PathValue("id")
	if executionID == "" {
		http.Error(w, "execution id is required", http.StatusBadRequest)
		return
	}

	exec, err := h.execRepo.GetByID(ctx, executionID)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	includeOutput := r.URL.Query().Get("include_output") == "true"

	var output any
	if includeOutput {
		if out, outErr := h.outputRepo.GetByExecution(ctx, executionID); outErr == nil && out != nil {
			if unmarshalErr := json.Unmarshal([]byte(out.Payload), &output); unmarshalErr != nil {
				output = out.Payload
			}
		}
	}

	var inputPayload any
	if exec.InputPayload != "" {
		if unmarshalErr := json.Unmarshal([]byte(exec.InputPayload), &inputPayload); unmarshalErr != nil {
			inputPayload = exec.InputPayload
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"execution_id":      exec.ID,
		"instance_id":       exec.InstanceID,
		"state":             exec.State,
		"attempt":           exec.Attempt,
		"status":            exec.Status,
		"error_class":       exec.ErrorClass,
		"error_message":     exec.ErrorMessage,
		"next_retry_at":     exec.NextRetryAt,
		"started_at":        exec.StartedAt,
		"finished_at":       exec.FinishedAt,
		"created_at":        exec.CreatedAt,
		"trace_id":          exec.TraceID,
		"input_payload":     inputPayload,
		"input_schema_hash": exec.InputSchemaHash,
		"output":            output,
	})
}

// Retry handles POST /api/v1/executions/{id}/retry.
func (h *ExecutionHandler) Retry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	executionID := r.PathValue("id")
	if executionID == "" {
		http.Error(w, "execution id is required", http.StatusBadRequest)
		return
	}

	exec, err := h.execRepo.GetByID(ctx, executionID)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	instance, err := h.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		http.Error(w, "instance not found", http.StatusNotFound)
		return
	}

	newExec, retryErr := createRetryExecution(ctx, h.runtimeRepo, h.auditRepo, exec, instance)
	if retryErr != nil {
		log.WithError(retryErr).Error("retry execution failed", "execution_id", executionID)
		http.Error(w, retryErr.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"execution_id": newExec.ID,
		"status":       newExec.Status,
	})
}
