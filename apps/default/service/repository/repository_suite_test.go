package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pitabwire/frame/data"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

type RepositorySuite struct {
	frametests.FrameBaseTestSuite

	dbPool         pool.Pool
	eventRepo      EventLogRepository
	auditRepo      AuditEventRepository
	defRepo        WorkflowDefinitionRepository
	instanceRepo   WorkflowInstanceRepository
	execRepo       WorkflowExecutionRepository
	retryRepo      RetryPolicyRepository
	scheduleRepo   ScheduleRepository
	schemaRepo     SchemaRegistryRepository
	triggerRepo    TriggerBindingRepository
	outputRepo     WorkflowOutputRepository
	timerRepo      WorkflowTimerRepository
	scopeRepo      WorkflowScopeRunRepository
	signalWaitRepo WorkflowSignalWaitRepository
	signalMsgRepo  WorkflowSignalMessageRepository
	runtimeRepo    WorkflowRuntimeRepository
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupSuite() {
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
		&models.WorkflowDefinition{},
		&models.WorkflowInstance{},
		&models.WorkflowStateExecution{},
		&models.WorkflowRetryPolicy{},
		&models.ScheduleDefinition{},
		&models.WorkflowStateSchema{},
		&models.TriggerBinding{},
		&models.WorkflowStateOutput{},
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
	s.eventRepo = NewEventLogRepository(p)
	s.auditRepo = NewAuditEventRepository(p)
	s.defRepo = NewWorkflowDefinitionRepository(p)
	s.instanceRepo = NewWorkflowInstanceRepository(p)
	s.execRepo = NewWorkflowExecutionRepository(p)
	s.retryRepo = NewRetryPolicyRepository(p)
	s.scheduleRepo = NewScheduleRepository(p)
	s.schemaRepo = NewSchemaRegistryRepository(p)
	s.triggerRepo = NewTriggerBindingRepository(p)
	s.outputRepo = NewWorkflowOutputRepository(p)
	s.timerRepo = NewWorkflowTimerRepository(p)
	s.scopeRepo = NewWorkflowScopeRunRepository(p)
	s.signalWaitRepo = NewWorkflowSignalWaitRepository(p)
	s.signalMsgRepo = NewWorkflowSignalMessageRepository(p)
	s.runtimeRepo = NewWorkflowRuntimeRepository(p)
}

func (s *RepositorySuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		`TRUNCATE event_log, workflow_audit_events, workflow_definitions, workflow_instances,
		 workflow_state_executions, workflow_retry_policies, schedule_definitions, workflow_state_schemas,
		 trigger_bindings, workflow_state_outputs, workflow_timers, workflow_scope_runs,
		 workflow_signal_waits, workflow_signal_messages CASCADE`,
	).Error)
}

func (s *RepositorySuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *RepositorySuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *RepositorySuite) createDefinition(
	ctx context.Context,
	name string,
	version int,
	status models.WorkflowDefinitionStatus,
) *models.WorkflowDefinition {
	s.T().Helper()
	def := &models.WorkflowDefinition{
		Name:            name,
		WorkflowVersion: version,
		Status:          status,
		DSLBlob:         `{"version":"1.0","name":"` + name + `","steps":[]}`,
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))
	return def
}

func (s *RepositorySuite) createInstance(
	ctx context.Context,
	state string,
	status models.WorkflowInstanceStatus,
) *models.WorkflowInstance {
	s.T().Helper()
	inst := &models.WorkflowInstance{
		WorkflowName:    "payments",
		WorkflowVersion: 1,
		CurrentState:    state,
		Status:          status,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, inst))
	return inst
}

func (s *RepositorySuite) createExecution(
	_ context.Context,
	db *gorm.DB,
	inst *models.WorkflowInstance,
	state string,
	status models.ExecutionStatus,
	token string,
) *models.WorkflowStateExecution {
	s.T().Helper()
	exec := &models.WorkflowStateExecution{
		InstanceID:      inst.ID,
		State:           state,
		StateVersion:    1,
		Attempt:         1,
		Status:          status,
		ExecutionToken:  token,
		InputSchemaHash: "input-hash",
		InputPayload:    `{"hello":"world"}`,
		TraceID:         "trace-1",
	}
	exec.CopyPartitionInfo(&inst.BaseModel)
	s.Require().NoError(db.Create(exec).Error)
	return exec
}

