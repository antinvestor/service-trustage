//nolint:testpackage // package-local tests cover unexported transport helpers intentionally.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/security"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
)

func TestConnectHelpers_AuthAndErrors(t *testing.T) {
	t.Parallel()

	t.Run("require auth rejects missing claims", func(t *testing.T) {
		t.Parallel()
		err := requireConnectAuth(context.Background())
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("code = %v", connect.CodeOf(err))
		}
	})

	t.Run("require auth accepts claims", func(t *testing.T) {
		t.Parallel()
		claims := &security.AuthenticationClaims{TenantID: "tenant", PartitionID: "partition"}
		claims.Subject = "subject"
		if err := requireConnectAuth(claims.ClaimsToContext(context.Background())); err != nil {
			t.Fatalf("requireConnectAuth() error = %v", err)
		}
	})

	t.Run("connectErrorForBusiness maps categories", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			err  error
			code connect.Code
		}{
			{name: "not found", err: business.ErrWorkflowNotFound, code: connect.CodeNotFound},
			{name: "invalid arg", err: business.ErrDSLValidationFailed, code: connect.CodeInvalidArgument},
			{name: "aborted", err: business.ErrInvalidToken, code: connect.CodeAborted},
			{name: "already exists", err: business.ErrWorkflowAlreadyActive, code: connect.CodeAlreadyExists},
			{name: "internal", err: errors.New("boom"), code: connect.CodeInternal},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if got := connect.CodeOf(connectErrorForBusiness(tc.err)); got != tc.code {
					t.Fatalf("code = %v want %v", got, tc.code)
				}
			})
		}
	})

	t.Run("connectLookupError maps record not found", func(t *testing.T) {
		t.Parallel()
		if got := connect.CodeOf(connectLookupError(gorm.ErrRecordNotFound, "missing")); got != connect.CodeNotFound {
			t.Fatalf("code = %v", got)
		}
		if got := connect.CodeOf(connectLookupError(errors.New("boom"), "missing")); got != connect.CodeInternal {
			t.Fatalf("code = %v", got)
		}
	})

	t.Run("duplicate detection works for wrapped and textual errors", func(t *testing.T) {
		t.Parallel()
		if !isDuplicateRecordError(gorm.ErrDuplicatedKey) {
			t.Fatal("expected duplicated key to be detected")
		}
		if !isDuplicateRecordError(errors.New("duplicate key value violates unique constraint")) {
			t.Fatal("expected duplicate text to be detected")
		}
		if isDuplicateRecordError(nil) {
			t.Fatal("nil should not be duplicate")
		}
	})
}

func TestConnectHelpers_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	value, err := structpb.NewStruct(map[string]any{"name": "alice"})
	if err != nil {
		t.Fatalf("NewStruct() error = %v", err)
	}

	raw, err := rawJSONFromStruct(value)
	if err != nil {
		t.Fatalf("rawJSONFromStruct() error = %v", err)
	}

	decoded, err := structFromJSONString(string(raw))
	if err != nil {
		t.Fatalf("structFromJSONString() error = %v", err)
	}
	if decoded.GetFields()["name"].GetStringValue() != "alice" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestConnectHelpers_LossyFallbackPreservesInvalidJSON(t *testing.T) {
	t.Parallel()

	value := lossyStructFromJSONString("{bad")
	if value == nil || value.GetFields()["raw_json"].GetStringValue() != "{bad" {
		t.Fatalf("lossyStructFromJSONString() = %+v", value)
	}
}

func TestConnectHelpers_WorkflowStatusConversions(t *testing.T) {
	t.Parallel()

	if workflowStatusToProto(models.WorkflowStatusArchived) == 0 {
		t.Fatal("archived should map to a non-default enum")
	}
	if filter, err := workflowStatusFilter(0); err != nil || filter != "" {
		t.Fatal("unspecified workflow filter should be empty")
	}
	if _, err := workflowStatusFilter(999); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("workflowStatusFilter invalid code = %v", connect.CodeOf(err))
	}
}

