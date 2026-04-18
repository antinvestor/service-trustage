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

//nolint:testpackage // package-local tests exercise unexported engine internals intentionally.
package business

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
)

func (s *BusinessSuite) TestStateEngine_CreateDispatchCommitAndTerminalPaths() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	definition := s.createWorkflow(tenantCtx, s.sampleDSL())

	instance := &models.WorkflowInstance{
		WorkflowName:    definition.Name,
		WorkflowVersion: definition.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	engine := s.stateEngine()
	inputPayload := json.RawMessage(`{"hello":"world"}`)
	cmd, err := engine.CreateInitialExecution(ctx, instance, inputPayload)
	s.Require().NoError(err)
	s.True(strings.HasPrefix(cmd.TraceID, "trc_"))

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)

	dispatchCmd, err := engine.Dispatch(ctx, exec)
	s.Require().NoError(err)
	s.Equal(exec.ID, dispatchCmd.ExecutionID)

	s.Require().NoError(engine.Commit(ctx, &CommitRequest{
		ExecutionID:    dispatchCmd.ExecutionID,
		ExecutionToken: dispatchCmd.ExecutionToken,
		Output:         json.RawMessage(`{"ok":true}`),
	}))

	output, err := s.outputRepo.GetByExecution(ctx, exec.ID)
	s.Require().NoError(err)
	s.NotEmpty(output.ID)

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, updatedInstance.Status)
}

func (s *BusinessSuite) TestStateEngine_CommitRetryAndFatal() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	var definition *models.WorkflowDefinition

	cases := []struct {
		name          string
		errorClass    string
		wantInstState models.WorkflowInstanceStatus
	}{
		{
			name:          "retryable leaves instance running",
			errorClass:    "retryable",
			wantInstState: models.InstanceStatusRunning,
		},
		{name: "fatal fails instance", errorClass: "fatal", wantInstState: models.InstanceStatusFailed},
	}

	for _, tc := range cases {
		s.SetupTest()
		definition = s.createWorkflow(tenantCtx, s.sampleDSL())
		s.Require().NoError(s.retryRepo.Store(tenantCtx, &models.WorkflowRetryPolicy{
			WorkflowName:    definition.Name,
			WorkflowVersion: definition.WorkflowVersion,
			State:           "log_step",
			MaxAttempts:     3,
			InitialDelayMs:  10,
			MaxDelayMs:      50,
			BackoffStrategy: "exponential",
		}))
		instance := &models.WorkflowInstance{
			WorkflowName:    definition.Name,
			WorkflowVersion: definition.WorkflowVersion,
			CurrentState:    "log_step",
			Status:          models.InstanceStatusRunning,
			Revision:        1,
		}
		s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

		engine := s.stateEngine()
		cmd, err := engine.CreateInitialExecution(tenantCtx, instance, json.RawMessage(`{"hello":"world"}`))
		s.Require().NoError(err)
		exec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
		s.Require().NoError(err)
		dispatchCmd, err := engine.Dispatch(tenantCtx, exec)
		s.Require().NoError(err)

		s.Require().NoError(engine.Commit(ctx, &CommitRequest{
			ExecutionID:    dispatchCmd.ExecutionID,
			ExecutionToken: dispatchCmd.ExecutionToken,
			Error: &CommitError{
				Class:   tc.errorClass,
				Code:    "FAILED",
				Message: "boom",
			},
		}))

		updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
		s.Require().NoError(err)
		s.Equal(tc.wantInstState, updatedInstance.Status)
	}
}

