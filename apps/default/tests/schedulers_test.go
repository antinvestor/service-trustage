package tests_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/schedulers"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
)

type publishedMessage struct {
	ref     string
	payload any
}

type fakeQueueManager struct {
	published []publishedMessage
}

var errQueueNotUsed = errors.New("queue manager not used in test")

func (f *fakeQueueManager) AddPublisher(_ context.Context, _ string, _ string) error {
	return nil
}
func (f *fakeQueueManager) GetPublisher(_ string) (queue.Publisher, error) {
	return nil, errQueueNotUsed
}
func (f *fakeQueueManager) DiscardPublisher(_ context.Context, _ string) error { return nil }
func (f *fakeQueueManager) AddSubscriber(_ context.Context, _ string, _ string, _ ...queue.SubscribeWorker) error {
	return nil
}
func (f *fakeQueueManager) DiscardSubscriber(_ context.Context, _ string) error { return nil }
func (f *fakeQueueManager) GetSubscriber(_ string) (queue.Subscriber, error) {
	return nil, errQueueNotUsed
}
func (f *fakeQueueManager) Publish(_ context.Context, reference string, payload any, _ ...map[string]string) error {
	f.published = append(f.published, publishedMessage{ref: reference, payload: payload})
	return nil
}
func (f *fakeQueueManager) Init(_ context.Context) error  { return nil }
func (f *fakeQueueManager) Close(_ context.Context) error { return nil }

func (s *DefaultServiceSuite) TestDispatchScheduler_RunOnce() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	def := s.createWorkflow(tenantCtx, s.sampleDSL())
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

	cfg := &config.Config{DispatchBatchSize: 10, QueueExecDispatchName: "exec-dispatch"}
	queueMgr := &fakeQueueManager{}

	engine := s.stateEngine()
	sched := schedulers.NewDispatchScheduler(s.execRepo, engine, queueMgr, cfg, s.metrics)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)
	s.Len(queueMgr.published, 1)
	s.Equal("exec-dispatch", queueMgr.published[0].ref)
}

func (s *DefaultServiceSuite) TestDispatchScheduler_RunUntilDrained() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	def := s.createWorkflow(tenantCtx, s.sampleDSL())
	for range 3 {
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
	}

	cfg := &config.Config{
		DispatchBatchSize:          1,
		DispatchMaxBatchesPerSweep: 3,
		QueueExecDispatchName:      "exec-dispatch",
	}
	queueMgr := &fakeQueueManager{}

	sched := schedulers.NewDispatchScheduler(s.execRepo, s.stateEngine(), queueMgr, cfg, s.metrics)
	count := sched.RunUntilDrained(ctx)
	s.Equal(3, count)
	s.Len(queueMgr.published, 3)
}

func (s *DefaultServiceSuite) TestOutboxScheduler_RunOnce() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	payloadBytes, _ := json.Marshal(map[string]any{"foo": "bar"})
	event := &models.EventLog{
		EventType: "user.created",
		Source:    "api",
		Payload:   string(payloadBytes),
	}
	s.Require().NoError(s.eventRepo.Create(tenantCtx, event))

	cfg := &config.Config{OutboxBatchSize: 10, QueueEventIngestName: "event-ingest"}
	queueMgr := &fakeQueueManager{}

	sched := schedulers.NewOutboxScheduler(s.eventRepo, queueMgr, cfg, s.metrics)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)
	s.Len(queueMgr.published, 1)
	s.Equal("event-ingest", queueMgr.published[0].ref)
}

func (s *DefaultServiceSuite) TestOutboxScheduler_RunUntilDrained() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	for i := range 3 {
		event := &models.EventLog{
			EventType: "user.created",
			Source:    "api",
			Payload:   `{"index":1}`,
		}
		if i == 2 {
			event.EventType = "user.updated"
		}
		s.Require().NoError(s.eventRepo.Create(tenantCtx, event))
	}

	cfg := &config.Config{
		OutboxBatchSize:          1,
		OutboxMaxBatchesPerSweep: 3,
		OutboxClaimTTLSeconds:    30,
		QueueEventIngestName:     "event-ingest",
	}
	queueMgr := &fakeQueueManager{}

	sched := schedulers.NewOutboxScheduler(s.eventRepo, queueMgr, cfg, s.metrics)
	count := sched.RunUntilDrained(ctx)
	s.Equal(3, count)
	s.Len(queueMgr.published, 3)
}

