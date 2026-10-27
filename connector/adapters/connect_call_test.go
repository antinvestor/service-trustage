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

package adapters_test

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/connector/adapters"
)

type recordedCall struct {
	method   string
	path     string
	headers  http.Header
	body     map[string]any
	rawBody  []byte
	callSeen bool
}

func connectServer(t *testing.T, status int, contentType, body string) (*httptest.Server, *recordedCall) {
	t.Helper()

	rec := &recordedCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.callSeen = true
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.headers = r.Header.Clone()
		rec.rawBody, _ = io.ReadAll(r.Body)
		_ = json.Unmarshal(rec.rawBody, &rec.body)

		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv, rec
}

func connectAdapter(t *testing.T, routes string) *adapters.ConnectCallAdapter {
	t.Helper()

	parsed, err := adapters.ParseConnectRoutes(routes)
	require.NoError(t, err)

	return adapters.NewConnectCallAdapter(&http.Client{}, parsed)
}

func connectInput(service, rpc string, request map[string]any) *connector.ExecuteRequest {
	return &connector.ExecuteRequest{Input: map[string]any{
		"service": service,
		"rpc":     rpc,
		"request": request,
	}}
}

func isRetry(class connector.ErrorClass) bool {
	return class == connector.ErrorRetryable || class == connector.ErrorExternalDependency
}

func TestConnectCall_SuccessPostsConnectJSONAndReturnsReply(t *testing.T) {
	srv, rec := connectServer(t, http.StatusOK, "application/json",
		`{"proposal":{"id":"p-1","state":"OPEN"},"count":"3"}`)
	adapter := connectAdapter(t, "finance.v1.="+srv.URL+"/api/")

	resp, execErr := adapter.Execute(context.Background(), connectInput(
		"finance.v1.ProposalService", "GetProposal", map[string]any{"proposal_id": "p-1"},
	))
	require.Nil(t, execErr)

	require.Equal(t, http.MethodPost, rec.method)
	require.Equal(t, "/api/finance.v1.ProposalService/GetProposal", rec.path)
	require.Equal(t, "application/json", rec.headers.Get("Content-Type"))
	require.Equal(t, "1", rec.headers.Get("Connect-Protocol-Version"))
	require.Equal(t, map[string]any{"proposal_id": "p-1"}, rec.body)

	require.Equal(t, map[string]any{
		"proposal": map[string]any{"id": "p-1", "state": "OPEN"},
		"count":    "3",
	}, resp.Output)
}

func TestConnectCall_EmptyReplyAndMissingRequest(t *testing.T) {
	srv, rec := connectServer(t, http.StatusOK, "application/json", ``)
	adapter := connectAdapter(t, "settlement.v1.="+srv.URL)

	resp, execErr := adapter.Execute(context.Background(), &connector.ExecuteRequest{Input: map[string]any{
		"service": "settlement.v1.SettlementService",
		"rpc":     "Tick",
	}})
	require.Nil(t, execErr)
	require.Equal(t, map[string]any{}, resp.Output)
	require.JSONEq(t, `{}`, string(rec.rawBody))
}

func TestConnectCall_LongestPrefixWins(t *testing.T) {
	general, generalRec := connectServer(t, http.StatusOK, "application/json", `{"from":"general"}`)
	specific, specificRec := connectServer(t, http.StatusOK, "application/json", `{"from":"specific"}`)
	adapter := connectAdapter(t,
		"finance.="+general.URL+", finance.v1.ProposalService="+specific.URL+",other.v1.=http://other.invalid")

	resp, execErr := adapter.Execute(context.Background(),
		connectInput("finance.v1.ProposalService", "Get", map[string]any{}))
	require.Nil(t, execErr)
	require.Equal(t, "specific", resp.Output["from"])
	require.True(t, specificRec.callSeen)
	require.False(t, generalRec.callSeen)

	resp, execErr = adapter.Execute(context.Background(),
		connectInput("finance.v2.LedgerService", "Get", map[string]any{}))
	require.Nil(t, execErr)
	require.Equal(t, "general", resp.Output["from"])
}

func TestConnectCall_NoRouteFailsWithoutRetry(t *testing.T) {
	adapter := connectAdapter(t, "finance.v1.=http://127.0.0.1:1")

	_, execErr := adapter.Execute(context.Background(),
		connectInput("settlement.v1.SettlementService", "Tick", map[string]any{}))
	require.NotNil(t, execErr)
	require.Equal(t, connector.ErrorFatal, execErr.Class)
	require.Equal(t, "CONNECT_NO_ROUTE", execErr.Code)
}

