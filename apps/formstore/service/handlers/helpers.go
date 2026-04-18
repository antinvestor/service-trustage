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

package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
)

// RequestIDHeader is the HTTP header for request ID propagation.
const RequestIDHeader = "X-Request-ID"

// LimitBodySize wraps an http.Handler with request body size limiting.
func LimitBodySize(next http.Handler, maxSize int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
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

// writeAuthzError writes an appropriate HTTP error response for authorisation failures.
func writeAuthzError(w http.ResponseWriter, err error) {
	if errors.Is(err, authorizer.ErrInvalidSubject) || errors.Is(err, authorizer.ErrInvalidObject) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	http.Error(w, err.Error(), http.StatusForbidden)
}

// httpStatusForError maps a business error to an HTTP status code and safe message.
func httpStatusForError(err error) (int, string) {
	switch {
	case errors.Is(err, business.ErrFormDefinitionNotFound),
		errors.Is(err, business.ErrFormSubmissionNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, business.ErrDuplicateFormID):
		return http.StatusConflict, "form_id already exists"
	case errors.Is(err, business.ErrInvalidFormData),
		errors.Is(err, business.ErrSchemaValidationFailed),
		errors.Is(err, business.ErrSubmissionTooLarge),
		errors.Is(err, business.ErrInvalidStatus):
		return http.StatusBadRequest, "invalid request data"
	case errors.Is(err, business.ErrFileUploadFailed):
		return http.StatusBadGateway, "file upload failed"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
