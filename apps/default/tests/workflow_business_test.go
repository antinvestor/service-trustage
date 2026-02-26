package tests

import (
	"errors"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *DefaultServiceSuite) TestWorkflowBusiness_CreateAndActivate() {
	ctx := s.tenantCtx()

	dslBlob := `{
  "version": "1.0",
  "name": "onboard",
  "steps": [
    {"id": "step_a", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "hello"}}},
    {"id": "step_b", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "world"}}}
  ]
}`

	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(dslBlob))
	s.Require().NoError(err)
	s.Require().NotEmpty(def.ID)
	s.Equal("onboard", def.Name)
	s.Equal(models.WorkflowStatusDraft, def.Status)

	// Schemas registered for each call step (input/output/error).
	_, err = s.schemaRepo.Lookup(ctx, "onboard", 1, "step_a", models.SchemaTypeInput)
	s.Require().NoError(err)
	_, err = s.schemaRepo.Lookup(ctx, "onboard", 1, "step_a", models.SchemaTypeOutput)
	s.Require().NoError(err)
	_, err = s.schemaRepo.Lookup(ctx, "onboard", 1, "step_a", models.SchemaTypeError)
	s.Require().NoError(err)

	// Activate workflow.
	s.Require().NoError(s.workflowBusiness().ActivateWorkflow(ctx, def.ID))
	updated, err := s.defRepo.GetByID(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusActive, updated.Status)

	// Activating again should fail.
	err = s.workflowBusiness().ActivateWorkflow(ctx, def.ID)
	s.Require().Error(err)
	s.True(errors.Is(err, business.ErrInvalidWorkflowStatus))
}
