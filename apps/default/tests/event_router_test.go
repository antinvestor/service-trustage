package tests_test

import (
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/events"
)

func (s *DefaultServiceSuite) TestEventRouter_RoutesMatchingTrigger() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL())
	s.createTrigger(ctx, "user.created", def)

	msg := &events.IngestedEventMessage{
		EventID:     "evt-001",
		EventType:   "user.created",
		TenantID:    testTenantID,
		PartitionID: testPartitionID,
		Payload: map[string]any{
			"user_id": "user-1",
		},
	}

	created, err := s.eventRouter().RouteEvent(ctx, msg)
	s.Require().NoError(err)
	s.Equal(1, created)

	instances, err := s.instanceRepo.List(ctx, "", "", 10)
	s.Require().NoError(err)
	s.Len(instances, 1)
	instance := instances[0]
	s.Equal(def.Name, instance.WorkflowName)
	s.Equal("log_step", instance.CurrentState)

	execs, err := s.execRepo.List(ctx, "", instance.ID, 10)
	s.Require().NoError(err)
	s.Len(execs, 1)
	s.Equal("log_step", execs[0].State)

	auditEvents, err := s.auditRepo.ListByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.NotEmpty(auditEvents)
}

func (s *DefaultServiceSuite) TestEventRouter_FilterBlocksNonMatching() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL())

	binding := &models.TriggerBinding{
		EventType:       "order.created",
		EventFilter:     "payload.amount > 10",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}
	s.Require().NoError(s.triggerRepo.Create(ctx, binding))

	msg := &events.IngestedEventMessage{
		EventID:     "evt-002",
		EventType:   "order.created",
		TenantID:    testTenantID,
		PartitionID: testPartitionID,
		Payload: map[string]any{
			"amount": 5,
		},
	}

	created, err := s.eventRouter().RouteEvent(ctx, msg)
	s.Require().NoError(err)
	s.Equal(0, created)

	instances, err := s.instanceRepo.List(ctx, "", "", 10)
	s.Require().NoError(err)
	s.Empty(instances)
}

func (s *DefaultServiceSuite) TestEventRouter_FilterHandlesInvalidExpression() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL())

	binding := &models.TriggerBinding{
		EventType:       "order.updated",
		EventFilter:     "payload.unknown >",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}
	s.Require().NoError(s.triggerRepo.Create(ctx, binding))

	msg := &events.IngestedEventMessage{
		EventID:     "evt-003",
		EventType:   "order.updated",
		TenantID:    testTenantID,
		PartitionID: testPartitionID,
		Payload: map[string]any{
			"amount": 20,
		},
	}

	created, err := s.eventRouter().RouteEvent(ctx, msg)
	s.Require().NoError(err)
	s.Equal(0, created)
}
