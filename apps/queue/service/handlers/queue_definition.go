package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/queue/service/authz"
	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// QueueDefinitionHandler handles queue definition HTTP endpoints.
type QueueDefinitionHandler struct {
	mgr   business.QueueManager
	authz authz.Middleware
}

const (
	defaultPriorityLevels = 3
	defaultSLAMinutes     = 30
)

// NewQueueDefinitionHandler creates a new QueueDefinitionHandler.
func NewQueueDefinitionHandler(mgr business.QueueManager, authz authz.Middleware) *QueueDefinitionHandler {
	return &QueueDefinitionHandler{mgr: mgr, authz: authz}
}

// Create handles POST /api/v1/queues.
func (h *QueueDefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	if err := h.authz.CanQueueManage(ctx); err != nil {
		writeAuthzError(w, err)
		return
	}

	var req struct {
		Name           string          `json:"name"`
		Description    string          `json:"description"`
		PriorityLevels *int            `json:"priority_levels"`
		MaxCapacity    *int            `json:"max_capacity"`
		SLAMinutes     *int            `json:"sla_minutes"`
		Config         json.RawMessage `json:"config"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	def := &models.QueueDefinition{
		Name:           req.Name,
		Description:    req.Description,
		Active:         true,
		PriorityLevels: defaultPriorityLevels,
		SLAMinutes:     defaultSLAMinutes,
	}

	if req.PriorityLevels != nil {
		def.PriorityLevels = *req.PriorityLevels
	}

	if req.MaxCapacity != nil {
		def.MaxCapacity = *req.MaxCapacity
	}

	if req.SLAMinutes != nil {
		def.SLAMinutes = *req.SLAMinutes
	}

	if req.Config != nil {
		def.Config = string(req.Config)
	}

	if err := h.mgr.CreateQueue(ctx, def); err != nil {
		log.WithError(err).Error("failed to create queue")
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(queueDefToJSON(def))
}

// List handles GET /api/v1/queues.
func (h *QueueDefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	if err := h.authz.CanQueueView(ctx); err != nil {
		writeAuthzError(w, err)
		return
	}

	activeOnly := r.URL.Query().Get("active") == "true"

	defs, err := h.mgr.ListQueues(ctx, activeOnly)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	items := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		items = append(items, queueDefToJSON(d))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// Get handles GET /api/v1/queues/{id}.
func (h *QueueDefinitionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	if err := h.authz.CanQueueView(ctx); err != nil {
		writeAuthzError(w, err)
		return
	}

	id := r.PathValue("id")

	def, err := h.mgr.GetQueue(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(queueDefToJSON(def))
}

// Update handles PUT /api/v1/queues/{id}.
func (h *QueueDefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	if err := h.authz.CanQueueManage(ctx); err != nil {
		writeAuthzError(w, err)
		return
	}

	id := r.PathValue("id")

	def, err := h.mgr.GetQueue(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	var req struct {
		Name           *string         `json:"name"`
		Description    *string         `json:"description"`
		Active         *bool           `json:"active"`
		PriorityLevels *int            `json:"priority_levels"`
		MaxCapacity    *int            `json:"max_capacity"`
		SLAMinutes     *int            `json:"sla_minutes"`
		Config         json.RawMessage `json:"config"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		def.Name = *req.Name
	}

	if req.Description != nil {
		def.Description = *req.Description
	}

	if req.Active != nil {
		def.Active = *req.Active
	}

	if req.PriorityLevels != nil {
		def.PriorityLevels = *req.PriorityLevels
	}

	if req.MaxCapacity != nil {
		def.MaxCapacity = *req.MaxCapacity
	}

	if req.SLAMinutes != nil {
		def.SLAMinutes = *req.SLAMinutes
	}

	if req.Config != nil {
		def.Config = string(req.Config)
	}

	if updateErr := h.mgr.UpdateQueue(ctx, def); updateErr != nil {
		log.WithError(updateErr).Error("failed to update queue")
		status, msg := httpStatusForError(updateErr)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(queueDefToJSON(def))
}

// Delete handles DELETE /api/v1/queues/{id}.
func (h *QueueDefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	if err := h.authz.CanQueueManage(ctx); err != nil {
		writeAuthzError(w, err)
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.DeleteQueue(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func queueDefToJSON(def *models.QueueDefinition) map[string]any {
	result := map[string]any{
		"id":              def.ID,
		"name":            def.Name,
		"description":     def.Description,
		"active":          def.Active,
		"priority_levels": def.PriorityLevels,
		"max_capacity":    def.MaxCapacity,
		"sla_minutes":     def.SLAMinutes,
		"created_at":      def.CreatedAt,
		"modified_at":     def.ModifiedAt,
	}

	if def.Config != "" {
		result["config"] = json.RawMessage(def.Config)
	}

	return result
}
