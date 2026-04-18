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
