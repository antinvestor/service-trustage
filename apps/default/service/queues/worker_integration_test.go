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

//nolint:testpackage // package-local worker tests exercise unexported queue internals intentionally.
package queues

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/connector/adapters"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

type WorkerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool pool.Pool
	cache  cache.RawCache

	metrics *telemetry.Metrics

	defRepo        repository.WorkflowDefinitionRepository
	schemaRepo     repository.SchemaRegistryRepository
	instanceRepo   repository.WorkflowInstanceRepository
	execRepo       repository.WorkflowExecutionRepository
	outputRepo     repository.WorkflowOutputRepository
	triggerRepo    repository.TriggerBindingRepository
	retryRepo      repository.RetryPolicyRepository
	scheduleRepo   repository.ScheduleRepository
	timerRepo      repository.WorkflowTimerRepository
	scopeRepo      repository.WorkflowScopeRunRepository
	signalWaitRepo repository.WorkflowSignalWaitRepository
	signalMsgRepo  repository.WorkflowSignalMessageRepository
	auditRepo      repository.AuditEventRepository
}

func TestWorkerSuite(t *testing.T) {
	suite.Run(t, new(WorkerSuite))
}

func (s *WorkerSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{testpostgres.New()}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()
	dsn := s.Resources()[0].GetDS(ctx)
	p := pool.NewPool(ctx)
	s.Require().NoError(p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(
		&models.EventLog{},
		&models.WorkflowAuditEvent{},
		&models.WorkflowStateOutput{},
		&models.WorkflowStateExecution{},
		&models.WorkflowScopeRun{},
		&models.WorkflowSignalWait{},
		&models.WorkflowSignalMessage{},
		&models.WorkflowTimer{},
		&models.WorkflowStateSchema{},
		&models.WorkflowInstance{},
		&models.WorkflowDefinition{},
		&models.WorkflowRetryPolicy{},
		&models.TriggerBinding{},
		&models.ScheduleDefinition{},
	))
	s.Require().NoError(db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_state_schema
		 ON workflow_state_schemas (tenant_id, workflow_name, workflow_version, state, schema_type)`,
	).Error)
	s.Require().NoError(db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_instance_trigger_dedupe
		 ON workflow_instances (tenant_id, partition_id, workflow_name, workflow_version, trigger_event_id)
		 WHERE trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL`,
	).Error)

	s.dbPool = p
	s.cache = cache.NewInMemoryCache()
	s.metrics = telemetry.NewMetrics()

	s.defRepo = repository.NewWorkflowDefinitionRepository(p)
	s.schemaRepo = repository.NewSchemaRegistryRepository(p)
	s.instanceRepo = repository.NewWorkflowInstanceRepository(p)
	s.execRepo = repository.NewWorkflowExecutionRepository(p)
	s.outputRepo = repository.NewWorkflowOutputRepository(p)
	s.triggerRepo = repository.NewTriggerBindingRepository(p)
	s.retryRepo = repository.NewRetryPolicyRepository(p)
	s.scheduleRepo = repository.NewScheduleRepository(p)
	s.timerRepo = repository.NewWorkflowTimerRepository(p)
	s.scopeRepo = repository.NewWorkflowScopeRunRepository(p)
	s.signalWaitRepo = repository.NewWorkflowSignalWaitRepository(p)
	s.signalMsgRepo = repository.NewWorkflowSignalMessageRepository(p)
	s.auditRepo = repository.NewAuditEventRepository(p)
}

func (s *WorkerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		`TRUNCATE event_log, workflow_audit_events, workflow_state_outputs, workflow_state_executions,
		 workflow_scope_runs, workflow_signal_waits, workflow_signal_messages, workflow_timers,
		 workflow_state_schemas, workflow_instances, workflow_definitions, workflow_retry_policies,
		 trigger_bindings CASCADE`,
	).Error)
	s.Require().NoError(s.cache.Flush(ctx))
}

func (s *WorkerSuite) TearDownSuite() {
	ctx := context.Background()
	if s.cache != nil {
		_ = s.cache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *WorkerSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant-001",
		PartitionID: "test-partition-001",
	}
	claims.Subject = "test-user-001"
	return claims.ClaimsToContext(context.Background())
}

func (s *WorkerSuite) schemaRegistry() business.SchemaRegistry {
	return business.NewSchemaRegistry(s.schemaRepo, s.cache)
}

func (s *WorkerSuite) workflowBusiness() business.WorkflowBusiness {
	return business.NewWorkflowBusiness(s.defRepo, s.scheduleRepo, s.schemaRegistry(), nil)
}

