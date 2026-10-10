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

//nolint:testpackage // package-local repository tests share the suite fixture.
package repository

import (
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// Regression: gorm's Updates runs BaseModel's save hooks on the empty
// Model(&T{}), which generated a fresh id and added it to the WHERE clause,
// so these updates silently matched no row.

func (s *RepositorySuite) TestEventLog_MarkPublishedIfUnpublishedPersists() {
	ctx := s.tenantCtx()

	event := &models.EventLog{EventType: "a.b", Source: "api", Payload: `{}`}
	s.Require().NoError(s.eventRepo.Create(ctx, event))

	s.Require().NoError(s.eventRepo.MarkPublishedIfUnpublished(ctx, event.ID))
	s.Require().ErrorIs(s.eventRepo.MarkPublishedIfUnpublished(ctx, event.ID), ErrStaleMutation)

	pending, err := s.eventRepo.FindUnpublished(ctx, 10)
	s.Require().NoError(err)
	s.Empty(pending)
}

func (s *RepositorySuite) TestSignalDelivery_MarksMessageDeliveredAndWaitMatched() {
	ctx := s.tenantCtx()
	db := s.dbPool.DB(ctx, false)

	inst := s.createInstance(ctx, "wait", models.InstanceStatusRunning)
	exec := s.createExecution(ctx, db, inst, "wait", models.ExecStatusWaiting, "token-w")

	wait := &models.WorkflowSignalWait{
		ExecutionID: exec.ID,
		InstanceID:  inst.ID,
		State:       "wait",
		SignalName:  "go",
		Status:      "waiting",
	}
	wait.CopyPartitionInfo(&inst.BaseModel)
	s.Require().NoError(db.Create(wait).Error)

	msg := &models.WorkflowSignalMessage{
		TargetInstanceID: inst.ID,
		SignalName:       "go",
		Payload:          `{}`,
		Status:           "pending",
	}
	s.Require().NoError(s.signalMsgRepo.Create(ctx, msg))

	claim, err := s.runtimeRepo.ClaimSignalDelivery(ctx, &ClaimSignalDeliveryRequest{
		InstanceID: inst.ID,
		SignalName: "go",
		Owner:      "owner-1",
		LeaseUntil: time.Now().Add(time.Minute),
	})
	s.Require().NoError(err)
	s.Require().NotNil(claim)

	stored, err := s.signalMsgRepo.ListByInstance(ctx, inst.ID, 10)
	s.Require().NoError(err)
	s.Require().Len(stored, 1)
	s.Equal("delivered", stored[0].Status)
	s.Equal(wait.ID, stored[0].WaitID)

	matched, err := s.signalWaitRepo.GetByExecutionID(ctx, exec.ID)
	s.Require().NoError(err)
	s.Equal("matched", matched.Status)

	again, err := s.runtimeRepo.ClaimSignalDelivery(ctx, &ClaimSignalDeliveryRequest{
		InstanceID: inst.ID,
		SignalName: "go",
		Owner:      "owner-2",
		LeaseUntil: time.Now().Add(time.Minute),
	})
	s.Require().NoError(err)
	s.Nil(again, "a delivered signal is never delivered twice")
}
