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
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
)

// TriggerSpec starts a new instance of the workflow when an event of
// EventType is ingested and Filter (CEL over payload) holds. Key is a CEL
// expression over the trigger payload whose value becomes the instance's
// correlation key; signal routes with a key match on it.
type TriggerSpec struct {
	EventType string `json:"event_type"`
	Filter    string `json:"filter,omitempty"`
	Key       string `json:"key,omitempty"`
}

// SignalRouteSpec delivers an ingested event of EventType as a signal to
// running instances of the same workflow version. Exactly one of Key (match
// instances whose correlation key equals the value) and Instance (the
// instance id carried in the payload) is set. SignalName defaults to
// EventType and must be waited on by some signal_wait step.
type SignalRouteSpec struct {
	EventType  string `json:"event_type"`
	SignalName string `json:"signal_name,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Key        string `json:"key,omitempty"`
	Instance   string `json:"instance,omitempty"`
}

// EffectiveSignalName returns the signal name the route delivers.
func (r *SignalRouteSpec) EffectiveSignalName() string {
	if strings.TrimSpace(r.SignalName) != "" {
		return r.SignalName
	}

	return r.EventType
}

// FormatCorrelationKey converts the value of a key or instance expression
// into its string form. Strings are returned as is; other scalars are
// formatted. Anything else (null, lists, maps) yields ok=false.
func FormatCorrelationKey(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, v != ""
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	default:
		return "", false
	}
}

func validateTriggers(spec *WorkflowSpec, result *ValidationResult) {
	if len(spec.Triggers) == 0 {
		return
	}

	env, err := NewExpressionEnv()
	if err != nil {
		result.AddError(ErrInvalidExpression, fmt.Sprintf("failed to create CEL env: %v", err))
		return
	}

	for i, trigger := range spec.Triggers {
		if trigger == nil {
			result.AddError(ErrInvalidTrigger, fmt.Sprintf("triggers[%d]: nil entry", i))
			continue
		}

		prefix := fmt.Sprintf("triggers[%d]", i)
		if strings.TrimSpace(trigger.EventType) == "" {
			result.AddError(ErrInvalidTrigger, prefix+": event_type is required")
		}

		validateRoutingFilter(env, prefix, trigger.Filter, ErrInvalidTrigger, result)
		validateRoutingExpr(env, prefix, "key", trigger.Key, ErrInvalidTrigger, result)
	}
}

func validateSignalRoutes(spec *WorkflowSpec, result *ValidationResult) {
	if len(spec.Signals) == 0 {
		return
	}

	env, err := NewExpressionEnv()
	if err != nil {
		result.AddError(ErrInvalidExpression, fmt.Sprintf("failed to create CEL env: %v", err))
		return
	}

	waited := waitedSignalNames(spec)

	for i, route := range spec.Signals {
		if route == nil {
			result.AddError(ErrInvalidSignalRoute, fmt.Sprintf("signals[%d]: nil entry", i))
			continue
		}

		prefix := fmt.Sprintf("signals[%d]", i)
		if strings.TrimSpace(route.EventType) == "" {
			result.AddError(ErrInvalidSignalRoute, prefix+": event_type is required")
		}

		validateRoutingFilter(env, prefix, route.Filter, ErrInvalidSignalRoute, result)

		hasKey := strings.TrimSpace(route.Key) != ""
		hasInstance := strings.TrimSpace(route.Instance) != ""

		if hasKey == hasInstance {
			result.AddError(ErrInvalidSignalRoute, prefix+": exactly one of key and instance is required")
		}

		validateRoutingExpr(env, prefix, "key", route.Key, ErrInvalidSignalRoute, result)
		validateRoutingExpr(env, prefix, "instance", route.Instance, ErrInvalidSignalRoute, result)

		if hasKey && !hasInstance {
			for j, trigger := range spec.Triggers {
				if trigger != nil && strings.TrimSpace(trigger.Key) == "" {
					result.AddError(ErrInvalidSignalRoute, fmt.Sprintf(
						"%s: a key route requires every trigger to declare key (triggers[%d] has none)",
						prefix, j,
					))

					break
				}
			}
		}

		name := route.EffectiveSignalName()
		if strings.TrimSpace(name) != "" {
			if _, ok := waited[name]; !ok {
				result.AddError(ErrInvalidSignalRoute, fmt.Sprintf(
					"%s: no signal_wait step waits on signal %q", prefix, name,
				))
			}
		}
	}
}

func validateRoutingFilter(
	env *cel.Env,
	prefix, filter string,
	code ValidationErrorCode,
	result *ValidationResult,
) {
	if strings.TrimSpace(filter) == "" {
		return
	}

	ast, err := CompileExpression(env, filter)
	if err != nil {
		result.AddError(code, fmt.Sprintf("%s: invalid filter: %v", prefix, err))
		return
	}

	out := ast.OutputType()
	if !out.IsExactType(cel.BoolType) && !out.IsExactType(cel.DynType) {
		result.AddError(code, fmt.Sprintf("%s: filter must evaluate to bool, got %s", prefix, out))
	}
}

func validateRoutingExpr(
	env *cel.Env,
	prefix, field, expr string,
	code ValidationErrorCode,
	result *ValidationResult,
) {
	if strings.TrimSpace(expr) == "" {
		return
	}

	if _, err := CompileExpression(env, expr); err != nil {
		result.AddError(code, fmt.Sprintf("%s: invalid %s: %v", prefix, field, err))
	}
}

// waitedSignalNames returns every signal name some signal_wait step in the
// workflow waits on, including steps nested in branches and error fallbacks.
func waitedSignalNames(spec *WorkflowSpec) map[string]struct{} {
	names := make(map[string]struct{})

	var walk func(steps []*StepSpec)
	walk = func(steps []*StepSpec) {
		for _, step := range steps {
			if step == nil {
				continue
			}

			if step.Type == StepTypeSignalWait && step.SignalWait != nil && step.SignalWait.SignalName != "" {
				names[step.SignalWait.SignalName] = struct{}{}
			}

			walk(step.AllSubSteps())

			if step.OnError != nil {
				walk(step.OnError.Fallback)
			}
		}
	}

	walk(spec.Steps)

	if spec.OnError != nil {
		walk(spec.OnError.Fallback)
	}

	return names
}
