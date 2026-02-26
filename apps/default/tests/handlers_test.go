package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

type allowAllAuthz struct{}

func (a allowAllAuthz) CanIngestEvent(_ context.Context) error  { return nil }
func (a allowAllAuthz) CanManageWorkflow(_ context.Context) error { return nil }
func (a allowAllAuthz) CanViewWorkflow(_ context.Context) error   { return nil }
func (a allowAllAuthz) CanViewInstance(_ context.Context) error   { return nil }
func (a allowAllAuthz) CanRetryInstance(_ context.Context) error  { return nil }
func (a allowAllAuthz) CanViewExecution(_ context.Context) error  { return nil }
func (a allowAllAuthz) CanRetryExecution(_ context.Context) error { return nil }

func (s *DefaultServiceSuite) TestEventHandler_IngestEvent_Idempotent() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	rateLimiter := handlers.NewRateLimiter(cache.NewInMemoryCache(), 100)
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, rateLimiter)

	body := map[string]any{
		"event_type":      "user.created",
		"source":          "api",
		"idempotency_key": "idem-123",
		"payload":         map[string]any{"user_id": "u1"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.IngestEvent(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	// Second request should be idempotent.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req2 = req2.WithContext(ctx)
	h.IngestEvent(w2, req2)
	s.Equal(http.StatusAccepted, w2.Code)

	rows, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(rows, 1)
}

func (s *DefaultServiceSuite) TestFormHandler_SubmitForm() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewFormHandler(s.eventRepo, allowAllAuthz{}, metrics, nil)

	body := map[string]any{
		"fields": map[string]any{"email": "user@example.com"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader(payload))
	req.SetPathValue("form_id", "form-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.SubmitForm(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	rows, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(rows, 1)
	s.Equal("form.submitted", rows[0].EventType)
}

func (s *DefaultServiceSuite) TestWebhookReceiveHandler_IncludesHeaders() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWebhookReceiveHandler(s.eventRepo, allowAllAuthz{}, metrics, nil)

	body := map[string]any{"hello": "world"}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hook-1", bytes.NewReader(payload))
	req.SetPathValue("webhook_id", "hook-1")
	req.Header.Set("User-Agent", "tester")
	req.Header.Set("X-Webhook-Signature", "sig")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ReceiveWebhook(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	rows, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(rows, 1)

	var stored map[string]any
	s.Require().NoError(json.Unmarshal([]byte(rows[0].Payload), &stored))
	headers, ok := stored["headers"].(map[string]any)
	s.Require().True(ok)
	s.Equal("tester", headers["User-Agent"])
	s.Equal("sig", headers["X-Webhook-Signature"])
}
