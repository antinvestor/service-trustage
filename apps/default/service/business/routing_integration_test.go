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

//nolint:testpackage // package-local tests exercise unexported business helpers intentionally.
package business

import (
	"context"
	"strings"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/events"
)

const routedVoteDSL = `{
  "version": "1.0",
  "name": "routed-vote",
  "description": "%DESC%",
  "triggers": [
    {"event_type": "proposal.created", "filter": "!has(payload.kind) || payload.kind != 'ignored'", "key": "payload.proposal_id"}
  ],
  "signals": [
    {"event_type": "proposal.cancelled", "key": "payload.proposal_id"},
    {"event_type": "intent.finalized", "signal_name": "finalized", "instance": "payload.meta.instance_id"}
  ],
  "steps": [
    {"id": "wait_cancel", "type": "signal_wait", "signal_wait": {"signal_name": "proposal.cancelled", "output_var": "cancel"}},
    {"id": "wait_final", "type": "signal_wait", "signal_wait": {"signal_name": "finalized"}},
    {"id": "done", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`

func routedVote(desc string) string {
	return strings.ReplaceAll(routedVoteDSL, "%DESC%", desc)
}

func (s *BusinessSuite) activeBindings(ctx context.Context, name string) map[int][]*models.TriggerBinding {
	all, err := s.triggerRepo.ListByWorkflow(ctx, name)
	s.Require().NoError(err)

	out := map[int][]*models.TriggerBinding{}
	for _, b := range all {
		if b.Active {
			out[b.WorkflowVersion] = append(out[b.WorkflowVersion], b)
		}
	}

	return out
}

func (s *BusinessSuite) activeRouteVersions(ctx context.Context, eventType string) map[int]int {
	routes, err := s.signalRouteRepo.FindActiveByEventType(ctx, eventType, 50)
	s.Require().NoError(err)

	out := map[int]int{}
	for _, r := range routes {
		out[r.WorkflowVersion]++
	}

	return out
}

func (s *BusinessSuite) TestWorkflowRouting_ActivateSwitchesVersionsAndArchiveDeactivates() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	v1, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	s.Empty(s.activeBindings(ctx, "routed-vote"), "a DRAFT version has no bindings")

	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))
	bindings := s.activeBindings(ctx, "routed-vote")
	s.Require().Len(bindings[1], 1)
	s.Equal("proposal.created", bindings[1][0].EventType)
	s.Equal("!has(payload.kind) || payload.kind != 'ignored'", bindings[1][0].EventFilter)
	s.Equal("payload.proposal_id", bindings[1][0].KeyExpr)
	s.Equal(map[int]int{1: 1}, s.activeRouteVersions(ctx, "proposal.cancelled"))

	// Activating twice is idempotent.
	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))
	s.Len(s.activeBindings(ctx, "routed-vote")[1], 1)

	v2, err := biz.CreateWorkflow(ctx, []byte(routedVote("v2")))
	s.Require().NoError(err)
	s.Equal(2, v2.WorkflowVersion)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v2.ID))

	bindings = s.activeBindings(ctx, "routed-vote")
	s.Empty(bindings[1], "activating v2 deactivates v1's bindings")
	s.Len(bindings[2], 1)
	s.Equal(map[int]int{1: 1, 2: 1}, s.activeRouteVersions(ctx, "proposal.cancelled"),
		"v1's routes stay in force for its running instances")

	s.Require().NoError(biz.ArchiveWorkflow(ctx, v2.ID))
	s.Empty(s.activeBindings(ctx, "routed-vote"), "archiving v2 deactivates its bindings")
	s.Equal(map[int]int{1: 1, 2: 1}, s.activeRouteVersions(ctx, "proposal.cancelled"),
		"archiving keeps the version's signal routes")
}

