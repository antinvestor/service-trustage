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

package tests_test

import (
	"encoding/json"

	commonv1 "buf.build/gen/go/antinvestor/common/protocolbuffers/go/common/v1"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	signalv1 "github.com/antinvestor/service-trustage/gen/go/signal/v1"
)

func (s *DefaultServiceSuite) TestRuntimeConnectServer_ListExecutionsAndGetInstanceRun() {
	ctx := s.tenantCtx()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        `{"tenant":"x"}`,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	waitingExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token-a",
		InputSchemaHash: "hash-a",
		InputPayload:    `{"hello":"world"}`,
		TraceID:         "trace-1",
	}
	s.Require().NoError(s.execRepo.Create(ctx, waitingExec))

	failedExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "notify",
		Attempt:         2,
		Status:          models.ExecStatusFailed,
		ExecutionToken:  "token-b",
		InputSchemaHash: "hash-b",
		InputPayload:    `{"bye":"world"}`,
		TraceID:         "trace-2",
		ErrorClass:      "fatal",
		ErrorMessage:    "boom",
	}
	s.Require().NoError(s.execRepo.Create(ctx, failedExec))

	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: failedExec.ID,
		InstanceID:  instance.ID,
		State:       failedExec.State,
		SchemaHash:  "out",
		Payload:     `{"ok":false}`,
	}))
	s.Require().NoError(s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  instance.ID,
		ExecutionID: failedExec.ID,
		EventType:   "state.failed",
		State:       failedExec.State,
		Payload:     `{"reason":"boom"}`,
		TraceID:     failedExec.TraceID,
	}))
	s.Require().NoError(s.scopeRepo.Create(ctx, &models.WorkflowScopeRun{
		ParentExecutionID: failedExec.ID,
		ParentInstanceID:  instance.ID,
		ParentState:       "fanout",
		ScopeType:         "parallel",
		Status:            "running",
		WaitAll:           true,
		TotalChildren:     2,
		ResultsPayload:    `[null,null]`,
	}))
	s.Require().NoError(s.signalWaitRepo.Create(ctx, &models.WorkflowSignalWait{
		ExecutionID: waitingExec.ID,
		InstanceID:  instance.ID,
		State:       waitingExec.State,
		SignalName:  "approved",
		OutputVar:   "approval",
		Status:      "waiting",
	}))
	s.Require().NoError(s.signalMsgRepo.Create(ctx, &models.WorkflowSignalMessage{
		TargetInstanceID: instance.ID,
		SignalName:       "approved",
		Payload:          `{"ok":true}`,
		Status:           "pending",
	}))

	server := handlers.NewRuntimeConnectServer(
		s.instanceRepo,
		s.execRepo,
		s.outputRepo,
		s.auditRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.stateEngine(),
	)

	listResp, err := server.ListExecutions(
		ctx,
		&runtimev1.ListExecutionsRequest{
			InstanceId: instance.ID,
			Search: &commonv1.SearchRequest{
				Cursor: &commonv1.PageCursor{Limit: 10},
			},
		},
	)
	s.Require().NoError(err)
	s.Len(listResp.GetItems(), 2)

	runResp, err := server.GetInstanceRun(
		ctx,
		&runtimev1.GetInstanceRunRequest{
			InstanceId:      instance.ID,
			ExecutionLimit:  10,
			TimelineLimit:   10,
			IncludePayloads: true,
		},
	)
	s.Require().NoError(err)
	s.Equal(instance.ID, runResp.GetInstance().GetId())
	s.Len(runResp.GetExecutions(), 2)
	s.Len(runResp.GetTimeline(), 1)
	s.Len(runResp.GetOutputs(), 1)
	s.Len(runResp.GetScopeRuns(), 1)
	s.Len(runResp.GetSignalWaits(), 1)
	s.Len(runResp.GetSignalMessages(), 1)
	s.Equal("retry_execution", runResp.GetResumeStrategy())
}

func (s *DefaultServiceSuite) TestSignalConnectServer_StoresPendingSignal() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        "{}",
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	server := handlers.NewSignalConnectServer(s.stateEngine())
	resp, err := server.SendSignal(
		ctx,
		&signalv1.SendSignalRequest{
			InstanceId: instance.ID,
			SignalName: "approved",
			Payload:    mustStructFromMap(map[string]any{"ok": true}),
		},
	)
	s.Require().NoError(err)
	s.False(resp.GetDelivered())

	messages, err := s.signalMsgRepo.ListByInstance(ctx, instance.ID, 10)
	s.Require().NoError(err)
	s.Len(messages, 1)

	var payload map[string]any
	s.Require().NoError(json.Unmarshal([]byte(messages[0].Payload), &payload))
	s.Equal(true, payload["ok"])
}

func (s *DefaultServiceSuite) TestRuntimeConnectServer_ResumeExecution() {
	ctx := s.tenantCtx()

	waitingDSL := `{
  "version": "1.0",
  "name": "delay-workflow",
  "steps": [
    {
      "id": "wait",
      "type": "delay",
      "delay": { "duration": "1m" }
    },
    {
      "id": "after",
      "type": "call",
      "call": { "action": "log.entry", "input": { "message": "after" } }
    }
  ]
}`

	def := s.createWorkflow(ctx, waitingDSL)
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        "{}",
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	waitingExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "",
		InputSchemaHash: "hash",
		InputPayload:    `{"message":"preserved"}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, waitingExec))

	failedExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "after",
		Attempt:         2,
		Status:          models.ExecStatusFailed,
		ExecutionToken:  "retry-token",
		InputSchemaHash: "hash",
		InputPayload:    `{"message":"retry"}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, failedExec))

	server := handlers.NewRuntimeConnectServer(
		s.instanceRepo,
		s.execRepo,
		s.outputRepo,
		s.auditRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.stateEngine(),
	)

	waitingResp, err := server.ResumeExecution(
		ctx,
		&runtimev1.ResumeExecutionRequest{
			ExecutionId: waitingExec.ID,
			Payload:     mustStructFromMap(map[string]any{}),
		},
	)
	s.Require().NoError(err)
	s.Equal("resumed_waiting_execution", waitingResp.GetAction())

	resumedExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", resumedExec.State)
	s.JSONEq(`{"message":"preserved"}`, resumedExec.InputPayload)

	retryResp, err := server.ResumeExecution(
		ctx,
		&runtimev1.ResumeExecutionRequest{
			ExecutionId: failedExec.ID,
		},
	)
	s.Require().NoError(err)
	s.Equal("created_retry_execution", retryResp.GetAction())
	s.Equal(runtimev1.ExecutionStatus_EXECUTION_STATUS_PENDING, retryResp.GetExecution().GetStatus())
}
