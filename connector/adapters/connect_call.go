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

package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	connectCallType        = "connect.call"
	connectCallDisplayName = "Connect RPC Call"
)

var (
	connectServicePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$`)
	connectRPCPattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ConnectRoute maps a fully-qualified service name prefix to the base URL
// that serves it.
type ConnectRoute struct {
	Prefix  string
	BaseURL string
}

// ConnectRoutes is the operator-configured routing table for connect.call,
// ordered longest prefix first.
type ConnectRoutes []ConnectRoute

// ParseConnectRoutes parses "<prefix>=<baseURL>,..." (the
// TRUSTAGE_CONNECT_ROUTES format). Blank entries are ignored; a base URL
// must be absolute http(s) with a host and no query or fragment.
func ParseConnectRoutes(spec string) (ConnectRoutes, error) {
	var routes ConnectRoutes

	seen := map[string]bool{}

	for _, entry := range strings.Split(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		prefix, base, found := strings.Cut(entry, "=")
		prefix = strings.TrimSpace(prefix)
		base = strings.TrimSpace(base)

		if !found || prefix == "" || base == "" {
			return nil, fmt.Errorf("connect route %q: want <service-prefix>=<base-url>", entry)
		}

		if seen[prefix] {
			return nil, fmt.Errorf("connect route %q: duplicate prefix", prefix)
		}
		seen[prefix] = true

		parsed, err := url.Parse(base)
		if err != nil {
			return nil, fmt.Errorf("connect route %q: invalid base url: %w", prefix, err)
		}

		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf("connect route %q: base url must be absolute http(s)", prefix)
		}

		if parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("connect route %q: base url must not carry a query or fragment", prefix)
		}

		routes = append(routes, ConnectRoute{Prefix: prefix, BaseURL: strings.TrimRight(base, "/")})
	}

	sort.SliceStable(routes, func(i, j int) bool {
		return len(routes[i].Prefix) > len(routes[j].Prefix)
	})

	return routes, nil
}

// Resolve returns the base URL of the longest prefix matching service.
func (r ConnectRoutes) Resolve(service string) (string, bool) {
	for _, route := range r {
		if strings.HasPrefix(service, route.Prefix) {
			return route.BaseURL, true
		}
	}

	return "", false
}

// ConnectCallAdapter makes a Connect unary call with the JSON codec to a
// service on an operator-configured route. Requests go through the frame
// HTTP client, so they carry the service's workload token. Only configured
// base URLs are ever called, which is why the private-network block that
// guards user-supplied URLs does not apply.
type ConnectCallAdapter struct {
	client *http.Client
	routes ConnectRoutes
}

// NewConnectCallAdapter creates a ConnectCallAdapter.
func NewConnectCallAdapter(client *http.Client, routes ConnectRoutes) *ConnectCallAdapter {
	return &ConnectCallAdapter{client: client, routes: routes}
}

func (a *ConnectCallAdapter) Type() string        { return connectCallType }
func (a *ConnectCallAdapter) DisplayName() string { return connectCallDisplayName }

func (a *ConnectCallAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["service", "rpc"],
		"properties": {
			"service": {"type": "string", "description": "Fully-qualified service, e.g. finance.v1.FinanceService"},
			"rpc": {"type": "string", "description": "Method name, e.g. GetProposal"},
			"request": {"type": "object", "description": "Request message in proto JSON"},
			"endpoint": {"type": "string", "format": "uri", "description": "Base URL of the service, e.g. http://service-loans.fintech.svc:80; an in-cluster host only. An operator route for the service takes precedence."}
		}
	}`)
}

func (a *ConnectCallAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *ConnectCallAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object", "description": "The reply message in proto JSON"}`)
}

func (a *ConnectCallAdapter) Validate(req *connector.ExecuteRequest) error {
	_, err := connectCallInput(req)
	return err
}

// connectCall is a validated connect.call step input.
type connectCall struct {
	service  string
	rpc      string
	request  map[string]any
	endpoint string
}

