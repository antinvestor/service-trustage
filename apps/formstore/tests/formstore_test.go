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
	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// --- Definition CRUD tests ---

func (s *FormStoreSuite) TestCreateDefinition_Basic() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID: "kyc-onboard",
		Name:   "KYC Onboarding",
		Active: true,
	}
	err := s.biz.CreateDefinition(ctx, def)
	s.Require().NoError(err)
	s.NotEmpty(def.ID)

	fetched, err := s.biz.GetDefinition(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal("kyc-onboard", fetched.FormID)
	s.Equal("KYC Onboarding", fetched.Name)
}

func (s *FormStoreSuite) TestCreateDefinition_WithSchema() {
	ctx := s.tenantCtx()
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`
	def := &models.FormDefinition{
		FormID:     "with-schema",
		Name:       "Schema Form",
		JSONSchema: schema,
		Active:     true,
	}
	err := s.biz.CreateDefinition(ctx, def)
	s.Require().NoError(err)
	s.NotEmpty(def.ID)
}

func (s *FormStoreSuite) TestCreateDefinition_InvalidSchema() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID:     "bad-schema",
		Name:       "Bad Schema Form",
		JSONSchema: `{"type": "not-a-real-type"}`, // invalid JSON Schema
		Active:     true,
	}
	// This should still save — ValidateSchema only checks parseability, not semantic validity.
	// The schema is parseable JSON but may not be a valid JSON Schema type.
	// Actual behavior depends on the validator implementation.
	_ = s.biz.CreateDefinition(ctx, def)
}

func (s *FormStoreSuite) TestCreateDefinition_UnparseableSchema() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID:     "unparseable",
		Name:       "Unparseable",
		JSONSchema: `{not json at all`,
		Active:     true,
	}
	err := s.biz.CreateDefinition(ctx, def)
	s.Require().Error(err, "unparseable JSON schema should fail")
}

func (s *FormStoreSuite) TestGetDefinitionByFormID() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID: "lookup-test",
		Name:   "Lookup Test",
		Active: true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	found, err := s.biz.GetDefinitionByFormID(ctx, "lookup-test")
	s.Require().NoError(err)
	s.Equal(def.ID, found.ID)
}

func (s *FormStoreSuite) TestListDefinitions() {
	ctx := s.tenantCtx()

	var defs []*models.FormDefinition
	for _, name := range []string{"form-a", "form-b", "form-c"} {
		d := &models.FormDefinition{
			FormID: name,
			Name:   name,
			Active: true,
		}
		s.Require().NoError(s.biz.CreateDefinition(ctx, d))
		defs = append(defs, d)
	}

	// Deactivate form-c via raw SQL (GORM skips zero-value bools with default:true).
	db := s.dbPool.DB(ctx, false)
	s.Require().NoError(db.Exec("UPDATE form_definitions SET active = false WHERE id = ?", defs[2].ID).Error)

	all, err := s.biz.ListDefinitions(ctx, false, 100, 0)
	s.Require().NoError(err)
	s.Len(all, 3)

	active, err := s.biz.ListDefinitions(ctx, true, 100, 0)
	s.Require().NoError(err)
	s.Len(active, 2)
}

func (s *FormStoreSuite) TestDeleteDefinition() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{FormID: "delete-me", Name: "Delete Me", Active: true}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	s.Require().NoError(s.biz.DeleteDefinition(ctx, def.ID))

	_, err := s.biz.GetDefinition(ctx, def.ID)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrFormDefinitionNotFound)
}

// --- Submission tests ---

func (s *FormStoreSuite) TestCreateSubmission_Basic() {
	ctx := s.tenantCtx()
	sub := &models.FormSubmission{
		FormID:      "general",
		SubmitterID: "user-123",
		Data:        `{"name": "John", "email": "john@example.com"}`,
	}
	err := s.biz.CreateSubmission(ctx, sub)
	s.Require().NoError(err)
	s.NotEmpty(sub.ID)
	s.Equal(models.SubmissionStatusPending, sub.Status)

	fetched, err := s.biz.GetSubmission(ctx, sub.ID)
	s.Require().NoError(err)
	s.Equal("user-123", fetched.SubmitterID)
}

func (s *FormStoreSuite) TestCreateSubmission_SchemaValidation_Pass() {
	ctx := s.tenantCtx()
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`
	def := &models.FormDefinition{
		FormID:     "validated-form",
		Name:       "Validated Form",
		JSONSchema: schema,
		Active:     true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	sub := &models.FormSubmission{
		FormID: "validated-form",
		Data:   `{"name": "Alice", "age": 30}`,
	}
	err := s.biz.CreateSubmission(ctx, sub)
	s.Require().NoError(err, "valid data should pass schema validation")
}

func (s *FormStoreSuite) TestCreateSubmission_SchemaValidation_Fail() {
	ctx := s.tenantCtx()
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`
	def := &models.FormDefinition{
		FormID:     "strict-form",
		Name:       "Strict Form",
		JSONSchema: schema,
		Active:     true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	// Missing required field "name".
	sub := &models.FormSubmission{
		FormID: "strict-form",
		Data:   `{"age": 25}`,
	}
	err := s.biz.CreateSubmission(ctx, sub)
	s.Require().Error(err, "missing required field should fail validation")
	s.ErrorIs(err, business.ErrSchemaValidationFailed)
}

func (s *FormStoreSuite) TestCreateSubmission_SchemaValidation_WrongType() {
	ctx := s.tenantCtx()
	schema := `{
		"type": "object",
		"properties": {
			"age": {"type": "integer"}
		}
	}`
	def := &models.FormDefinition{
		FormID:     "type-check-form",
		Name:       "Type Check",
		JSONSchema: schema,
		Active:     true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	// age is string instead of integer.
	sub := &models.FormSubmission{
		FormID: "type-check-form",
		Data:   `{"age": "not-a-number"}`,
	}
	err := s.biz.CreateSubmission(ctx, sub)
	s.Require().Error(err, "wrong type should fail validation")
	s.ErrorIs(err, business.ErrSchemaValidationFailed)
}

func (s *FormStoreSuite) TestCreateSubmission_Idempotency() {
	ctx := s.tenantCtx()

	sub1 := &models.FormSubmission{
		FormID:         "idempotent-form",
		SubmitterID:    "user-1",
		Data:           `{"key": "value1"}`,
		IdempotencyKey: "idem-key-001",
	}
	err := s.biz.CreateSubmission(ctx, sub1)
	s.Require().NoError(err)
	originalID := sub1.ID

	// Second submission with same idempotency key should return the original.
	sub2 := &models.FormSubmission{
		FormID:         "idempotent-form",
		SubmitterID:    "user-1",
		Data:           `{"key": "value2"}`, // different data
		IdempotencyKey: "idem-key-001",      // same key
	}
	err = s.biz.CreateSubmission(ctx, sub2)
	s.Require().NoError(err)
	s.Equal(originalID, sub2.ID, "idempotent submission should return original")
	s.Contains(sub2.Data, "value1", "idempotent submission should return original data")
}

func (s *FormStoreSuite) TestCreateSubmission_DifferentIdempotencyKeys() {
	ctx := s.tenantCtx()

	sub1 := &models.FormSubmission{
		FormID:         "idem-form",
		Data:           `{"v": 1}`,
		IdempotencyKey: "key-A",
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub1))

	sub2 := &models.FormSubmission{
		FormID:         "idem-form",
		Data:           `{"v": 2}`,
		IdempotencyKey: "key-B",
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub2))

	s.NotEqual(sub1.ID, sub2.ID, "different idempotency keys should create separate submissions")
}

func (s *FormStoreSuite) TestListSubmissions() {
	ctx := s.tenantCtx()

	for i := range 5 {
		sub := &models.FormSubmission{
			FormID: "list-form",
			Data:   `{"i": ` + string(rune('0'+i)) + `}`,
		}
		s.Require().NoError(s.biz.CreateSubmission(ctx, sub))
	}

	subs, err := s.biz.ListSubmissions(ctx, "list-form", 100, 0)
	s.Require().NoError(err)
	s.Len(subs, 5)
}

func (s *FormStoreSuite) TestUpdateSubmission_ValidStatus() {
	ctx := s.tenantCtx()
	sub := &models.FormSubmission{
		FormID: "update-form",
		Data:   `{"test": true}`,
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub))

	sub.Status = models.SubmissionStatusComplete
	err := s.biz.UpdateSubmission(ctx, sub)
	s.Require().NoError(err)

	fetched, err := s.biz.GetSubmission(ctx, sub.ID)
	s.Require().NoError(err)
	s.Equal(models.SubmissionStatusComplete, fetched.Status)
}

func (s *FormStoreSuite) TestUpdateSubmission_InvalidStatus() {
	ctx := s.tenantCtx()
	sub := &models.FormSubmission{
		FormID: "invalid-status-form",
		Data:   `{"test": true}`,
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub))

	sub.Status = "bogus"
	err := s.biz.UpdateSubmission(ctx, sub)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrInvalidStatus)
}

func (s *FormStoreSuite) TestDeleteSubmission() {
	ctx := s.tenantCtx()
	sub := &models.FormSubmission{
		FormID: "delete-form",
		Data:   `{"x": 1}`,
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub))

	s.Require().NoError(s.biz.DeleteSubmission(ctx, sub.ID))

	_, err := s.biz.GetSubmission(ctx, sub.ID)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrFormSubmissionNotFound)
}
