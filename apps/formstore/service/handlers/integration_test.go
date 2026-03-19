package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

func (s *HandlerSuite) TestFormDefinitionHandler_Lifecycle() {
	ctx := s.tenantCtx()
	h := NewFormDefinitionHandler(s.biz, allowAllAuthz{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/form-definitions", encodeBody(map[string]any{
		"form_id":     "loan-application",
		"name":        "Loan Application",
		"description": "Loan form",
		"json_schema": map[string]any{"type": "object"},
	}))
	createReq = createReq.WithContext(ctx)
	createW := httptest.NewRecorder()
	h.Create(createW, createReq)
	s.Equal(http.StatusCreated, createW.Code)

	var created map[string]any
	s.Require().NoError(json.Unmarshal(createW.Body.Bytes(), &created))
	defID, _ := created["id"].(string)
	s.NotEmpty(defID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/form-definitions?active=true", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/form-definitions/"+defID, nil)
	getReq.SetPathValue("id", defID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/form-definitions/"+defID, encodeBody(map[string]any{
		"name":        "Loan Application Updated",
		"description": "Updated",
		"active":      false,
	}))
	updateReq.SetPathValue("id", defID)
	updateReq = updateReq.WithContext(ctx)
	updateW := httptest.NewRecorder()
	h.Update(updateW, updateReq)
	s.Equal(http.StatusOK, updateW.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/form-definitions/"+defID, nil)
	deleteReq.SetPathValue("id", defID)
	deleteReq = deleteReq.WithContext(ctx)
	deleteW := httptest.NewRecorder()
	h.Delete(deleteW, deleteReq)
	s.Equal(http.StatusNoContent, deleteW.Code)
}

func (s *HandlerSuite) TestFormSubmissionHandler_Lifecycle() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID:     "savings-form",
		Name:       "Savings",
		JSONSchema: `{"type":"object"}`,
		Active:     true,
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	h := NewFormSubmissionHandler(s.biz, allowAllAuthz{}, nil)

	submitReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/forms/"+def.FormID+"/submissions",
		encodeBody(map[string]any{
			"submitter_id":    "user-1",
			"data":            map[string]any{"amount": 100},
			"idempotency_key": "idem-1",
			"metadata":        map[string]any{"source": "web"},
		}),
	)
	submitReq.SetPathValue("form_id", def.FormID)
	submitReq = submitReq.WithContext(ctx)
	submitW := httptest.NewRecorder()
	h.Submit(submitW, submitReq)
	s.Equal(http.StatusCreated, submitW.Code)

	var created map[string]any
	s.Require().NoError(json.Unmarshal(submitW.Body.Bytes(), &created))
	subID, _ := created["id"].(string)
	s.NotEmpty(subID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/forms/"+def.FormID+"/submissions?limit=10", nil)
	listReq.SetPathValue("form_id", def.FormID)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.ListByForm(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+subID, nil)
	getReq.SetPathValue("id", subID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/submissions/"+subID, encodeBody(map[string]any{
		"data":     map[string]any{"amount": 150},
		"status":   "complete",
		"metadata": map[string]any{"source": "api"},
	}))
	updateReq.SetPathValue("id", subID)
	updateReq = updateReq.WithContext(ctx)
	updateW := httptest.NewRecorder()
	h.Update(updateW, updateReq)
	s.Equal(http.StatusOK, updateW.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/submissions/"+subID, nil)
	deleteReq.SetPathValue("id", subID)
	deleteReq = deleteReq.WithContext(ctx)
	deleteW := httptest.NewRecorder()
	h.Delete(deleteW, deleteReq)
	s.Equal(http.StatusNoContent, deleteW.Code)
}

