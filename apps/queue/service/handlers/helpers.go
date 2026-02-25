package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
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

		w.Header().Set(RequestIDHeader, requestID)

		ctx := r.Context()
		log := util.Log(ctx).WithField("request_id", requestID)
		ctx = util.ContextWithLogger(ctx, log)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ErrMissingAuth is returned when auth claims are missing from the request context.
var ErrMissingAuth = errors.New("authentication required")

// requireAuth validates auth claims exist and writes an error response if missing.
func requireAuth(ctx context.Context, w http.ResponseWriter) bool {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		http.Error(w, ErrMissingAuth.Error(), http.StatusUnauthorized)
		return false
	}

	return true
}

// httpStatusForError maps a business error to an HTTP status code and safe message.
func httpStatusForError(err error) (int, string) {
	switch {
	case errors.Is(err, business.ErrQueueNotFound),
		errors.Is(err, business.ErrQueueItemNotFound),
		errors.Is(err, business.ErrCounterNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, business.ErrNoWaitingItems):
		return http.StatusNotFound, "no waiting items"
	case errors.Is(err, business.ErrQueueFull):
		return http.StatusConflict, "queue is at maximum capacity"
	case errors.Is(err, business.ErrDuplicateQueueName):
		return http.StatusConflict, "queue name already exists"
	case errors.Is(err, business.ErrCounterBusy):
		return http.StatusConflict, "counter is currently serving another item"
	case errors.Is(err, business.ErrCounterNotOpen):
		return http.StatusBadRequest, "counter is not open"
	case errors.Is(err, business.ErrCounterNotServing):
		return http.StatusBadRequest, "counter is not serving"
	case errors.Is(err, business.ErrInvalidTransition):
		return http.StatusBadRequest, "invalid status transition"
	case errors.Is(err, business.ErrItemNotWaiting):
		return http.StatusBadRequest, "item is not waiting"
	case errors.Is(err, business.ErrItemNotNoShow):
		return http.StatusBadRequest, "item is not in no-show status"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
