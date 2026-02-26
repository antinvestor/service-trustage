package tests

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

	if _, err := compiler.Compile("schema.json"); err != nil {
		t.Fatalf("compile failed: %v", err)
	}
}
