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

//nolint:testpackage // package-local integration tests use unexported handler fixtures intentionally.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pitabwire/frame/cache"
	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

func (s *HandlerSuite) TestQueueDefinitionHandler_Lifecycle() {
	ctx := s.tenantCtx()
	h := NewQueueDefinitionHandler(s.manager)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/queues", encodeBody(map[string]any{
		"name":            "main-queue",
		"description":     "Main queue",
		"priority_levels": 3,
		"max_capacity":    10,
		"sla_minutes":     30,
		"config":          map[string]any{"kind": "walkin"},
	}))
	createReq = createReq.WithContext(ctx)
	createW := httptest.NewRecorder()
	h.Create(createW, createReq)
	s.Equal(http.StatusCreated, createW.Code)

	var created map[string]any
	s.Require().NoError(json.Unmarshal(createW.Body.Bytes(), &created))
	queueID, _ := created["id"].(string)
	s.NotEmpty(queueID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues?active=true", nil)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues/"+queueID, nil)
	getReq.SetPathValue("id", queueID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	h.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/queues/"+queueID, encodeBody(map[string]any{
		"description": "Updated queue",
		"active":      false,
	}))
	updateReq.SetPathValue("id", queueID)
	updateReq = updateReq.WithContext(ctx)
	updateW := httptest.NewRecorder()
	h.Update(updateW, updateReq)
	s.Equal(http.StatusOK, updateW.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/queues/"+queueID, nil)
	deleteReq.SetPathValue("id", queueID)
	deleteReq = deleteReq.WithContext(ctx)
	deleteW := httptest.NewRecorder()
	h.Delete(deleteW, deleteReq)
	s.Equal(http.StatusNoContent, deleteW.Code)
}