func (s *BusinessSuite) TestStateEngine_SignalWaitDelayAndResumePaths() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()

	dslBlob := `{
  "version": "1.0",
  "name": "signal-delay-workflow",
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
	s.Require().NoError(engine.StartSignalWait(tenantCtx, &ExecutionCommand{
		ExecutionID:    exec.ID,
		InstanceID:     instance.ID,
		ExecutionToken: rawToken,
		InputPayload:   json.RawMessage(`{}`),
	}, step))

	wait, err := s.signalWaitRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal("waiting", wait.Status)

	delivered, err := engine.SendSignal(
		tenantCtx,
		instance.ID,
		"approval_response",
		json.RawMessage(`{"approved":true}`),
	)
	s.Require().NoError(err)
	s.True(delivered)

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", nextExec.State)
	s.Equal(models.ExecStatusPending, nextExec.Status)

	delayDef := s.createWorkflow(tenantCtx, `{
  "version": "1.0",
  "name": "delay-workflow",
  "steps": [
    {
      "id": "wait",
      "type": "delay",
      "delay": { "duration": "1m" }
    },
    {
      "id": "after_delay",
      "type": "call",
      "call": { "action": "log.entry", "input": {} }
    }
  ]
}`)

	delayInstance := &models.WorkflowInstance{
		WorkflowName:    delayDef.Name,
		WorkflowVersion: delayDef.WorkflowVersion,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, delayInstance))
	delayCmd, err := engine.CreateInitialExecution(tenantCtx, delayInstance, json.RawMessage(`{"message":"preserved"}`))
	s.Require().NoError(err)
	delayExec, err := s.execRepo.GetByID(ctx, delayCmd.ExecutionID)
	s.Require().NoError(err)
	dispatchCmd, err := engine.Dispatch(tenantCtx, delayExec)
	s.Require().NoError(err)

	s.Require().
		NoError(engine.ParkExecutionUntil(ctx, dispatchCmd.ExecutionID, dispatchCmd.ExecutionToken, time.Now().Add(time.Minute)))
	timer, err := s.timerRepo.GetByExecutionID(ctx, dispatchCmd.ExecutionID)
	s.Require().NoError(err)
	s.Equal(dispatchCmd.ExecutionID, timer.ExecutionID)

	s.Require().NoError(engine.ResumeWaitingExecution(ctx, dispatchCmd.ExecutionID, json.RawMessage(`{}`)))
	resumedExec, err := s.execRepo.GetLatestByInstance(ctx, delayInstance.ID)
	s.Require().NoError(err)
	s.Equal("after_delay", resumedExec.State)
}

func (s *BusinessSuite) TestStateEngine_InvalidTokenAndStalePaths() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	engine := s.stateEngine()
	def := s.createWorkflow(tenantCtx, s.sampleDSL())

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	cmd, err := engine.CreateInitialExecution(tenantCtx, instance, json.RawMessage(`{"hello":"world"}`))
	s.Require().NoError(err)
	exec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
	s.Require().NoError(err)
	dispatchCmd, err := engine.Dispatch(tenantCtx, exec)
	s.Require().NoError(err)

	err = engine.Commit(ctx, &CommitRequest{
		ExecutionID:    dispatchCmd.ExecutionID,
		ExecutionToken: "wrong-token",
		Output:         json.RawMessage(`{"ok":true}`),
	})
	s.Require().ErrorIs(err, ErrInvalidToken)

	s.Require().NoError(engine.Commit(ctx, &CommitRequest{
		ExecutionID:    dispatchCmd.ExecutionID,
		ExecutionToken: dispatchCmd.ExecutionToken,
		Output:         json.RawMessage(`{"ok":true}`),
	}))

	err = engine.Commit(ctx, &CommitRequest{
		ExecutionID:    dispatchCmd.ExecutionID,
		ExecutionToken: dispatchCmd.ExecutionToken,
		Output:         json.RawMessage(`{"ok":true}`),
	})
	s.Require().ErrorIs(err, ErrInvalidToken)

	err = engine.ParkExecutionUntil(
		ctx,
		dispatchCmd.ExecutionID,
		dispatchCmd.ExecutionToken,
		time.Now().Add(time.Minute),
	)
	s.Require().ErrorIs(err, ErrInvalidToken)

	err = engine.ResumeWaitingExecution(ctx, dispatchCmd.ExecutionID, json.RawMessage(`{}`))
	s.Require().ErrorIs(err, ErrStaleExecution)

	_, err = engine.SendSignal(ctx, "", "approved", json.RawMessage(`{}`))
	s.Require().EqualError(err, "instance_id is required")

	_, err = engine.SendSignal(ctx, instance.ID, "", json.RawMessage(`{}`))
	s.Require().EqualError(err, "signal_name is required")
}

func (s *BusinessSuite) TestStateEngine_ForeachScopeLaunchesChildrenAndResumesParent() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	engine := s.stateEngine()
	runtimeRepo := repository.NewWorkflowRuntimeRepository(s.dbPool)

	dslBlob := `{
  "version": "1.0",
  "name": "foreach-workflow",
  "steps": [
    {
      "id": "fanout",
      "type": "foreach",
      "foreach": {
        "items": "payload.items",
        "item_var": "item",
        "index_var": "index",
        "max_concurrency": 1,
        "steps": [
          {
            "id": "child_step",
            "type": "call",
            "call": {
              "action": "log.entry",
              "input": {
                "value": "{{ item }}",
                "index": "{{ index }}"
              }
            }
          }
        ]
      }
    },
    {
      "id": "after",
      "type": "call",
      "call": { "action": "log.entry", "input": {} }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)
	spec, err := dsl.Parse([]byte(dslBlob))
	s.Require().NoError(err)
	step := dsl.FindStep(spec, "fanout")
	s.Require().NotNil(step)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "fanout",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	cmd, err := engine.CreateInitialExecution(
		tenantCtx,
		instance,
		json.RawMessage(`{"items":[{"approved":true},{"approved":false},{"approved":true}]}`),
	)
	s.Require().NoError(err)
	parentExec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
	s.Require().NoError(err)
	dispatchCmd, err := engine.Dispatch(tenantCtx, parentExec)
	s.Require().NoError(err)

	s.Require().NoError(engine.StartBranchScope(tenantCtx, dispatchCmd, step))

	scope, err := s.scopeRepo.GetByParentExecutionID(ctx, parentExec.ID)
	s.Require().NoError(err)
	s.Equal(string(dsl.StepTypeForeach), scope.ScopeType)
	s.Equal(3, scope.TotalChildren)
	s.Equal(1, scope.NextChildIndex)

	for index := range 3 {
		children, listErr := s.instanceRepo.ListByParentExecutionID(ctx, parentExec.ID)
		s.Require().NoError(listErr)
		s.Require().Len(children, index+1)

		child := children[index]
		childExec, execErr := s.execRepo.GetLatestByInstance(ctx, child.ID)
		s.Require().NoError(execErr)

		s.Require().NoError(runtimeRepo.CommitExecution(ctx, &repository.CommitExecutionRequest{
			Execution:      childExec,
			Instance:       child,
			VerifyToken:    false,
			ExpectedStatus: models.ExecStatusPending,
			OutputPayload:  `{"index":` + strconv.Itoa(index) + `,"ok":true}`,
		}))

		s.Require().NoError(engine.ReconcileBranchScope(ctx, scope.ID))
		scope, err = s.scopeRepo.GetByID(ctx, scope.ID)
		s.Require().NoError(err)
	}

	parentLatest, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", parentLatest.State)
	s.Equal(models.ExecStatusPending, parentLatest.Status)

	parentOutput, err := s.outputRepo.GetByExecution(ctx, parentExec.ID)
	s.Require().NoError(err)
	s.Contains(parentOutput.Payload, `"items"`)

	reloadedScope, err := s.scopeRepo.GetByID(ctx, scope.ID)
	s.Require().NoError(err)
	s.Equal("completed", reloadedScope.Status)
	s.Equal(3, reloadedScope.CompletedChildren)
}

func (s *BusinessSuite) TestStateEngine_ParallelScopeFailureFailsParentAndCancelsSiblings() {
	tenantCtx := s.tenantCtx()
	ctx := context.Background()
	engine := s.stateEngine()

	dslBlob := `{
  "version": "1.0",
  "name": "parallel-workflow",
  "steps": [
    {
      "id": "fanout",
      "type": "parallel",
      "parallel": {
        "wait_all": true,
        "steps": [
          { "id": "branch_a", "type": "call", "call": { "action": "log.entry", "input": {} } },
          { "id": "branch_b", "type": "call", "call": { "action": "log.entry", "input": {} } }
        ]
      }
    },
    {
      "id": "after",
      "type": "call",
      "call": { "action": "log.entry", "input": {} }
    }
  ]
}`

	def := s.createWorkflow(tenantCtx, dslBlob)
	spec, err := dsl.Parse([]byte(dslBlob))
	s.Require().NoError(err)
	step := dsl.FindStep(spec, "fanout")
	s.Require().NotNil(step)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "fanout",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	cmd, err := engine.CreateInitialExecution(tenantCtx, instance, json.RawMessage(`{}`))
	s.Require().NoError(err)
	parentExec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
	s.Require().NoError(err)
	dispatchCmd, err := engine.Dispatch(tenantCtx, parentExec)
	s.Require().NoError(err)

	s.Require().NoError(engine.StartBranchScope(tenantCtx, dispatchCmd, step))
	scope, err := s.scopeRepo.GetByParentExecutionID(ctx, parentExec.ID)
	s.Require().NoError(err)

	children, err := s.instanceRepo.ListByParentExecutionID(ctx, parentExec.ID)
	s.Require().NoError(err)
	s.Require().Len(children, 2)

	failedChild := children[0]
	failedExec, err := s.execRepo.GetLatestByInstance(ctx, failedChild.ID)
	s.Require().NoError(err)
	s.Require().NoError(s.execRepo.UpdateStatus(ctx, failedExec.ID, models.ExecStatusFatal, map[string]any{
		"error_class":   "fatal",
		"error_message": "branch exploded",
		"finished_at":   time.Now(),
	}))
	s.Require().NoError(s.instanceRepo.UpdateStatus(ctx, failedChild.ID, models.InstanceStatusFailed))

	s.Require().NoError(engine.ReconcileBranchScope(ctx, scope.ID))

	parentLatest, err := s.execRepo.GetByID(ctx, parentExec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusFatal, parentLatest.Status)
	s.Equal("fatal", parentLatest.ErrorClass)
	s.Equal("branch exploded", parentLatest.ErrorMessage)

	parentInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusFailed, parentInstance.Status)

	reloadedScope, err := s.scopeRepo.GetByID(ctx, scope.ID)
	s.Require().NoError(err)
	s.Equal("failed", reloadedScope.Status)
	s.Equal(1, reloadedScope.FailedChildren)

	cancelledChild, err := s.instanceRepo.GetByID(ctx, children[1].ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCancelled, cancelledChild.Status)
}

