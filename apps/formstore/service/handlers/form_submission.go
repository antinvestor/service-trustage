package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// FormSubmissionHandler handles form submission HTTP endpoints.
type FormSubmissionHandler struct {
	biz     business.FormStoreBusiness
	limiter *RateLimiter
}

// NewFormSubmissionHandler creates a new FormSubmissionHandler.
func NewFormSubmissionHandler(
	biz business.FormStoreBusiness,
	limiter *RateLimiter,
) *FormSubmissionHandler {
	return &FormSubmissionHandler{biz: biz, limiter: limiter}
}

// Submit handles POST /api/v1/forms/{form_id}/submissions.
func (h *FormSubmissionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	if h.limiter != nil && !h.limiter.Allow(ctx) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	formID := r.PathValue("form_id")

	var req struct {
		SubmitterID    string          `json:"submitter_id"`
		Data           json.RawMessage `json:"data"`
		IdempotencyKey string          `json:"idempotency_key"`
		Metadata       json.RawMessage `json:"metadata"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if len(req.Data) == 0 {
		http.Error(w, "data field is required", http.StatusBadRequest)
		return
	}

	sub := &models.FormSubmission{
		FormID:         formID,
		SubmitterID:    req.SubmitterID,
		Status:         models.SubmissionStatusPending,
		Data:           string(req.Data),
		IdempotencyKey: req.IdempotencyKey,
	}

	if req.Metadata != nil {
		sub.Metadata = string(req.Metadata)
	}

	if err := h.biz.CreateSubmission(ctx, sub); err != nil {
		log.WithError(err).Error("failed to create form submission")
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(submissionToJSON(sub))
}

// ListByForm handles GET /api/v1/forms/{form_id}/submissions.
func (h *FormSubmissionHandler) ListByForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	formID := r.PathValue("form_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	subs, err := h.biz.ListSubmissions(ctx, formID, limit, offset)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	items := make([]map[string]any, 0, len(subs))
	for _, s := range subs {
		items = append(items, submissionToJSON(s))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// Get handles GET /api/v1/submissions/{id}.
func (h *FormSubmissionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	sub, err := h.biz.GetSubmission(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submissionToJSON(sub))
}

// Update handles PUT /api/v1/submissions/{id}.
func (h *FormSubmissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	sub, err := h.biz.GetSubmission(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	var req struct {
		Data     json.RawMessage `json:"data"`
		Status   *string         `json:"status"`
		Metadata json.RawMessage `json:"metadata"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.Data != nil {
		sub.Data = string(req.Data)
	}

	if req.Status != nil {
		sub.Status = models.FormSubmissionStatus(*req.Status)
	}

	if req.Metadata != nil {
		sub.Metadata = string(req.Metadata)
	}

	if updateErr := h.biz.UpdateSubmission(ctx, sub); updateErr != nil {
		log.WithError(updateErr).Error("failed to update form submission")
		status, msg := httpStatusForError(updateErr)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(submissionToJSON(sub))
}

// Delete handles DELETE /api/v1/submissions/{id}.
func (h *FormSubmissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.biz.DeleteSubmission(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func submissionToJSON(sub *models.FormSubmission) map[string]any {
	result := map[string]any{
		"id":              sub.ID,
		"form_id":         sub.FormID,
		"submitter_id":    sub.SubmitterID,
		"status":          sub.Status,
		"data":            json.RawMessage(sub.Data),
		"file_count":      sub.FileCount,
		"idempotency_key": sub.IdempotencyKey,
		"created_at":      sub.CreatedAt,
		"modified_at":     sub.ModifiedAt,
	}

	if sub.Metadata != "" {
		result["metadata"] = json.RawMessage(sub.Metadata)
	}

	return result
}
