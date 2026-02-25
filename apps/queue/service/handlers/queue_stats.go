package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
)

// QueueStatsHandler handles queue statistics HTTP endpoints.
type QueueStatsHandler struct {
	stats business.QueueStatsService
}

// NewQueueStatsHandler creates a new QueueStatsHandler.
func NewQueueStatsHandler(stats business.QueueStatsService) *QueueStatsHandler {
	return &QueueStatsHandler{stats: stats}
}

// GetStats handles GET /api/v1/queues/{queue_id}/stats.
func (h *QueueStatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	queueID := r.PathValue("queue_id")

	stats, err := h.stats.GetStats(ctx, queueID)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
