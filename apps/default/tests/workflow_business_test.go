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
	"context"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func (s *DefaultServiceSuite) TestWorkflowBusiness_RegistersSchemasForDelayIfAndSequence() {
	ctx := s.tenantCtx()

	dslBlob := `{
  "version": "1.0",
  "name": "schema-workflow",
  "steps": [
    {
      "id": "seq",
      "type": "sequence",
      "sequence": {
        "steps": [
          {
            "id": "wait",
            "type": "delay",
            "delay": { "duration": "1m" }
          },
          {
            "id": "check",
            "type": "if",
            "if": {
              "expr": "payload.amount > 10",
              "then": [
                {"id": "high", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ],
              "else": [
                {"id": "low", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ]
            }
          }
        ]
      }
    }
  ]
}`

	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(dslBlob))
	s.Require().NoError(err)

	delayOutput, err := s.schemaRepo.Lookup(
		context.Background(),
		def.Name,
		def.WorkflowVersion,
		"wait",
		models.SchemaTypeOutput,
	)
	s.Require().NoError(err)
	s.NotEmpty(delayOutput.SchemaHash)

	ifOutput, err := s.schemaRepo.Lookup(
		context.Background(),
		def.Name,
		def.WorkflowVersion,
		"check",
		models.SchemaTypeOutput,
	)
	s.Require().NoError(err)
	s.Contains(string(ifOutput.SchemaBlob), `"branch"`)

	sequenceInput, err := s.schemaRepo.Lookup(
		context.Background(),
		def.Name,
		def.WorkflowVersion,
		"seq",
		models.SchemaTypeInput,
	)
	s.Require().NoError(err)
	s.NotEmpty(sequenceInput.SchemaHash)
}

func (s *DefaultServiceSuite) TestWorkflowBusiness_AllowsParallelRuntime() {
	ctx := s.tenantCtx()

	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "parallel-supported",
  "steps": [
    {
      "id": "fanout",
      "type": "parallel",
      "parallel": {
        "steps": [
          {"id": "child", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
    }
  ]
}`))
	s.Require().NoError(err)

	schema, lookupErr := s.schemaRepo.Lookup(
		context.Background(),
		def.Name,
		def.WorkflowVersion,
		"fanout",
		models.SchemaTypeOutput,
	)
	s.Require().NoError(lookupErr)
	s.Contains(string(schema.SchemaBlob), `"branches"`)
}