func TestConnectCall_ConnectErrorCodes(t *testing.T) {
	tests := []struct {
		code      string
		status    int
		wantRetry bool
	}{
		{"unavailable", http.StatusServiceUnavailable, true},
		{"deadline_exceeded", http.StatusGatewayTimeout, true},
		{"resource_exhausted", http.StatusTooManyRequests, true},
		{"aborted", http.StatusConflict, true},
		{"internal", http.StatusInternalServerError, true},
		{"unknown", http.StatusInternalServerError, true},
		{"not_found", http.StatusNotFound, false},
		{"invalid_argument", http.StatusBadRequest, false},
		{"failed_precondition", http.StatusBadRequest, false},
		{"permission_denied", http.StatusForbidden, false},
		{"unauthenticated", http.StatusUnauthorized, false},
		{"already_exists", http.StatusConflict, false},
		{"unimplemented", http.StatusNotImplemented, false},
		{"canceled", http.StatusRequestTimeout, false},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			srv, _ := connectServer(t, tc.status, "application/json",
				`{"code":"`+tc.code+`","message":"boom `+tc.code+`"}`)
			adapter := connectAdapter(t, "finance.v1.="+srv.URL)

			_, execErr := adapter.Execute(context.Background(),
				connectInput("finance.v1.ProposalService", "Get", map[string]any{}))
			require.NotNil(t, execErr)
			require.Equal(t, tc.wantRetry, isRetry(execErr.Class), "class %s", execErr.Class)
			require.Contains(t, execErr.Message, "boom "+tc.code)
			require.Equal(t, tc.code, execErr.Details["connect_code"])
		})
	}
}

func TestConnectCall_NonJSONErrorsClassifiedByStatus(t *testing.T) {
	tests := []struct {
		status    int
		wantRetry bool
	}{
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusInternalServerError, true},
		{http.StatusTooManyRequests, true},
		{http.StatusNotFound, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
	}

	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv, _ := connectServer(t, tc.status, "text/html", "<html>gateway</html>")
			adapter := connectAdapter(t, "finance.v1.="+srv.URL)

			_, execErr := adapter.Execute(context.Background(),
				connectInput("finance.v1.ProposalService", "Get", map[string]any{}))
			require.NotNil(t, execErr)
			require.Equal(t, tc.wantRetry, isRetry(execErr.Class), "class %s", execErr.Class)
		})
	}
}

func TestConnectCall_TransportFailureRetries(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close()

	adapter := connectAdapter(t, "finance.v1.="+base)

	_, execErr := adapter.Execute(context.Background(),
		connectInput("finance.v1.ProposalService", "Get", map[string]any{}))
	require.NotNil(t, execErr)
	require.True(t, isRetry(execErr.Class))
}

func TestConnectCall_InvalidReplyFails(t *testing.T) {
	srv, _ := connectServer(t, http.StatusOK, "application/json", `[1,2]`)
	adapter := connectAdapter(t, "finance.v1.="+srv.URL)

	_, execErr := adapter.Execute(context.Background(),
		connectInput("finance.v1.ProposalService", "Get", map[string]any{}))
	require.NotNil(t, execErr)
	require.Equal(t, connector.ErrorFatal, execErr.Class)
}

func TestConnectCall_ValidateInput(t *testing.T) {
	adapter := connectAdapter(t, "finance.v1.=http://finance.local")

	tests := []struct {
		name  string
		input map[string]any
		ok    bool
	}{
		{
			"valid",
			map[string]any{"service": "finance.v1.ProposalService", "rpc": "Get", "request": map[string]any{}},
			true,
		},
		{"request optional", map[string]any{"service": "finance.v1.ProposalService", "rpc": "Get"}, true},
		{"missing service", map[string]any{"rpc": "Get"}, false},
		{"missing rpc", map[string]any{"service": "finance.v1.ProposalService"}, false},
		{"unqualified service", map[string]any{"service": "ProposalService", "rpc": "Get"}, false},
		{"path in service", map[string]any{"service": "finance.v1/../admin.Svc", "rpc": "Get"}, false},
		{"path in rpc", map[string]any{"service": "finance.v1.ProposalService", "rpc": "Get/../../x"}, false},
		{
			"request not object",
			map[string]any{"service": "finance.v1.ProposalService", "rpc": "Get", "request": "x"},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Validate(&connector.ExecuteRequest{Input: tc.input})
			if tc.ok {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)

			_, execErr := adapter.Execute(context.Background(), &connector.ExecuteRequest{Input: tc.input})
			require.NotNil(t, execErr)
			require.Equal(t, connector.ErrorFatal, execErr.Class)
		})
	}
}

