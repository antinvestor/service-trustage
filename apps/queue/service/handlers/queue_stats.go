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