func (s *RepositorySuite) TestWorkflowRepositories_DefinitionsInstancesAndExecutions() {
	ctx := s.tenantCtx()
	def := s.createDefinition(ctx, "payments", 1, models.WorkflowStatusActive)
	s.createDefinition(ctx, "payments", 2, models.WorkflowStatusArchived)

	loaded, err := s.defRepo.GetByID(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(def.ID, loaded.ID)

	byVersion, err := s.defRepo.GetByNameAndVersion(ctx, "payments", 1)
	s.Require().NoError(err)
	s.Equal(def.ID, byVersion.ID)

	active, err := s.defRepo.ListActiveByName(ctx, "payments", 10)
	s.Require().NoError(err)
	s.Len(active, 1)

	db := s.dbPool.DB(ctx, false)
	inst := &models.WorkflowInstance{
		WorkflowName:      "payments",
		WorkflowVersion:   1,
		CurrentState:      "start",
		Status:            models.InstanceStatusRunning,
		Revision:          1,
		TriggerEventID:    "evt-1",
		ParentExecutionID: "exec-parent",
		ScopeIndex:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, inst))

	child := &models.WorkflowInstance{
		WorkflowName:      "payments",
		WorkflowVersion:   1,
		CurrentState:      "child",
		Status:            models.InstanceStatusRunning,
		Revision:          1,
		ParentExecutionID: inst.ParentExecutionID,
		ScopeIndex:        2,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, child))

	children, err := s.instanceRepo.ListByParentExecutionID(ctx, inst.ParentExecutionID)
	s.Require().NoError(err)
	s.Len(children, 2)
	s.Equal(child.ID, children[1].ID)

	foundInst, err := s.instanceRepo.FindByTriggerEvent(
		ctx,
		inst.WorkflowName,
		inst.WorkflowVersion,
		inst.TriggerEventID,
	)
	s.Require().NoError(err)
	s.Equal(inst.ID, foundInst.ID)

	listed, err := s.instanceRepo.List(ctx, string(models.InstanceStatusRunning), "payments", 10)
	s.Require().NoError(err)
	s.Len(listed, 2)

	s.Require().NoError(s.instanceRepo.CASTransition(ctx, inst.ID, "start", 1, "review"))
	reloaded, err := s.instanceRepo.GetByID(ctx, inst.ID)
	s.Require().NoError(err)
	s.Equal("review", reloaded.CurrentState)
	s.Equal(int64(2), reloaded.Revision)

	err = s.instanceRepo.CASTransition(ctx, inst.ID, "start", 1, "done")
	s.Require().Error(err)

	execOne := s.createExecution(ctx, db, inst, "review", models.ExecStatusPending, "tok-a")
	retryAt := time.Now().Add(-time.Minute)
	startedAt := time.Now().Add(-2 * time.Minute)
	execTwo := &models.WorkflowStateExecution{
		InstanceID:      inst.ID,
		State:           "review",
		StateVersion:    1,
		Attempt:         1,
		Status:          models.ExecStatusRetryScheduled,
		ExecutionToken:  "tok-b",
		InputSchemaHash: "input-hash",
		InputPayload:    `{"hello":"world"}`,
		TraceID:         "trace-1",
		NextRetryAt:     &retryAt,
	}
	execTwo.CopyPartitionInfo(&inst.BaseModel)
	s.Require().NoError(s.execRepo.Create(ctx, execTwo))
	timeoutExec := &models.WorkflowStateExecution{
		InstanceID:      inst.ID,
		State:           "review",
		StateVersion:    1,
		Attempt:         1,
		Status:          models.ExecStatusDispatched,
		ExecutionToken:  "tok-c",
		InputSchemaHash: "input-hash",
		InputPayload:    `{"hello":"world"}`,
		TraceID:         "trace-1",
		StartedAt:       &startedAt,
	}
	timeoutExec.CopyPartitionInfo(&inst.BaseModel)
	s.Require().NoError(s.execRepo.Create(ctx, timeoutExec))

	pending, err := s.execRepo.FindPending(ctx, 10)
	s.Require().NoError(err)
	s.Len(pending, 1)

	retryDue, err := s.execRepo.FindRetryDue(ctx, 10)
	s.Require().NoError(err)
	s.Len(retryDue, 1)

	timedOut, err := s.execRepo.FindTimedOut(ctx, 30, 10)
	s.Require().NoError(err)
	s.Len(timedOut, 1)

	s.Require().
		NoError(db.Model(&models.WorkflowStateExecution{}).Where("id = ?", timeoutExec.ID).Update("status", models.ExecStatusDispatched).Error)
	consumed, err := s.execRepo.VerifyAndConsumeToken(ctx, timeoutExec.ID, "tok-c")
	s.Require().NoError(err)
	s.Equal(timeoutExec.ID, consumed.ID)

	stored, err := s.execRepo.GetByID(ctx, timeoutExec.ID)
	s.Require().NoError(err)
	s.Empty(stored.ExecutionToken)

	s.Require().
		NoError(s.execRepo.UpdateStatus(ctx, execOne.ID, models.ExecStatusCompleted, map[string]any{"error_class": "none"}))
	updatedExec, err := s.execRepo.GetByID(ctx, execOne.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusCompleted, updatedExec.Status)
	s.NotNil(updatedExec.FinishedAt)

	s.Require().NoError(s.execRepo.MarkStale(ctx, execTwo.ID))
	stale, err := s.execRepo.GetByID(ctx, execTwo.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusStale, stale.Status)
}

