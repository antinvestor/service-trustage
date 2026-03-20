//nolint:testpackage // package-local tests cover unexported HTTP helpers intentionally.
package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	framecache "github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
)

func TestHTTPHelpers_MiddlewareAndErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("request id middleware reuses existing header", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(RequestIDHeader, "req-xyz")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "req-xyz", rec.Header().Get(RequestIDHeader))
	})

	t.Run("request id middleware generates header", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.NotEmpty(t, rec.Header().Get(RequestIDHeader))
	})

	t.Run("limit body size rejects oversized read", func(t *testing.T) {
		t.Parallel()
		handler := LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err == nil {
				t.Fatal("expected oversized body read to fail")
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec,
			httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", MaxRequestBodySize+1))),
		)
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("require auth validates claims", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		require.False(t, requireAuth(context.Background(), rec))
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		claims := &security.AuthenticationClaims{TenantID: "tenant", PartitionID: "partition"}
		claims.Subject = "user"
		rec = httptest.NewRecorder()
		require.True(t, requireAuth(claims.ClaimsToContext(context.Background()), rec))
	})

	statusCases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "workflow not found",
			err:        business.ErrWorkflowNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "instance not found",
			err:        business.ErrInstanceNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "dsl invalid",
			err:        business.ErrDSLValidationFailed,
			wantStatus: http.StatusBadRequest,
			wantMsg:    business.ErrDSLValidationFailed.Error(),
		},
		{
			name:       "duplicate active",
			err:        business.ErrWorkflowAlreadyActive,
			wantStatus: http.StatusConflict,
			wantMsg:    "workflow already active",
		},
		{
			name:       "stale token",
			err:        business.ErrInvalidToken,
			wantStatus: http.StatusConflict,
			wantMsg:    "stale execution or invalid token",
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
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	t.Run("nil limiter allows", func(t *testing.T) {
		t.Parallel()
		var limiter *RateLimiter
		require.True(t, limiter.Allow(context.Background()))
	})

	t.Run("named limiter enforces tenant window", func(t *testing.T) {
		t.Parallel()
		limiter := NewNamedRateLimiter(framecache.NewInMemoryCache(), "trustage:test", 1)
		require.NotNil(t, limiter)

		claims := &security.AuthenticationClaims{TenantID: "tenant-a", PartitionID: "p1"}
		claims.Subject = "user"
		ctx := claims.ClaimsToContext(context.Background())

		require.True(t, limiter.Allow(ctx))
		require.False(t, limiter.Allow(ctx))
		require.True(t, limiter.Allow(context.Background()))
	})
}
