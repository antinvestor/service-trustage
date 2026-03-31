package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// WebhookReceiveHandler handles inbound webhooks that trigger workflows.
// Single purpose: accept external webhook payloads and create workflow-triggering events.
type WebhookReceiveHandler struct {
	eventRepo   repository.EventLogRepository
	metrics     *telemetry.Metrics
	rateLimiter *RateLimiter
}

// NewWebhookReceiveHandler creates a new WebhookReceiveHandler.
func NewWebhookReceiveHandler(
	eventRepo repository.EventLogRepository,
	metrics *telemetry.Metrics,
	rateLimiter *RateLimiter,
) *WebhookReceiveHandler {
	return &WebhookReceiveHandler{
		eventRepo:   eventRepo,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// ReceiveWebhook handles POST /api/v1/webhooks/{webhook_id}.
// Creates an event with type "webhook.received" that trigger bindings can match.
func (h *WebhookReceiveHandler) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
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

	webhookID := r.PathValue("webhook_id")
	if webhookID == "" {
		http.Error(w, "webhook_id is required", http.StatusBadRequest)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Build event payload with webhook metadata.
	payload := map[string]any{
		"webhook_id": webhookID,
		"body":       body,
		"headers":    extractHeaders(r),
	}

	payloadBytes, _ := json.Marshal(payload)

	// Use webhook_id + content hash for idempotency.
	idempotencyKey := r.Header.Get("Idempotency-Key")

	eventLog := &models.EventLog{
		EventType:      "webhook.received",
		Source:         "webhook:" + webhookID,
		IdempotencyKey: idempotencyKey,
		Payload:        string(payloadBytes),
	}

	if err := h.eventRepo.Create(ctx, eventLog); err != nil {
		log.WithError(err).Error("failed to store webhook event", "webhook_id", webhookID)
		http.Error(w, "failed to process webhook", http.StatusInternalServerError)

		return
	}

	log.Debug("webhook received",
		"event_id", eventLog.ID,
		"webhook_id", webhookID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"event_id":   eventLog.ID,
		"webhook_id": webhookID,
	})
}

// extractHeaders captures select safe headers from the request for context.
func extractHeaders(r *http.Request) map[string]string {
	safeHeaders := []string{
		"Content-Type",
		"User-Agent",
		"X-Request-ID",
		"X-Webhook-Signature",
		"X-Webhook-Event",
	}

	headers := make(map[string]string)

	for _, h := range safeHeaders {
		if v := r.Header.Get(h); v != "" {
			headers[h] = v
		}
	}

	return headers
}