func (s *RepositorySuite) TestAuxiliaryRepositories_EventAuditScheduleSchemaOutputAndSignals() {
	ctx := s.tenantCtx()
	db := s.dbPool.DB(ctx, false)
	inst := s.createInstance(ctx, "start", models.InstanceStatusRunning)
	exec := s.createExecution(ctx, db, inst, "start", models.ExecStatusDispatched, "token-x")

	event := &models.EventLog{
		EventType:      "payment.requested",
		Source:         "api",
		IdempotencyKey: "idem-1",
		Payload:        `{"ok":true}`,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, event))
	foundEvent, err := s.eventRepo.FindByIdempotencyKey(ctx, "idem-1")
	s.Require().NoError(err)
	s.Equal(event.ID, foundEvent.ID)

	claimed, err := s.eventRepo.ClaimUnpublished(ctx, 10, "publisher-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimed, 1)
	s.Require().NoError(s.eventRepo.ReleaseClaim(ctx, event.ID, "publisher-1"))
	claimed, err = s.eventRepo.ClaimUnpublished(ctx, 10, "publisher-2", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimed, 1)
	s.Require().NoError(s.eventRepo.MarkPublishedByOwner(ctx, event.ID, "publisher-2", time.Now()))

	audit := &models.WorkflowAuditEvent{
		InstanceID:  inst.ID,
		ExecutionID: exec.ID,
		EventType:   "state.started",
		State:       "start",
	}
	s.Require().NoError(s.auditRepo.Append(ctx, audit))
	audits, err := s.auditRepo.ListByInstanceWithLimit(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Len(audits, 1)

	output := &models.WorkflowStateOutput{
		ExecutionID: exec.ID,
		InstanceID:  inst.ID,
		State:       "start",
		SchemaHash:  "hash-1",
		Payload:     `{"done":true}`,
	}
	s.Require().NoError(s.outputRepo.Store(ctx, output))
	gotOutput, err := s.outputRepo.GetByExecution(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(output.ID, gotOutput.ID)
	byState, err := s.outputRepo.GetByInstanceAndState(ctx, inst.ID, "start")
	s.Require().NoError(err)
	s.Equal(output.ID, byState.ID)

	policy := &models.WorkflowRetryPolicy{WorkflowName: "payments", WorkflowVersion: 1, State: "start", MaxAttempts: 3}
	s.Require().NoError(s.retryRepo.Store(ctx, policy))
	gotPolicy, err := s.retryRepo.Lookup(ctx, "payments", 1, "start")
	s.Require().NoError(err)
	s.Equal(policy.ID, gotPolicy.ID)

	fireAt := time.Now().Add(-time.Minute)
	sched := &models.ScheduleDefinition{
		Name:            "nightly",
		CronExpr:        "1h",
		WorkflowName:    "payments",
		WorkflowVersion: 1,
		Active:          true,
		NextFireAt:      &fireAt,
	}
	s.Require().NoError(s.scheduleRepo.Create(ctx, sched))
	due, err := s.scheduleRepo.FindDue(ctx, time.Now(), 10)
	s.Require().NoError(err)
	s.Len(due, 1)
	nextFire := time.Now().Add(time.Hour)
	s.Require().NoError(s.scheduleRepo.UpdateFireTimes(ctx, sched.ID, time.Now(), &nextFire))

	schema := &models.WorkflowStateSchema{
		WorkflowName:    "payments",
		WorkflowVersion: 1,
		State:           "start",
		SchemaType:      models.SchemaTypeInput,
		SchemaHash:      "hash-a",
		SchemaBlob:      []byte(`{"type":"object"}`),
	}
	s.Require().NoError(s.schemaRepo.Store(ctx, schema))
	schemaDup := *schema
	schemaDup.BaseModel.ID = ""
	s.Require().NoError(s.schemaRepo.Store(ctx, &schemaDup))
	gotSchema, err := s.schemaRepo.Lookup(ctx, "payments", 1, "start", models.SchemaTypeInput)
	s.Require().NoError(err)
	s.Equal(schema.ID, gotSchema.ID)
	gotByHash, err := s.schemaRepo.LookupByHash(ctx, "hash-a")
	s.Require().NoError(err)
	s.Equal(schema.ID, gotByHash.ID)

	trigger := &models.TriggerBinding{
		EventType:       "payment.requested",
		WorkflowName:    "payments",
		WorkflowVersion: 1,
		Active:          true,
	}
	s.Require().NoError(s.triggerRepo.Create(ctx, trigger))
	bindings, err := s.triggerRepo.FindByEventType(ctx, "payment.requested")
	s.Require().NoError(err)
	s.Len(bindings, 1)

	timer := &models.WorkflowTimer{
		ExecutionID: exec.ID,
		InstanceID:  inst.ID,
		State:       "start",
		FiresAt:     time.Now().Add(-time.Minute),
	}
	s.Require().NoError(s.timerRepo.Create(ctx, timer))
	timers, err := s.timerRepo.ClaimDue(ctx, time.Now(), 10, "timer-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(timers, 1)
	s.Require().NoError(s.timerRepo.ReleaseClaim(ctx, timer.ID, "timer-1"))
	timers, err = s.timerRepo.ClaimDue(ctx, time.Now(), 10, "timer-2", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Require().NoError(s.timerRepo.MarkFiredByOwner(ctx, timer.ID, "timer-2", time.Now()))

	scope := &models.WorkflowScopeRun{
		ParentExecutionID: exec.ID,
		ParentInstanceID:  inst.ID,
		ParentState:       "fanout",
		ScopeType:         "parallel",
		Status:            "running",
		TotalChildren:     2,
	}
	s.Require().NoError(s.scopeRepo.Create(ctx, scope))
	scopes, err := s.scopeRepo.ClaimRunning(ctx, 10, "scope-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(scopes, 1)
	s.Require().NoError(s.scopeRepo.ReleaseClaim(ctx, scope.ID, "scope-1"))

	wait := &models.WorkflowSignalWait{
		ExecutionID: exec.ID,
		InstanceID:  inst.ID,
		State:       "wait",
		SignalName:  "approved",
		Status:      "waiting",
		TimeoutAt:   timePtr(time.Now().Add(-time.Minute)),
	}
	s.Require().NoError(s.signalWaitRepo.Create(ctx, wait))
	waits, err := s.signalWaitRepo.ClaimTimedOut(ctx, time.Now(), 10, "wait-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(waits, 1)
	s.Require().NoError(s.signalWaitRepo.ReleaseClaim(ctx, wait.ID, "wait-1"))
	waits, err = s.signalWaitRepo.ClaimTimedOut(ctx, time.Now(), 10, "wait-2", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Require().NoError(s.signalWaitRepo.MarkTimedOutByOwner(ctx, wait.ID, "wait-2", time.Now()))

	message := &models.WorkflowSignalMessage{
		TargetInstanceID: inst.ID,
		SignalName:       "approved",
		Payload:          `{"approved":true}`,
		Status:           "pending",
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, message))
	claimedMsg, err := s.signalMsgRepo.ClaimOldestPendingForTarget(
		ctx,
		inst.ID,
		"approved",
		"msg-1",
		time.Now().Add(time.Minute),
	)
	s.Require().NoError(err)
	s.NotNil(claimedMsg)
	s.Require().NoError(s.signalMsgRepo.MarkDeliveredByOwner(ctx, message.ID, "msg-1", wait.ID, time.Now()))
}

func (s *RepositorySuite) TestWorkflowRuntimeRepository_CommitAndWaitFlows() {
	ctx := s.tenantCtx()
	db := s.dbPool.DB(ctx, false)
	inst := s.createInstance(ctx, "start", models.InstanceStatusRunning)
	exec := s.createExecution(ctx, db, inst, "start", models.ExecStatusDispatched, "token-1")

	err := s.runtimeRepo.CommitExecution(ctx, &CommitExecutionRequest{
		Execution:           exec,
		Instance:            inst,
		TokenHash:           "token-1",
		VerifyToken:         true,
		OutputPayload:       `{"ok":true}`,
		ExpectedStatus:      models.ExecStatusDispatched,
		NextState:           "next",
		NextInputPayload:    `{"carry":true}`,
		NextInputSchemaHash: "hash-next",
	})
	s.Require().NoError(err)

	reloadedInst, err := s.instanceRepo.GetByID(ctx, inst.ID)
	s.Require().NoError(err)
	s.Equal("next", reloadedInst.CurrentState)

	execs, err := s.execRepo.List(ctx, "", inst.ID, 10)
	s.Require().NoError(err)
	s.Len(execs, 2)

	waitInst := s.createInstance(ctx, "delay", models.InstanceStatusRunning)
	waitExec := s.createExecution(ctx, db, waitInst, "delay", models.ExecStatusDispatched, "token-delay")
	s.Require().NoError(s.runtimeRepo.ParkExecution(ctx, &ParkExecutionRequest{
		Execution:  waitExec,
		Instance:   waitInst,
		TokenHash:  "token-delay",
		FireAt:     time.Now().Add(time.Minute),
		AuditTrace: "trace-delay",
	}))
	timer, err := s.timerRepo.GetByExecutionID(ctx, waitExec.ID)
	s.Require().NoError(err)
	s.Equal(waitExec.ID, timer.ExecutionID)

	signalInst := s.createInstance(ctx, "signal", models.InstanceStatusRunning)
	signalExec := s.createExecution(ctx, db, signalInst, "signal", models.ExecStatusDispatched, "token-signal")
	timeoutAt := time.Now().Add(time.Minute)
	s.Require().NoError(s.runtimeRepo.StartSignalWait(ctx, &StartSignalWaitRequest{
		Execution:  signalExec,
		Instance:   signalInst,
		TokenHash:  "token-signal",
		SignalName: "approved",
		OutputVar:  "approval",
		TimeoutAt:  &timeoutAt,
		AuditTrace: "trace-signal",
	}))

	wait, err := s.signalWaitRepo.GetByExecutionID(ctx, signalExec.ID)
	s.Require().NoError(err)
	s.Equal("approved", wait.SignalName)

	msg := &models.WorkflowSignalMessage{
		TargetInstanceID: signalInst.ID,
		SignalName:       "approved",
		Payload:          `{"approved":true}`,
		Status:           "pending",
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, msg))
	claim, err := s.runtimeRepo.ClaimSignalDelivery(ctx, &ClaimSignalDeliveryRequest{
		InstanceID: signalInst.ID,
		SignalName: "approved",
		Owner:      "delivery-1",
		LeaseUntil: time.Now().Add(time.Minute),
	})
	s.Require().NoError(err)
	s.NotNil(claim)
	s.NotNil(claim.Message)
	s.NotNil(claim.Wait)

	failInst := s.createInstance(ctx, "fatal", models.InstanceStatusRunning)
	failExec := s.createExecution(ctx, db, failInst, "fatal", models.ExecStatusDispatched, "token-fail")
	s.Require().NoError(s.runtimeRepo.FailExecution(ctx, &FailExecutionRequest{
		Execution:      failExec,
		Instance:       failInst,
		TokenHash:      "token-fail",
		VerifyToken:    true,
		ExpectedStatus: models.ExecStatusDispatched,
		Status:         models.ExecStatusFailed,
		ErrorClass:     "external_dependency",
		ErrorMessage:   "boom",
		AuditTrace:     "trace-fail",
		AuditPayload:   `{"boom":true}`,
	}))
	failedInst, err := s.instanceRepo.GetByID(ctx, failInst.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusFailed, failedInst.Status)

	retryInst := s.createInstance(ctx, "retry", models.InstanceStatusFailed)
	retryExec := s.createExecution(ctx, db, retryInst, "retry", models.ExecStatusTimedOut, "")
	newExec := &models.WorkflowStateExecution{
		InstanceID:      retryInst.ID,
		State:           "retry",
		StateVersion:    1,
		Attempt:         2,
		Status:          models.ExecStatusPending,
		ExecutionToken:  "token-retry",
		InputSchemaHash: "hash-retry",
		InputPayload:    `{"retry":true}`,
		TraceID:         "trace-retry",
	}
	_, err = s.runtimeRepo.CreateRetryExecution(ctx, &CreateRetryExecutionRequest{
		Execution:    retryExec,
		Instance:     retryInst,
		NewExecution: newExec,
	})
	s.Require().NoError(err)

	reloadedRetryInst, err := s.instanceRepo.GetByID(ctx, retryInst.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusRunning, reloadedRetryInst.Status)

	scopeInst := s.createInstance(ctx, "parallel", models.InstanceStatusRunning)
	scopeExec := s.createExecution(ctx, db, scopeInst, "parallel", models.ExecStatusDispatched, "token-scope")
	scope := &models.WorkflowScopeRun{
		ParentExecutionID: scopeExec.ID,
		ParentInstanceID:  scopeInst.ID,
		ParentState:       "parallel",
		ScopeType:         "parallel",
		Status:            "running",
		TotalChildren:     1,
		NextChildIndex:    1,
	}
	childInst := &models.WorkflowInstance{
		BaseModel: data.BaseModel{
			ID: util.IDString(),
		},
		WorkflowName:      "payments",
		WorkflowVersion:   1,
		CurrentState:      "child",
		Status:            models.InstanceStatusRunning,
		Revision:          1,
		ParentInstanceID:  scopeInst.ID,
		ParentExecutionID: scopeExec.ID,
	}
	childInst.CopyPartitionInfo(&scopeInst.BaseModel)
	childExec := &models.WorkflowStateExecution{
		BaseModel: data.BaseModel{
			ID: util.IDString(),
		},
		InstanceID:      childInst.ID,
		State:           "child",
		StateVersion:    1,
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  "child-token",
		InputSchemaHash: "child-hash",
		InputPayload:    `{}`,
		TraceID:         "trace-child",
	}
	childExec.CopyPartitionInfo(&scopeInst.BaseModel)
	s.Require().NoError(s.runtimeRepo.StartBranchScope(ctx, &StartBranchScopeRequest{
		Execution: scopeExec,
		Instance:  scopeInst,
		TokenHash: "token-scope",
		Scope:     scope,
		LaunchChildren: []*ScopedChildRecord{{
			Instance:  childInst,
			Execution: childExec,
		}},
		AuditTrace: "trace-scope",
	}))

	children, err := s.instanceRepo.ListByParentExecutionID(ctx, scopeExec.ID)
	s.Require().NoError(err)
	s.Len(children, 1)

	s.Require().NoError(s.runtimeRepo.UpdateScope(ctx, &UpdateScopeRequest{
		ScopeID:               scope.ID,
		Status:                "completed",
		CompletedChildren:     1,
		FailedChildren:        0,
		NextChildIndex:        1,
		ResultsPayload:        `[{"ok":true}]`,
		ReleaseClaim:          true,
		CancelRunningChildren: true,
		ParentExecutionID:     scopeExec.ID,
	}))
}

func (s *RepositorySuite) TestRepositoryHelpers_EventAuditSignalAndProjectionPaths() {
	ctx := s.tenantCtx()
	db := s.dbPool.DB(ctx, false)

	def := s.createDefinition(ctx, "payments", 1, models.WorkflowStatusDraft)
	def.TimeoutSeconds = 45
	s.Require().NoError(s.defRepo.Update(ctx, def))
	updatedDef, err := s.defRepo.GetByID(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(int64(45), updatedDef.TimeoutSeconds)

	inst := s.createInstance(ctx, "wait", models.InstanceStatusRunning)
	s.Require().NoError(s.instanceRepo.UpdateStatus(ctx, inst.ID, models.InstanceStatusCompleted))
	completedInst, err := s.instanceRepo.GetByID(ctx, inst.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, completedInst.Status)

	execOne := s.createExecution(ctx, db, inst, "wait", models.ExecStatusPending, "tok-one")
	time.Sleep(10 * time.Millisecond)
	execTwo := s.createExecution(ctx, db, inst, "after", models.ExecStatusPending, "tok-two")
	latestExec, err := s.execRepo.GetLatestByInstance(ctx, inst.ID)
	s.Require().NoError(err)
	s.Equal(execTwo.ID, latestExec.ID)

	s.Require().NoError(s.runtimeRepo.UpdateExecutionStatus(ctx, execOne, models.ExecStatusRunning))
	updatedExec, err := s.execRepo.GetByID(ctx, execOne.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusRunning, updatedExec.Status)

	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: execOne.ID,
		InstanceID:  inst.ID,
		State:       "wait",
		SchemaHash:  "hash-1",
		Payload:     `{"ok":true}`,
	}))
	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: execTwo.ID,
		InstanceID:  inst.ID,
		State:       "after",
		SchemaHash:  "hash-2",
		Payload:     `{"ok":false}`,
	}))
	outputs, err := s.outputRepo.ListByInstance(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Len(outputs, 2)

	oldAudit := &models.WorkflowAuditEvent{InstanceID: inst.ID, EventType: "old"}
	newAudit := &models.WorkflowAuditEvent{InstanceID: inst.ID, EventType: "new"}
	s.Require().NoError(s.auditRepo.Append(ctx, oldAudit))
	s.Require().NoError(s.auditRepo.Append(ctx, newAudit))
	oldTime := time.Now().Add(-48 * time.Hour)
	s.Require().NoError(s.dbPool.DB(ctx, false).
		Model(&models.WorkflowAuditEvent{}).
		Where("id = ?", oldAudit.ID).
		UpdateColumn("created_at", oldTime).Error)

	audits, err := s.auditRepo.ListByInstance(ctx, inst.ID)
	s.Require().NoError(err)
	s.Len(audits, 2)

	deletedAuditCount, err := s.auditRepo.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 10)
	s.Require().NoError(err)
	s.Equal(int64(1), deletedAuditCount)

	unpublished := &models.EventLog{EventType: "evt.pending", Source: "api", Payload: `{"ok":true}`}
	claimedCompat := &models.EventLog{EventType: "evt.compat", Source: "api", Payload: `{"ok":2}`}
	oldPublishedAt := time.Now().Add(-48 * time.Hour)
	published := &models.EventLog{
		EventType:   "evt.done",
		Source:      "api",
		Payload:     `{"ok":3}`,
		Published:   true,
		PublishedAt: &oldPublishedAt,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, unpublished))
	s.Require().NoError(s.eventRepo.Create(ctx, claimedCompat))
	s.Require().NoError(s.eventRepo.Create(ctx, published))

	unpublishedItems, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(unpublishedItems, 2)

	s.Require().NoError(s.eventRepo.MarkPublished(ctx, unpublished.ID))

	processed, err := s.eventRepo.FindAndProcessUnpublished(ctx, 10, func(event *models.EventLog) error {
		if event.ID == claimedCompat.ID {
			return nil
		}
		return gorm.ErrInvalidData
	})
	s.Require().NoError(err)
	s.Equal(1, processed)

	deletedEventCount, err := s.eventRepo.DeletePublishedBefore(ctx, time.Now().Add(-24*time.Hour), 10)
	s.Require().NoError(err)
	s.Equal(int64(1), deletedEventCount)

	scope := &models.WorkflowScopeRun{
		ParentExecutionID: execTwo.ID,
		ParentInstanceID:  inst.ID,
		ParentState:       "fanout",
		ScopeType:         "parallel",
		Status:            "running",
		TotalChildren:     1,
	}
	s.Require().NoError(s.scopeRepo.Create(ctx, scope))
	gotScope, err := s.scopeRepo.GetByID(ctx, scope.ID)
	s.Require().NoError(err)
	s.Equal(scope.ID, gotScope.ID)
	gotScopeByExec, err := s.scopeRepo.GetByParentExecutionID(ctx, execTwo.ID)
	s.Require().NoError(err)
	s.Equal(scope.ID, gotScopeByExec.ID)
	scopeList, err := s.scopeRepo.ListByInstance(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Len(scopeList, 1)
	s.NotNil(s.scopeRepo.Pool())

	waitTimeout := time.Now().Add(-time.Minute)
	wait := &models.WorkflowSignalWait{
		ExecutionID: execTwo.ID,
		InstanceID:  inst.ID,
		State:       "wait",
		SignalName:  "approved",
		OutputVar:   "approval",
		Status:      "waiting",
		TimeoutAt:   &waitTimeout,
	}
	s.Require().NoError(s.signalWaitRepo.Create(ctx, wait))
	foundWait, err := s.signalWaitRepo.FindActiveByInstanceAndSignal(ctx, inst.ID, "approved")
	s.Require().NoError(err)
	s.Equal(wait.ID, foundWait.ID)
	waits, err := s.signalWaitRepo.ListByInstance(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Len(waits, 1)
	claimedWaits, err := s.signalWaitRepo.ClaimTimedOut(ctx, time.Now(), 10, "wait-owner", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimedWaits, 1)
	s.Require().NoError(s.signalWaitRepo.ReleaseClaim(ctx, wait.ID, "wait-owner"))
	claimedWaits, err = s.signalWaitRepo.ClaimTimedOut(ctx, time.Now(), 10, "wait-owner-2", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimedWaits, 1)
	s.Require().NoError(s.signalWaitRepo.MarkCompletedByOwner(ctx, wait.ID, "wait-owner-2", "msg-1", time.Now()))

	message := &models.WorkflowSignalMessage{
		TargetInstanceID: inst.ID,
		SignalName:       "approved",
		Payload:          `{"ok":true}`,
		Status:           "pending",
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, message))
	messages, err := s.signalMsgRepo.ListByInstance(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Len(messages, 1)
	claimedMessage, err := s.signalMsgRepo.ClaimOldestPendingForTarget(
		ctx,
		inst.ID,
		"approved",
		"msg-owner",
		time.Now().Add(time.Minute),
	)
	s.Require().NoError(err)
	s.Require().NotNil(claimedMessage)
	s.Require().NoError(s.signalMsgRepo.ReleaseClaim(ctx, message.ID, "msg-owner"))
	claimedMessage, err = s.signalMsgRepo.ClaimOldestPendingForTarget(
		ctx,
		inst.ID,
		"approved",
		"msg-owner-2",
		time.Now().Add(time.Minute),
	)
	s.Require().NoError(err)
	s.Require().NotNil(claimedMessage)
	s.Require().NoError(s.signalMsgRepo.MarkDeliveredByOwner(ctx, message.ID, "msg-owner-2", wait.ID, time.Now()))
}

func timePtr(t time.Time) *time.Time { return &t }
