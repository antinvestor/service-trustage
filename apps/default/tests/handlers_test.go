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

package tests_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func (s *DefaultServiceSuite) TestEventHandler_IngestEvent_Idempotent() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	rateLimiter := handlers.NewRateLimiter(cache.NewInMemoryCache(), 100)
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, metrics, rateLimiter)

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
	h := handlers.NewFormHandler(s.eventRepo, metrics, nil)

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
	h := handlers.NewWebhookReceiveHandler(s.eventRepo, metrics, nil)

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
