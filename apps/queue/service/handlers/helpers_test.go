package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

func TestHandlerHelpers_MiddlewareAndErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("request id middleware uses existing header", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(RequestIDHeader, "req-321")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "req-321", rec.Header().Get(RequestIDHeader))
	})

	t.Run("request id middleware generates header when absent", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.NotEmpty(t, rec.Header().Get(RequestIDHeader))
	})

	t.Run("limit body size rejects oversized reads", func(t *testing.T) {
		t.Parallel()
		handler := LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.Error(t, err)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec,
			httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", MaxRequestBodySize+1))),
		)
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("require auth enforces claims", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		require.False(t, requireAuth(context.Background(), rec))
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		claims := &security.AuthenticationClaims{TenantID: "tenant", PartitionID: "partition"}
		claims.Subject = "user"
		rec = httptest.NewRecorder()
		require.True(t, requireAuth(claims.ClaimsToContext(context.Background()), rec))
	})

	t.Run("authz error mapping distinguishes invalid subject", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		writeAuthzError(rec, authorizer.ErrInvalidSubject)
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		rec = httptest.NewRecorder()
		writeAuthzError(rec, errors.New("forbidden"))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandlerHelpers_HTTPStatusAndJSONMappers(t *testing.T) {
	t.Parallel()

	statusCases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "queue not found",
			err:        business.ErrQueueNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "no waiting",
			err:        business.ErrNoWaitingItems,
			wantStatus: http.StatusNotFound,
			wantMsg:    "no waiting items",
		},
		{
			name:       "queue full",
			err:        business.ErrQueueFull,
			wantStatus: http.StatusConflict,
			wantMsg:    "queue is at maximum capacity",
		},
		{
			name:       "counter not open",
			err:        business.ErrCounterNotOpen,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "counter is not open",
		},
		{
			name:       "fallback",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
	}
	for _, tc := range statusCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStatus, gotMsg := httpStatusForError(tc.err)
			require.Equal(t, tc.wantStatus, gotStatus)
			require.Equal(t, tc.wantMsg, gotMsg)
		})
	}

	def := queueDefToJSON(
		&models.QueueDefinition{
			Name:           "main",
			Description:    "Main queue",
			Active:         true,
			PriorityLevels: 3,
			MaxCapacity:    10,
			SLAMinutes:     30,
			Config:         `{"kind":"walkin"}`,
		},
	)
	require.Equal(t, "main", def["name"])

	item := itemToJSON(
		&models.QueueItem{
			QueueID:  "queue-1",
			Priority: 2,
			Status:   models.ItemStatusWaiting,
			TicketNo: "A-001",
			Metadata: `{"source":"web"}`,
		},
	)
	require.Equal(t, "A-001", item["ticket_no"])

	counter := counterToJSON(
		&models.QueueCounter{
			QueueID:    "queue-1",
			Name:       "Desk 1",
			Status:     models.CounterStatusOpen,
			Categories: `["vip"]`,
		},
	)
	require.Equal(t, "Desk 1", counter["name"])
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	t.Run("nil limiter allows", func(t *testing.T) {
		t.Parallel()
		var limiter *RateLimiter
		require.True(t, limiter.Allow(context.Background()))
	})

	t.Run("tenant scoped limiter applies budget", func(t *testing.T) {
		t.Parallel()
		limiter := NewRateLimiter(cache.NewInMemoryCache(), 1)
		require.NotNil(t, limiter)

		claims := &security.AuthenticationClaims{TenantID: "tenant-a", PartitionID: "p1"}
		claims.Subject = "user"
		ctx := claims.ClaimsToContext(context.Background())

		require.True(t, limiter.Allow(ctx))
		require.False(t, limiter.Allow(ctx))
		require.True(t, limiter.Allow(context.Background()))
	})
}
