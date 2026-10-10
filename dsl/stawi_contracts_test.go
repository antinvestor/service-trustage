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

//nolint:testpackage // package-local DSL tests exercise unexported helpers intentionally.
package dsl

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Stawi process contracts (testdata/stawi, copied from
// stawilabs/stawi/workflows) are real-world DSL 1.2 documents that use
// triggers, signal routes, metadata templates and instance vars. They must
// keep parsing and validating, and every vars/metadata reference must name
// something the engine provides.

var celVarsRef = regexp.MustCompile(`\bvars\.([A-Za-z_][A-Za-z0-9_]*)`)

func TestStawiContractsParseAndValidate(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "stawi", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no Stawi contracts in testdata/stawi")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			raw, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatal(readErr)
			}

			spec, parseErr := Parse(raw)
			if parseErr != nil {
				t.Fatalf("parse: %v", parseErr)
			}

			if res := Validate(spec); !res.Valid() {
				t.Fatalf("validate: %v", res.Errors)
			}

			checkContractReferences(t, spec)
		})
	}
}

func checkContractReferences(t *testing.T, spec *WorkflowSpec) {
	t.Helper()

	outputVars := map[string]bool{}
	for _, step := range CollectAllSteps(spec) {
		if step.Call != nil && step.Call.OutputVar != "" {
			outputVars[step.Call.OutputVar] = true
		}
		if step.SignalWait != nil && step.SignalWait.OutputVar != "" {
			outputVars[step.SignalWait.OutputVar] = true
		}
	}

	// foreach bodies also read their item and index variables.
	loopVars := map[string]bool{}
	for _, step := range CollectAllSteps(spec) {
		if step.Foreach != nil {
			loopVars[firstNonEmpty(step.Foreach.ItemVar, "item")] = true
			loopVars[firstNonEmpty(step.Foreach.IndexVar, "index")] = true
		}
	}

	metadataNames := map[string]bool{
		"instance_id": true, "execution_id": true, "workflow_name": true,
		"workflow_version": true, "correlation_key": true,
	}

	checkPath := func(stepID, path string) {
		parts := strings.Split(path, ".")
		switch parts[0] {
		case "payload":
		case "vars":
			if len(parts) < 2 || !outputVars[parts[1]] {
				t.Errorf("step %s: %q names no output_var", stepID, path)
			}
		case "metadata":
			if len(parts) != 2 || !metadataNames[parts[1]] {
				t.Errorf("step %s: %q is not an execution metadata variable", stepID, path)
			}
		default:
			if loopVars[parts[0]] {
				return
			}
			t.Errorf("step %s: template root %q is not payload, vars or metadata", stepID, parts[0])
		}
	}

	checkCEL := func(stepID, expr string) {
		for _, m := range celVarsRef.FindAllStringSubmatch(expr, -1) {
			if !outputVars[m[1]] {
				t.Errorf("step %s: expression %q reads vars.%s, which no step sets", stepID, expr, m[1])
			}
		}
	}

	for _, step := range CollectAllSteps(spec) {
		if step.Call != nil {
			walkTemplateStrings(step.Call.Input, func(s string) {
				for _, path := range ExtractTemplateVars(s) {
					checkPath(step.ID, path)
				}
			})
		}
		if step.SignalSend != nil {
			walkTemplateStrings(step.SignalSend.Payload, func(s string) {
				for _, path := range ExtractTemplateVars(s) {
					checkPath(step.ID, path)
				}
			})
		}
		for _, ct := range step.OnSuccess.Conditional {
			checkCEL(step.ID, ct.Condition)
		}
		for _, ct := range step.OnFailure.Conditional {
			checkCEL(step.ID, ct.Condition)
		}
		if step.If != nil {
			checkCEL(step.ID, step.If.Expr)
		}
		if step.Delay != nil {
			checkCEL(step.ID, step.Delay.Until)
		}
		if step.Foreach != nil {
			checkCEL(step.ID, step.Foreach.Items)
		}
	}
}

func walkTemplateStrings(value any, fn func(string)) {
	switch v := value.(type) {
	case string:
		fn(v)
	case map[string]any:
		for _, inner := range v {
			walkTemplateStrings(inner, fn)
		}
	case []any:
		for _, inner := range v {
			walkTemplateStrings(inner, fn)
		}
	}
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