func (s *BusinessSuite) TestCreateWorkflow_IdenticalActiveVersionReassertsRouting() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	def, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, def.ID))

	// Simulate a version registered before routing existed: no live bindings
	// and no routes.
	target := repository.WorkflowRoutingTarget{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		TenantID:        def.TenantID,
		PartitionID:     def.PartitionID,
	}
	s.Require().NoError(s.triggerRepo.DeactivateByWorkflowVersion(ctx, target))
	s.Require().NoError(s.signalRouteRepo.ReplaceForWorkflowVersion(ctx, target, nil))
	s.Empty(s.activeBindings(ctx, def.Name))
	s.Empty(s.activeRouteVersions(ctx, "proposal.cancelled"))

	again, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	s.Equal(def.ID, again.ID)
	s.Equal(models.WorkflowStatusActive, again.Status)

	s.Len(s.activeBindings(ctx, def.Name)[def.WorkflowVersion], 1)
	s.Equal(map[int]int{1: 1}, s.activeRouteVersions(ctx, "proposal.cancelled"))
	s.Equal(map[int]int{1: 1}, s.activeRouteVersions(ctx, "intent.finalized"))
}

func (s *BusinessSuite) TestCreateWorkflow_IdenticalDraftDoesNotBind() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	_, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	_, err = biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)

	s.Empty(s.activeBindings(ctx, "routed-vote"))
}

func (s *BusinessSuite) ingest(ctx context.Context, eventID, eventType string, payload map[string]any) int {
	created, err := s.eventRouter().RouteEvent(ctx, &events.IngestedEventMessage{
		EventID:     eventID,
		TenantID:    "test-tenant-001",
		PartitionID: "test-partition-001",
		EventType:   eventType,
		Payload:     payload,
	})
	s.Require().NoError(err)

	return created
}

func (s *BusinessSuite) pendingSignals(ctx context.Context, instanceID, signalName string) int {
	msgs, err := s.signalMsgRepo.ListByInstance(ctx, instanceID, 50)
	s.Require().NoError(err)

	n := 0
	for _, m := range msgs {
		if m.SignalName == signalName && m.Status == "pending" {
			n++
		}
	}

	return n
}

func (s *BusinessSuite) TestEventRouter_TriggersSetCorrelationKeyAndSignalsRouteIdempotently() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	def, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, def.ID))

	s.Equal(1, s.ingest(ctx, "evt-p1", "proposal.created", map[string]any{"proposal_id": "p-1"}))
	s.Equal(1, s.ingest(ctx, "evt-p2", "proposal.created", map[string]any{"proposal_id": 42}))
	s.Equal(1, s.ingest(ctx, "evt-nokey", "proposal.created", map[string]any{"other": true}))
	s.Equal(0, s.ingest(ctx, "evt-skip", "proposal.created", map[string]any{"proposal_id": "p-9", "kind": "ignored"}))

	inst1, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, "evt-p1")
	s.Require().NoError(err)
	s.Equal("p-1", inst1.CorrelationKey)

	inst2, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, "evt-p2")
	s.Require().NoError(err)
	s.Equal("42", inst2.CorrelationKey, "non-string scalars are formatted")

	noKey, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, "evt-nokey")
	s.Require().NoError(err)
	s.Empty(noKey.CorrelationKey, "a key that fails to evaluate leaves the key empty")

	// The cancel arrives before the instance waits: it is buffered once,
	// even when the event is redelivered.
	s.Equal(0, s.ingest(ctx, "evt-c1", "proposal.cancelled", map[string]any{"proposal_id": "p-1"}))
	s.Equal(0, s.ingest(ctx, "evt-c1", "proposal.cancelled", map[string]any{"proposal_id": "p-1"}))
	s.Equal(1, s.pendingSignals(ctx, inst1.ID, "proposal.cancelled"))
	s.Equal(0, s.pendingSignals(ctx, inst2.ID, "proposal.cancelled"))

	// The instance reaches its signal_wait and consumes the buffered signal.
	engine := s.stateEngine()
	exec, err := s.execRepo.GetLatestByInstance(ctx, inst1.ID)
	s.Require().NoError(err)
	cmd, err := engine.Dispatch(ctx, exec)
	s.Require().NoError(err)
	s.Equal("p-1", cmd.CorrelationKey)

	spec, err := dsl.Parse([]byte(def.DSLBlob))
	s.Require().NoError(err)
	s.Require().NoError(engine.StartSignalWait(ctx, cmd, dsl.FindStep(spec, "wait_cancel")))

	next, err := s.execRepo.GetLatestByInstance(ctx, inst1.ID)
	s.Require().NoError(err)
	s.Equal("wait_final", next.State)
	s.Equal(0, s.pendingSignals(ctx, inst1.ID, "proposal.cancelled"))

	// A redelivery after consumption must not buffer a second signal.
	s.Equal(0, s.ingest(ctx, "evt-c1", "proposal.cancelled", map[string]any{"proposal_id": "p-1"}))
	s.Equal(0, s.pendingSignals(ctx, inst1.ID, "proposal.cancelled"))

	// Instance routes use the id carried in the payload, under the route's
	// signal name.
	finalized := map[string]any{"meta": map[string]any{"instance_id": inst1.ID}}
	s.Equal(0, s.ingest(ctx, "evt-f1", "intent.finalized", finalized))
	s.Equal(0, s.ingest(ctx, "evt-f1", "intent.finalized", finalized))
	s.Equal(1, s.pendingSignals(ctx, inst1.ID, "finalized"))

	// Unknown or foreign instance ids are ignored without failing the event.
	s.Equal(0, s.ingest(ctx, "evt-f2", "intent.finalized",
		map[string]any{"meta": map[string]any{"instance_id": "wfi_does_not_exist"}}))
}

