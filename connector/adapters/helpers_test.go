package adapters

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/antinvestor/service-trustage/connector"
)

func TestURLValidationAndHTTPClassification(t *testing.T) {
	t.Parallel()

	if err := validateExternalURL(context.Background(), "ftp://example.com"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
	if err := validateExternalURL(context.Background(), "http://localhost/path"); err == nil {
		t.Fatal("expected localhost to be blocked")
	}
	if err := validateExternalURL(context.Background(), "http://example.com/path"); err != nil {
		t.Fatalf("validateExternalURL() error = %v", err)
	}

	if !isPrivateIP(net.ParseIP("10.0.0.1")) {
		t.Fatal("expected private range")
	}
	if isPrivateIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public ip should not be private")
	}

	cases := []struct {
		status int
		class  connector.ErrorClass
	}{
		{status: http.StatusBadRequest, class: connector.ErrorFatal},
		{status: http.StatusTooManyRequests, class: connector.ErrorRetryable},
		{status: http.StatusBadGateway, class: connector.ErrorExternalDependency},
		{status: http.StatusPermanentRedirect, class: connector.ErrorRetryable},
	}

	for _, tc := range cases {

		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			t.Parallel()
			err := classifyHTTPStatus(tc.status, "body")
			if err == nil || err.Class != tc.class {
				t.Fatalf("classifyHTTPStatus(%d) = %+v", tc.status, err)
			}
		})
	}

	if classifyHTTPStatus(http.StatusOK, "ok") != nil {
		t.Fatal("200 should not produce execution error")
	}
	if got := truncateBody(strings.Repeat("x", maxErrorBodyLen+10)); len(got) <= maxErrorBodyLen {
		t.Fatalf("truncateBody() = %q", got)
	}
}

func TestExecuteAPIPostAndAdapterHelpers(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Idempotency-Key") != "idem-1" {
			t.Fatalf("idempotency = %q", r.Header.Get("Idempotency-Key"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"req-1","status":"queued"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	req := &connector.ExecuteRequest{
		Input:          map[string]any{"approver": "user-1", "title": "Approve"},
		Config:         map[string]any{"api_url": "https://example.com/approvals"},
		Credentials:    map[string]string{"api_key": "secret"},
		IdempotencyKey: "idem-1",
		Metadata:       map[string]string{"execution_id": "exec-1", "instance_id": "inst-1"},
	}

	parsed, execErr := executeAPIPost(context.Background(), client, req, map[string]any{"hello": "world"})
	if execErr != nil {
		t.Fatalf("executeAPIPost() error = %+v", execErr)
	}
	if parsed["id"] != "req-1" {
		t.Fatalf("parsed = %+v", parsed)
	}

	approval := NewApprovalRequestAdapter(client)
	resp, execErr := approval.Execute(context.Background(), req)
	if execErr != nil {
		t.Fatalf("approval.Execute() error = %+v", execErr)
	}
	if resp.Output["request_id"] != "req-1" {
		t.Fatalf("approval response = %+v", resp.Output)
	}

	payload := approval.buildPayload(req)
	if payload["execution_id"] != "exec-1" || payload["instance_id"] != "inst-1" {
		t.Fatalf("approval payload = %+v", payload)
	}

	badReq := &connector.ExecuteRequest{}
	if _, execErr = executeAPIPost(context.Background(), client, badReq, map[string]any{}); execErr == nil {
		t.Fatal("expected missing api_url error")
	}
}

func TestFormValidateAndTransformAdapters(t *testing.T) {
	t.Parallel()

	validate := NewFormValidateAdapter()
	success, execErr := validate.Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"fields":          map[string]any{"name": "alice", "age": float64(30)},
			"required_fields": []any{"name"},
			"field_types":     map[string]any{"age": "number"},
		},
	})
	if execErr != nil || success.Output["valid"] != true {
		t.Fatalf("validate success = %+v err=%+v", success, execErr)
	}

	if _, execErr = validate.Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"fields":          map[string]any{"name": ""},
			"required_fields": []any{"name"},
			"field_types":     map[string]any{"name": "string"},
		},
	}); execErr == nil || execErr.Code != "VALIDATION_FAILED" {
		t.Fatalf("expected validation failure, got %+v", execErr)
	}

	transform := NewDataTransformAdapter()
	resp, execErr := transform.Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"source":     map[string]any{"amount": float64(7)},
			"expression": "payload.amount",
			"mappings":   map[string]any{"amount": "payload.amount"},
		},
	})
	if execErr != nil {
		t.Fatalf("transform.Execute() error = %+v", execErr)
	}
	if resp.Output["result"].(float64) != 7 {
		t.Fatalf("transform result = %+v", resp.Output)
	}
	if resp.Output["data"].(map[string]any)["amount"].(float64) != 7 {
		t.Fatalf("transform data = %+v", resp.Output)
	}

	if _, execErr = transform.Execute(context.Background(), &connector.ExecuteRequest{
		Input: map[string]any{
			"source":   map[string]any{},
			"mappings": map[string]any{"broken": float64(3)},
		},
	}); execErr == nil || execErr.Code != "MAPPING_ERROR" {
		t.Fatalf("expected mapping error, got %+v", execErr)
	}
}

func TestExecuteAPIPost_ErrorClassification(t *testing.T) {
	t.Parallel()

	rateLimitedClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader("rate limited")),
			Header:     make(http.Header),
		}, nil
	})}

	_, execErr := executeAPIPost(context.Background(), rateLimitedClient, &connector.ExecuteRequest{
		Config: map[string]any{"api_url": "https://example.com/ratelimited"},
	}, map[string]any{"hello": "world"})
	if execErr == nil || execErr.Class != connector.ErrorRetryable {
		t.Fatalf("executeAPIPost() error = %+v", execErr)
	}

	failingClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	})}
	_, execErr = executeAPIPost(context.Background(), failingClient, &connector.ExecuteRequest{
		Config: map[string]any{"api_url": "https://example.com"},
	}, map[string]any{"hello": "world"})
	if execErr == nil || execErr.Code != "HTTP_ERROR" {
		t.Fatalf("executeAPIPost() network error = %+v", execErr)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (r roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
