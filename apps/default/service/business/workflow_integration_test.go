package business

import (
	"context"
	"encoding/json"

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
	s.Require().Error(err)
	s.ErrorIs(err, ErrSchemaNotFound)

	_, err = s.schemaRegistry().ValidateInput(ctx, "payments", 1, "step_a", json.RawMessage(`{"amount":"bad"}`))
	s.Require().Error(err)
	s.ErrorIs(err, ErrInputContractViolation)
}
