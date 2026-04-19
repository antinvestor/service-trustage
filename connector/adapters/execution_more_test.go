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

//nolint:testpackage // package-local adapter tests exercise unexported helpers intentionally.
package adapters

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/antinvestor/service-trustage/connector"
)

func TestWithAdapterTimeout_BoundedByDefault(t *testing.T) {
	t.Parallel()

	// Verify withAdapterTimeout applies a deadline within defaultAdapterHTTPTimeout.
	ctx, cancel := withAdapterTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline after withAdapterTimeout")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > defaultAdapterHTTPTimeout {
		t.Fatalf("expected deadline within %v, got %v", defaultAdapterHTTPTimeout, remaining)
	}
}

func TestWithAdapterTimeout_RespectsShortParentDeadline(t *testing.T) {
	t.Parallel()

	// A parent with a shorter deadline must not be extended.
	short := 5 * time.Second
	parent, parentCancel := context.WithTimeout(context.Background(), short)
	defer parentCancel()

	ctx, cancel := withAdapterTimeout(parent)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	// The resulting deadline must be ≤ the parent deadline.
	parentDeadline, _ := parent.Deadline()
	if deadline.After(parentDeadline) {
		t.Fatalf("child deadline %v is after parent deadline %v", deadline, parentDeadline)
	}
}

func TestWebhookAdapter_TimeoutPropagated(t *testing.T) {
	t.Parallel()

	// The adapter must surface a context error when the server is too slow.
	blocker := make(chan struct{})
	defer close(blocker)

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-blocker:
			return nil, errors.New("unreachable")
		}
	})}

	// Give it a context that expires very quickly so the test runs fast.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, execErr := NewWebhookAdapter(client).Execute(ctx, &connector.ExecuteRequest{
		Input: map[string]any{
			"url":    "https://example.com/slow",
			"method": http.MethodPost,
		},
	})
	if execErr == nil {
		t.Fatal("expected an error from a blocking server with a short context")
	}
	if execErr.Code != "HTTP_ERROR" {
		t.Fatalf("expected HTTP_ERROR, got %s", execErr.Code)
	}
}

func TestAdapters_DisplayNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  string
		want string
	}{
		{name: "ai", got: NewAIChatAdapter().DisplayName(), want: aiChatDisplayName},
		{
			name: "approval",
			got:  NewApprovalRequestAdapter(&http.Client{}).DisplayName(),
			want: approvalRequestDisplayName,
		},
		{name: "form validate", got: NewFormValidateAdapter().DisplayName(), want: formValidateDisplayName},
		{name: "http", got: NewHTTPAdapter(&http.Client{}).DisplayName(), want: httpDisplayName},
		{name: "log", got: NewLogEntryAdapter().DisplayName(), want: logEntryDisplayName},
		{
			name: "notification",
			got:  NewNotificationSendAdapter(&http.Client{}).DisplayName(),
			want: notificationSendDisplayName,
		},
		{
			name: "notification status",
			got:  NewNotificationStatusAdapter(&http.Client{}).DisplayName(),
			want: notificationStatusDisplayName,
		},
		{
			name: "payment",
			got:  NewPaymentInitiateAdapter(&http.Client{}).DisplayName(),
			want: paymentInitiateDisplayName,
		},
		{
			name: "payment verify",
			got:  NewPaymentVerifyAdapter(&http.Client{}).DisplayName(),
			want: paymentVerifyDisplayName,
		},
		{name: "transform", got: NewDataTransformAdapter().DisplayName(), want: dataTransformDisplayName},
		{name: "webhook", got: NewWebhookAdapter(&http.Client{}).DisplayName(), want: webhookDisplayName},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("DisplayName() = %q want %q", tc.got, tc.want)
			}
		})
	}
}

func TestHTTPAndWebhookAdapters_Execute(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Idempotency-Key") != "idem-1" {
			t.Fatalf("idempotency header = %q", r.Header.Get("Idempotency-Key"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     http.Header{"X-Test": []string{"yes"}},
		}, nil
	})}

	httpResp, execErr := NewHTTPAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"url":         "https://example.com/resource",
			"method":      http.MethodPost,
			"query":       map[string]any{"page": "1"},
			"body":        map[string]any{"hello": "world"},
			"headers":     map[string]any{"X-Custom": "value"},
			"auth_header": "Bearer secret",
		},
		IdempotencyKey: "idem-1",
	})
	if execErr != nil {
		t.Fatalf("HTTPAdapter.Execute() error = %+v", execErr)
	}
	if httpResp.Output["status_code"].(int) != http.StatusOK {
		t.Fatalf("HTTPAdapter.Execute() output = %+v", httpResp.Output)
	}

	webhookResp, execErr := NewWebhookAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"url":    "https://example.com/webhook",
			"method": http.MethodPost,
			"body":   map[string]any{"hello": "world"},
			"headers": map[string]any{
				"X-Webhook": "yes",
			},
		},
		IdempotencyKey: "idem-1",
	})
	if execErr != nil {
		t.Fatalf("WebhookAdapter.Execute() error = %+v", execErr)
	}
	if webhookResp.Output["status_code"].(int) != http.StatusOK {
		t.Fatalf("WebhookAdapter.Execute() output = %+v", webhookResp.Output)
	}
}

