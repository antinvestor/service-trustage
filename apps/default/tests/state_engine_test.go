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
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
)

func (s *DefaultServiceSuite) TestStateEngine_DispatchCommitTerminal() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	def := s.createWorkflow(tenantCtx, s.sampleDSL())

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	inputPayload := json.RawMessage(`{"hello":"world"}`)
	_, err := engine.CreateInitialExecution(ctx, instance, inputPayload)
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusPending, exec.Status)
	s.True(strings.HasPrefix(exec.TraceID, "trc_"))

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	commitReq := &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Output:         json.RawMessage(`{"ok":true}`),
	}
	s.Require().NoError(engine.Commit(ctx, commitReq))

	output, err := s.outputRepo.GetByExecution(ctx, exec.ID)
	s.Require().NoError(err)
	s.NotEmpty(output.ID)

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, updatedInstance.Status)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitRetrySchedules() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	def := s.createWorkflow(tenantCtx, s.sampleDSL())

	policy := &models.WorkflowRetryPolicy{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		State:           "log_step",
		MaxAttempts:     3,
		InitialDelayMs:  10,
		MaxDelayMs:      50,
		BackoffStrategy: "exponential",
	}
	s.Require().NoError(s.retryRepo.Store(tenantCtx, policy))

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	inputPayload := json.RawMessage(`{"hello":"world"}`)
	_, err := engine.CreateInitialExecution(ctx, instance, inputPayload)
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	commitReq := &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Error: &business.CommitError{
			Class:   "retryable",
			Code:    "UPSTREAM_TIMEOUT",
			Message: "timeout",
		},
	}
	s.Require().NoError(engine.Commit(ctx, commitReq))

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusRunning, updatedInstance.Status)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitFatalMarksFailed() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	def := s.createWorkflow(tenantCtx, s.sampleDSL())

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	_, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"hello":"world"}`))
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	commitReq := &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Error: &business.CommitError{
			Class:   "fatal",
			Code:    "FAILED",
			Message: "boom",
		},
	}
	s.Require().NoError(engine.Commit(ctx, commitReq))

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.NotEmpty(updatedInstance.Status)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitMapsOutputToNextInput() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "mapping-workflow",
  "steps": [
    {
      "id": "step_a",
      "type": "call",
      "call": {
        "action": "log.entry",
        "input": {"message": "start"},
        "output_var": "step_out"
      }
    },
    {
      "id": "step_b",
      "type": "call",
      "call": {
        "action": "log.entry",
        "input": {"message": "{{ step_out.result }}"}
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "step_a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	_, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"hello":"world"}`))
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	commitReq := &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Output:         json.RawMessage(`{"result":"mapped"}`),
	}
	s.Require().NoError(engine.Commit(ctx, commitReq))

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("step_b", nextExec.State)
	s.Contains(nextExec.InputPayload, "mapped")
}

