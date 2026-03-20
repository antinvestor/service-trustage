//nolint:testpackage // package-local integration tests use unexported handler fixtures intentionally.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *HandlerSuite) TestWorkflowHandler_Lifecycle() {
	ctx := s.tenantCtx()
	h := NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, s.metrics)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte(s.sampleDSL())))
	createReq = createReq.WithContext(ctx)
	createW := httptest.NewRecorder()
	h.CreateWorkflow(createW, createReq)
	s.Equal(http.StatusCreated, createW.Code)

	var createResp map[string]any
	s.Require().NoError(json.Unmarshal(createW.Body.Bytes(), &createResp))
	workflowID, _ := createResp["id"].(string)
	s.NotEmpty(workflowID)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+workflowID, nil)
	getReq.SetPathValue("id", workflowID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.GetWorkflow(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	activateReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+workflowID+"/activate", nil)
	activateReq.SetPathValue("id", workflowID)
	activateReq = activateReq.WithContext(ctx)
	activateW := httptest.NewRecorder()
	h.ActivateWorkflow(activateW, activateReq)
	s.Equal(http.StatusOK, activateW.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?status=active", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.ListWorkflows(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *HandlerSuite) TestInstanceAndExecutionHandlers_Lifecycle() {
	ctx := s.tenantCtx()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusFailed,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusFailed,
		InputPayload:   `{"hello":"world"}`,
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))
	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "step-a",
		SchemaHash:  "hash",
		Payload:     `{"result":"ok"}`,
	}))

	instanceHandler := NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})
	executionHandler := NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	listInstancesReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances?status=failed", nil)
	listInstancesReq = listInstancesReq.WithContext(ctx)
	listInstancesW := httptest.NewRecorder()
	instanceHandler.List(listInstancesW, listInstancesReq)
	s.Equal(http.StatusOK, listInstancesW.Code)

	retryInstanceReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+instance.ID+"/retry", nil)
	retryInstanceReq.SetPathValue("id", instance.ID)
	retryInstanceReq = retryInstanceReq.WithContext(ctx)
	retryInstanceW := httptest.NewRecorder()
	instanceHandler.Retry(retryInstanceW, retryInstanceReq)
	s.Equal(http.StatusOK, retryInstanceW.Code)

	listExecReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	listExecReq = listExecReq.WithContext(ctx)
	listExecW := httptest.NewRecorder()
	executionHandler.List(listExecW, listExecReq)
	s.Equal(http.StatusOK, listExecW.Code)

	getExecReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+exec.ID+"?include_output=true", nil)
	getExecReq.SetPathValue("id", exec.ID)
	getExecReq = getExecReq.WithContext(ctx)
	getExecW := httptest.NewRecorder()
	executionHandler.Get(getExecW, getExecReq)
	s.Equal(http.StatusOK, getExecW.Code)

	retryExecReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+exec.ID+"/retry", nil)
	retryExecReq.SetPathValue("id", exec.ID)
	retryExecReq = retryExecReq.WithContext(ctx)
	retryExecW := httptest.NewRecorder()
	executionHandler.Retry(retryExecW, retryExecReq)
	s.Equal(http.StatusOK, retryExecW.Code)
}

func (s *HandlerSuite) TestEventFormAndWebhookHandlers() {
	ctx := s.tenantCtx()
	eventHandler := NewEventHandler(
		s.eventRepo,
		s.auditRepo,
		allowAllAuthz{},
		s.metrics,
		NewRateLimiter(cache.NewInMemoryCache(), 100),
	)
	formHandler := NewFormHandler(s.eventRepo, allowAllAuthz{}, s.metrics, nil)
	webhookHandler := NewWebhookReceiveHandler(s.eventRepo, allowAllAuthz{}, s.metrics, nil)

	eventBody, err := json.Marshal(map[string]any{
		"event_type":      "user.created",
		"source":          "api",
		"idempotency_key": "idem-123",
		"payload":         map[string]any{"user_id": "u1"},
	})
	s.Require().NoError(err)

	eventReq := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(eventBody))
	eventReq = eventReq.WithContext(ctx)
	eventW := httptest.NewRecorder()
	eventHandler.IngestEvent(eventW, eventReq)
	s.Equal(http.StatusAccepted, eventW.Code)

	secondEventReq := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(eventBody))
	secondEventReq = secondEventReq.WithContext(ctx)
	secondEventW := httptest.NewRecorder()
	eventHandler.IngestEvent(secondEventW, secondEventReq)
	s.Equal(http.StatusAccepted, secondEventW.Code)

	s.Require().NoError(s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID: "inst-1",
		EventType:  "state.started",
		State:      "step-a",
	}))
	timelineReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/timeline", nil)
	timelineReq.SetPathValue("id", "inst-1")
	timelineReq = timelineReq.WithContext(ctx)
	timelineW := httptest.NewRecorder()
	eventHandler.GetInstanceTimeline(timelineW, timelineReq)
	s.Equal(http.StatusOK, timelineW.Code)

	formBody, err := json.Marshal(map[string]any{
		"fields": map[string]any{"email": "user@example.com"},
	})
	s.Require().NoError(err)

	formReq := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader(formBody))
	formReq.SetPathValue("form_id", "form-1")
	formReq = formReq.WithContext(ctx)
	formW := httptest.NewRecorder()
	formHandler.SubmitForm(formW, formReq)
	s.Equal(http.StatusAccepted, formW.Code)

	webhookBody, err := json.Marshal(map[string]any{"hello": "world"})
	s.Require().NoError(err)

	webhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hook-1", bytes.NewReader(webhookBody))
	webhookReq.SetPathValue("webhook_id", "hook-1")
	webhookReq.Header.Set("User-Agent", "tester")
	webhookReq.Header.Set("X-Webhook-Signature", "sig")
	webhookReq = webhookReq.WithContext(ctx)
	webhookW := httptest.NewRecorder()
	webhookHandler.ReceiveWebhook(webhookW, webhookReq)
	s.Equal(http.StatusAccepted, webhookW.Code)
}

