package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// EventHandler handles event ingestion and timeline HTTP endpoints.
type EventHandler struct {
	eventRepo   repository.EventLogRepository
	auditRepo   repository.AuditEventRepository
	metrics     *telemetry.Metrics
	rateLimiter *RateLimiter
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(
	eventRepo repository.EventLogRepository,
	auditRepo repository.AuditEventRepository,
	metrics *telemetry.Metrics,
	rateLimiter *RateLimiter,
) *EventHandler {
	return &EventHandler{
		eventRepo:   eventRepo,
		auditRepo:   auditRepo,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// IngestEventRequest is the request body for event ingestion.
type IngestEventRequest struct {
	EventType      string         `json:"event_type"`
	Source         string         `json:"source"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
}

// IngestEvent handles POST /api/v1/events.
func (h *EventHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	// Rate limit per tenant.
	if h.rateLimiter != nil && !h.rateLimiter.Allow(ctx) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req IngestEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}

	// If an idempotency key is provided, check for duplicate.
	if req.IdempotencyKey != "" {
		existing, _ := h.eventRepo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
		if existing != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"event_id":   existing.ID,
				"idempotent": true,
			})

			return
		}
	}

	// Store in event log for outbox publishing.
	payloadBytes, _ := json.Marshal(req.Payload)

	eventLog := &models.EventLog{
		EventType:      req.EventType,
		Source:         req.Source,
		IdempotencyKey: req.IdempotencyKey,
		Payload:        string(payloadBytes),
	}

	if err := h.eventRepo.Create(ctx, eventLog); err != nil {
		log.WithError(err).Error("failed to store event", "event_type", req.EventType)
		http.Error(w, "failed to store event", http.StatusInternalServerError)

		return
	}

	log.Debug("event ingested", "event_id", eventLog.ID, "event_type", req.EventType)

	// Event is stored in the outbox table. The outbox scheduler will publish it
	// to NATS, where the event router worker will process it. This avoids
	// double-routing that would occur if we also routed synchronously here.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"event_id": eventLog.ID,
	})
}

// GetInstanceTimeline handles GET /api/v1/instances/{id}/timeline.
func (h *EventHandler) GetInstanceTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	instanceID := r.PathValue("id")

	auditEvents, err := h.auditRepo.ListByInstance(ctx, instanceID)
	if err != nil {
		http.Error(w, "failed to fetch timeline", http.StatusInternalServerError)
		return
	}

	h.writeTimeline(ctx, w, auditEvents)
}

func (h *EventHandler) writeTimeline(
	_ context.Context,
	w http.ResponseWriter,
	auditEvents []*models.WorkflowAuditEvent,
) {
	type timelineEntry struct {
		EventType string `json:"event_type"`
		State     string `json:"state,omitempty"`
		CreatedAt string `json:"created_at"`
	}

	entries := make([]timelineEntry, 0, len(auditEvents))
	for _, e := range auditEvents {
		entries = append(entries, timelineEntry{
			EventType: e.EventType,
			State:     e.State,
			CreatedAt: e.CreatedAt.String(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}
