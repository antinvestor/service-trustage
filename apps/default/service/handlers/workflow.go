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
	"fmt"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// WorkflowHandler handles workflow management HTTP endpoints.
// Uses plain HTTP until proto generation is set up.
type WorkflowHandler struct {
	workflowBiz business.WorkflowBusiness
	metrics     *telemetry.Metrics
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(
	biz business.WorkflowBusiness,
	metrics *telemetry.Metrics,
) *WorkflowHandler {
	return &WorkflowHandler{
		workflowBiz: biz,
		metrics:     metrics,
	}
}

// CreateWorkflow handles POST /api/v1/workflows.
func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, telemetry.SpanCreateWorkflow)
	defer func() { telemetry.EndSpan(span, nil) }()

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	def, err := h.workflowBiz.CreateWorkflow(ctx, body)
	if err != nil {
		log.WithError(err).Error("failed to create workflow")
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      def.ID,
		"name":    def.Name,
		"version": def.WorkflowVersion,
		"status":  def.Status,
	})
}

// GetWorkflow handles GET /api/v1/workflows/{id}.
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	def, schedules, err := h.workflowBiz.GetWorkflowWithSchedules(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)
		return
	}

	schedItems := make([]map[string]any, 0, len(schedules))
	for _, s := range schedules {
		item := map[string]any{
			"id":               s.ID,
			"name":             s.Name,
			"cron_expr":        s.CronExpr,
			"workflow_name":    s.WorkflowName,
			"workflow_version": s.WorkflowVersion,
			"active":           s.Active,
			"jitter_seconds":   s.JitterSeconds,
			"next_fire_at":     s.NextFireAt,
			"last_fired_at":    s.LastFiredAt,
		}
		schedItems = append(schedItems, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":        def.ID,
		"name":      def.Name,
		"version":   def.WorkflowVersion,
		"status":    def.Status,
		"dsl":       json.RawMessage(def.DSLBlob),
		"schedules": schedItems,
	})
}

// ListWorkflows handles GET /api/v1/workflows.
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" && status != string(models.WorkflowStatusActive) {
		http.Error(w, "unsupported status filter", http.StatusBadRequest)
		return
	}

	name := r.URL.Query().Get("name")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	defs, err := h.workflowBiz.ListWorkflows(ctx, name, limit)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)
		return
	}

	items := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		items = append(items, map[string]any{
			"id":      def.ID,
			"name":    def.Name,
			"version": def.WorkflowVersion,
			"status":  def.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// ActivateWorkflow handles POST /api/v1/workflows/{id}/activate.
func (h *WorkflowHandler) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.workflowBiz.ActivateWorkflow(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "active"})
}