func (s *DefaultServiceSuite) TestRetryScheduler_RunOnce() {
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

	cfg := &config.Config{RetryBatchSize: 10}
	sched := schedulers.NewRetryScheduler(s.execRepo, s.instanceRepo, cfg, s.metrics)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)

	var total int64
	s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM workflow_state_executions WHERE instance_id = ?",
		instance.ID,
	).Scan(&total)
	s.Equal(int64(2), total)
}

func (s *DefaultServiceSuite) TestTimerScheduler_RunOnce() {
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

	cfg := &config.Config{TimerBatchSize: 10, TimerClaimTTLSeconds: 30}
	sched := schedulers.NewTimerScheduler(s.timerRepo, s.stateEngine(), cfg, s.metrics)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)

	nextExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("after", nextExec.State)

	storedTimer, err := s.timerRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.NotNil(storedTimer.FiredAt)
}

func (s *DefaultServiceSuite) TestTimeoutScheduler_RunOnce() {
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

	cfg := &config.Config{TimeoutBatchSize: 10, DefaultExecutionTimeoutSeconds: 30}
	sched := schedulers.NewTimeoutScheduler(s.execRepo, s.instanceRepo, s.retryRepo, s.auditRepo, cfg, s.metrics)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)

	var total int64
	s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM workflow_state_executions WHERE instance_id = ?",
		instance.ID,
	).Scan(&total)
	s.Equal(int64(2), total)
}

func (s *DefaultServiceSuite) TestSignalScheduler_RunOnce() {
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

	s.execRepo.Pool().DB(ctx, false).Exec(
		`UPDATE workflow_signal_waits SET timeout_at = ? WHERE id = ?`,
		past, wait.ID,
	)

	cfg := &config.Config{SignalBatchSize: 10, SignalClaimTTLSeconds: 30}
	sched := schedulers.NewSignalScheduler(s.signalWaitRepo, engine, cfg)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)

	updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusTimedOut, updatedExec.Status)
}

func (s *DefaultServiceSuite) TestScopeScheduler_RunOnce_ParallelWaitAll() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

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
          {"id": "child_a", "type": "call", "call": {"action": "log.entry", "input": {}}},
          {"id": "child_b", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
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

	rawToken, err := cryptoutil.GenerateToken()
	s.Require().NoError(err)
	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "fanout",
		Attempt:         1,
		Status:          models.ExecStatusDispatched,
		ExecutionToken:  cryptoutil.HashToken(rawToken),
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(tenantCtx, exec))

	engine := s.stateEngine()
	s.Require().NoError(engine.StartBranchScope(ctx, &business.ExecutionCommand{
		ExecutionID:    exec.ID,
		InstanceID:     instance.ID,
		ExecutionToken: rawToken,
		InputPayload:   json.RawMessage(`{}`),
	}, step))

	var childInstances []*models.WorkflowInstance
	s.execRepo.Pool().DB(ctx, false).
		Where("parent_execution_id = ? AND deleted_at IS NULL", exec.ID).
		Order("scope_index").
		Find(&childInstances)
	s.Require().Len(childInstances, 2)

	for index, child := range childInstances {
		childExec, getErr := s.execRepo.GetLatestByInstance(ctx, child.ID)
		s.Require().NoError(getErr)

		now := time.Now()
		s.execRepo.Pool().DB(ctx, false).Exec(
			`UPDATE workflow_state_executions SET status = ?, finished_at = ?, output_schema_hash = ? WHERE id = ?`,
			string(models.ExecStatusCompleted), now, "hash", childExec.ID,
		)
		s.execRepo.Pool().DB(ctx, false).Exec(
			`UPDATE workflow_instances SET status = ?, finished_at = ? WHERE id = ?`,
			string(models.InstanceStatusCompleted), now, child.ID,
		)
		s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
			ExecutionID: childExec.ID,
			InstanceID:  child.ID,
			State:       child.CurrentState,
			SchemaHash:  "hash",
			Payload:     string(mustJSON(map[string]any{"branch": index})),
		}))
	}

	cfg := &config.Config{ScopeBatchSize: 10, ScopeClaimTTLSeconds: 30}
	sched := schedulers.NewScopeScheduler(s.scopeRepo, engine, cfg)
	count := sched.RunOnce(ctx)
	s.Equal(1, count)

	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, updatedInstance.Status)
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return payload
}