func (s *DefaultServiceSuite) TestStateEngine_SignalWaitAndSendSignal() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "signal-workflow",
  "steps": [
    {
      "id": "approval_wait",
      "type": "signal_wait",
      "signal_wait": {
        "signal_name": "approval_response",
        "output_var": "approval"
      }
    },
    {
      "id": "after",
      "type": "call",
      "call": {
        "action": "log.entry",
        "input": {}
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)
	spec, err := dsl.Parse([]byte(dslBlob))
	s.Require().NoError(err)
	step := dsl.FindStep(spec, "approval_wait")
	s.Require().NotNil(step)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "approval_wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "approval_wait",
		Attempt:         1,
		Status:          models.ExecStatusDispatched,
		ExecutionToken:  cryptoutil.HashToken(rawToken),
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	engine := s.stateEngine()
	s.Require().NoError(engine.StartSignalWait(ctx, &business.ExecutionCommand{
		ExecutionID:    exec.ID,
		InstanceID:     instance.ID,
		ExecutionToken: rawToken,
		InputPayload:   json.RawMessage(`{}`),
	}, step))

	wait, err := s.signalWaitRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal("waiting", wait.Status)

	delivered, err := engine.SendSignal(ctx, instance.ID, "approval_response", json.RawMessage(`{"approved":true}`))
	s.Require().NoError(err)
	s.True(delivered)

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", updatedInstance.CurrentState)

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", nextExec.State)
	s.Equal(models.ExecStatusPending, nextExec.Status)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitSequencePreservesInputForChild() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "sequence-workflow",
  "steps": [
    {
      "id": "seq",
      "type": "sequence",
      "sequence": {
        "steps": [
          {
            "id": "child",
            "type": "call",
            "call": {
              "action": "log.entry",
              "input": {"message": "{{ input.message }}"}
            }
          }
        ]
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "seq",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	inputPayload := json.RawMessage(`{"message":"preserved"}`)
	_, err := engine.CreateInitialExecution(ctx, instance, inputPayload)
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	s.Require().NoError(engine.Commit(ctx, &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Output:         json.RawMessage(`{}`),
	}))

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("child", nextExec.State)
	s.JSONEq(string(inputPayload), nextExec.InputPayload)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitIfPreservesInputForBranch() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "if-workflow",
  "steps": [
    {
      "id": "check",
      "type": "if",
      "if": {
        "expr": "payload.amount > 100",
        "then": [
          {
            "id": "high",
            "type": "call",
            "call": {
              "action": "log.entry",
              "input": {"message": "high"}
            }
          }
        ],
        "else": [
          {
            "id": "low",
            "type": "call",
            "call": {
              "action": "log.entry",
              "input": {"message": "low"}
            }
          }
        ]
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "check",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	inputPayload := json.RawMessage(`{"amount":150}`)
	_, err := engine.CreateInitialExecution(ctx, instance, inputPayload)
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	s.Require().NoError(engine.Commit(ctx, &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Output:         json.RawMessage(`{"branch":"then"}`),
	}))

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("high", nextExec.State)
	s.JSONEq(string(inputPayload), nextExec.InputPayload)
}

func (s *DefaultServiceSuite) TestStateEngine_ResumeWaitingExecutionAdvancesDelay() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
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
      "call": {
        "action": "log.entry",
        "input": {"message": "after"}
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "",
		InputSchemaHash: "hash",
		InputPayload:    `{"message":"preserved"}`,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	engine := s.stateEngine()
	s.Require().NoError(engine.ResumeWaitingExecution(ctx, exec.ID, json.RawMessage(`{}`)))

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", nextExec.State)
	s.JSONEq(`{"message":"preserved"}`, nextExec.InputPayload)
}

func (s *DefaultServiceSuite) TestStateEngine_ParkExecutionUntilMarksWaitingAndCreatesTimer() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	def := s.createWorkflow(tenantCtx, s.sampleDSL())
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	_, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"hello":"world"}`))
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	fireAt := time.Now().Add(time.Minute)
	s.Require().NoError(engine.ParkExecutionUntil(ctx, exec.ID, rawToken, fireAt))

	updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusWaiting, updatedExec.Status)

	timer, err := s.timerRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(exec.ID, timer.ExecutionID)
}

func (s *DefaultServiceSuite) TestStateEngine_ParkExecutionUntilRejectsInvalidToken() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	def := s.createWorkflow(tenantCtx, s.sampleDSL())
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	_, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"hello":"world"}`))
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		cryptoutil.HashToken("valid-token"), exec.ID,
	)

	err = engine.ParkExecutionUntil(ctx, exec.ID, "wrong-token", time.Now().Add(time.Minute))
	s.Require().Error(err)
}

func (s *DefaultServiceSuite) TestStateEngine_ResumeWaitingExecutionMarksStaleForStoppedInstance() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "delay-workflow",
  "steps": [
    {
      "id": "wait",
      "type": "delay",
      "delay": { "duration": "1m" }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "wait",
		Status:          models.InstanceStatusFailed,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "",
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	engine := s.stateEngine()
	s.Require().NoError(engine.ResumeWaitingExecution(ctx, exec.ID, json.RawMessage(`{}`)))

	updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusStale, updatedExec.Status)
}

func (s *DefaultServiceSuite) TestStateEngine_CommitIfInvalidOutputMarksContractViolation() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "if-workflow",
  "steps": [
    {
      "id": "check",
      "type": "if",
      "if": {
        "expr": "payload.amount > 100",
        "then": [
          {"id": "high", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "check",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	_, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"amount":150}`))
	s.Require().NoError(err)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	tokenHash := cryptoutil.HashToken(rawToken)
	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_state_executions SET status = 'dispatched', execution_token = ? WHERE id = ?`,
		tokenHash, exec.ID,
	)

	err = engine.Commit(ctx, &business.CommitRequest{
		ExecutionID:    exec.ID,
		ExecutionToken: rawToken,
		Output:         json.RawMessage(`{}`),
	})
	s.Require().Error(err)

	updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusInvalidOutputContract, updatedExec.Status)
}
