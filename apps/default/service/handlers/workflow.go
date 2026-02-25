package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// WorkflowHandler handles workflow management HTTP endpoints.
// Uses plain HTTP until proto generation is set up.
type WorkflowHandler struct {
	workflowBiz business.WorkflowBusiness
	authz       authz.Middleware
	metrics     *telemetry.Metrics
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(biz business.WorkflowBusiness, authzMiddleware authz.Middleware, metrics *telemetry.Metrics) *WorkflowHandler {
	return &WorkflowHandler{
		workflowBiz: biz,
		authz:       authzMiddleware,
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

	if h.authz != nil {
		if err := h.authz.CanManageWorkflow(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
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

	if h.authz != nil {
		if err := h.authz.CanViewWorkflow(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	id := r.PathValue("id")

	def, err := h.workflowBiz.GetWorkflow(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      def.ID,
		"name":    def.Name,
		"version": def.WorkflowVersion,
		"status":  def.Status,
		"dsl":     json.RawMessage(def.DSLBlob),
	})
}

// ListWorkflows handles GET /api/v1/workflows.
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	if h.authz != nil {
		if err := h.authz.CanViewWorkflow(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
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

	if h.authz != nil {
		if err := h.authz.CanManageWorkflow(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
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
