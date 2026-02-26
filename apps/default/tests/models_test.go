package tests_test

import "github.com/antinvestor/service-trustage/apps/default/service/models"

func (s *DefaultServiceSuite) TestWorkflowInstance_TransitionTo() {
	inst := &models.WorkflowInstance{Status: models.InstanceStatusRunning}
	s.Require().NoError(inst.TransitionTo(models.InstanceStatusCompleted))
	s.Equal(models.InstanceStatusCompleted, inst.Status)

	err := inst.TransitionTo(models.InstanceStatusRunning)
	s.Require().Error(err)
}

func (s *DefaultServiceSuite) TestWorkflowDefinition_TransitionTo() {
	def := &models.WorkflowDefinition{Status: models.WorkflowStatusDraft}
	s.Require().NoError(def.TransitionTo(models.WorkflowStatusActive))
	s.Equal(models.WorkflowStatusActive, def.Status)

	err := def.TransitionTo(models.WorkflowStatusDraft)
	s.Require().Error(err)
}
