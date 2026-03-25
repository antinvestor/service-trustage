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

// FormHandler handles form submission HTTP endpoints.
// Single purpose: accept form data and create workflow-triggering events.
type FormHandler struct {
	eventRepo   repository.EventLogRepository
	metrics     *telemetry.Metrics
	rateLimiter *RateLimiter
}

// NewFormHandler creates a new FormHandler.
func NewFormHandler(
	eventRepo repository.EventLogRepository,
	metrics *telemetry.Metrics,
	rateLimiter *RateLimiter,
) *FormHandler {
	return &FormHandler{
		eventRepo:   eventRepo,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// FormSubmitRequest is the request body for form submission.
type FormSubmitRequest struct {
	Fields         map[string]any `json:"fields"`
	SubmitterID    string         `json:"submitter_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
}

// SubmitForm handles POST /api/v1/forms/{form_id}/submit.
// It creates an event with type "form.submitted" that trigger bindings can match.
func (h *FormHandler) SubmitForm(w http.ResponseWriter, r *http.Request) {
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

	formID := r.PathValue("form_id")
	if formID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}

	var req FormSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Fields) == 0 {
		http.Error(w, "fields are required", http.StatusBadRequest)
		return
	}

	// Check idempotency.
	if req.IdempotencyKey != "" {
		existing, _ := h.eventRepo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
		if existing != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"submission_id": existing.ID,
				"form_id":       formID,
				"idempotent":    true,
			})

			return
		}
	}

	// Build event payload with form metadata.
	payload := map[string]any{
		"form_id": formID,
		"fields":  req.Fields,
	}

	if req.SubmitterID != "" {
		payload["submitter_id"] = req.SubmitterID
	}

	payloadBytes, _ := json.Marshal(payload)

	eventLog := &models.EventLog{
		EventType:      "form.submitted",
		Source:         "form:" + formID,
		IdempotencyKey: req.IdempotencyKey,
		Payload:        string(payloadBytes),
	}

	if err := h.eventRepo.Create(ctx, eventLog); err != nil {
		log.WithError(err).Error("failed to store form submission")
		http.Error(w, "failed to store form submission", http.StatusInternalServerError)

		return
	}

	log.Debug("form submitted",
		"submission_id", eventLog.ID,
		"form_id", formID,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"submission_id": eventLog.ID,
		"form_id":       formID,
	})
}