func TestParseConnectRoutes(t *testing.T) {
	routes, err := adapters.ParseConnectRoutes("")
	require.NoError(t, err)
	require.Empty(t, routes)

	routes, err = adapters.ParseConnectRoutes(
		" finance.v1.=http://finance:8080/ , settlement.v1.=https://settle.internal/rpc ,",
	)
	require.NoError(t, err)

	base, ok := routes.Resolve("finance.v1.FinanceService")
	require.True(t, ok)
	require.Equal(t, "http://finance:8080", base)

	base, ok = routes.Resolve("settlement.v1.SettlementService")
	require.True(t, ok)
	require.Equal(t, "https://settle.internal/rpc", base)

	_, ok = routes.Resolve("notification.v1.NotificationService")
	require.False(t, ok)

	for _, bad := range []string{
		"finance.v1.",
		"=http://x",
		"finance.v1.=",
		"finance.v1.=ftp://x",
		"finance.v1.=http://",
		"finance.v1.=http://x?q=1",
		"finance.v1.=http://a,finance.v1.=http://b",
	} {
		_, err = adapters.ParseConnectRoutes(bad)
		require.Error(t, err, bad)
	}
}

// inClusterAdapter sends every connection to srv, so a workflow can name an
// in-cluster host and the test still reaches the local server.
func inClusterAdapter(t *testing.T, srv *httptest.Server, routes string) *adapters.ConnectCallAdapter {
	t.Helper()

	parsed, err := adapters.ParseConnectRoutes(routes)
	require.NoError(t, err)

	addr := srv.Listener.Addr().String()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}}

	return adapters.NewConnectCallAdapter(client, parsed)
}

func endpointInput(endpoint string) *connector.ExecuteRequest {
	req := connectInput("loans.v1.LoanManagementService", "LoanJobRun", map[string]any{"job": "X"})
	req.Input["endpoint"] = endpoint

	return req
}

// A workflow names its own service's in-cluster endpoint; trustage needs no
// route for it.
func TestConnectCall_WorkflowEndpointReachesItsService(t *testing.T) {
	srv, rec := connectServer(t, http.StatusOK, "application/json", `{"processed":1}`)
	adapter := inClusterAdapter(t, srv, "")

	resp, execErr := adapter.Execute(context.Background(), endpointInput("http://service-loans.fintech.svc:80"))
	require.Nil(t, execErr)
	require.Equal(t, "/loans.v1.LoanManagementService/LoanJobRun", rec.path)
	require.Equal(t, map[string]any{"processed": float64(1)}, resp.Output)
}

// An operator route for the service still wins over the workflow's endpoint.
func TestConnectCall_OperatorRouteOverridesTheEndpoint(t *testing.T) {
	srv, rec := connectServer(t, http.StatusOK, "application/json", `{}`)
	adapter := inClusterAdapter(t, srv, "loans.v1.=http://override.ops.svc/base")

	_, execErr := adapter.Execute(context.Background(), endpointInput("http://service-loans.fintech.svc:80"))
	require.Nil(t, execErr)
	require.Equal(t, "/base/loans.v1.LoanManagementService/LoanJobRun", rec.path)
}

// The call carries trustage's own token, so a workflow can only name an
// in-cluster host, never point the token anywhere else.
func TestConnectCall_EndpointMustBeATrustedInClusterHost(t *testing.T) {
	adapter := connectAdapter(t, "")

	for _, endpoint := range []string{
		"https://attacker.example.com",
		"http://169.254.169.254",
		"http://user:pass@service-loans.fintech.svc",
		"http://service-loans.fintech.svc/path",
		"http://service-loans.fintech.svc?x=1",
		"ftp://service-loans.fintech.svc",
		"service-loans.fintech.svc",
	} {
		require.Error(t, adapter.Validate(endpointInput(endpoint)), endpoint)
		_, execErr := adapter.Execute(context.Background(), endpointInput(endpoint))
		require.NotNil(t, execErr, endpoint)
		require.Equal(t, "INVALID_INPUT", execErr.Code, endpoint)
	}

	require.NoError(t, adapter.Validate(endpointInput("https://service-loans.fintech.svc.cluster.local:8443")))
}
