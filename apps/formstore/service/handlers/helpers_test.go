//nolint:testpackage // package-local tests cover unexported handler helpers intentionally.
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

	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
)

func TestHandlerHelpers_MiddlewareAndErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("request id middleware reuses supplied header", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(RequestIDHeader, "req-123")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "req-123", rec.Header().Get(RequestIDHeader))
		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("request id middleware generates missing header", func(t *testing.T) {
		t.Parallel()
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.NotEmpty(t, rec.Header().Get(RequestIDHeader))
	})

	t.Run("require auth rejects missing claims", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		ok := requireAuth(context.Background(), rec)
		require.False(t, ok)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("require auth accepts claims", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		claims := &security.AuthenticationClaims{TenantID: "tenant", PartitionID: "partition"}
		claims.Subject = "user"
		ok := requireAuth(claims.ClaimsToContext(context.Background()), rec)
		require.True(t, ok)
	})

	t.Run("authz error maps invalid subject to unauthorized", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		writeAuthzError(rec, authorizer.ErrInvalidSubject)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("authz error maps generic error to forbidden", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		writeAuthzError(rec, errors.New("denied"))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("limit body size caps reads", func(t *testing.T) {
		t.Parallel()
		handler := LimitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err == nil {
				t.Fatal("expected oversized body read to fail")
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}), 8)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789")))
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
}

func TestHandlerHelpers_HTTPStatusForError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "definition not found",
			err:        business.ErrFormDefinitionNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "submission not found",
			err:        business.ErrFormSubmissionNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "resource not found",
		},
		{
			name:       "duplicate",
			err:        business.ErrDuplicateFormID,
			wantStatus: http.StatusConflict,
			wantMsg:    "form_id already exists",
		},
		{
			name:       "invalid request",
			err:        business.ErrInvalidFormData,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid request data",
		},
		{
			name:       "upload failed",
			err:        business.ErrFileUploadFailed,
			wantStatus: http.StatusBadGateway,
			wantMsg:    "file upload failed",
		},
		{
			name:       "fallback",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
	}

	for _, tc := range cases {
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

	t.Run("tenant scoped limiter enforces configured window", func(t *testing.T) {
		t.Parallel()
		limiter := NewRateLimiter(cache.NewInMemoryCache(), 1)
		require.NotNil(t, limiter)

		claims := &security.AuthenticationClaims{TenantID: "tenant-a", PartitionID: "p1"}
		claims.Subject = "user-a"
		ctx := claims.ClaimsToContext(context.Background())

		require.True(t, limiter.Allow(ctx))
		require.False(t, limiter.Allow(ctx))
		require.True(t, limiter.Allow(context.Background()))
	})
}