func (s *WorkerSuite) stateEngine() business.StateEngine {
	return business.NewStateEngine(
		s.instanceRepo,
		s.execRepo,
		repository.NewWorkflowRuntimeRepository(s.dbPool),
		s.timerRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.outputRepo,
		s.auditRepo,
		s.defRepo,
		s.retryRepo,
		s.schemaRegistry(),
		s.metrics,
		s.cache,
	)
}

func (s *WorkerSuite) eventRouter() business.EventRouter {
	return business.NewEventRouter(
		s.triggerRepo,
		s.defRepo,
		s.instanceRepo,
		s.auditRepo,
		s.stateEngine(),
		s.metrics,
	)
}

func (s *WorkerSuite) TestEventRouterWorker_HandleRoutesEvent() {
	ctx := s.tenantCtx()
	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "route-workflow",
  "steps": [
    {"id": "log_step", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "hello"}}}
  ]
}`))
	s.Require().NoError(err)
	def.Status = models.WorkflowStatusActive
	s.Require().NoError(s.defRepo.Update(ctx, def))

	s.Require().NoError(s.triggerRepo.Create(ctx, &models.TriggerBinding{
		EventType:       "customer.created",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}))

	worker := NewEventRouterWorker(s.eventRouter())
	message, err := json.Marshal(&events.IngestedEventMessage{
		EventID:     "evt-1",
		TenantID:    "test-tenant-001",
		PartitionID: "test-partition-001",
		EventType:   "customer.created",
		Payload:     map[string]any{"name": "Alice"},
	})
	s.Require().NoError(err)

	s.Require().NoError(worker.Handle(context.Background(), nil, message))

	instance, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, "evt-1")
	s.Require().NoError(err)
	s.Equal("log_step", instance.CurrentState)
}

func (s *WorkerSuite) TestExecutionWorker_HandleProcessesCallStep() {
	ctx := s.tenantCtx()
	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "worker-workflow",
  "steps": [
    {"id": "log_step", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "{{ payload.message }}"}}}
  ]
}`))
	s.Require().NoError(err)

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	engine := s.stateEngine()
	cmd, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(`{"message":"hello"}`))
	s.Require().NoError(err)
	exec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
	s.Require().NoError(err)
	dispatchCmd, err := engine.Dispatch(ctx, exec)
	s.Require().NoError(err)

	registry := connector.NewRegistry()
	s.Require().NoError(registry.Register(adapters.NewLogEntryAdapter()))
	worker := NewExecutionWorker(engine, s.defRepo, registry)
	payload, err := json.Marshal(dispatchCmd)
	s.Require().NoError(err)

	s.Require().NoError(worker.Handle(context.Background(), nil, payload))

	output, err := s.outputRepo.GetByExecution(context.Background(), dispatchCmd.ExecutionID)
	s.Require().NoError(err)
	s.Contains(output.Payload, `"logged": true`)

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, updatedInstance.Status)
}

