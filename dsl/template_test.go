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

//nolint:testpackage // package-local DSL tests exercise unexported template helpers intentionally.
package dsl

import (
	"reflect"
	"testing"
)

func TestResolveTemplateValueNestedStructures(t *testing.T) {
	value := map[string]any{
		"message": "hello {{ payload.name }}",
		"items":   []any{"{{ payload.city }}", 2},
	}

	resolved, err := ResolveTemplateValue(value, map[string]any{
		"payload": map[string]any{
			"name": "Jane",
			"city": "Kampala",
		},
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	expected := map[string]any{
		"message": "hello Jane",
		"items":   []any{"Kampala", 2},
	}
	if !reflect.DeepEqual(expected, resolved) {
		t.Fatalf("expected %#v, got %#v", expected, resolved)
	}
}

func TestResolveTemplateMissingPathReturnsError(t *testing.T) {
	_, err := ResolveTemplate("{{ payload.missing }}", map[string]any{
		"payload": map[string]any{"name": "Jane"},
	})
	if err == nil {
		t.Fatal("expected template resolution error")
	}
}

func TestValidateTemplateDetectsEmptyExpression(t *testing.T) {
	errors := ValidateTemplate("hello {{   }}")
	if len(errors) == 0 {
		t.Fatal("expected template validation errors")
	}
}

func TestExtractTemplateVars(t *testing.T) {
	vars := ExtractTemplateVars("{{ payload.name }} {{ payload.email }}")
	if len(vars) != 2 || vars[0] != "payload.name" || vars[1] != "payload.email" {
		t.Fatalf("unexpected extracted vars: %#v", vars)
	}
}
