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

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// QueueItemHandler handles queue item HTTP endpoints.
type QueueItemHandler struct {
	mgr     business.QueueManager
	limiter *RateLimiter
}

// NewQueueItemHandler creates a new QueueItemHandler.
func NewQueueItemHandler(mgr business.QueueManager, limiter *RateLimiter) *QueueItemHandler {
	return &QueueItemHandler{mgr: mgr, limiter: limiter}
}

// Enqueue handles POST /api/v1/queues/{queue_id}/items.
func (h *QueueItemHandler) Enqueue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	if h.limiter != nil && !h.limiter.Allow(ctx) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	queueID := r.PathValue("queue_id")

	var req struct {
		Priority   *int            `json:"priority"`
		Category   string          `json:"category"`
		CustomerID string          `json:"customer_id"`
		Metadata   json.RawMessage `json:"metadata"`
		TicketNo   string          `json:"ticket_no"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	item := &models.QueueItem{
		QueueID:    queueID,
		Priority:   1,
		Category:   req.Category,
		CustomerID: req.CustomerID,
		TicketNo:   req.TicketNo,
	}

	if req.Priority != nil {
		item.Priority = *req.Priority
	}

	if req.Metadata != nil {
		item.Metadata = string(req.Metadata)
	}

	if err := h.mgr.Enqueue(ctx, item); err != nil {
		log.WithError(err).Error("failed to enqueue item", "queue_id", queueID)
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(itemToJSON(item))
}

// ListWaiting handles GET /api/v1/queues/{queue_id}/items.
func (h *QueueItemHandler) ListWaiting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	queueID := r.PathValue("queue_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, err := h.mgr.ListWaitingItems(ctx, queueID, limit, offset)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		results = append(results, itemToJSON(item))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": results})
}

// Get handles GET /api/v1/items/{id}.
func (h *QueueItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	item, err := h.mgr.GetItem(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(itemToJSON(item))
}

// GetPosition handles GET /api/v1/items/{id}/position.
func (h *QueueItemHandler) GetPosition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	position, err := h.mgr.GetItemPosition(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"position": position})
}

// Cancel handles POST /api/v1/items/{id}/cancel.
func (h *QueueItemHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.CancelItem(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// NoShow handles POST /api/v1/items/{id}/no-show.
func (h *QueueItemHandler) NoShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.NoShowItem(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "no_show"})
}

// Requeue handles POST /api/v1/items/{id}/requeue.
func (h *QueueItemHandler) Requeue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.mgr.RequeueItem(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "waiting"})
}

// Transfer handles POST /api/v1/items/{id}/transfer.
func (h *QueueItemHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	var req struct {
		QueueID string `json:"queue_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.QueueID == "" {
		http.Error(w, "queue_id is required", http.StatusBadRequest)
		return
	}

	if err := h.mgr.TransferItem(ctx, id, req.QueueID); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "transferred"})
}

func itemToJSON(item *models.QueueItem) map[string]any {
	result := map[string]any{
		"id":          item.ID,
		"queue_id":    item.QueueID,
		"priority":    item.Priority,
		"status":      item.Status,
		"ticket_no":   item.TicketNo,
		"category":    item.Category,
		"customer_id": item.CustomerID,
		"counter_id":  item.CounterID,
		"served_by":   item.ServedBy,
		"joined_at":   item.JoinedAt,
		"created_at":  item.CreatedAt,
	}

	if item.CalledAt != nil {
		result["called_at"] = item.CalledAt
	}

	if item.ServiceStart != nil {
		result["service_start"] = item.ServiceStart
	}

	if item.ServiceEnd != nil {
		result["service_end"] = item.ServiceEnd
	}

	if item.Metadata != "" {
		result["metadata"] = json.RawMessage(item.Metadata)
	}

	return result
}