func (s *DefaultServiceSuite) TestCronScheduler_RunOnce() {
	ctx := context.Background()
	tenantCtx := s.tenantCtx()

	nextFire := time.Now().Add(-time.Minute)
	schedDef := &models.ScheduleDefinition{
		Name:            "sched-1",
		CronExpr:        "1h",
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		InputPayload:    `{"extra":"x"}`,
		Active:          true,
		NextFireAt:      &nextFire,
	}
	s.Require().NoError(s.scheduleRepo.Create(tenantCtx, schedDef))

	cron := schedulers.NewCronScheduler(s.scheduleRepo, s.eventRepo, &config.Config{})
	count := cron.RunOnce(ctx)
	s.Equal(1, count)

	var total int64
	s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM event_log",
	).Scan(&total)
	s.Equal(int64(1), total)
}

func (s *DefaultServiceSuite) TestCleanupScheduler_RunOnce() {
	ctx := security.SkipTenancyChecksOnClaims(s.tenantCtx())

	event := &models.EventLog{
		EventType: "user.created",
		Source:    "api",
		Payload:   `{"id":"1"}`,
		Published: true,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, event))

	audit := &models.WorkflowAuditEvent{
		InstanceID: "inst-1",
		EventType:  "state.completed",
		State:      "step-a",
	}
	s.Require().NoError(s.auditRepo.Append(ctx, audit))

	oldTime := time.Now().Add(-48 * time.Hour)
	s.execRepo.Pool().DB(ctx, false).Exec(
		"UPDATE event_log SET published_at = ? WHERE id = ?",
		oldTime, event.ID,
	)
	s.execRepo.Pool().DB(ctx, false).Exec(
		"UPDATE workflow_audit_events SET created_at = ? WHERE id = ?",
		oldTime, audit.ID,
	)

	cfg := &config.Config{RetentionDays: 1}
	cleanup := schedulers.NewCleanupScheduler(s.eventRepo, s.auditRepo, cfg)
	deleted := cleanup.RunOnce(ctx)
	s.Equal(int64(2), deleted)
}

func (s *DefaultServiceSuite) TestSchedulers_Start_CancelledContext() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := &config.Config{
		DispatchIntervalSeconds: 1,
		RetryIntervalSeconds:    1,
		TimerIntervalSeconds:    1,
		TimeoutIntervalSeconds:  1,
		OutboxIntervalSeconds:   1,
		TimerClaimTTLSeconds:    1,
		CleanupIntervalHours:    1,
	}

	queueMgr := &fakeQueueManager{}

	dispatch := schedulers.NewDispatchScheduler(s.execRepo, s.stateEngine(), queueMgr, cfg, s.metrics)
	retry := schedulers.NewRetryScheduler(s.execRepo, s.instanceRepo, cfg, s.metrics)
	timer := schedulers.NewTimerScheduler(s.timerRepo, s.stateEngine(), cfg, s.metrics)
	timeout := schedulers.NewTimeoutScheduler(s.execRepo, s.instanceRepo, s.retryRepo, s.auditRepo, cfg, s.metrics)
	outbox := schedulers.NewOutboxScheduler(s.eventRepo, queueMgr, cfg, s.metrics)
	cron := schedulers.NewCronScheduler(s.scheduleRepo, s.eventRepo, cfg)
	cleanup := schedulers.NewCleanupScheduler(s.eventRepo, s.auditRepo, cfg)

	done := make(chan struct{})
	go func() { dispatch.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { retry.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { timer.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { timeout.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { outbox.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { cron.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { cleanup.Start(ctx); done <- struct{}{} }()
	<-done
}
