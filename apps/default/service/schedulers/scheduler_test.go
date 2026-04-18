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

//nolint:testpackage // package-local scheduler tests exercise unexported run-once helpers intentionally.
package schedulers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/pitabwire/frame"
	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

type SchedulerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool         pool.Pool
	rawCache       cache.RawCache
	cfg            *config.Config
	eventRepo      repository.EventLogRepository
	auditRepo      repository.AuditEventRepository
	scheduleRepo   repository.ScheduleRepository
	defRepo        repository.WorkflowDefinitionRepository
	schemaRepo     repository.SchemaRegistryRepository
	instanceRepo   repository.WorkflowInstanceRepository
	execRepo       repository.WorkflowExecutionRepository
	outputRepo     repository.WorkflowOutputRepository
	retryRepo      repository.RetryPolicyRepository
	timerRepo      repository.WorkflowTimerRepository
	scopeRepo      repository.WorkflowScopeRunRepository
	signalWaitRepo repository.WorkflowSignalWaitRepository
	signalMsgRepo  repository.WorkflowSignalMessageRepository
}

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, new(SchedulerSuite))
}

func (s *SchedulerSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{testpostgres.New()}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()
	dsn := s.Resources()[0].GetDS(ctx)
	p := pool.NewPool(ctx)
	s.Require().NoError(p.AddConnection(
		ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(
		&models.EventLog{},
		&models.WorkflowAuditEvent{},
		&models.ScheduleDefinition{},
		&models.WorkflowDefinition{},
		&models.WorkflowStateSchema{},
		&models.WorkflowInstance{},
		&models.WorkflowStateExecution{},
		&models.WorkflowStateOutput{},
		&models.WorkflowRetryPolicy{},
		&models.WorkflowTimer{},
		&models.WorkflowScopeRun{},
		&models.WorkflowSignalWait{},
		&models.WorkflowSignalMessage{},
	))
	s.Require().NoError(db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_state_schema
		 ON workflow_state_schemas (tenant_id, workflow_name, workflow_version, state, schema_type)`,
	).Error)

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()
	s.cfg = &config.Config{
		RetentionDays: 1,
	}
	s.eventRepo = repository.NewEventLogRepository(p)
	s.auditRepo = repository.NewAuditEventRepository(p)
	s.scheduleRepo = repository.NewScheduleRepository(p)
	s.defRepo = repository.NewWorkflowDefinitionRepository(p)
	s.schemaRepo = repository.NewSchemaRegistryRepository(p)
	s.instanceRepo = repository.NewWorkflowInstanceRepository(p)
	s.execRepo = repository.NewWorkflowExecutionRepository(p)
	s.outputRepo = repository.NewWorkflowOutputRepository(p)
	s.retryRepo = repository.NewRetryPolicyRepository(p)
	s.timerRepo = repository.NewWorkflowTimerRepository(p)
	s.scopeRepo = repository.NewWorkflowScopeRunRepository(p)
	s.signalWaitRepo = repository.NewWorkflowSignalWaitRepository(p)
	s.signalMsgRepo = repository.NewWorkflowSignalMessageRepository(p)
}

func (s *SchedulerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		`TRUNCATE event_log, workflow_audit_events, schedule_definitions, workflow_definitions,
		 workflow_state_schemas, workflow_instances, workflow_state_executions, workflow_state_outputs,
		 workflow_retry_policies, workflow_timers, workflow_scope_runs, workflow_signal_waits,
		 workflow_signal_messages CASCADE`,
	).Error)
	s.Require().NoError(s.rawCache.Flush(ctx))
}

func (s *SchedulerSuite) TearDownSuite() {
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *SchedulerSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *SchedulerSuite) schemaRegistry() business.SchemaRegistry {
	return business.NewSchemaRegistry(s.schemaRepo, s.rawCache)
}

func (s *SchedulerSuite) workflowBusiness() business.WorkflowBusiness {
	return business.NewWorkflowBusiness(s.defRepo, s.scheduleRepo, s.schemaRegistry())
}

func (s *SchedulerSuite) stateEngine() business.StateEngine {
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
		telemetry.NewMetrics(),
		s.rawCache,
	)
}

func (s *SchedulerSuite) createWorkflow(ctx context.Context, dslBlob string) *models.WorkflowDefinition {
	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(dslBlob))
	s.Require().NoError(err)
	return def
}

type captureWorker struct {
	ch chan []byte
}

func (c *captureWorker) Handle(_ context.Context, _ map[string]string, message []byte) error {
	payload := append([]byte(nil), message...)
	c.ch <- payload
	return nil
}

func (s *SchedulerSuite) TestBuildIngestedEventMessage() {
	msg, err := buildIngestedEventMessage(&models.EventLog{
		BaseModel: models.EventLog{}.BaseModel,
		EventType: "payment.requested",
		Source:    "api",
		Payload:   `{"amount":100}`,
	})
	s.Require().NoError(err)
	s.Equal("payment.requested", msg.EventType)

	_, err = buildIngestedEventMessage(&models.EventLog{Payload: `{bad`})
	s.Require().Error(err)
}

func (s *SchedulerSuite) TestCleanupScheduler_RunOnceDeletesExpiredRows() {
	ctx := s.tenantCtx()
	oldTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now().Add(-2 * time.Hour)

	oldEvent := &models.EventLog{
		EventType:   "old",
		Source:      "api",
		Payload:     `{"x":1}`,
		Published:   true,
		PublishedAt: &oldTime,
	}
	recentEvent := &models.EventLog{
		EventType:   "recent",
		Source:      "api",
		Payload:     `{"x":2}`,
		Published:   true,
		PublishedAt: &recentTime,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, oldEvent))
	s.Require().NoError(s.eventRepo.Create(ctx, recentEvent))
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).Where("id = ?", oldEvent.ID).UpdateColumn("created_at", oldTime).Error)
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).Where("id = ?", recentEvent.ID).UpdateColumn("created_at", recentTime).Error)

	oldAudit := &models.WorkflowAuditEvent{InstanceID: "inst-1", EventType: "old.audit"}
	recentAudit := &models.WorkflowAuditEvent{InstanceID: "inst-1", EventType: "recent.audit"}
	s.Require().NoError(s.auditRepo.Append(ctx, oldAudit))
	s.Require().NoError(s.auditRepo.Append(ctx, recentAudit))
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.WorkflowAuditEvent{}).Where("id = ?", oldAudit.ID).UpdateColumn("created_at", oldTime).Error)
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.WorkflowAuditEvent{}).Where("id = ?", recentAudit.ID).UpdateColumn("created_at", recentTime).Error)

	scheduler := NewCleanupScheduler(s.eventRepo, s.auditRepo, s.cfg)
	deleted := scheduler.RunOnce(ctx)
	s.Equal(int64(2), deleted)

	remainingEvents, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Empty(remainingEvents)
}

func (s *SchedulerSuite) TestCronScheduler_RunOnceCreatesEventAndAdvancesSchedule() {
	ctx := s.tenantCtx()
	fireAt := time.Now().Add(-time.Minute).UTC()
	sched := &models.ScheduleDefinition{
		Name:            "hourly",
		CronExpr:        "*/5 * * * *",
		WorkflowName:    "payments",
		WorkflowVersion: 1,
		InputPayload:    `{"country":"UG"}`,
		Active:          true,
		NextFireAt:      &fireAt,
	}
	s.Require().NoError(s.scheduleRepo.Create(ctx, sched))

	scheduler := NewCronScheduler(s.scheduleRepo, s.eventRepo, s.cfg)
	fired := scheduler.RunOnce(ctx)
	s.Equal(1, fired)

	unpublished, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(unpublished, 1)
	s.Contains(unpublished[0].Payload, `"country": "UG"`)
	s.Contains(unpublished[0].Payload, `"schedule_name": "hourly"`)

	// Verify next_fire_at was advanced: schedule should now be due again in the future
	// (ClaimAndFireBatch updates next_fire_at; a subsequent sweep in the near future should
	// NOT claim the same row since its next_fire_at now points ahead).
	var reloaded models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).First(&reloaded, "id = ?", sched.ID).Error)
	s.NotNil(reloaded.NextFireAt)
	s.True(reloaded.NextFireAt.After(time.Now()), "next_fire_at must be in the future after firing")
}

func (s *SchedulerSuite) TestRetryScheduler_RunOnceCreatesNewAttempt() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	nextRetryAt := time.Now().Add(-time.Minute)
	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "step-a",
		Attempt:         1,
		Status:          models.ExecStatusRetryScheduled,
		ExecutionToken:  "token",
		InputSchemaHash: "hash",
		InputPayload:    "{}",
		NextRetryAt:     &nextRetryAt,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	scheduler := NewRetryScheduler(
		s.execRepo,
		s.instanceRepo,
		&config.Config{RetryBatchSize: 10},
		telemetry.NewMetrics(),
	)
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	var total int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.WorkflowStateExecution{}).
		Where("instance_id = ?", instance.ID).Count(&total).Error)
	s.Equal(int64(2), total)
}

func (s *SchedulerSuite) TestDispatchScheduler_RunOncePublishesToQueue() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	def := s.createWorkflow(tenantCtx, `{
  "version": "1.0",
  "name": "dispatch-workflow",
  "steps": [
    {"id": "log_step", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`)
	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "log_step",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "log_step",
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  "token",
		InputSchemaHash: "hash",
		InputPayload:    "{}",
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	topic := fmt.Sprintf("mem://dispatch-%s", util.IDString())
	worker := &captureWorker{ch: make(chan []byte, 1)}
	svcCtx, svc := frame.NewService(
		frame.WithName("dispatch-test"),
		frametests.WithNoopDriver(),
		frame.WithRegisterPublisher("exec-dispatch", topic),
		frame.WithRegisterSubscriber("exec-dispatch", topic, worker),
	)
	s.Require().NoError(svc.Run(svcCtx, ""))
	defer svc.Stop(svcCtx)

	scheduler := NewDispatchScheduler(s.execRepo, s.stateEngine(), svc.QueueManager(), &config.Config{
		DispatchBatchSize:     10,
		QueueExecDispatchName: "exec-dispatch",
	}, telemetry.NewMetrics())
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	select {
	case <-worker.ch:
	case <-time.After(2 * time.Second):
		s.Fail("expected dispatch message to be published")
	}
}

func (s *SchedulerSuite) TestDispatchScheduler_RunUntilDrainedPublishesMultipleBatches() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	def := s.createWorkflow(tenantCtx, `{
  "version": "1.0",
  "name": "dispatch-drain-workflow",
  "steps": [
    {"id": "log_step", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`)

	for range 2 {
		instance := &models.WorkflowInstance{
			WorkflowName:    def.Name,
			WorkflowVersion: def.WorkflowVersion,
			CurrentState:    "log_step",
			Status:          models.InstanceStatusRunning,
			Revision:        1,
		}
		s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))
		s.Require().NoError(s.execRepo.Create(tenantCtx, &models.WorkflowStateExecution{
			InstanceID:      instance.ID,
			State:           "log_step",
			Attempt:         1,
			Status:          models.ExecStatusPending,
			ExecutionToken:  "token",
			InputSchemaHash: "hash",
			InputPayload:    "{}",
		}))
	}

	topic := fmt.Sprintf("mem://dispatch-drain-%s", util.IDString())
	worker := &captureWorker{ch: make(chan []byte, 2)}
	svcCtx, svc := frame.NewService(
		frame.WithName("dispatch-drain-test"),
		frametests.WithNoopDriver(),
		frame.WithRegisterPublisher("exec-dispatch", topic),
		frame.WithRegisterSubscriber("exec-dispatch", topic, worker),
	)
	s.Require().NoError(svc.Run(svcCtx, ""))
	defer svc.Stop(svcCtx)

	scheduler := NewDispatchScheduler(s.execRepo, s.stateEngine(), svc.QueueManager(), &config.Config{
		DispatchBatchSize:          1,
		DispatchMaxBatchesPerSweep: 3,
		QueueExecDispatchName:      "exec-dispatch",
	}, telemetry.NewMetrics())

	count := scheduler.RunUntilDrained(ctx)
	s.Equal(2, count)
}

func (s *SchedulerSuite) TestOutboxScheduler_RunOncePublishesToQueue() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	event := &models.EventLog{
		EventType: "user.created",
		Source:    "api",
		Payload:   `{"foo":"bar"}`,
	}
	s.Require().NoError(s.eventRepo.Create(tenantCtx, event))

	topic := fmt.Sprintf("mem://outbox-%s", util.IDString())
	worker := &captureWorker{ch: make(chan []byte, 1)}
	svcCtx, svc := frame.NewService(
		frame.WithName("outbox-test"),
		frametests.WithNoopDriver(),
		frame.WithRegisterPublisher("event-ingest", topic),
		frame.WithRegisterSubscriber("event-ingest", topic, worker),
	)
	s.Require().NoError(svc.Run(svcCtx, ""))
	defer svc.Stop(svcCtx)

	scheduler := NewOutboxScheduler(s.eventRepo, svc.QueueManager(), &config.Config{
		OutboxBatchSize:       10,
		OutboxClaimTTLSeconds: 30,
		QueueEventIngestName:  "event-ingest",
	}, telemetry.NewMetrics())
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	select {
	case <-worker.ch:
	case <-time.After(2 * time.Second):
		s.Fail("expected outbox message to be published")
	}
}

func (s *SchedulerSuite) TestOutboxScheduler_RunUntilDrainedPublishesMultipleBatches() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	for index := range 2 {
		s.Require().NoError(s.eventRepo.Create(tenantCtx, &models.EventLog{
			EventType: fmt.Sprintf("user.created.%d", index),
			Source:    "api",
			Payload:   fmt.Sprintf(`{"idx":%d}`, index),
		}))
	}

	topic := fmt.Sprintf("mem://outbox-drain-%s", util.IDString())
	worker := &captureWorker{ch: make(chan []byte, 2)}
	svcCtx, svc := frame.NewService(
		frame.WithName("outbox-drain-test"),
		frametests.WithNoopDriver(),
		frame.WithRegisterPublisher("event-ingest", topic),
		frame.WithRegisterSubscriber("event-ingest", topic, worker),
	)
	s.Require().NoError(svc.Run(svcCtx, ""))
	defer svc.Stop(svcCtx)

	scheduler := NewOutboxScheduler(s.eventRepo, svc.QueueManager(), &config.Config{
		OutboxBatchSize:          1,
		OutboxMaxBatchesPerSweep: 3,
		OutboxClaimTTLSeconds:    30,
		QueueEventIngestName:     "event-ingest",
	}, telemetry.NewMetrics())

	count := scheduler.RunUntilDrained(ctx)
	s.Equal(2, count)
}

func (s *SchedulerSuite) TestTimerScheduler_RunOnceResumesWaitingExecution() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

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
      "call": { "action": "log.entry", "input": { "message": "after" } }
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

	firesAt := time.Now().Add(-time.Minute)
	timer := &models.WorkflowTimer{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "wait",
		FiresAt:     firesAt,
	}
	s.Require().NoError(s.timerRepo.Create(tenantCtx, timer))

	scheduler := NewTimerScheduler(s.timerRepo, s.stateEngine(), &config.Config{
		TimerBatchSize:       10,
		TimerClaimTTLSeconds: 30,
	}, telemetry.NewMetrics())
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", nextExec.State)

	storedTimer, err := s.timerRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.NotNil(storedTimer.FiredAt)
}

func (s *SchedulerSuite) TestTimeoutScheduler_RunOnceSchedulesRetry() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(tenantCtx, instance))

	policy := &models.WorkflowRetryPolicy{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		State:           "step-a",
		MaxAttempts:     3,
		InitialDelayMs:  10,
		MaxDelayMs:      100,
		BackoffStrategy: "exponential",
	}
	s.Require().NoError(s.retryRepo.Store(tenantCtx, policy))

	startedAt := time.Now().Add(-5 * time.Minute)
	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "step-a",
		Attempt:         1,
		Status:          models.ExecStatusDispatched,
		ExecutionToken:  "token",
		InputSchemaHash: "hash",
		InputPayload:    "{}",
		StartedAt:       &startedAt,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	scheduler := NewTimeoutScheduler(
		s.execRepo,
		s.instanceRepo,
		s.retryRepo,
		s.auditRepo,
		&config.Config{TimeoutBatchSize: 10, DefaultExecutionTimeoutSeconds: 30},
		telemetry.NewMetrics(),
	)
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	var total int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.WorkflowStateExecution{}).
		Where("instance_id = ?", instance.ID).Count(&total).Error)
	s.Equal(int64(2), total)
}

func (s *SchedulerSuite) TestSignalScheduler_RunOnceTimesOutWait() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	dslBlob := `{
  "version": "1.0",
  "name": "signal-timeout-workflow",
  "steps": [
    {
      "id": "approval_wait",
      "type": "signal_wait",
      "signal_wait": {
        "signal_name": "approval_response",
        "timeout": "1m"
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
	past := time.Now().Add(-time.Minute)
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.WorkflowSignalWait{}).
		Where("id = ?", wait.ID).UpdateColumn("timeout_at", past).Error)

	scheduler := NewSignalScheduler(s.signalWaitRepo, engine, &config.Config{
		SignalBatchSize:       10,
		SignalClaimTTLSeconds: 30,
	})
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusTimedOut, updatedExec.Status)
}

func (s *SchedulerSuite) TestScopeScheduler_RunOnceReconcilesBranchScope() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()
	engine := s.stateEngine()
	runtimeRepo := repository.NewWorkflowRuntimeRepository(s.dbPool)

	dslBlob := `{
  "version": "1.0",
  "name": "scope-workflow",
  "steps": [
    {
      "id": "fanout",
      "type": "parallel",
      "parallel": {
        "wait_all": true,
        "steps": [
          { "id": "branch_a", "type": "call", "call": { "action": "log.entry", "input": {} } }
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
	s.Require().Len(children, 1)

	childExec, err := s.execRepo.GetLatestByInstance(ctx, children[0].ID)
	s.Require().NoError(err)
	s.Require().NoError(runtimeRepo.CommitExecution(ctx, &repository.CommitExecutionRequest{
		Execution:      childExec,
		Instance:       children[0],
		VerifyToken:    false,
		ExpectedStatus: models.ExecStatusPending,
		OutputPayload:  `{"ok":true}`,
	}))

	scheduler := NewScopeScheduler(s.scopeRepo, engine, &config.Config{
		ScopeBatchSize:       10,
		ScopeClaimTTLSeconds: 30,
	})
	count := scheduler.RunOnce(ctx)
	s.Equal(1, count)

	updatedScope, err := s.scopeRepo.GetByID(ctx, scope.ID)
	s.Require().NoError(err)
	s.Equal("completed", updatedScope.Status)

	latestExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", latestExec.State)
}

func TestJitterFor_Deterministic(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	a := jitterFor("sched-1", sched, nominal)
	b := jitterFor("sched-1", sched, nominal)
	require.Equal(t, a, b, "jitter must be deterministic per schedule id")
}

func TestJitterFor_RespectsCap(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *") // 5-min period → cap = min(period/10, 30s) = 30s.
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	for i := range 50 {
		j := jitterFor(fmt.Sprintf("s-%d", i), sched, nominal)
		require.True(t, j >= 0 && j < 30*time.Second, "jitter %v out of bounds", j)
	}
}

var _ = telemetry.NewMetrics()