func (s *WorkerSuite) TestExecutionWorker_HandleControlSteps() {
	ctx := s.tenantCtx()
	registry := connector.NewRegistry()
	s.Require().NoError(registry.Register(adapters.NewLogEntryAdapter()))

	tests := []struct {
		name   string
		dsl    string
		state  string
		input  string
		assert func(cmd *business.ExecutionCommand, instance *models.WorkflowInstance)
	}{
		{
			name: "if step commits branch and advances",
			dsl: `{
  "version": "1.0",
  "name": "if-workflow",
  "steps": [
    {
      "id": "check",
      "type": "if",
      "if": {
        "expr": "payload.amount > 100",
        "then": [{"id": "then_call", "type": "call", "call": {"action": "log.entry", "input": {}}}],
        "else": [{"id": "else_call", "type": "call", "call": {"action": "log.entry", "input": {}}}]
      }
    },
    {"id": "after", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`,
			state: "check",
			input: `{"amount":150}`,
			assert: func(cmd *business.ExecutionCommand, instance *models.WorkflowInstance) {
				output, err := s.outputRepo.GetByExecution(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.JSONEq(`{"branch":"then"}`, output.Payload)

				nextExec, err := s.execRepo.GetLatestByInstance(context.Background(), instance.ID)
				s.Require().NoError(err)
				s.Equal("then_call", nextExec.State)
				s.Equal(models.ExecStatusPending, nextExec.Status)
			},
		},
		{
			name: "delay step parks execution",
			dsl: `{
  "version": "1.0",
  "name": "delay-worker-workflow",
  "steps": [
    {"id": "wait", "type": "delay", "delay": {"duration": "1m"}},
    {"id": "after", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`,
			state: "wait",
			input: `{}`,
			assert: func(cmd *business.ExecutionCommand, _ *models.WorkflowInstance) {
				timer, err := s.timerRepo.GetByExecutionID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Equal(cmd.ExecutionID, timer.ExecutionID)

				exec, err := s.execRepo.GetByID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Equal(models.ExecStatusWaiting, exec.Status)
			},
		},
		{
			name: "signal wait step records durable wait",
			dsl: `{
  "version": "1.0",
  "name": "signal-worker-workflow",
  "steps": [
    {"id": "wait_signal", "type": "signal_wait", "signal_wait": {"signal_name": "approved", "output_var": "approval"}},
    {"id": "after", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`,
			state: "wait_signal",
			input: `{}`,
			assert: func(cmd *business.ExecutionCommand, _ *models.WorkflowInstance) {
				wait, err := s.signalWaitRepo.GetByExecutionID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Equal("approved", wait.SignalName)
				s.Equal("waiting", wait.Status)

				exec, err := s.execRepo.GetByID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Equal(models.ExecStatusWaiting, exec.Status)
			},
		},
		{
			name: "parallel step creates branch scope",
			dsl: `{
  "version": "1.0",
  "name": "parallel-worker-workflow",
  "steps": [
    {
      "id": "fanout",
      "type": "parallel",
      "parallel": {
        "wait_all": true,
        "steps": [
          {"id": "branch_a", "type": "call", "call": {"action": "log.entry", "input": {}}},
          {"id": "branch_b", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
    },
    {"id": "after", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`,
			state: "fanout",
			input: `{}`,
			assert: func(cmd *business.ExecutionCommand, _ *models.WorkflowInstance) {
				scope, err := s.scopeRepo.GetByParentExecutionID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Equal(string(dsl.StepTypeParallel), scope.ScopeType)
				s.Equal(2, scope.TotalChildren)

				children, err := s.instanceRepo.ListByParentExecutionID(context.Background(), cmd.ExecutionID)
				s.Require().NoError(err)
				s.Len(children, 2)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(tc.dsl))
			s.Require().NoError(err)

			instance := &models.WorkflowInstance{
				WorkflowName:    def.Name,
				WorkflowVersion: def.WorkflowVersion,
				CurrentState:    tc.state,
				Status:          models.InstanceStatusRunning,
				Revision:        1,
			}
			s.Require().NoError(s.instanceRepo.Create(ctx, instance))

			engine := s.stateEngine()
			cmd, err := engine.CreateInitialExecution(ctx, instance, json.RawMessage(tc.input))
			s.Require().NoError(err)
			exec, err := s.execRepo.GetByID(ctx, cmd.ExecutionID)
			s.Require().NoError(err)
			dispatchCmd, err := engine.Dispatch(ctx, exec)
			s.Require().NoError(err)

			worker := NewExecutionWorker(engine, s.defRepo, registry)
			payload, err := json.Marshal(dispatchCmd)
			s.Require().NoError(err)
			s.Require().NoError(worker.Handle(context.Background(), nil, payload))

			tc.assert(dispatchCmd, instance)
		})
	}
}

func TestQueueHelpers_TableDriven(t *testing.T) {
	t.Parallel()

	t.Run("payload vars extracts payload and top level aliases", func(t *testing.T) {
		t.Parallel()
		vars, err := payloadVars(json.RawMessage(`{"message":"hello","amount":10}`))
		require.NoError(t, err)
		require.Equal(t, "hello", vars["message"])
		require.Equal(t, map[string]any{"message": "hello", "amount": float64(10)}, vars["payload"])
	})

	t.Run("resolve step input uses payload when step input empty", func(t *testing.T) {
		t.Parallel()
		resolved, err := resolveStepInput(nil, json.RawMessage(`{"message":"hello"}`))
		require.NoError(t, err)
		require.Equal(t, "hello", resolved["message"])
	})

	t.Run("resolve signal send templating", func(t *testing.T) {
		t.Parallel()
		target, payload, err := resolveSignalSend(&dsl.SignalSendSpec{
			SignalName:       "approved",
			TargetWorkflowID: "{{ payload.instance_id }}",
			Payload: map[string]any{
				"approved": "{{ payload.approved }}",
			},
		}, json.RawMessage(`{"instance_id":"inst-1","approved":true}`))
		require.NoError(t, err)
		require.Equal(t, "inst-1", target)
		require.Contains(t, string(payload), `"approved":"true"`)
	})
}