func TestResolveNextStepForInstance(t *testing.T) {
	t.Parallel()

	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:   "parallel_root",
				Type: dsl.StepTypeParallel,
				Parallel: &dsl.ParallelSpec{
					Steps: []*dsl.StepSpec{
						{ID: "branch_a", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "log.entry"}},
						{ID: "branch_b", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "log.entry"}},
					},
				},
			},
			{
				ID:   "foreach_root",
				Type: dsl.StepTypeForeach,
				Foreach: &dsl.ForeachSpec{
					ItemVar: "item",
					Steps: []*dsl.StepSpec{
						{ID: "loop_step", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "log.entry"}},
					},
				},
			},
			{ID: "after", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "log.entry"}},
		},
	}

	tests := []struct {
		name         string
		instance     *models.WorkflowInstance
		currentState string
		wantStepID   string
	}{
		{
			name:         "root workflow uses normal navigation",
			instance:     &models.WorkflowInstance{},
			currentState: "parallel_root",
			wantStepID:   "foreach_root",
		},
		{
			name: "parallel child resolves within subtree",
			instance: &models.WorkflowInstance{
				ParentExecutionID: "parent-exec",
				ScopeType:         string(dsl.StepTypeParallel),
				ScopeEntryState:   "branch_a",
			},
			currentState: "branch_a",
			wantStepID:   "",
		},
		{
			name: "foreach child resolves in container",
			instance: &models.WorkflowInstance{
				ParentExecutionID: "parent-exec",
				ScopeType:         string(dsl.StepTypeForeach),
				ScopeParentState:  "foreach_root",
			},
			currentState: "loop_step",
			wantStepID:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			next, err := resolveNextStepForInstance(spec, tc.instance, tc.currentState, map[string]any{})
			require.NoError(t, err)
			if tc.wantStepID == "" {
				require.Nil(t, next)
				return
			}
			require.NotNil(t, next)
			require.Equal(t, tc.wantStepID, next.ID)
		})
	}
}

