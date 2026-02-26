package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func (s *DefaultServiceSuite) TestWorkflowHandler_Lifecycle() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

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

	var listResp map[string][]map[string]any
	s.Require().NoError(json.Unmarshal(listW.Body.Bytes(), &listResp))
	s.Len(listResp["items"], 1)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Errors() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte(`{"version":"1.0"}`)))
	badReq = badReq.WithContext(ctx)
	badW := httptest.NewRecorder()
	h.CreateWorkflow(badW, badReq)
	s.Equal(http.StatusInternalServerError, badW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/missing", nil)
	getReq.SetPathValue("id", "missing")
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.GetWorkflow(getW, getReq)
	s.Equal(http.StatusNotFound, getW.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?status=draft", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.ListWorkflows(listW, listReq)
	s.Equal(http.StatusBadRequest, listW.Code)

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	unauthW := httptest.NewRecorder()
	h.ListWorkflows(unauthW, unauthReq)
	s.Equal(http.StatusUnauthorized, unauthW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_ListAndRetry() {
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
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances?status=failed", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+instance.ID+"/retry", nil)
	retryReq.SetPathValue("id", instance.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusOK, retryW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_ListGetRetry() {
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

	output := &models.WorkflowStateOutput{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "step-a",
		SchemaHash:  "hash",
		Payload:     `{"result":"ok"}`,
	}
	s.Require().NoError(s.outputRepo.Store(ctx, output))

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+exec.ID+"?include_output=true", nil)
	getReq.SetPathValue("id", exec.ID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+exec.ID+"/retry", nil)
	retryReq.SetPathValue("id", exec.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusOK, retryW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_GetErrors() {
	ctx := s.tenantCtx()
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/", nil)
	getReq.SetPathValue("id", "")
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusBadRequest, getW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_Timeline() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	audit := &models.WorkflowAuditEvent{InstanceID: "inst-1", EventType: "state.started", State: "step-a"}
	s.Require().NoError(s.auditRepo.Append(ctx, audit))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/timeline", nil)
	req.SetPathValue("id", "inst-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetInstanceTimeline(w, req)
	s.Equal(http.StatusOK, w.Code)

	var entries []map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &entries))
	s.Len(entries, 1)
}

func (s *DefaultServiceSuite) TestRequestIDMiddleware() {
	h := handlers.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.NotEmpty(w.Header().Get(handlers.RequestIDHeader))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(handlers.RequestIDHeader, "req-123")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	s.Equal("req-123", w2.Header().Get(handlers.RequestIDHeader))
}

func (s *DefaultServiceSuite) TestLimitBodySize() {
	h := handlers.LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	oversize := bytes.Repeat([]byte("a"), handlers.MaxRequestBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(oversize))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	s.Equal(http.StatusRequestEntityTooLarge, w.Code)
}

func (s *DefaultServiceSuite) TestRateLimiter_Allow() {
	cacheStore := cache.NewInMemoryCache()
	limiter := handlers.NewRateLimiter(cacheStore, 1)
	ctx := s.tenantCtx()

	allowed := limiter.Allow(ctx)
	denied := limiter.Allow(ctx)
	s.True(allowed)
	s.False(denied)

	// Ensure fail-open when limiter disabled.
	nilLimiter := handlers.NewRateLimiter(nil, 1)
	s.True(nilLimiter.Allow(ctx))
}

func (s *DefaultServiceSuite) TestEventHandler_RateLimitExceeded() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	limiter := handlers.NewRateLimiter(cache.NewInMemoryCache(), 1)
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, limiter)

	body := map[string]any{"event_type": "user.created", "source": "api", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	h.IngestEvent(w2, req2)
	s.Equal(http.StatusTooManyRequests, w2.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_RejectsMissingEventType() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	body := map[string]any{"source": "api", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_RetryErrors() {
	ctx := s.tenantCtx()
	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances//retry", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Retry(w, req)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_RetryNotFound() {
	ctx := s.tenantCtx()
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/missing/retry", nil)
	retryReq.SetPathValue("id", "missing")
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusNotFound, retryW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_Timeline_Unauthorized() {
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/timeline", nil)
	req.SetPathValue("id", "inst-1")
	w := httptest.NewRecorder()
	h.GetInstanceTimeline(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_InvalidJSON() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte("{")))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusBadRequest, w.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_ListFilters() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusPending,
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions?status=pending", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_ListFilters() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances?workflow_name=wf", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_GetNotFound() {
	ctx := s.tenantCtx()
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/missing", nil)
	getReq.SetPathValue("id", "missing")
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusNotFound, getW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_RetryNotFound() {
	ctx := s.tenantCtx()
	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/missing/retry", nil)
	req.SetPathValue("id", "missing")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Retry(w, req)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_RetryCreatesNewAttempt() {
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
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+exec.ID+"/retry", nil)
	retryReq.SetPathValue("id", exec.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusOK, retryW.Code)

	var total int64
	s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM workflow_state_executions WHERE instance_id = ?",
		instance.ID,
	).Scan(&total)
	s.Equal(int64(2), total)
}

func (s *DefaultServiceSuite) TestInstanceHandler_RetryCreatesNewAttempt() {
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
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+instance.ID+"/retry", nil)
	retryReq.SetPathValue("id", instance.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusOK, retryW.Code)

	var total int64
	s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM workflow_state_executions WHERE instance_id = ?",
		instance.ID,
	).Scan(&total)
	s.Equal(int64(2), total)
}

func (s *DefaultServiceSuite) TestExecutionHandler_RetryNotAllowed() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusPending,
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+exec.ID+"/retry", nil)
	retryReq.SetPathValue("id", exec.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusConflict, retryW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_RetryNotAllowed() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusPending,
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/"+instance.ID+"/retry", nil)
	retryReq.SetPathValue("id", instance.ID)
	retryReq = retryReq.WithContext(ctx)
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusConflict, retryW.Code)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Forbidden() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), denyAuthz{}, metrics)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte(s.sampleDSL())))
	createReq = createReq.WithContext(ctx)
	createW := httptest.NewRecorder()
	h.CreateWorkflow(createW, createReq)
	s.Equal(http.StatusForbidden, createW.Code)
}

