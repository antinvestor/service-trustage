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
	"fmt"
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

// contractScopes lists the names a contract's references may use.
type contractScopes struct {
	outputVars map[string]bool
	loopVars   map[string]bool
}

var contractMetadataNames = map[string]bool{ //nolint:gochecknoglobals // test fixture
	"instance_id": true, "execution_id": true, "workflow_name": true,
	"workflow_version": true, "correlation_key": true,
}

func scopesOf(spec *WorkflowSpec) contractScopes {
	scopes := contractScopes{outputVars: map[string]bool{}, loopVars: map[string]bool{}}

	for _, step := range CollectAllSteps(spec) {
		if step.Call != nil && step.Call.OutputVar != "" {
			scopes.outputVars[step.Call.OutputVar] = true
		}
		if step.SignalWait != nil && step.SignalWait.OutputVar != "" {
			scopes.outputVars[step.SignalWait.OutputVar] = true
		}
		// foreach bodies also read their item and index variables.
		if step.Foreach != nil {
			scopes.loopVars[firstNonEmpty(step.Foreach.ItemVar, "item")] = true
			scopes.loopVars[firstNonEmpty(step.Foreach.IndexVar, "index")] = true
		}
	}

	return scopes
}

// templatePathProblem explains why a template path names nothing the
// engine provides, or returns "".
func (c contractScopes) templatePathProblem(path string) string {
	parts := strings.Split(path, ".")

	switch {
	case parts[0] == "payload", c.loopVars[parts[0]]:
		return ""
	case parts[0] == "vars":
		if len(parts) < 2 || !c.outputVars[parts[1]] {
			return fmt.Sprintf("%q names no output_var", path)
		}
	case parts[0] == "metadata":
		if len(parts) != 2 || !contractMetadataNames[parts[1]] {
			return fmt.Sprintf("%q is not an execution metadata variable", path)
		}
	default:
		return fmt.Sprintf("template root %q is not payload, vars or metadata", parts[0])
	}

	return ""
}

// stepExpressions returns every CEL expression a step evaluates.
func stepExpressions(step *StepSpec) []string {
	var exprs []string
	for _, ct := range step.OnSuccess.Conditional {
		exprs = append(exprs, ct.Condition)
	}
	for _, ct := range step.OnFailure.Conditional {
		exprs = append(exprs, ct.Condition)
	}
	if step.If != nil {
		exprs = append(exprs, step.If.Expr)
	}
	if step.Delay != nil {
		exprs = append(exprs, step.Delay.Until)
	}
	if step.Foreach != nil {
		exprs = append(exprs, step.Foreach.Items)
	}

	return exprs
}

// stepTemplatePaths returns every {{ }} path in a step's call input and
// signal_send payload.
func stepTemplatePaths(step *StepSpec) []string {
	var paths []string
	collect := func(s string) { paths = append(paths, ExtractTemplateVars(s)...) }

	if step.Call != nil {
		walkTemplateStrings(step.Call.Input, collect)
	}
	if step.SignalSend != nil {
		walkTemplateStrings(step.SignalSend.Payload, collect)
	}

	return paths
}

func checkContractReferences(t *testing.T, spec *WorkflowSpec) {
	t.Helper()

	scopes := scopesOf(spec)

	for _, step := range CollectAllSteps(spec) {
		for _, path := range stepTemplatePaths(step) {
			if problem := scopes.templatePathProblem(path); problem != "" {
				t.Errorf("step %s: %s", step.ID, problem)
			}
		}

		for _, expr := range stepExpressions(step) {
			for _, m := range celVarsRef.FindAllStringSubmatch(expr, -1) {
				if !scopes.outputVars[m[1]] {
					t.Errorf("step %s: expression %q reads vars.%s, which no step sets", step.ID, expr, m[1])
				}
			}
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
