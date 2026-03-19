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