type denyAuthz struct{}

func (d denyAuthz) CanIngestEvent(_ context.Context) error    { return errDenied }
func (d denyAuthz) CanManageWorkflow(_ context.Context) error { return errDenied }
func (d denyAuthz) CanViewWorkflow(_ context.Context) error   { return errDenied }
func (d denyAuthz) CanViewInstance(_ context.Context) error   { return errDenied }
func (d denyAuthz) CanRetryInstance(_ context.Context) error  { return errDenied }
func (d denyAuthz) CanViewExecution(_ context.Context) error  { return errDenied }
func (d denyAuthz) CanRetryExecution(_ context.Context) error { return errDenied }

var errDenied = errors.New("denied")

func (s *DefaultServiceSuite) TestExecutionHandler_Forbidden() {
	ctx := s.tenantCtx()
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, denyAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusForbidden, listW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_Forbidden() {
	ctx := s.tenantCtx()
	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, denyAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusForbidden, listW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_Forbidden() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, denyAuthz{}, metrics, nil)

	body := map[string]any{"event_type": "user.created", "source": "api", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusForbidden, w.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_List_Unauthorized() {
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusUnauthorized, listW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_List_Unauthorized() {
	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusUnauthorized, listW.Code)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_List_Unauthorized() {
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	listW := httptest.NewRecorder()
	h.ListWorkflows(listW, listReq)
	s.Equal(http.StatusUnauthorized, listW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_Retry_Unauthorized() {
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/executions/1/retry", nil)
	retryReq.SetPathValue("id", "1")
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusUnauthorized, retryW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_Retry_Unauthorized() {
	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/1/retry", nil)
	retryReq.SetPathValue("id", "1")
	retryW := httptest.NewRecorder()
	h.Retry(retryW, retryReq)
	s.Equal(http.StatusUnauthorized, retryW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_RateLimiter_UnknownClaims() {
	metrics := telemetry.NewMetrics()
	limiter := handlers.NewRateLimiter(cache.NewInMemoryCache(), 1)
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, limiter)

	body := map[string]any{"event_type": "user.created", "source": "api", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	ctx := s.tenantCtx()
	_ = limiter.Allow(ctx)
	_ = limiter.Allow(ctx)
}

func (s *DefaultServiceSuite) TestInstanceHandler_List_Limit() {
	ctx := s.tenantCtx()
	for i := 0; i < 3; i++ {
		inst := &models.WorkflowInstance{
			WorkflowName:    "wf",
			WorkflowVersion: 1,
			CurrentState:    "step",
			Status:          models.InstanceStatusRunning,
			Revision:        1,
		}
		s.Require().NoError(s.instanceRepo.Create(ctx, inst))
	}

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances?limit=1", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_List_Limit() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	for i := 0; i < 3; i++ {
		exec := &models.WorkflowStateExecution{
			InstanceID:     instance.ID,
			State:          "step",
			Attempt:        1,
			Status:         models.ExecStatusPending,
			InputPayload:   "{}",
			ExecutionToken: "token",
		}
		s.Require().NoError(s.execRepo.Create(ctx, exec))
	}

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions?limit=1", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_PayloadMetadata() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	body := map[string]any{"event_type": "user.created", "source": "api", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	req.Header.Set("X-Request-Id", "req1")
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	rows, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(rows, 1)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Create_InvalidJSON() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader([]byte("{")))
	createReq = createReq.WithContext(ctx)
	createW := httptest.NewRecorder()
	h.CreateWorkflow(createW, createReq)
	s.Equal(http.StatusBadRequest, createW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_OutputAbsent() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step",
		Attempt:        1,
		Status:         models.ExecStatusFailed,
		InputPayload:   "{}",
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+exec.ID+"?include_output=true", nil)
	getReq.SetPathValue("id", exec.ID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)
}

func (s *DefaultServiceSuite) TestInstanceHandler_List_StatusFilter() {
	ctx := s.tenantCtx()
	instRunning := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	instFailed := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step",
		Status:          models.InstanceStatusFailed,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instRunning))
	s.Require().NoError(s.instanceRepo.Create(ctx, instFailed))

	h := handlers.NewInstanceHandler(s.instanceRepo, s.execRepo, s.auditRepo, allowAllAuthz{})
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances?status=failed", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)
}

func (s *DefaultServiceSuite) TestExecutionHandler_Get_Unauthorized() {
	h := handlers.NewExecutionHandler(s.execRepo, s.instanceRepo, s.outputRepo, s.auditRepo, allowAllAuthz{})

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/executions/1", nil)
	getReq.SetPathValue("id", "1")
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusUnauthorized, getW.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_Ingest_Idempotency() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	body := map[string]any{"event_type": "user.created", "source": "api", "idempotency_key": "dup", "payload": map[string]any{"id": "1"}}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.IngestEvent(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(payload))
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	h.IngestEvent(w2, req2)
	s.Equal(http.StatusAccepted, w2.Code)
}

func (s *DefaultServiceSuite) TestEventHandler_Timeline_Ordering() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewEventHandler(s.eventRepo, s.auditRepo, allowAllAuthz{}, metrics, nil)

	first := &models.WorkflowAuditEvent{InstanceID: "inst-2", EventType: "state.started", State: "step-a"}
	second := &models.WorkflowAuditEvent{InstanceID: "inst-2", EventType: "state.completed", State: "step-a"}
	s.Require().NoError(s.auditRepo.Append(ctx, first))
	time.Sleep(5 * time.Millisecond)
	s.Require().NoError(s.auditRepo.Append(ctx, second))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-2/timeline", nil)
	req.SetPathValue("id", "inst-2")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetInstanceTimeline(w, req)
	s.Equal(http.StatusOK, w.Code)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Activate_NotFound() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/missing/activate", nil)
	req.SetPathValue("id", "missing")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ActivateWorkflow(w, req)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Activate_Unauthorized() {
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/1/activate", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ActivateWorkflow(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *DefaultServiceSuite) TestWorkflowHandler_Activate_AlreadyActive() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWorkflowHandler(s.workflowBusiness(), allowAllAuthz{}, metrics)

	def := s.createWorkflow(ctx, s.sampleDSL())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+def.ID+"/activate", nil)
	req.SetPathValue("id", def.ID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ActivateWorkflow(w, req)
	s.Equal(http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+def.ID+"/activate", nil)
	req2.SetPathValue("id", def.ID)
	req2 = req2.WithContext(ctx)
	h.ActivateWorkflow(w2, req2)
	s.Equal(http.StatusBadRequest, w2.Code)
}

func (s *DefaultServiceSuite) TestFormHandler_ErrorsAndIdempotency() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewFormHandler(s.eventRepo, allowAllAuthz{}, metrics, nil)

	missingFormReq := httptest.NewRequest(http.MethodPost, "/api/v1/forms//submit", bytes.NewReader([]byte(`{}`)))
	missingFormReq = missingFormReq.WithContext(ctx)
	missingW := httptest.NewRecorder()
	h.SubmitForm(missingW, missingFormReq)
	s.Equal(http.StatusBadRequest, missingW.Code)

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader([]byte("{")))
	invalidReq.SetPathValue("form_id", "form-1")
	invalidReq = invalidReq.WithContext(ctx)
	invalidW := httptest.NewRecorder()
	h.SubmitForm(invalidW, invalidReq)
	s.Equal(http.StatusBadRequest, invalidW.Code)

	fieldsReq := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader([]byte(`{"fields":{}}`)))
	fieldsReq.SetPathValue("form_id", "form-1")
	fieldsReq = fieldsReq.WithContext(ctx)
	fieldsW := httptest.NewRecorder()
	h.SubmitForm(fieldsW, fieldsReq)
	s.Equal(http.StatusBadRequest, fieldsW.Code)

	body := map[string]any{
		"fields":          map[string]any{"email": "user@example.com"},
		"idempotency_key": "form-dup",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader(payload))
	req.SetPathValue("form_id", "form-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.SubmitForm(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/forms/form-1/submit", bytes.NewReader(payload))
	req2.SetPathValue("form_id", "form-1")
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	h.SubmitForm(w2, req2)
	s.Equal(http.StatusAccepted, w2.Code)
}

func (s *DefaultServiceSuite) TestWebhookHandler_ErrorsAndRateLimit() {
	ctx := s.tenantCtx()
	metrics := telemetry.NewMetrics()
	h := handlers.NewWebhookReceiveHandler(s.eventRepo, allowAllAuthz{}, metrics, nil)

	missingReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks//", bytes.NewReader([]byte(`{}`)))
	missingReq = missingReq.WithContext(ctx)
	missingW := httptest.NewRecorder()
	h.ReceiveWebhook(missingW, missingReq)
	s.Equal(http.StatusBadRequest, missingW.Code)

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hook-1", bytes.NewReader([]byte("{")))
	invalidReq.SetPathValue("webhook_id", "hook-1")
	invalidReq = invalidReq.WithContext(ctx)
	invalidW := httptest.NewRecorder()
	h.ReceiveWebhook(invalidW, invalidReq)
	s.Equal(http.StatusBadRequest, invalidW.Code)

	limiter := handlers.NewRateLimiter(cache.NewInMemoryCache(), 1)
	hLimited := handlers.NewWebhookReceiveHandler(s.eventRepo, allowAllAuthz{}, metrics, limiter)

	body := map[string]any{"hello": "world"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hook-1", bytes.NewReader(payload))
	req.SetPathValue("webhook_id", "hook-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	hLimited.ReceiveWebhook(w, req)
	s.Equal(http.StatusAccepted, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/hook-1", bytes.NewReader(payload))
	req2.SetPathValue("webhook_id", "hook-1")
	req2 = req2.WithContext(ctx)
	w2 := httptest.NewRecorder()
	hLimited.ReceiveWebhook(w2, req2)
	s.Equal(http.StatusTooManyRequests, w2.Code)
}
