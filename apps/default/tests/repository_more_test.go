package tests_test

import (
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *DefaultServiceSuite) TestRepository_ScopeSignalAndOutputLifecycle() {
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

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token",
		InputSchemaHash: "hash",
		InputPayload:    `{"hello":"world"}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	scope := &models.WorkflowScopeRun{
		ParentExecutionID: exec.ID,
		ParentInstanceID:  instance.ID,
		ParentState:       "fanout",
		ScopeType:         "parallel",
		Status:            "running",
		WaitAll:           true,
		TotalChildren:     2,
		ResultsPayload:    `[null,null]`,
	}
	s.Require().NoError(s.scopeRepo.Create(ctx, scope))

	foundScope, err := s.scopeRepo.GetByParentExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(scope.ID, foundScope.ID)

	scopeItems, err := s.scopeRepo.ListByInstance(ctx, instance.ID, 10)
	s.Require().NoError(err)
	s.Len(scopeItems, 1)

	claimedScopes, err := s.scopeRepo.ClaimRunning(ctx, 10, "owner-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimedScopes, 1)
	s.Require().NoError(s.scopeRepo.ReleaseClaim(ctx, scope.ID, "owner-1"))

	timeoutAt := time.Now().Add(-time.Minute)
	wait := &models.WorkflowSignalWait{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "wait",
		SignalName:  "approved",
		OutputVar:   "approval",
		Status:      "waiting",
		TimeoutAt:   &timeoutAt,
	}
	s.Require().NoError(s.signalWaitRepo.Create(ctx, wait))

	foundWait, err := s.signalWaitRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal(wait.ID, foundWait.ID)

	activeWait, err := s.signalWaitRepo.FindActiveByInstanceAndSignal(ctx, instance.ID, "approved")
	s.Require().NoError(err)
	s.Equal(wait.ID, activeWait.ID)

	waitItems, err := s.signalWaitRepo.ListByInstance(ctx, instance.ID, 10)
	s.Require().NoError(err)
	s.Len(waitItems, 1)

	claimedWaits, err := s.signalWaitRepo.ClaimTimedOut(
		ctx,
		time.Now().Add(time.Minute),
		10,
		"wait-owner",
		time.Now().Add(time.Minute),
	)
	s.Require().NoError(err)
	s.Len(claimedWaits, 1)
	s.Require().NoError(s.signalWaitRepo.ReleaseClaim(ctx, wait.ID, "wait-owner"))

	message := &models.WorkflowSignalMessage{
		TargetInstanceID: instance.ID,
		SignalName:       "approved",
		Payload:          `{"ok":true}`,
		Status:           "pending",
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, message))

	messageItems, err := s.signalMsgRepo.ListByInstance(ctx, instance.ID, 10)
	s.Require().NoError(err)
	s.Len(messageItems, 1)

	claimedMessage, err := s.signalMsgRepo.ClaimOldestPendingForTarget(
		ctx,
		instance.ID,
		"approved",
		"msg-owner",
		time.Now().Add(time.Minute),
	)
	s.Require().NoError(err)
	s.Require().NotNil(claimedMessage)
	s.Require().NoError(s.signalMsgRepo.ReleaseClaim(ctx, message.ID, "msg-owner"))

	output := &models.WorkflowStateOutput{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "wait",
		SchemaHash:  "hash",
		Payload:     `{"done":true}`,
	}
	s.Require().NoError(s.outputRepo.Store(ctx, output))

	outputItems, err := s.outputRepo.ListByInstance(ctx, instance.ID, 10)
	s.Require().NoError(err)
	s.Len(outputItems, 1)
}

func (s *DefaultServiceSuite) TestRepository_EventLogClaimReleaseAndPublish() {
	ctx := s.tenantCtx()

	event := &models.EventLog{
		EventType: "user.created",
		Source:    "api",
		Payload:   `{"user_id":"u-1"}`,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, event))

	claimed, err := s.eventRepo.ClaimUnpublished(ctx, 10, "publisher-1", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimed, 1)
	s.Require().NoError(s.eventRepo.ReleaseClaim(ctx, event.ID, "publisher-1"))

	claimed, err = s.eventRepo.ClaimUnpublished(ctx, 10, "publisher-2", time.Now().Add(time.Minute))
	s.Require().NoError(err)
	s.Len(claimed, 1)

	stored := &models.EventLog{}
	s.Require().NoError(
		s.dbPool.DB(ctx, true).Where("id = ?", event.ID).First(stored).Error,
	)
	s.Equal("publisher-2", stored.PublishClaimOwner)
	s.NotNil(stored.PublishClaimUntil)

	s.Require().NoError(s.eventRepo.MarkPublishedByOwner(ctx, event.ID, "publisher-2", time.Now()))

	stored = &models.EventLog{}
	s.Require().NoError(
		s.dbPool.DB(ctx, true).Where("id = ?", event.ID).First(stored).Error,
	)
	s.True(stored.Published)
	s.NotNil(stored.PublishedAt)
	s.Empty(stored.PublishClaimOwner)
	s.Nil(stored.PublishClaimUntil)

	unpublished, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Empty(unpublished)
}

func (s *DefaultServiceSuite) TestRepository_InstanceSchemaSignalAndTimerMutations() {
	ctx := s.tenantCtx()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        "{}",
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	s.Require().NoError(s.instanceRepo.CASTransition(ctx, instance.ID, "step-a", 1, "step-b"))
	updatedInstance, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal("step-b", updatedInstance.CurrentState)
	s.Equal(int64(2), updatedInstance.Revision)

	err = s.instanceRepo.CASTransition(ctx, instance.ID, "step-a", 1, "step-c")
	s.Require().Error(err)

	s.Require().NoError(s.instanceRepo.UpdateStatus(ctx, instance.ID, models.InstanceStatusCompleted))
	updatedInstance, err = s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, updatedInstance.Status)
	s.NotNil(updatedInstance.FinishedAt)

	schema := &models.WorkflowStateSchema{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		State:           "step-a",
		SchemaType:      models.SchemaTypeInput,
		SchemaHash:      "hash-1",
		SchemaBlob:      []byte(`{"type":"object"}`),
	}
	s.Require().NoError(s.schemaRepo.Store(ctx, schema))
	s.Require().NoError(s.schemaRepo.Store(ctx, schema))

	storedSchema, err := s.schemaRepo.Lookup(ctx, "wf", 1, "step-a", models.SchemaTypeInput)
	s.Require().NoError(err)
	s.Equal(schema.SchemaHash, storedSchema.SchemaHash)

	storedByHash, err := s.schemaRepo.LookupByHash(ctx, schema.SchemaHash)
	s.Require().NoError(err)
	s.Equal(schema.ID, storedByHash.ID)

	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "step-b",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token",
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	claimUntil := time.Now().Add(time.Minute)
	wait := &models.WorkflowSignalWait{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       exec.State,
		SignalName:  "approved",
		Status:      "waiting",
		ClaimOwner:  "wait-owner",
		ClaimUntil:  &claimUntil,
	}
	s.Require().NoError(s.signalWaitRepo.Create(ctx, wait))
	s.Require().NoError(
		s.signalWaitRepo.MarkCompletedByOwner(ctx, wait.ID, "wait-owner", "message-1", time.Now()),
	)

	completedWait, err := s.signalWaitRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal("matched", completedWait.Status)
	s.Equal("message-1", completedWait.MessageID)
	s.NotNil(completedWait.MatchedAt)
	s.Empty(completedWait.ClaimOwner)

	timeoutExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "step-c",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token-2",
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, timeoutExec))

	timeoutWait := &models.WorkflowSignalWait{
		ExecutionID: timeoutExec.ID,
		InstanceID:  instance.ID,
		State:       timeoutExec.State,
		SignalName:  "approved",
		Status:      "waiting",
		ClaimOwner:  "wait-owner-2",
		ClaimUntil:  &claimUntil,
	}
	s.Require().NoError(s.signalWaitRepo.Create(ctx, timeoutWait))
	s.Require().NoError(s.signalWaitRepo.MarkTimedOutByOwner(ctx, timeoutWait.ID, "wait-owner-2", time.Now()))

	timedOutWait, err := s.signalWaitRepo.GetByExecutionID(ctx, timeoutExec.ID)
	s.Require().NoError(err)
	s.Equal("timed_out", timedOutWait.Status)
	s.NotNil(timedOutWait.TimedOutAt)
	s.Empty(timedOutWait.ClaimOwner)

	message := &models.WorkflowSignalMessage{
		TargetInstanceID: instance.ID,
		SignalName:       "approved",
		Payload:          `{}`,
		Status:           "pending",
		ClaimOwner:       "msg-owner",
		ClaimUntil:       &claimUntil,
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, message))
	s.Require().NoError(
		s.signalMsgRepo.MarkDeliveredByOwner(ctx, message.ID, "msg-owner", wait.ID, time.Now()),
	)

	deliveredMessage := &models.WorkflowSignalMessage{}
	s.Require().NoError(s.dbPool.DB(ctx, true).Where("id = ?", message.ID).First(deliveredMessage).Error)
	s.Equal("delivered", deliveredMessage.Status)
	s.Equal(wait.ID, deliveredMessage.WaitID)
	s.NotNil(deliveredMessage.DeliveredAt)
	s.Empty(deliveredMessage.ClaimOwner)

	timer := &models.WorkflowTimer{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       exec.State,
		FiresAt:     time.Now().Add(time.Minute),
		ClaimOwner:  "timer-owner",
		ClaimUntil:  &claimUntil,
	}
	s.Require().NoError(s.timerRepo.Create(ctx, timer))
	s.Require().NoError(s.timerRepo.ReleaseClaim(ctx, timer.ID, "timer-owner"))

	releasedTimer, err := s.timerRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Empty(releasedTimer.ClaimOwner)
	s.Nil(releasedTimer.ClaimUntil)

	fireExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "step-d",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token-3",
		InputSchemaHash: "hash",
		InputPayload:    `{}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, fireExec))

	fireTimer := &models.WorkflowTimer{
		ExecutionID: fireExec.ID,
		InstanceID:  instance.ID,
		State:       fireExec.State,
		FiresAt:     time.Now().Add(time.Minute),
		ClaimOwner:  "timer-owner-2",
		ClaimUntil:  &claimUntil,
	}
	s.Require().NoError(s.timerRepo.Create(ctx, fireTimer))
	s.Require().NoError(s.timerRepo.MarkFiredByOwner(ctx, fireTimer.ID, "timer-owner-2", time.Now()))

	firedTimer, err := s.timerRepo.GetByExecutionID(ctx, fireExec.ID)
	s.Require().NoError(err)
	s.NotNil(firedTimer.FiredAt)
	s.Empty(firedTimer.ClaimOwner)
	s.Nil(firedTimer.ClaimUntil)
}
