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
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *DefaultServiceSuite) TestSchemaRegistry_RegisterAndValidate() {
	ctx := s.tenantCtx()
	registry := s.schemaRegistry()

	schema := json.RawMessage(`{"type":"object"}`)
	hash, err := registry.RegisterSchema(ctx, "wf", 1, "step_a", models.SchemaTypeInput, schema)
	s.Require().NoError(err)
	s.NotEmpty(hash)

	lookup, err := s.schemaRepo.Lookup(ctx, "wf", 1, "step_a", models.SchemaTypeInput)
	s.Require().NoError(err)
	s.True(json.Valid(lookup.SchemaBlob), "schema blob should be valid json")

	validPayload := json.RawMessage(`{"foo":"bar"}`)

	hash2, err := registry.ValidateInput(ctx, "wf", 1, "step_a", validPayload)
	s.Require().NoError(err)
	s.Equal(hash, hash2)

	outputSchema := json.RawMessage(`{"type":"object","properties":{"result":{"type":"boolean"}}}`)
	_, err = registry.RegisterSchema(ctx, "wf", 1, "step_a", models.SchemaTypeOutput, outputSchema)
	s.Require().NoError(err)

	output := json.RawMessage(`{"result":true}`)
	s.Require().NoError(registry.ValidateOutput(ctx, "wf", 1, "step_a", output))

	badOutput := json.RawMessage(`{"result":"nope"}`)
	s.Require().Error(registry.ValidateOutput(ctx, "wf", 1, "step_a", badOutput))
}

func (s *DefaultServiceSuite) TestSchemaRegistry_MissingSchemasAndOptionalErrorSchema() {
	ctx := s.tenantCtx()
	registry := s.schemaRegistry()

	_, err := registry.ValidateInput(ctx, "wf", 1, "missing", json.RawMessage(`{}`))
	s.Require().Error(err)

	err = registry.ValidateError(ctx, "wf", 1, "missing", json.RawMessage(`{"class":"fatal"}`))
	s.Require().NoError(err)
}

func TestJsonSchemaCompilerSimple(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	err = compiler.AddResource("schema.json", doc)
	if err != nil {
		t.Fatalf("add resource failed: %v", err)
	}

	if _, compileErr := compiler.Compile("schema.json"); compileErr != nil {
		t.Fatalf("compile failed: %v", compileErr)
	}
}