func TestEvaluateMapping_TableDriven(t *testing.T) {
	t.Parallel()

	engine := &stateEngine{}
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:   "call_step",
				Type: dsl.StepTypeCall,
				Call: &dsl.CallSpec{
					Action:    "log.entry",
					OutputVar: "result",
				},
			},
			{
				ID:   "wait_signal",
				Type: dsl.StepTypeSignalWait,
				SignalWait: &dsl.SignalWaitSpec{
					SignalName: "approved",
					OutputVar:  "approval",
				},
			},
			{
				ID:   "if_step",
				Type: dsl.StepTypeIf,
				If:   &dsl.IfSpec{Expr: "true"},
			},
			{
				ID:    "delay_step",
				Type:  dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{Duration: dsl.Duration{Duration: time.Second}},
			},
			{
				ID:   "next_from_call",
				Type: dsl.StepTypeCall,
				Call: &dsl.CallSpec{
					Action: "log.entry",
					Input: map[string]any{
						"value": "{{ result.value }}",
					},
				},
			},
			{
				ID:   "next_from_signal",
				Type: dsl.StepTypeCall,
				Call: &dsl.CallSpec{
					Action: "log.entry",
					Input: map[string]any{
						"approved": "{{ approval.approved }}",
					},
				},
			},
			{
				ID:   "plain_next",
				Type: dsl.StepTypeCall,
				Call: &dsl.CallSpec{
					Action: "log.entry",
				},
			},
		},
	}

	tests := []struct {
		name         string
		currentState string
		nextStepID   string
		currentInput json.RawMessage
		output       json.RawMessage
		wantJSON     string
	}{
		{
			name:         "call output var maps into next input",
			currentState: "call_step",
			nextStepID:   "next_from_call",
			currentInput: json.RawMessage(`{"original":true}`),
			output:       json.RawMessage(`{"value":"ok"}`),
			wantJSON:     `{"value":"ok"}`,
		},
		{
			name:         "signal wait output var maps into next input",
			currentState: "wait_signal",
			nextStepID:   "next_from_signal",
			currentInput: json.RawMessage(`{"original":true}`),
			output:       json.RawMessage(`{"approved":true}`),
			wantJSON:     `{"approved":"true"}`,
		},
		{
			name:         "if step preserves current input",
			currentState: "if_step",
			nextStepID:   "plain_next",
			currentInput: json.RawMessage(`{"keep":"me"}`),
			output:       json.RawMessage(`{"branch":"then"}`),
			wantJSON:     `{"keep":"me"}`,
		},
		{
			name:         "delay step preserves current input",
			currentState: "delay_step",
			nextStepID:   "plain_next",
			currentInput: json.RawMessage(`{"carry":"forward"}`),
			output:       json.RawMessage(`{}`),
			wantJSON:     `{"carry":"forward"}`,
		},
		{
			name:         "no next input passes output through",
			currentState: "call_step",
			nextStepID:   "plain_next",
			currentInput: json.RawMessage(`{"original":true}`),
			output:       json.RawMessage(`{"value":"ok"}`),
			wantJSON:     `{"value":"ok"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			next := dsl.FindStep(spec, tc.nextStepID)
			require.NotNil(t, next)

			got, err := engine.evaluateMapping(spec, tc.currentState, next, tc.currentInput, tc.output)
			require.NoError(t, err)
			require.JSONEq(t, tc.wantJSON, string(got))
		})
	}
}
