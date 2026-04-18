//nolint:testpackage // package-local tests exercise unexported business helpers intentionally.
package business

import (
	"context"
	"encoding/json"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/events"
)

func (s *BusinessSuite) TestWorkflowBusiness_CreateActivateAndSchemaRegistration() {
	ctx := s.tenantCtx()

	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "schema-workflow",
  "steps": [
    {
      "id": "seq",
      "type": "sequence",
      "sequence": {
        "steps": [
          {
            "id": "wait",
            "type": "delay",
            "delay": { "duration": "1m" }
          },
          {
            "id": "check",
            "type": "if",
            "if": {
              "expr": "payload.amount > 10",
              "then": [
                {"id": "high", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ],
              "else": [
                {"id": "low", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ]
            }
          }
        ]
      }
    }
  ]
}`))
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusDraft, def.Status)

	s.Require().NoError(s.workflowBusiness().ActivateWorkflow(ctx, def.ID))
	active, err := s.workflowBusiness().GetWorkflow(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusActive, active.Status)

	listed, err := s.workflowBusiness().ListWorkflows(ctx, def.Name, 10)
	s.Require().NoError(err)
	s.Len(listed, 1)

	cases := []struct {
		state      string
		schemaType models.SchemaType
		contains   string
	}{
		{state: "wait", schemaType: models.SchemaTypeOutput, contains: `"type": "object"`},
		{state: "check", schemaType: models.SchemaTypeOutput, contains: `"branch"`},
		{state: "seq", schemaType: models.SchemaTypeInput, contains: `"type": "object"`},
	}

	for _, tc := range cases {
		schema, lookupErr := s.schemaRepo.Lookup(
			context.Background(),
			def.Name,
			def.WorkflowVersion,
			tc.state,
			tc.schemaType,
		)
		s.Require().NoError(lookupErr)
		s.Contains(string(schema.SchemaBlob), tc.contains)
	}
}

func (s *BusinessSuite) TestCreateWorkflow_MaterialisesSchedules() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "nightly", "cron_expr": "0 2 * * *"},
			{"name": "hourly",  "cron_expr": "0 * * * *"}
		]
	}`)

	def, err := s.workflowBusiness().CreateWorkflow(ctx, dslBlob)
	s.Require().NoError(err)

	// Workflow in DRAFT.
	s.Equal(models.WorkflowStatusDraft, def.Status)

	// Schedules materialised, active=false, next_fire_at=nil.
	out, listErr := s.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	s.Require().NoError(listErr)
	s.Len(out, 2)
	for _, sch := range out {
		s.False(sch.Active, "new schedule should be inactive")
		s.Nil(sch.NextFireAt, "new schedule should have no next_fire_at")
		s.Equal(def.Name, sch.WorkflowName)
		s.Equal(def.WorkflowVersion, sch.WorkflowVersion)
	}
}

func (s *BusinessSuite) TestCreateWorkflow_InvalidScheduleCronRejected() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-bad-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "bad", "cron_expr": "not-a-cron"}]
	}`)

	_, err := s.workflowBusiness().CreateWorkflow(ctx, dslBlob)
	s.Require().Error(err)
	s.Contains(err.Error(), "cron")
}

func (s *BusinessSuite) TestWorkflowBusiness_RejectsUnsupportedStepType() {
	ctx := s.tenantCtx()

	_, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "bad-workflow",
  "steps": [
    {
      "id": "unknown",
      "type": "unsupported"
    }
  ]
}`))
	s.Require().Error(err)
	s.ErrorIs(err, ErrDSLValidationFailed)
}

func (s *BusinessSuite) TestEventRouter_RouteEventCreatesAndDeduplicatesInstance() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL())
	s.Require().NoError(s.defRepo.Update(ctx, &models.WorkflowDefinition{
		BaseModel:       def.BaseModel,
		Name:            def.Name,
		WorkflowVersion: def.WorkflowVersion,
		Status:          models.WorkflowStatusActive,
		DSLBlob:         def.DSLBlob,
		TimeoutSeconds:  def.TimeoutSeconds,
	}))

	binding := &models.TriggerBinding{
		EventType:       "payment.requested",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}
	s.Require().NoError(s.triggerRepo.Create(ctx, binding))

	router := s.eventRouter()
	event := &events.IngestedEventMessage{
		EventID:     "evt-1",
		TenantID:    "test-tenant-001",
		PartitionID: "test-partition-001",
		EventType:   "payment.requested",
		Payload: map[string]any{
			"amount": 150,
		},
	}

	created, err := router.RouteEvent(ctx, event)
	s.Require().NoError(err)
	s.Equal(1, created)

	instance, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, event.EventID)
	s.Require().NoError(err)
	s.Equal("log_step", instance.CurrentState)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusPending, exec.Status)

	created, err = router.RouteEvent(ctx, event)
	s.Require().NoError(err)
	s.Equal(0, created)

	audits, err := s.auditRepo.ListByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.NotEmpty(audits)
}