func (s *BusinessSuite) TestEventRouter_SignalRoutesNeverCrossVersions() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	v1, err := biz.CreateWorkflow(ctx, []byte(routedVote("v1")))
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))
	s.Equal(1, s.ingest(ctx, "evt-a", "proposal.created", map[string]any{"proposal_id": "p-1"}))

	v2, err := biz.CreateWorkflow(ctx, []byte(routedVote("v2")))
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v2.ID))
	s.Equal(1, s.ingest(ctx, "evt-b", "proposal.created", map[string]any{"proposal_id": "p-1"}))

	onV1, err := s.instanceRepo.FindByTriggerEvent(ctx, "routed-vote", 1, "evt-a")
	s.Require().NoError(err)
	onV2, err := s.instanceRepo.FindByTriggerEvent(ctx, "routed-vote", 2, "evt-b")
	s.Require().NoError(err)

	_, err = s.instanceRepo.FindByTriggerEvent(ctx, "routed-vote", 1, "evt-b")
	s.Require().Error(err, "the new trigger event must start only the ACTIVE version")

	s.Equal(0, s.ingest(ctx, "evt-c", "proposal.cancelled", map[string]any{"proposal_id": "p-1"}))
	s.Equal(1, s.pendingSignals(ctx, onV1.ID, "proposal.cancelled"))
	s.Equal(1, s.pendingSignals(ctx, onV2.ID, "proposal.cancelled"))

	// An instance route naming a v1 instance is delivered only by v1's route.
	s.Equal(0, s.ingest(ctx, "evt-f", "intent.finalized",
		map[string]any{"meta": map[string]any{"instance_id": onV1.ID}}))
	s.Equal(1, s.pendingSignals(ctx, onV1.ID, "finalized"))
}

func (s *BusinessSuite) TestEventRouter_SkipsBindingOfInactiveVersion() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL()) // DRAFT

	s.Require().NoError(s.triggerRepo.Create(ctx, &models.TriggerBinding{
		EventType:       "payment.requested",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}))

	s.Equal(0, s.ingest(ctx, "evt-draft", "payment.requested", map[string]any{"amount": 1}))

	_, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, "evt-draft")
	s.Require().Error(err)
}