func TestConnectHelpers_InstanceStatusMappings(t *testing.T) {
	t.Parallel()

	if instanceStatusFilter(0) != "" {
		t.Fatal("unspecified instance filter should be empty")
	}

	cases := []struct {
		name       string
		model      models.WorkflowInstanceStatus
		proto      int32
		filterFrom int32
		wantFilter string
	}{
		{
			name:       "running",
			model:      models.InstanceStatusRunning,
			proto:      int32(instanceStatusToProto(models.InstanceStatusRunning)),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_RUNNING),
			wantFilter: string(models.InstanceStatusRunning),
		},
		{
			name:       "completed",
			model:      models.InstanceStatusCompleted,
			proto:      int32(instanceStatusToProto(models.InstanceStatusCompleted)),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_COMPLETED),
			wantFilter: string(models.InstanceStatusCompleted),
		},
		{
			name:       "failed",
			model:      models.InstanceStatusFailed,
			proto:      int32(instanceStatusToProto(models.InstanceStatusFailed)),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_FAILED),
			wantFilter: string(models.InstanceStatusFailed),
		},
		{
			name:       "cancelled",
			model:      models.InstanceStatusCancelled,
			proto:      int32(instanceStatusToProto(models.InstanceStatusCancelled)),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_CANCELLED),
			wantFilter: string(models.InstanceStatusCancelled),
		},
		{
			name:       "suspended",
			model:      models.InstanceStatusSuspended,
			proto:      int32(instanceStatusToProto(models.InstanceStatusSuspended)),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_SUSPENDED),
			wantFilter: string(models.InstanceStatusSuspended),
		},
		{
			name:       "default",
			model:      "unknown",
			proto:      int32(runtimev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED),
			filterFrom: int32(runtimev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED),
			wantFilter: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := instanceStatusToProto(tc.model); int32(got) != tc.proto {
				t.Fatalf("instanceStatusToProto() = %v want %v", got, tc.proto)
			}
			if got := instanceStatusFilter(runtimev1.InstanceStatus(tc.filterFrom)); got != tc.wantFilter {
				t.Fatalf("instanceStatusFilter() = %q want %q", got, tc.wantFilter)
			}
		})
	}
}

func TestConnectHelpers_ExecutionStatusMappings(t *testing.T) {
	t.Parallel()

	if executionStatusFilter(0) != "" {
		t.Fatal("unspecified execution filter should be empty")
	}

	cases := []struct {
		name       string
		model      models.ExecutionStatus
		proto      runtimev1.ExecutionStatus
		wantFilter string
	}{
		{
			name:       "pending",
			model:      models.ExecStatusPending,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_PENDING,
			wantFilter: string(models.ExecStatusPending),
		},
		{
			name:       "dispatched",
			model:      models.ExecStatusDispatched,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_DISPATCHED,
			wantFilter: string(models.ExecStatusDispatched),
		},
		{
			name:       "running",
			model:      models.ExecStatusRunning,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_RUNNING,
			wantFilter: string(models.ExecStatusRunning),
		},
		{
			name:       "completed",
			model:      models.ExecStatusCompleted,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
			wantFilter: string(models.ExecStatusCompleted),
		},
		{
			name:       "failed",
			model:      models.ExecStatusFailed,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_FAILED,
			wantFilter: string(models.ExecStatusFailed),
		},
		{
			name:       "fatal",
			model:      models.ExecStatusFatal,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_FATAL,
			wantFilter: string(models.ExecStatusFatal),
		},
		{
			name:       "timed out",
			model:      models.ExecStatusTimedOut,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_TIMED_OUT,
			wantFilter: string(models.ExecStatusTimedOut),
		},
		{
			name:       "invalid input",
			model:      models.ExecStatusInvalidInputContract,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_INPUT_CONTRACT,
			wantFilter: string(models.ExecStatusInvalidInputContract),
		},
		{
			name:       "invalid output",
			model:      models.ExecStatusInvalidOutputContract,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT,
			wantFilter: string(models.ExecStatusInvalidOutputContract),
		},
		{
			name:       "stale",
			model:      models.ExecStatusStale,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_STALE,
			wantFilter: string(models.ExecStatusStale),
		},
		{
			name:       "retry scheduled",
			model:      models.ExecStatusRetryScheduled,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_RETRY_SCHEDULED,
			wantFilter: string(models.ExecStatusRetryScheduled),
		},
		{
			name:       "waiting",
			model:      models.ExecStatusWaiting,
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_WAITING,
			wantFilter: string(models.ExecStatusWaiting),
		},
		{
			name:       "default",
			model:      "unknown",
			proto:      runtimev1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED,
			wantFilter: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := executionStatusToProto(tc.model); got != tc.proto {
				t.Fatalf("executionStatusToProto() = %v want %v", got, tc.proto)
			}
			if got := executionStatusFilter(tc.proto); got != tc.wantFilter {
				t.Fatalf("executionStatusFilter() = %q want %q", got, tc.wantFilter)
			}
		})
	}
}

