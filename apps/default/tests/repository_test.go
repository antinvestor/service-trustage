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
	"time"

	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
)

func (s *DefaultServiceSuite) TestEventLogRepository_Lifecycle() {
	ctx := s.tenantCtx()

	event := &models.EventLog{
		EventType:      "user.created",
		Source:         "api",
		IdempotencyKey: "idem-1",
		Payload:        `{"user_id":"u1"}`,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, event))
	s.NotEmpty(event.ID)

	found, err := s.eventRepo.FindByIdempotencyKey(ctx, "idem-1")
	s.Require().NoError(err)
	s.Equal(event.ID, found.ID)

	unpublished, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Len(unpublished, 1)

	processed := 0
	unscopedCtx := security.SkipTenancyChecksOnClaims(ctx)
	count, err := s.eventRepo.FindAndProcessUnpublished(unscopedCtx, 10, func(_ *models.EventLog) error {
		processed++
		return nil
	})
	s.Require().NoError(err)
	s.Equal(processed, count)

	s.Require().NoError(s.eventRepo.MarkPublished(unscopedCtx, event.ID))

	_, err = s.eventRepo.DeletePublishedBefore(unscopedCtx, time.Now().Add(time.Minute), 10)
	s.Require().NoError(err)
}

func (s *DefaultServiceSuite) TestAuditRepository_DeleteBefore() {
	ctx := s.tenantCtx()

	event := &models.WorkflowAuditEvent{
		InstanceID: "inst-1",
		EventType:  "state.completed",
		State:      "step-a",
	}
	s.Require().NoError(s.auditRepo.Append(ctx, event))

	deleted, err := s.auditRepo.DeleteBefore(ctx, time.Now().Add(time.Minute), 10)
	s.Require().NoError(err)
	s.Equal(int64(1), deleted)

	events, err := s.auditRepo.ListByInstance(ctx, "inst-1")
	s.Require().NoError(err)
	s.Empty(events)
}

func (s *DefaultServiceSuite) TestWorkflowExecutionRepository_VerifyAndConsumeToken() {
	ctx := s.tenantCtx()

	exec := &models.WorkflowStateExecution{
		InstanceID:     "inst-1",
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusDispatched,
		InputPayload:   "{}",
		ExecutionToken: cryptoutil.HashToken("token-1"),
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	fetched, err := s.execRepo.VerifyAndConsumeToken(ctx, exec.ID, cryptoutil.HashToken("token-1"))
	s.Require().NoError(err)
	s.Equal(exec.ID, fetched.ID)

	execAfter, err := s.execRepo.GetByID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Empty(execAfter.ExecutionToken)
}

func (s *DefaultServiceSuite) TestWorkflowExecutionRepository_FindPendingRetryTimedOut() {
	ctx := s.tenantCtx()

	pending := &models.WorkflowStateExecution{
		InstanceID:   "inst-1",
		State:        "step-a",
		Attempt:      1,
		Status:       models.ExecStatusPending,
		InputPayload: "{}",
	}
	retryDue := &models.WorkflowStateExecution{
		InstanceID:   "inst-2",
		State:        "step-b",
		Attempt:      1,
		Status:       models.ExecStatusRetryScheduled,
		InputPayload: "{}",
		NextRetryAt:  func() *time.Time { t := time.Now().Add(-time.Minute); return &t }(),
	}
	timedOut := &models.WorkflowStateExecution{
		InstanceID:   "inst-3",
		State:        "step-c",
		Attempt:      1,
		Status:       models.ExecStatusDispatched,
		InputPayload: "{}",
		StartedAt:    func() *time.Time { t := time.Now().Add(-2 * time.Minute); return &t }(),
	}

	s.Require().NoError(s.execRepo.Create(ctx, pending))
	s.Require().NoError(s.execRepo.Create(ctx, retryDue))
	s.Require().NoError(s.execRepo.Create(ctx, timedOut))

	pendingList, err := s.execRepo.FindPending(ctx, 10)
	s.Require().NoError(err)
	s.Len(pendingList, 1)

	retryList, err := s.execRepo.FindRetryDue(ctx, 10)
	s.Require().NoError(err)
	s.Len(retryList, 1)

	timeoutList, err := s.execRepo.FindTimedOut(ctx, 30, 10)
	s.Require().NoError(err)
	s.Len(timeoutList, 1)
}

func (s *DefaultServiceSuite) TestScheduleRepository_ClaimAndFire() {
	ctx := s.tenantCtx()
	now := time.Now().UTC()
	due := now.Add(-time.Minute)

	schedule := &models.ScheduleDefinition{
		Name:            "sched-1",
		CronExpr:        "*/5 * * * *",
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		InputPayload:    "{}",
		Active:          true,
		NextFireAt:      &due,
	}
	s.Require().NoError(s.scheduleRepo.Create(ctx, schedule))

	plan := func(_ context.Context, sched *models.ScheduleDefinition) (*models.EventLog, *time.Time, int, error) {
		next := now.Add(5 * time.Minute)
		payload := events.BuildScheduleFiredPayload(sched.ID, sched.Name, now.Format(time.RFC3339), nil)
		raw, _ := payload.ToJSON()
		ev := &models.EventLog{
			EventType:      events.ScheduleFiredType,
			Source:         "schedule:" + sched.ID,
			IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
			Payload:        raw,
		}
		ev.CopyPartitionInfo(&sched.BaseModel)
		return ev, &next, 0, nil
	}

	count, _, err := s.scheduleRepo.ClaimAndFireBatch(ctx, plan, now, 10)
	s.Require().NoError(err)
	s.Equal(1, count)
}
