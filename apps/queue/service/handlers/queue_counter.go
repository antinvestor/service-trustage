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

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// QueueCounterHandler handles queue counter HTTP endpoints.
type QueueCounterHandler struct {
	mgr business.QueueManager
}

// NewQueueCounterHandler creates a new QueueCounterHandler.
func NewQueueCounterHandler(mgr business.QueueManager) *QueueCounterHandler {
	return &QueueCounterHandler{mgr: mgr}
}

// Create handles POST /api/v1/queues/{queue_id}/counters.
func (h *QueueCounterHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	queueID := r.PathValue("queue_id")

	var req struct {
		Name       string          `json:"name"`
		Categories json.RawMessage `json:"categories"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	counter := &models.QueueCounter{
		QueueID: queueID,
		Name:    req.Name,
	}

	if req.Categories != nil {
		counter.Categories = string(req.Categories)
	}

	if err := h.mgr.CreateCounter(ctx, counter); err != nil {
		log.WithError(err).Error("failed to create counter", "queue_id", queueID)
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(counterToJSON(counter))
}

// List handles GET /api/v1/queues/{queue_id}/counters.
func (h *QueueCounterHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	queueID := r.PathValue("queue_id")

	counters, err := h.mgr.ListCounters(ctx, queueID)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	items := make([]map[string]any, 0, len(counters))
	for _, c := range counters {
		items = append(items, counterToJSON(c))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// Open handles POST /api/v1/counters/{id}/open.
func (h *QueueCounterHandler) Open(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	var req struct {
		StaffID string `json:"staff_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if err := h.mgr.OpenCounter(ctx, id, req.StaffID); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "open"})
}

// Close handles POST /api/v1/counters/{id}/close.
func (h *QueueCounterHandler) Close(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.CloseCounter(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "closed"})
}

// Pause handles POST /api/v1/counters/{id}/pause.
func (h *QueueCounterHandler) Pause(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.PauseCounter(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// CallNext handles POST /api/v1/counters/{id}/call-next.
func (h *QueueCounterHandler) CallNext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	item, err := h.mgr.CallNext(ctx, id)
	if err != nil {
		log.WithError(err).Error("failed to call next", "counter_id", id)
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(itemToJSON(item))
}

// BeginService handles POST /api/v1/counters/{id}/begin-service.
func (h *QueueCounterHandler) BeginService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.BeginService(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "service_started"})
}

// CompleteService handles POST /api/v1/counters/{id}/complete-service.
func (h *QueueCounterHandler) CompleteService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.CompleteService(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

func counterToJSON(counter *models.QueueCounter) map[string]any {
	result := map[string]any{
		"id":              counter.ID,
		"queue_id":        counter.QueueID,
		"name":            counter.Name,
		"status":          counter.Status,
		"current_item_id": counter.CurrentItemID,
		"served_by":       counter.ServedBy,
		"total_served":    counter.TotalServed,
		"created_at":      counter.CreatedAt,
		"modified_at":     counter.ModifiedAt,
	}

	if counter.Categories != "" {
		result["categories"] = json.RawMessage(counter.Categories)
	}

	return result
}
