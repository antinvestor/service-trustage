package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
)

// MaxRequestBodySize is the maximum allowed request body size (1 MB).
const MaxRequestBodySize = 1 << 20 // 1 MB

// RequestIDHeader is the HTTP header for request ID propagation.
const RequestIDHeader = "X-Request-ID"

// LimitBodySize wraps an http.Handler with request body size limiting.
func LimitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware propagates or generates a request ID and adds it to the
// logger context for log correlation.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = util.IDString()
		}

		// Add to response header for client correlation.
		w.Header().Set(RequestIDHeader, requestID)

		// Add request ID to logging context.
		ctx := r.Context()
		log := util.Log(ctx).WithField("request_id", requestID)
		ctx = util.ContextWithLogger(ctx, log)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ErrMissingAuth is returned when auth claims are missing from the request context.
var ErrMissingAuth = errors.New("missing authentication claims")

// requireAuth validates auth claims exist and writes an error response if missing.
func requireAuth(ctx context.Context, w http.ResponseWriter) bool {
	if security.ClaimsFromContext(ctx) == nil {
		http.Error(w, ErrMissingAuth.Error(), http.StatusUnauthorized)
		return false
	}

	return true
}

// httpStatusForError maps a business error to an HTTP status code and safe message.
// Internal errors return a generic message to avoid leaking implementation details.
func httpStatusForError(err error) (int, string) {
	switch {
	case errors.Is(err, business.ErrWorkflowNotFound),
		errors.Is(err, business.ErrInstanceNotFound),
		errors.Is(err, business.ErrExecutionNotFound),
		errors.Is(err, business.ErrSchemaNotFound),
		errors.Is(err, business.ErrTriggerNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, business.ErrInputContractViolation),
		errors.Is(err, business.ErrOutputContractViolation),
		errors.Is(err, business.ErrDSLValidationFailed),
		errors.Is(err, business.ErrInvalidWorkflowStatus):
		// Validation errors are safe to return to clients.
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, business.ErrStaleExecution),
		errors.Is(err, business.ErrInvalidToken):
		return http.StatusConflict, "stale execution or invalid token"
	case errors.Is(err, business.ErrWorkflowAlreadyActive):
		return http.StatusConflict, "workflow already active"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