func (s *HandlerSuite) TestQueueItemCounterAndStatsHandlers_Flow() {
	ctx := s.tenantCtx()
	queueDef := &models.QueueDefinition{
		Name:           "service-queue",
		Active:         true,
		PriorityLevels: 3,
		MaxCapacity:    20,
		SLAMinutes:     30,
		Config:         "{}",
	}
	s.Require().NoError(s.defRepo.Create(ctx, queueDef))

	itemHandler := NewQueueItemHandler(s.manager, NewRateLimiter(cache.NewInMemoryCache(), 100))
	counterHandler := NewQueueCounterHandler(s.manager)
	statsHandler := NewQueueStatsHandler(s.stats)

	enqueueReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/queues/"+queueDef.ID+"/items",
		encodeBody(map[string]any{
			"priority":    2,
			"category":    "vip",
			"customer_id": "customer-1",
			"metadata":    map[string]any{"source": "web"},
			"ticket_no":   "VIP-001",
		}),
	)
	enqueueReq.SetPathValue("queue_id", queueDef.ID)
	enqueueReq = enqueueReq.WithContext(ctx)
	enqueueW := httptest.NewRecorder()
	itemHandler.Enqueue(enqueueW, enqueueReq)
	s.Equal(http.StatusCreated, enqueueW.Code)

	var itemResp map[string]any
	s.Require().NoError(json.Unmarshal(enqueueW.Body.Bytes(), &itemResp))
	itemID, _ := itemResp["id"].(string)
	s.NotEmpty(itemID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues/"+queueDef.ID+"/items?limit=10", nil)
	listReq.SetPathValue("queue_id", queueDef.ID)
	listReq = listReq.WithContext(ctx)
	listW := httptest.NewRecorder()
	itemHandler.ListWaiting(listW, listReq)
	s.Equal(http.StatusOK, listW.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+itemID, nil)
	getReq.SetPathValue("id", itemID)
	getReq = getReq.WithContext(ctx)
	getW := httptest.NewRecorder()
	itemHandler.Get(getW, getReq)
	s.Equal(http.StatusOK, getW.Code)

	positionReq := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+itemID+"/position", nil)
	positionReq.SetPathValue("id", itemID)
	positionReq = positionReq.WithContext(ctx)
	positionW := httptest.NewRecorder()
	itemHandler.GetPosition(positionW, positionReq)
	s.Equal(http.StatusOK, positionW.Code)

	createCounterReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/queues/"+queueDef.ID+"/counters",
		encodeBody(map[string]any{
			"name":       "Desk 1",
			"categories": []string{"vip"},
		}),
	)
	createCounterReq.SetPathValue("queue_id", queueDef.ID)
	createCounterReq = createCounterReq.WithContext(ctx)
	createCounterW := httptest.NewRecorder()
	counterHandler.Create(createCounterW, createCounterReq)
	s.Equal(http.StatusCreated, createCounterW.Code)

	var counterResp map[string]any
	s.Require().NoError(json.Unmarshal(createCounterW.Body.Bytes(), &counterResp))
	counterID, _ := counterResp["id"].(string)
	s.NotEmpty(counterID)

	listCountersReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues/"+queueDef.ID+"/counters", nil)
	listCountersReq.SetPathValue("queue_id", queueDef.ID)
	listCountersReq = listCountersReq.WithContext(ctx)
	listCountersW := httptest.NewRecorder()
	counterHandler.List(listCountersW, listCountersReq)
	s.Equal(http.StatusOK, listCountersW.Code)

	openReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/counters/"+counterID+"/open",
		encodeBody(map[string]any{"staff_id": "staff-1"}),
	)
	openReq.SetPathValue("id", counterID)
	openReq = openReq.WithContext(ctx)
	openW := httptest.NewRecorder()
	counterHandler.Open(openW, openReq)
	s.Equal(http.StatusOK, openW.Code)

	callNextReq := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counterID+"/call-next", nil)
	callNextReq.SetPathValue("id", counterID)
	callNextReq = callNextReq.WithContext(ctx)
	callNextW := httptest.NewRecorder()
	counterHandler.CallNext(callNextW, callNextReq)
	s.Equal(http.StatusOK, callNextW.Code)

	beginReq := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counterID+"/begin-service", nil)
	beginReq.SetPathValue("id", counterID)
	beginReq = beginReq.WithContext(ctx)
	beginW := httptest.NewRecorder()
	counterHandler.BeginService(beginW, beginReq)
	s.Equal(http.StatusOK, beginW.Code)

	completeReq := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counterID+"/complete-service", nil)
	completeReq.SetPathValue("id", counterID)
	completeReq = completeReq.WithContext(ctx)
	completeW := httptest.NewRecorder()
	counterHandler.CompleteService(completeW, completeReq)
	s.Equal(http.StatusOK, completeW.Code)

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counterID+"/pause", nil)
	pauseReq.SetPathValue("id", counterID)
	pauseReq = pauseReq.WithContext(ctx)
	pauseW := httptest.NewRecorder()
	counterHandler.Pause(pauseW, pauseReq)
	s.Equal(http.StatusOK, pauseW.Code)

	closeReq := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counterID+"/close", nil)
	closeReq.SetPathValue("id", counterID)
	closeReq = closeReq.WithContext(ctx)
	closeW := httptest.NewRecorder()
	counterHandler.Close(closeW, closeReq)
	s.Equal(http.StatusOK, closeW.Code)

	statsReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues/"+queueDef.ID+"/stats", nil)
	statsReq.SetPathValue("queue_id", queueDef.ID)
	statsReq = statsReq.WithContext(ctx)
	statsW := httptest.NewRecorder()
	statsHandler.GetStats(statsW, statsReq)
	s.Equal(http.StatusOK, statsW.Code)

	item := &models.QueueItem{
		QueueID:    queueDef.ID,
		Priority:   1,
		Status:     models.ItemStatusWaiting,
		TicketNo:   "VIP-002",
		Category:   "vip",
		Metadata:   "{}",
		JoinedAt:   time.Now(),
		CustomerID: "customer-2",
	}
	s.Require().NoError(s.itemRepo.Create(ctx, item))

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+item.ID+"/cancel", nil)
	cancelReq.SetPathValue("id", item.ID)
	cancelReq = cancelReq.WithContext(ctx)
	cancelW := httptest.NewRecorder()
	itemHandler.Cancel(cancelW, cancelReq)
	s.Equal(http.StatusOK, cancelW.Code)
}

