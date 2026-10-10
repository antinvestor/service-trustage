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
	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *RepositorySuite) routingTarget(name string, version int) WorkflowRoutingTarget {
	return WorkflowRoutingTarget{
		WorkflowName:    name,
		WorkflowVersion: version,
		TenantID:        "test-tenant",
		PartitionID:     "test-partition",
	}
}

func (s *RepositorySuite) TestTriggerBindings_ReplaceForWorkflowSwitchesVersions() {
	ctx := s.tenantCtx()

	v1 := []*models.TriggerBinding{
		{EventType: "proposal.created", EventFilter: "payload.ok", KeyExpr: "payload.id"},
	}
	s.Require().NoError(s.triggerRepo.ReplaceForWorkflow(ctx, s.routingTarget("vote", 1), v1))
	// Idempotent: replaying the same set changes nothing.
	s.Require().NoError(s.triggerRepo.ReplaceForWorkflow(ctx, s.routingTarget("vote", 1), v1))

	found, err := s.triggerRepo.FindByEventType(ctx, "proposal.created", 10)
	s.Require().NoError(err)
	s.Require().Len(found, 1)
	s.Equal(1, found[0].WorkflowVersion)
	s.Equal("payload.id", found[0].KeyExpr)
	s.Equal("payload.ok", found[0].EventFilter)

	v2 := []*models.TriggerBinding{
		{EventType: "proposal.created", KeyExpr: "payload.proposal_id"},
		{EventType: "proposal.reopened"},
	}
	s.Require().NoError(s.triggerRepo.ReplaceForWorkflow(ctx, s.routingTarget("vote", 2), v2))

	found, err = s.triggerRepo.FindByEventType(ctx, "proposal.created", 10)
	s.Require().NoError(err)
	s.Require().Len(found, 1, "activating v2 must deactivate v1's bindings")
	s.Equal(2, found[0].WorkflowVersion)
	s.Equal("payload.proposal_id", found[0].KeyExpr)

	// Dropping a trigger from the version deactivates its stale row.
	s.Require().NoError(s.triggerRepo.ReplaceForWorkflow(ctx, s.routingTarget("vote", 2), v2[:1]))
	found, err = s.triggerRepo.FindByEventType(ctx, "proposal.reopened", 10)
	s.Require().NoError(err)
	s.Empty(found)

	// Re-activating v1 swaps back.
	s.Require().NoError(s.triggerRepo.ReplaceForWorkflow(ctx, s.routingTarget("vote", 1), v1))
	found, err = s.triggerRepo.FindByEventType(ctx, "proposal.created", 10)
	s.Require().NoError(err)
	s.Require().Len(found, 1)
	s.Equal(1, found[0].WorkflowVersion)

	s.Require().NoError(s.triggerRepo.DeactivateByWorkflowVersion(ctx, s.routingTarget("vote", 1)))
	found, err = s.triggerRepo.FindByEventType(ctx, "proposal.created", 10)
	s.Require().NoError(err)
	s.Empty(found)

	all, err := s.triggerRepo.ListByWorkflow(ctx, "vote")
	s.Require().NoError(err)
	s.Len(all, 3, "rows are deactivated, never duplicated")
}

func (s *RepositorySuite) TestSignalRoutes_ReplaceForWorkflowVersionKeepsOtherVersions() {
	ctx := s.tenantCtx()

	v1 := []*models.SignalRoute{
		{EventType: "proposal.cancelled", SignalName: "proposal.cancelled", KeyExpr: "payload.id"},
	}
	v2 := []*models.SignalRoute{
		{EventType: "proposal.cancelled", SignalName: "cancelled", InstanceExpr: "payload.meta.instance_id"},
	}
	s.Require().NoError(s.signalRouteRepo.ReplaceForWorkflowVersion(ctx, s.routingTarget("vote", 1), v1))
	s.Require().NoError(s.signalRouteRepo.ReplaceForWorkflowVersion(ctx, s.routingTarget("vote", 2), v2))
	s.Require().NoError(s.signalRouteRepo.ReplaceForWorkflowVersion(ctx, s.routingTarget("vote", 2), v2))

	routes, err := s.signalRouteRepo.FindActiveByEventType(ctx, "proposal.cancelled", 10)
	s.Require().NoError(err)
	s.Require().Len(routes, 2, "routes of an older version stay in force for its running instances")

	byVersion := map[int]*models.SignalRoute{}
	for _, r := range routes {
		byVersion[r.WorkflowVersion] = r
	}
	s.Equal("payload.id", byVersion[1].KeyExpr)
	s.Equal("cancelled", byVersion[2].SignalName)
	s.Equal("payload.meta.instance_id", byVersion[2].InstanceExpr)

	s.Require().NoError(s.signalRouteRepo.ReplaceForWorkflowVersion(ctx, s.routingTarget("vote", 2), nil))
	routes, err = s.signalRouteRepo.FindActiveByEventType(ctx, "proposal.cancelled", 10)
	s.Require().NoError(err)
	s.Require().Len(routes, 1)
	s.Equal(1, routes[0].WorkflowVersion)

	limited, err := s.signalRouteRepo.FindActiveByEventType(ctx, "proposal.cancelled", 0)
	s.Require().NoError(err)
	s.Len(limited, 1)
}

func (s *RepositorySuite) TestWorkflowInstances_ListRunningByCorrelationKey() {
	ctx := s.tenantCtx()

	mk := func(version int, key string, status models.WorkflowInstanceStatus) *models.WorkflowInstance {
		inst := &models.WorkflowInstance{
			WorkflowName:    "vote",
			WorkflowVersion: version,
			CurrentState:    "wait",
			Status:          status,
			Revision:        1,
			CorrelationKey:  key,
		}
		s.Require().NoError(s.instanceRepo.Create(ctx, inst))
		return inst
	}

	running := mk(1, "p-1", models.InstanceStatusRunning)
	mk(1, "p-1", models.InstanceStatusCompleted)
	mk(2, "p-1", models.InstanceStatusRunning)
	mk(1, "p-2", models.InstanceStatusRunning)

	got, err := s.instanceRepo.ListRunningByCorrelationKey(ctx, "vote", 1, "p-1", 10)
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(running.ID, got[0].ID)
	s.Equal("p-1", got[0].CorrelationKey)

	none, err := s.instanceRepo.ListRunningByCorrelationKey(ctx, "vote", 1, "", 10)
	s.Require().NoError(err)
	s.Empty(none, "an empty key never matches")
}

func (s *RepositorySuite) TestSignalMessages_CreateIfAbsentIsIdempotent() {
	ctx := s.tenantCtx()

	msg := func() *models.WorkflowSignalMessage {
		m := &models.WorkflowSignalMessage{
			TargetInstanceID: "inst-1",
			SignalName:       "proposal.cancelled",
			Payload:          `{"a":1}`,
			Status:           "pending",
		}
		m.ID = "wsg_fixed_id"
		return m
	}

	created, err := s.signalMsgRepo.CreateIfAbsent(ctx, msg())
	s.Require().NoError(err)
	s.True(created)

	created, err = s.signalMsgRepo.CreateIfAbsent(ctx, msg())
	s.Require().NoError(err)
	s.False(created)

	list, err := s.signalMsgRepo.ListByInstance(ctx, "inst-1", 10)
	s.Require().NoError(err)
	s.Len(list, 1)
}
