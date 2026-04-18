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

import "github.com/antinvestor/service-trustage/apps/formstore/service/models"

func (s *FormStoreSuite) TestUpdateDefinition() {
	ctx := s.tenantCtx()

	def := &models.FormDefinition{
		FormID: "update-form",
		Name:   "Original",
		Active: true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	def.Name = "Updated"
	def.JSONSchema = `{"type":"object","properties":{"name":{"type":"string"}}}`
	s.Require().NoError(s.biz.UpdateDefinition(ctx, def))

	updated, err := s.biz.GetDefinition(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal("Updated", updated.Name)
	s.JSONEq(def.JSONSchema, updated.JSONSchema)
}
