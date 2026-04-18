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

package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

var templatePattern = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// ResolveTemplate replaces all {{ expr }} placeholders in the input string
// with values from the vars map using dot-notation path resolution.
func ResolveTemplate(template string, vars map[string]any) (string, error) {
	var resolveErr error

	result := templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		if resolveErr != nil {
			return match
		}

		submatch := templatePattern.FindStringSubmatch(match)
		if len(submatch) < 2 { //nolint:mnd // regex group count
			return match
		}

		path := strings.TrimSpace(submatch[1])

		val, err := resolvePath(path, vars)
		if err != nil {
			resolveErr = fmt.Errorf("resolve %q: %w", path, err)
			return match
		}

		return fmt.Sprintf("%v", val)
	})

	if resolveErr != nil {
		return "", resolveErr
	}

	return result, nil
}

// ValidateTemplate checks that all {{ }} references in the template are syntactically valid.
func ValidateTemplate(template string) []string {
	var errors []string

	matches := templatePattern.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) < 2 { //nolint:mnd // regex group count
			continue
		}

		path := strings.TrimSpace(match[1])
		if path == "" {
			errors = append(errors, "empty template expression")
			continue
		}

		parts := strings.Split(path, ".")
		if len(parts) == 0 || parts[0] == "" {
			errors = append(errors, fmt.Sprintf("invalid template path: %q", path))
		}
	}

	return errors
}

// ExtractTemplateVars extracts all variable paths referenced in {{ }} templates.
func ExtractTemplateVars(template string) []string {
	matches := templatePattern.FindAllStringSubmatch(template, -1)
	vars := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) >= 2 { //nolint:mnd // regex group count
			vars = append(vars, strings.TrimSpace(match[1]))
		}
	}

	return vars
}

// resolvePath resolves a dot-notation path against a nested map.
func resolvePath(path string, vars map[string]any) (any, error) {
	parts := strings.Split(path, ".")
	var current any = vars

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse into non-map at %q", part)
		}

		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found", part)
		}

		current = val
	}

	return current, nil
}

// ResolveTemplateValue resolves templates within any value type (string, map, or slice).
func ResolveTemplateValue(value any, vars map[string]any) (any, error) {
	switch v := value.(type) {
	case string:
		return ResolveTemplate(v, vars)
	case map[string]any:
		result := make(map[string]any, len(v))

		for k, val := range v {
			resolved, err := ResolveTemplateValue(val, vars)
			if err != nil {
				return nil, fmt.Errorf("resolve key %q: %w", k, err)
			}

			result[k] = resolved
		}

		return result, nil
	case []any:
		result := make([]any, len(v))

		for i, val := range v {
			resolved, err := ResolveTemplateValue(val, vars)
			if err != nil {
				return nil, fmt.Errorf("resolve index %d: %w", i, err)
			}

			result[i] = resolved
		}

		return result, nil
	default:
		return value, nil
	}
}