func (s *BusinessSuite) TestActivateWorkflow_ActivatesSchedulesAndDeactivatesPrevious() {
	ctx := s.tenantCtx()

	// v1 with two schedules.
	v1DSL := []byte(`{
		"version": "v1",
		"name": "w-activate",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "a", "cron_expr": "*/5 * * * *"},
			{"name": "b", "cron_expr": "0 * * * *"}
		]
	}`)
	biz := s.workflowBusiness()
	v1, err := biz.CreateWorkflow(ctx, v1DSL)
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))

	// Both v1 schedules must now be active with next_fire_at set.
	v1Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1Scheds, 2)
	for _, sch := range v1Scheds {
		s.True(sch.Active, "schedule %s must be active after workflow activation", sch.Name)
		s.NotNil(sch.NextFireAt)
		s.True(sch.NextFireAt.After(time.Now()), "next_fire_at must be in the future")
	}

	// v2 of the same workflow — manually created with workflow_version=2 since
	// CreateWorkflow always uses version=1 and the unique constraint prevents duplicates.
	v2Def := &models.WorkflowDefinition{
		Name:            v1.Name,
		WorkflowVersion: 2,
		Status:          models.WorkflowStatusDraft,
		DSLBlob:         string(v1DSL),
	}
	v2Def.CopyPartitionInfo(&v1.BaseModel)
	s.Require().NoError(s.defRepo.Create(ctx, v2Def))

	v2Sched := &models.ScheduleDefinition{
		Name:            "only",
		CronExpr:        "*/10 * * * *",
		WorkflowName:    v2Def.Name,
		WorkflowVersion: v2Def.WorkflowVersion,
		InputPayload:    "{}",
		Active:          false,
	}
	v2Sched.CopyPartitionInfo(&v2Def.BaseModel)
	s.Require().NoError(s.scheduleRepo.Create(ctx, v2Sched))

	s.NotEqual(v1.WorkflowVersion, v2Def.WorkflowVersion, "v2 must have a different workflow_version")

	s.Require().NoError(biz.ActivateWorkflow(ctx, v2Def.ID))

	// v2 schedule now active.
	v2Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v2Def.Name, v2Def.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v2Scheds, 1)
	s.True(v2Scheds[0].Active)
	s.NotNil(v2Scheds[0].NextFireAt)

	// v1 schedules must be deactivated.
	v1After, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1After, 2)
	for _, sch := range v1After {
		s.False(sch.Active, "v1 schedule %s must be deactivated after v2 activation", sch.Name)
	}
}

func (s *BusinessSuite) TestSchemaRegistry_RegisterValidateAndCache() {
	ctx := s.tenantCtx()

	schemaBlob := json.RawMessage(`{"type":"object","required":["amount"],"properties":{"amount":{"type":"number"}}}`)
	hash, err := s.schemaRegistry().RegisterSchema(ctx, "payments", 1, "step_a", models.SchemaTypeInput, schemaBlob)
	s.Require().NoError(err)
	s.NotEmpty(hash)

	gotHash, err := s.schemaRegistry().ValidateInput(ctx, "payments", 1, "step_a", json.RawMessage(`{"amount":100}`))
	s.Require().NoError(err)
	s.Equal(hash, gotHash)

	err = s.schemaRegistry().ValidateOutput(ctx, "payments", 1, "step_a", json.RawMessage(`{"bad":true}`))
	s.Require().ErrorIs(err, ErrSchemaNotFound)

	_, err = s.schemaRegistry().ValidateInput(ctx, "payments", 1, "step_a", json.RawMessage(`{"amount":"bad"}`))
	s.Require().Error(err)
	s.ErrorIs(err, ErrInputContractViolation)
}
