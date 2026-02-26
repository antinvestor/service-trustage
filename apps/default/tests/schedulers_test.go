package tests

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/schedulers"
)

type publishedMessage struct {
	ref     string
	payload any
}

type fakeQueueManager struct {
	published []publishedMessage
}

func (f *fakeQueueManager) AddPublisher(ctx context.Context, reference string, queueURL string) error {
	return nil
}
func (f *fakeQueueManager) GetPublisher(reference string) (queue.Publisher, error) { return nil, nil }
func (f *fakeQueueManager) DiscardPublisher(ctx context.Context, reference string) error { return nil }
func (f *fakeQueueManager) AddSubscriber(ctx context.Context, reference string, queueURL string, handlers ...queue.SubscribeWorker) error {
	return nil
}
func (f *fakeQueueManager) DiscardSubscriber(ctx context.Context, reference string) error { return nil }
func (f *fakeQueueManager) GetSubscriber(reference string) (queue.Subscriber, error) { return nil, nil }
func (f *fakeQueueManager) Publish(ctx context.Context, reference string, payload any, headers ...map[string]string) error {
	f.published = append(f.published, publishedMessage{ref: reference, payload: payload})
	return nil
}
func (f *fakeQueueManager) Init(ctx context.Context) error { return nil }
func (f *fakeQueueManager) Close(ctx context.Context) error { return nil }

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
		TimeoutIntervalSeconds:  1,
		OutboxIntervalSeconds:   1,
		CleanupIntervalHours:    1,
	}

	queueMgr := &fakeQueueManager{}

	dispatch := schedulers.NewDispatchScheduler(s.execRepo, s.stateEngine(), queueMgr, cfg, s.metrics)
	retry := schedulers.NewRetryScheduler(s.execRepo, s.instanceRepo, cfg, s.metrics)
	timeout := schedulers.NewTimeoutScheduler(s.execRepo, s.instanceRepo, s.retryRepo, s.auditRepo, cfg, s.metrics)
	outbox := schedulers.NewOutboxScheduler(s.eventRepo, queueMgr, cfg, s.metrics)
	cron := schedulers.NewCronScheduler(s.scheduleRepo, s.eventRepo, cfg)
	cleanup := schedulers.NewCleanupScheduler(s.eventRepo, s.auditRepo, cfg)

	done := make(chan struct{})
	go func() { dispatch.Start(ctx); done <- struct{}{} }()
	<-done
	go func() { retry.Start(ctx); done <- struct{}{} }()
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