func TestNotificationAndPaymentAdapters_Execute(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(r.URL.Path, "/notifications/"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(
					strings.NewReader(`{"status":"delivered","delivered_at":"2026-03-19T00:00:00Z"}`),
				),
				Header: make(http.Header),
			}, nil
		case strings.Contains(r.URL.Path, "/payments/"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"completed","amount":100,"currency":"USD"}`)),
				Header:     make(http.Header),
			}, nil
		case strings.Contains(r.URL.String(), "notifications"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"notif-1","status":"queued"}`)),
				Header:     make(http.Header),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"pay-1","status":"processing"}`)),
				Header:     make(http.Header),
			}, nil
		}
	})}

	notifyResp, execErr := NewNotificationSendAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"recipient": "user@example.com",
			"channel":   "email",
			"subject":   "Hello",
			"body":      "World",
		},
		Config: map[string]any{"api_url": "https://example.com/notifications"},
	})
	if execErr != nil || notifyResp.Output["notification_id"] != "notif-1" {
		t.Fatalf("NotificationSendAdapter.Execute() resp=%+v err=%+v", notifyResp, execErr)
	}

	statusResp, execErr := NewNotificationStatusAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input:       map[string]any{"notification_id": "notif-1"},
		Config:      map[string]any{"api_url": "https://example.com/notifications"},
		Credentials: map[string]string{"api_key": "secret"},
	})
	if execErr != nil || statusResp.Output["status"] != "delivered" {
		t.Fatalf("NotificationStatusAdapter.Execute() resp=%+v err=%+v", statusResp, execErr)
	}

	paymentResp, execErr := NewPaymentInitiateAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"amount":    100.0,
			"currency":  "USD",
			"recipient": "256700000000",
			"reference": "ref-1",
		},
		Config: map[string]any{"api_url": "https://example.com/payments"},
	})
	if execErr != nil || paymentResp.Output["payment_id"] != "pay-1" {
		t.Fatalf("PaymentInitiateAdapter.Execute() resp=%+v err=%+v", paymentResp, execErr)
	}

	verifyResp, execErr := NewPaymentVerifyAdapter(client).Execute(context.Background(), &connector.ExecuteRequest{
		Input:       map[string]any{"payment_id": "pay-1"},
		Config:      map[string]any{"api_url": "https://example.com/payments"},
		Credentials: map[string]string{"api_key": "secret"},
	})
	if execErr != nil || verifyResp.Output["status"] != "completed" {
		t.Fatalf("PaymentVerifyAdapter.Execute() resp=%+v err=%+v", verifyResp, execErr)
	}
}

func TestLogAndAIHelpers(t *testing.T) {
	t.Parallel()

	logResp, execErr := NewLogEntryAdapter().Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"level":   "info",
			"message": "hello",
			"data":    map[string]any{"ok": true},
		},
	})
	if execErr != nil || logResp.Output["logged"] != true {
		t.Fatalf("LogEntryAdapter.Execute() resp=%+v err=%+v", logResp, execErr)
	}

	ai := NewAIChatAdapter()
	_, execErr = ai.Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{"messages": []any{}},
	})
	if execErr == nil || execErr.Code != "CONFIG_ERROR" {
		t.Fatalf("AIChatAdapter.Execute() missing config err = %+v", execErr)
	}

	_, execErr = ai.Execute(context.Background(), &connector.ExecuteRequest{
		Input:  map[string]any{"messages": []any{}},
		Config: map[string]any{"provider": "openai", "model": "gpt-4o"},
	})
	if execErr == nil || execErr.Code != "CREDENTIALS_ERROR" {
		t.Fatalf("AIChatAdapter.Execute() missing creds err = %+v", execErr)
	}

	classCases := []struct {
		name string
		err  error
		code string
	}{
		{name: "auth", err: errors.New("invalid api key"), code: "AUTH_ERROR"},
		{name: "rate limit", err: errors.New("rate limit exceeded"), code: "RATE_LIMITED"},
		{name: "timeout", err: errors.New("deadline exceeded"), code: "TIMEOUT"},
		{name: "cancelled", err: errors.New("context canceled"), code: "CANCELLED"},
		{name: "dns", err: errors.New("no such host"), code: "CONNECTION_ERROR"},
		{name: "provider", err: errors.New("503 service unavailable"), code: "PROVIDER_ERROR"},
		{name: "request", err: errors.New("400 invalid model"), code: "REQUEST_ERROR"},
		{name: "fallback", err: errors.New("something else"), code: "LLM_ERROR"},
	}

	for _, tc := range classCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			classifiedErr := classifyLLMError(tc.err)
			if classifiedErr == nil || classifiedErr.Code != tc.code {
				t.Fatalf("classifyLLMError() = %+v", classifiedErr)
			}
		})
	}

	if !containsAny("Hello WORLD", "world") {
		t.Fatal("containsAny should be case insensitive")
	}
	if len(truncateError(strings.Repeat("x", 600))) <= 512 {
		t.Fatal("truncateError should append ellipsis after truncation")
	}
}
