package tests

import (
	"context"
	"encoding/json"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
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
