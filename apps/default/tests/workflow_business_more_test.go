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