func connectCallInput(req *connector.ExecuteRequest) (*connectCall, error) {
	call := &connectCall{request: map[string]any{}}
	call.service, _ = req.Input["service"].(string)
	call.rpc, _ = req.Input["rpc"].(string)

	if !connectServicePattern.MatchString(call.service) {
		return nil, fmt.Errorf("service must be a fully-qualified service name, got %q", call.service)
	}

	if !connectRPCPattern.MatchString(call.rpc) {
		return nil, fmt.Errorf("rpc must be a method name, got %q", call.rpc)
	}

	if raw, ok := req.Input["request"]; ok && raw != nil {
		obj, isObj := raw.(map[string]any)
		if !isObj {
			return nil, errors.New("request must be an object")
		}
		call.request = obj
	}

	if raw, ok := req.Input["endpoint"]; ok && raw != nil {
		endpoint, isString := raw.(string)
		if !isString {
			return nil, errors.New("endpoint must be a string")
		}
		base, err := connectEndpoint(endpoint)
		if err != nil {
			return nil, err
		}
		call.endpoint = base
	}

	return call, nil
}

// connectEndpoint checks a workflow-supplied service base URL. The call
// carries trustage's own service token, so only an in-cluster host it
// already trusts is accepted: a workflow can name its service, never point
// the token anywhere else.
func connectEndpoint(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("endpoint: %w", err)
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("endpoint %q must be an absolute http(s) URL", raw)
	}

	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("endpoint %q must be a bare base URL", raw)
	}

	if !isTrustedHostSuffix(strings.ToLower(parsed.Hostname())) {
		return "", fmt.Errorf("endpoint %q is not an in-cluster host trustage trusts", raw)
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

func (a *ConnectCallAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	call, err := connectCallInput(req)
	if err != nil {
		return nil, &connector.ExecutionError{Class: connector.ErrorFatal, Code: "INVALID_INPUT", Message: err.Error()}
	}
	service, rpc, request := call.service, call.rpc, call.request

	// An operator route wins; otherwise the workflow names its service.
	base, ok := a.routes.Resolve(service)
	if !ok {
		base, ok = call.endpoint, call.endpoint != ""
	}
	if !ok {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "CONNECT_NO_ROUTE",
			Message: fmt.Sprintf("no endpoint or TRUSTAGE_CONNECT_ROUTES entry serves %s", service),
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "MARSHAL_ERROR",
			Message: fmt.Sprintf("marshal request: %v", err),
		}
	}

	ctx, cancel := withAdapterTimeout(ctx)
	defer cancel()

	endpoint := base + "/" + service + "/" + rpc

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("build request: %v", err),
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    "CONNECT_TRANSPORT",
			Message: fmt.Sprintf("%s/%s: %v", service, rpc, err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("read reply: %v", err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, classifyConnectError(resp.StatusCode, respBody)
	}

	output := map[string]any{}
	if len(bytes.TrimSpace(respBody)) > 0 {
		if unmarshalErr := json.Unmarshal(respBody, &output); unmarshalErr != nil {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "CONNECT_INVALID_REPLY",
				Message: fmt.Sprintf("%s/%s replied with a non-object body: %v", service, rpc, unmarshalErr),
			}
		}
	}

	return &connector.ExecuteResponse{Output: output, RawBody: respBody}, nil
}

// connectRetryableCodes are the Connect codes that describe a transient
// condition; every other code fails the step at once.
//
//nolint:gochecknoglobals // immutable lookup table
var connectRetryableCodes = map[string]connector.ErrorClass{
	"unavailable":        connector.ErrorExternalDependency,
	"deadline_exceeded":  connector.ErrorExternalDependency,
	"resource_exhausted": connector.ErrorRetryable,
	"aborted":            connector.ErrorRetryable,
	"internal":           connector.ErrorRetryable,
	"unknown":            connector.ErrorRetryable,
}

func classifyConnectError(status int, body []byte) *connector.ExecutionError {
	var connectErr struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if json.Unmarshal(body, &connectErr) == nil && connectErr.Code != "" {
		class, retry := connectRetryableCodes[connectErr.Code]
		if !retry {
			class = connector.ErrorFatal
		}

		return &connector.ExecutionError{
			Class:   class,
			Code:    "CONNECT_" + strings.ToUpper(connectErr.Code),
			Message: truncateBody(connectErr.Message),
			Details: map[string]any{"connect_code": connectErr.Code, "http_status": status},
		}
	}

	details := map[string]any{"http_status": status}

	switch {
	case status == http.StatusTooManyRequests:
		return &connector.ExecutionError{
			Class: connector.ErrorRetryable, Code: "HTTP_429", Message: "rate limited", Details: details,
		}
	case status >= http.StatusInternalServerError:
		return &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    fmt.Sprintf("HTTP_%d", status),
			Message: truncateBody(string(body)),
			Details: details,
		}
	default:
		return &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    fmt.Sprintf("HTTP_%d", status),
			Message: truncateBody(string(body)),
			Details: details,
		}
	}
}