func (s *HandlerSuite) TestQueueItemHandler_NoShowRequeueAndTransferFlow() {
	ctx := s.tenantCtx()
	queueA := &models.QueueDefinition{
		Name:           "queue-a",
		Active:         true,
		PriorityLevels: 3,
		MaxCapacity:    20,
		SLAMinutes:     30,
		Config:         "{}",
	}
	queueB := &models.QueueDefinition{
		Name:           "queue-b",
		Active:         true,
		PriorityLevels: 3,
		MaxCapacity:    20,
		SLAMinutes:     30,
		Config:         "{}",
	}
	s.Require().NoError(s.defRepo.Create(ctx, queueA))
	s.Require().NoError(s.defRepo.Create(ctx, queueB))

	counter := &models.QueueCounter{
		QueueID:    queueA.ID,
		Name:       "Desk 1",
		Status:     models.CounterStatusOpen,
		ServedBy:   "staff-1",
		Categories: `["vip"]`,
	}
	s.Require().NoError(s.counterRepo.Create(ctx, counter))

	item := &models.QueueItem{
		QueueID:    queueA.ID,
		Priority:   1,
		Status:     models.ItemStatusWaiting,
		TicketNo:   "VIP-100",
		Category:   "vip",
		Metadata:   "{}",
		JoinedAt:   time.Now(),
		CustomerID: "customer-100",
	}
	s.Require().NoError(s.itemRepo.Create(ctx, item))
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	itemHandler := NewQueueItemHandler(s.manager, NewRateLimiter(cache.NewInMemoryCache(), 100))

	noShowReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+item.ID+"/no-show", nil)
	noShowReq.SetPathValue("id", item.ID)
	noShowReq = noShowReq.WithContext(ctx)
	noShowW := httptest.NewRecorder()
	itemHandler.NoShow(noShowW, noShowReq)
	s.Equal(http.StatusOK, noShowW.Code)

	reloadedItem, err := s.itemRepo.GetByID(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusNoShow, reloadedItem.Status)
	s.Empty(reloadedItem.CounterID)

	reloadedCounter, err := s.counterRepo.GetByID(ctx, counter.ID)
	s.Require().NoError(err)
	s.Empty(reloadedCounter.CurrentItemID)

	requeueReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+item.ID+"/requeue", nil)
	requeueReq.SetPathValue("id", item.ID)
	requeueReq = requeueReq.WithContext(ctx)
	requeueW := httptest.NewRecorder()
	itemHandler.Requeue(requeueW, requeueReq)
	s.Equal(http.StatusOK, requeueW.Code)

	reloadedItem, err = s.itemRepo.GetByID(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, reloadedItem.Status)
	s.Equal(queueA.ID, reloadedItem.QueueID)

	transferReq := httptest.NewRequest(http.MethodPost, "/api/v1/items/"+item.ID+"/transfer", encodeBody(map[string]any{
		"queue_id": queueB.ID,
	}))
	transferReq.SetPathValue("id", item.ID)
	transferReq = transferReq.WithContext(ctx)
	transferW := httptest.NewRecorder()
	itemHandler.Transfer(transferW, transferReq)
	s.Equal(http.StatusOK, transferW.Code)

	transferredItem, err := s.itemRepo.GetByID(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, transferredItem.Status)
	s.Equal(queueB.ID, transferredItem.QueueID)
}