func (s *HandlerSuite) TestFormDefinitionHandler_ValidationAndNotFound() {
	ctx := s.tenantCtx()
	h := NewFormDefinitionHandler(s.biz, allowAllAuthz{})

	tests := []struct {
		name       string
		method     string
		target     string
		id         string
		body       any
		wantStatus int
	}{
		{
			name:       "create rejects invalid json",
			method:     http.MethodPost,
			target:     "/api/v1/form-definitions",
			body:       "{",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create requires form id and name",
			method:     http.MethodPost,
			target:     "/api/v1/form-definitions",
			body:       map[string]any{"form_id": "", "name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get missing returns not found",
			method:     http.MethodGet,
			target:     "/api/v1/form-definitions/missing",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete missing returns not found",
			method:     http.MethodDelete,
			target:     "/api/v1/form-definitions/missing",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var req *http.Request
			switch body := tc.body.(type) {
			case string:
				req = httptest.NewRequest(tc.method, tc.target, strings.NewReader(body))
			case nil:
				req = httptest.NewRequest(tc.method, tc.target, nil)
			default:
				req = httptest.NewRequest(tc.method, tc.target, encodeBody(body))
			}
			if tc.id != "" {
				req.SetPathValue("id", tc.id)
			}
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			switch tc.method {
			case http.MethodPost:
				h.Create(w, req)
			case http.MethodGet:
				h.Get(w, req)
			case http.MethodDelete:
				h.Delete(w, req)
			}
			s.Equal(tc.wantStatus, w.Code)
		})
	}
}

func (s *HandlerSuite) TestFormSubmissionHandler_ValidationRateLimitAndNotFound() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID:     "submit-form",
		Name:       "Submit",
		JSONSchema: `{"type":"object"}`,
		Active:     true,
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	rateLimited := NewFormSubmissionHandler(s.biz, allowAllAuthz{}, NewRateLimiter(cache.NewInMemoryCache(), 1))
	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/forms/"+def.FormID+"/submissions", encodeBody(map[string]any{
		"submitter_id": "user-1",
		"data":         map[string]any{"amount": 100},
	}))
	okReq.SetPathValue("form_id", def.FormID)
	okReq = okReq.WithContext(ctx)
	okW := httptest.NewRecorder()
	rateLimited.Submit(okW, okReq)
	s.Equal(http.StatusCreated, okW.Code)

	rateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/forms/"+def.FormID+"/submissions",
		encodeBody(map[string]any{
			"submitter_id": "user-1",
			"data":         map[string]any{"amount": 200},
		}),
	)
	rateReq.SetPathValue("form_id", def.FormID)
	rateReq = rateReq.WithContext(ctx)
	rateW := httptest.NewRecorder()
	rateLimited.Submit(rateW, rateReq)
	s.Equal(http.StatusTooManyRequests, rateW.Code)

	noLimit := NewFormSubmissionHandler(s.biz, allowAllAuthz{}, nil)
	tests := []struct {
		name       string
		method     string
		target     string
		id         string
		formID     string
		body       any
		wantStatus int
	}{
		{
			name:       "submit rejects invalid json",
			method:     http.MethodPost,
			target:     "/api/v1/forms/" + def.FormID + "/submissions",
			formID:     def.FormID,
			body:       "{",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "submit requires data",
			method:     http.MethodPost,
			target:     "/api/v1/forms/" + def.FormID + "/submissions",
			formID:     def.FormID,
			body:       map[string]any{"submitter_id": "user-2"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get missing returns not found",
			method:     http.MethodGet,
			target:     "/api/v1/submissions/missing",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete missing returns not found",
			method:     http.MethodDelete,
			target:     "/api/v1/submissions/missing",
			id:         "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var req *http.Request
			switch body := tc.body.(type) {
			case string:
				req = httptest.NewRequest(tc.method, tc.target, strings.NewReader(body))
			case nil:
				req = httptest.NewRequest(tc.method, tc.target, nil)
			default:
				req = httptest.NewRequest(tc.method, tc.target, encodeBody(body))
			}
			if tc.id != "" {
				req.SetPathValue("id", tc.id)
			}
			if tc.formID != "" {
				req.SetPathValue("form_id", tc.formID)
			}
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			switch tc.method {
			case http.MethodPost:
				noLimit.Submit(w, req)
			case http.MethodGet:
				noLimit.Get(w, req)
			case http.MethodDelete:
				noLimit.Delete(w, req)
			}
			s.Equal(tc.wantStatus, w.Code)
		})
	}
}