func TestConnectHelpers_ProtoConversions(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	def := &models.WorkflowDefinition{
		WorkflowVersion: 3,
		Name:            "wf",
		Status:          models.WorkflowStatusActive,
		DSLBlob:         `{"version":"1.0","name":"wf","steps":[]}`,
		TimeoutSeconds:  60,
	}
	def.CreatedAt = now
	def.ModifiedAt = now

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 3,
		CurrentState:    "start",
		Status:          models.InstanceStatusRunning,
		Revision:        4,
		Metadata:        `{"key":"value"}`,
		ScopeType:       "parallel",
		ScopeIndex:      2,
	}
	instance.CreatedAt = now
	instance.ModifiedAt = now

	exec := &models.WorkflowStateExecution{
		InstanceID:       "inst-1",
		State:            "start",
		StateVersion:     2,
		Attempt:          3,
		Status:           models.ExecStatusWaiting,
		InputSchemaHash:  "input-hash",
		InputPayload:     `{"hello":"world"}`,
		OutputSchemaHash: "output-hash",
		TraceID:          "trace-1",
	}
	exec.CreatedAt = now
	exec.ModifiedAt = now

	scope := &models.WorkflowScopeRun{
		ParentExecutionID: "exec-1",
		ParentState:       "fanout",
		ScopeType:         "foreach",
		Status:            "running",
		WaitAll:           true,
		TotalChildren:     3,
		CompletedChildren: 1,
		FailedChildren:    0,
		NextChildIndex:    2,
		MaxConcurrency:    4,
		ItemVar:           "item",
		IndexVar:          "idx",
		ItemsPayload:      `[1,2,3]`,
		ResultsPayload:    `{"ok":true}`,
	}
	scope.CreatedAt = now
	scope.ModifiedAt = now

	wait := &models.WorkflowSignalWait{
		ExecutionID: "exec-1",
		State:       "wait",
		SignalName:  "approved",
		Status:      "waiting",
		Attempts:    2,
	}
	wait.CreatedAt = now
	wait.ModifiedAt = now

	message := &models.WorkflowSignalMessage{
		SignalName: "approved",
		Status:     "pending",
		Attempts:   1,
		Payload:    `{"value":"ok"}`,
	}
	message.CreatedAt = now
	message.ModifiedAt = now

	output := &models.WorkflowStateOutput{
		ExecutionID: "exec-1",
		State:       "start",
		SchemaHash:  "hash",
		Payload:     `{"done":true}`,
	}
	output.CreatedAt = now

	event := &models.WorkflowAuditEvent{
		EventType:   "state.completed",
		State:       "start",
		FromState:   "start",
		ToState:     "finish",
		ExecutionID: "exec-1",
		TraceID:     "trace-1",
		Payload:     `{"ok":true}`,
	}
	event.CreatedAt = now

	if got := workflowDefinitionToProto(def); got.GetName() != "wf" || got.GetVersion() != 3 {
		t.Fatalf("workflowDefinitionToProto() = %+v", got)
	}
	if got := workflowInstanceToProto(instance); got.GetScopeIndex() != 2 || got.GetStatus() == 0 {
		t.Fatalf("workflowInstanceToProto() = %+v", got)
	}
	if got := workflowExecutionToProto(exec, `{"done":true}`, true); got.GetAttempt() != 3 || got.GetOutput() == nil {
		t.Fatalf("workflowExecutionToProto() = %+v", got)
	}
	if got := timelineEntryToProto(event); got.GetExecutionId() != "exec-1" {
		t.Fatalf("timelineEntryToProto() = %+v", got)
	}
	if got := runtimeTimelineEntryToProto(event, true); got.GetPayload() == nil {
		t.Fatalf("runtimeTimelineEntryToProto() = %+v", got)
	}
	if got := stateOutputToProto(output, true); got.GetPayload() == nil {
		t.Fatalf("stateOutputToProto() = %+v", got)
	}
	if got := scopeRunToProto(scope, true); got.GetTotalChildren() != 3 || got.GetItemsPayload() == nil {
		t.Fatalf("scopeRunToProto() = %+v", got)
	}
	if got := signalWaitToProto(wait); got.GetAttempts() != 2 {
		t.Fatalf("signalWaitToProto() = %+v", got)
	}
	if got := signalMessageToProto(message, true); got.GetPayload() == nil {
		t.Fatalf("signalMessageToProto() = %+v", got)
	}
	record := eventRecordToProto("evt-1", "user.created", "api", "idem", map[string]any{"x": 1})
	if record.GetPayload().GetFields()["x"].GetNumberValue() != 1 {
		t.Fatalf("eventRecordToProto() = %+v", record)
	}
}

func TestEmbeddedSpecHandler(t *testing.T) {
	t.Parallel()

	spec := []byte("openapi: 3.0.0\n")
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	EmbeddedSpecHandler(spec).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("content-type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "openapi") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestLossyStructFromJSONString_Empty(t *testing.T) {
	t.Parallel()
	if lossyStructFromJSONString("") != nil {
		t.Fatal("expected nil for empty json")
	}
}

func TestStructFromMap_Nil(t *testing.T) {
	t.Parallel()
	value := structFromMap(nil)
	if value == nil {
		t.Fatal("expected empty struct")
	}
	if len(value.GetFields()) != 0 {
		encoded, _ := json.Marshal(value.AsMap())
		t.Fatalf("expected empty struct, got %s", encoded)
	}
}