func (s *HandlerSuite) TestQueueItemHandler_TransferValidationAndNotFound() {
	ctx := s.tenantCtx()
	itemHandler := NewQueueItemHandler(s.manager, NewRateLimiter(cache.NewInMemoryCache(), 100))

	tests := []struct {
		name       string
		id         string
		body       any
		wantStatus int
	}{
		{
			name:       "invalid json returns bad request",
			id:         "item-1",
			body:       "{",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing queue id returns bad request",
			id:         "item-1",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing item returns not found",
			id:         "missing-item",
			body:       map[string]any{"queue_id": "missing-queue"},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var req *http.Request
			switch body := tc.body.(type) {
			case string:
				req = httptest.NewRequest(http.MethodPost, "/api/v1/items/"+tc.id+"/transfer", strings.NewReader(body))
			default:
				req = httptest.NewRequest(http.MethodPost, "/api/v1/items/"+tc.id+"/transfer", encodeBody(body))
			}
			req.SetPathValue("id", tc.id)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			itemHandler.Transfer(w, req)
			s.Equal(tc.wantStatus, w.Code)
		})
	}
}

func (s *HandlerSuite) TestQueueDefinitionAndCounterHandlers_ValidationAndErrorPaths() {
	ctx := s.tenantCtx()
	queueHandler := NewQueueDefinitionHandler(s.manager)
	counterHandler := NewQueueCounterHandler(s.manager)

	validQueue := &models.QueueDefinition{
		Name:           "ops-queue",
		Active:         true,
		PriorityLevels: 3,
		MaxCapacity:    5,
		SLAMinutes:     15,
		Config:         "{}",
	}
	s.Require().NoError(s.defRepo.Create(ctx, validQueue))

	tests := []struct {
		name       string
		exec       func() *httptest.ResponseRecorder
		wantStatus int
	}{
		{
			name: "queue create rejects invalid json",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/queues", strings.NewReader("{"))
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				queueHandler.Create(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "queue create requires name",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/queues", encodeBody(map[string]any{}))
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				queueHandler.Create(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "queue get missing returns not found",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/queues/missing", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				queueHandler.Get(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "queue update rejects invalid json",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPut, "/api/v1/queues/"+validQueue.ID, strings.NewReader("{"))
				req.SetPathValue("id", validQueue.ID)
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				queueHandler.Update(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "queue delete missing returns not found",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/queues/missing", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				queueHandler.Delete(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "counter create rejects invalid json",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/queues/"+validQueue.ID+"/counters",
					strings.NewReader("{"),
				)
				req.SetPathValue("queue_id", validQueue.ID)
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.Create(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "counter create requires name",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/queues/"+validQueue.ID+"/counters",
					encodeBody(map[string]any{}),
				)
				req.SetPathValue("queue_id", validQueue.ID)
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.Create(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "counter open rejects invalid json",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/counters/missing/open", strings.NewReader("{"))
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.Open(w, req)
				return w
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "counter open missing returns not found",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/counters/missing/open",
					encodeBody(map[string]any{"staff_id": "s1"}),
				)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.Open(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "counter call next with no items returns not found",
			exec: func() *httptest.ResponseRecorder {
				counter := &models.QueueCounter{
					QueueID:    validQueue.ID,
					Name:       "Desk Empty",
					Status:     models.CounterStatusOpen,
					Categories: `[]`,
				}
				s.Require().NoError(s.counterRepo.Create(ctx, counter))
				req := httptest.NewRequest(http.MethodPost, "/api/v1/counters/"+counter.ID+"/call-next", nil)
				req.SetPathValue("id", counter.ID)
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.CallNext(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "counter begin service missing returns not found",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/counters/missing/begin-service", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.BeginService(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "counter complete service missing returns not found",
			exec: func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/counters/missing/complete-service", nil)
				req.SetPathValue("id", "missing")
				req = req.WithContext(ctx)
				w := httptest.NewRecorder()
				counterHandler.CompleteService(w, req)
				return w
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			w := tc.exec()
			s.Equal(tc.wantStatus, w.Code)
		})
	}
}

func TestHTTPStatusForError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found maps to 404", err: business.ErrQueueItemNotFound, wantStatus: http.StatusNotFound},
		{name: "validation maps to 400", err: business.ErrInvalidTransition, wantStatus: http.StatusBadRequest},
		{name: "capacity maps to 409", err: business.ErrQueueFull, wantStatus: http.StatusConflict},
		{name: "default maps to 500", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, _ := httpStatusForError(tc.err)
			require.Equal(t, tc.wantStatus, status)
		})
	}
}