func (s *HandlerSuite) TestHTTPHandlers_ValidationAndNotFoundPaths() {
	ctx := s.tenantCtx()

	workflowHandler := NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, s.metrics)
	instanceHandler := NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})
	executionHandler := NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})
	eventHandler := NewEventHandler(
		s.eventRepo,
		s.auditRepo,
		allowAllAuthz{},
		s.metrics,
		NewRateLimiter(cache.NewInMemoryCache(), 1),
	)
	formHandler := NewFormHandler(s.eventRepo, allowAllAuthz{}, s.metrics, nil)
	webhookHandler := NewWebhookReceiveHandler(s.eventRepo, allowAllAuthz{}, s.metrics, nil)

	tests := []struct {
		name       string
		wantStatus int
		exec       func() *httptest.ResponseRecorder
	}{
		{
			name:       "workflow create rejects invalid json",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", strings.NewReader("{"))
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				workflowHandler.CreateWorkflow(w, req)
				return w
			},
		},
		{
			name:       "workflow get missing returns not found",
			wantStatus: http.StatusNotFound,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/missing", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				workflowHandler.GetWorkflow(w, req)
				return w
			},
		},
		{
			name:       "instance retry missing returns not found",
			wantStatus: http.StatusNotFound,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/missing/retry", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				instanceHandler.Retry(w, req)
				return w
			},
		},
		{
			name:       "execution retry missing returns not found",
			wantStatus: http.StatusNotFound,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/missing/retry", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				executionHandler.Retry(w, req)
				return w
			},
		},
		{
			name:       "event ingest rejects invalid json",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader("{"))
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				eventHandler.IngestEvent(w, req)
				return w
			},
		},
		{
			name:       "event ingest rate limits second call",
			wantStatus: http.StatusTooManyRequests,
			exec: func() *httptest.ResponseRecorder {
				body := []byte(`{"event_type":"x","source":"api","payload":{}}`)
				first := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
				first = first.WithContext(ctx)
				eventHandler.IngestEvent(httptest.NewRecorder(), first)

				second := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
				second = second.WithContext(ctx)
				w := httptest.NewRecorder()
				eventHandler.IngestEvent(w, second)
				return w
			},
		},
		{
			name:       "form submit rejects invalid json",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/forms/f1/submit", strings.NewReader("{"))
				req.SetPathValue("form_id", "f1")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				formHandler.SubmitForm(w, req)
				return w
			},
		},
		{
			name:       "form submit requires fields",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/forms/f1/submit",
					bytes.NewReader([]byte(`{"fields":{}}`)),
				)
				req.SetPathValue("form_id", "f1")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				formHandler.SubmitForm(w, req)
				return w
			},
		},
		{
			name:       "form submit requires form id",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/forms//submit",
					bytes.NewReader([]byte(`{"fields":{"x":1}}`)),
				)
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				formHandler.SubmitForm(w, req)
				return w
			},
		},
		{
			name:       "webhook receive rejects invalid json",
			wantStatus: http.StatusBadRequest,
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/h1", strings.NewReader("{"))
				req.SetPathValue("webhook_id", "h1")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				webhookHandler.ReceiveWebhook(w, req)
				return w
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			w := tc.exec()
			s.Equal(tc.wantStatus, w.Code)
		})
	}
}

func (s *HandlerSuite) TestFormHandler_IdempotentAndWorkflowActivateInvalidTransition() {
	ctx := s.tenantCtx()
	formHandler := NewFormHandler(s.eventRepo, allowAllAuthz{}, s.metrics, nil)
	workflowHandler := NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, s.metrics)

	first := httptest.NewRequest(http.MethodPost, "/api/v1/forms/f1/submit", bytes.NewReader([]byte(`{
		"fields":{"email":"user@example.com"},
		"idempotency_key":"idem-form-1"
	}`)))
	first.SetPathValue("form_id", "f1")
	first = first.WithContext(ctx)
	firstW := httptest.NewRecorder()
	formHandler.SubmitForm(firstW, first)
	s.Equal(http.StatusAccepted, firstW.Code)

	second := httptest.NewRequest(http.MethodPost, "/api/v1/forms/f1/submit", bytes.NewReader([]byte(`{
		"fields":{"email":"user@example.com"},
		"idempotency_key":"idem-form-1"
	}`)))
	second.SetPathValue("form_id", "f1")
	second = second.WithContext(ctx)
	secondW := httptest.NewRecorder()
	formHandler.SubmitForm(secondW, second)
	s.Equal(http.StatusAccepted, secondW.Code)
	s.Contains(secondW.Body.String(), `"idempotent":true`)

	def := s.createWorkflow(ctx, s.sampleDSL())
	s.Require().NoError(def.TransitionTo(models.WorkflowStatusActive))
	s.Require().NoError(s.defRepo.Update(ctx, def))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+def.ID+"/activate", nil)
	req.SetPathValue("id", def.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	workflowHandler.ActivateWorkflow(w, req)
	s.Equal(http.StatusBadRequest, w.Code)
}
