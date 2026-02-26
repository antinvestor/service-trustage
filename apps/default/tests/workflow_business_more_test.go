package tests_test

import (
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *DefaultServiceSuite) TestWorkflowBusiness_GetAndList() {
	ctx := s.tenantCtx()

	def := &models.WorkflowDefinition{
		Name:            "wf",
		WorkflowVersion: 1,
		Status:          models.WorkflowStatusActive,
		DSLBlob:         "{}",
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	biz := s.workflowBusiness()
	found, err := biz.GetWorkflow(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(def.ID, found.ID)

	list, err := biz.ListWorkflows(ctx, "wf", 10)
	s.Require().NoError(err)
	s.Len(list, 1)
}

func (s *DefaultServiceSuite) TestWorkflowBusiness_Get_NotFound() {
	ctx := s.tenantCtx()
	biz := s.workflowBusiness()

	_, err := biz.GetWorkflow(ctx, "missing")
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrWorkflowNotFound)
}
