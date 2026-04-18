package dsl

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WorkflowSpec is the top-level DSL document structure.
type WorkflowSpec struct {
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Input       map[string]string `json:"input,omitempty"`
	Config      map[string]any    `json:"config,omitempty"`
	Timeout     Duration          `json:"timeout,omitempty"`
	OnError     *ErrorPolicy      `json:"on_error,omitempty"`
	Steps       []*StepSpec       `json:"steps"`
	Schedules   []*ScheduleSpec   `json:"schedules,omitempty"`
}

// ScheduleSpec declares a cron-triggered workflow schedule inside a WorkflowSpec.
// Schedules are materialised into schedule_definitions rows at CreateWorkflow time
// and follow the workflow's lifecycle — they activate when the workflow activates
// and deactivate when another version of the same workflow is activated.
type ScheduleSpec struct {
	Name         string         `json:"name"`
	CronExpr     string         `json:"cron_expr"`
	InputPayload map[string]any `json:"input_payload,omitempty"`
	// Active is an optional default. Nil means "active once the workflow is activated".
	// Explicitly false ships the schedule disabled even under an active workflow.
	Active *bool `json:"active,omitempty"`
}

// StepType enumerates all supported step types.
type StepType string

const (
	StepTypeCall       StepType = "call"
	StepTypeDelay      StepType = "delay"
	StepTypeIf         StepType = "if"
	StepTypeSequence   StepType = "sequence"
	StepTypeParallel   StepType = "parallel"
	StepTypeForeach    StepType = "foreach"
	StepTypeSignalWait StepType = "signal_wait"
	StepTypeSignalSend StepType = "signal_send"
)

// IsValid returns true if the step type is known.
func (t StepType) IsValid() bool {
	switch t {
	case StepTypeCall, StepTypeDelay, StepTypeIf, StepTypeSequence,
		StepTypeParallel, StepTypeForeach, StepTypeSignalWait, StepTypeSignalSend:
		return true
	default:
		return false
	}
}

// StepSpec is a single step in the workflow.
type StepSpec struct {
	ID        string       `json:"id"`
	Type      StepType     `json:"type"`
	Name      string       `json:"name,omitempty"`
	DependsOn string       `json:"depends_on,omitempty"`
	Retry     *RetrySpec   `json:"retry,omitempty"`
	Timeout   Duration     `json:"timeout,omitempty"`
	OnError   *ErrorPolicy `json:"on_error,omitempty"`

	// Transition fields. If set, these override the default implicit sequential ordering.
	// Each can be a single step ID string or a list of conditional targets.
	OnSuccess TransitionTarget `json:"on_success,omitempty"`
	OnFailure TransitionTarget `json:"on_failure,omitempty"`

	// Type-specific fields.
	Call       *CallSpec       `json:"call,omitempty"`
	Delay      *DelaySpec      `json:"delay,omitempty"`
	If         *IfSpec         `json:"if,omitempty"`
	Sequence   *SequenceSpec   `json:"sequence,omitempty"`
	Parallel   *ParallelSpec   `json:"parallel,omitempty"`
	Foreach    *ForeachSpec    `json:"foreach,omitempty"`
	SignalWait *SignalWaitSpec `json:"signal_wait,omitempty"`
	SignalSend *SignalSendSpec `json:"signal_send,omitempty"`
}

// TransitionTarget represents a transition destination. It can be either:
// - A simple string (target step ID)
// - An array of ConditionalTarget (CEL conditions evaluated in order, first match wins).
type TransitionTarget struct {
	// Static is set when the transition is a simple step ID string.
	Static string
	// Conditional is set when the transition is an array of conditional targets.
	Conditional []ConditionalTarget
}

// ConditionalTarget is a single entry in a conditional transition array.
type ConditionalTarget struct {
	Condition string `json:"condition"`
	Target    string `json:"target"`
}

// ErrInvalidTransitionTarget is returned when a transition target cannot be parsed.
var ErrInvalidTransitionTarget = errors.New("transition target must be a string or array of {condition, target}")

// IsEmpty returns true if no transition is defined.
func (t *TransitionTarget) IsEmpty() bool {
	return t.Static == "" && len(t.Conditional) == 0
}

// MarshalJSON implements json.Marshaler.
func (t *TransitionTarget) MarshalJSON() ([]byte, error) {
	if len(t.Conditional) > 0 {
		return json.Marshal(t.Conditional)
	}

	if t.Static != "" {
		return json.Marshal(t.Static)
	}

	return []byte("null"), nil
}

// UnmarshalJSON implements json.Unmarshaler.
// Handles both "step_id" (string) and [{"condition":"...","target":"..."}] (array).
func (t *TransitionTarget) UnmarshalJSON(b []byte) error {
	// Try string first.
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		t.Static = s
		return nil
	}

	// Try array of conditional targets.
	var targets []ConditionalTarget
	if err := json.Unmarshal(b, &targets); err == nil {
		t.Conditional = targets
		return nil
	}

	return ErrInvalidTransitionTarget
}

// AllSubSteps returns all nested sub-steps from this step (recursive).
func (s *StepSpec) AllSubSteps() []*StepSpec {
	var result []*StepSpec

	switch s.Type { //nolint:exhaustive // only compound types have sub-steps
	case StepTypeIf:
		if s.If != nil {
			result = append(result, s.If.Then...)
			result = append(result, s.If.Else...)
		}
	case StepTypeSequence:
		if s.Sequence != nil {
			result = append(result, s.Sequence.Steps...)
		}
	case StepTypeParallel:
		if s.Parallel != nil {
			result = append(result, s.Parallel.Steps...)
		}
	case StepTypeForeach:
		if s.Foreach != nil {
			result = append(result, s.Foreach.Steps...)
		}
	}

	return result
}

// CallSpec describes a connector invocation.
type CallSpec struct {
	Action    string         `json:"action"`
	Input     map[string]any `json:"input"`
	OutputVar string         `json:"output_var,omitempty"`
}

// DelaySpec describes a durable wait.
type DelaySpec struct {
	Duration Duration `json:"duration,omitempty"`
	Until    string   `json:"until,omitempty"`
}

// IfSpec describes a conditional branch.
type IfSpec struct {
	Expr string      `json:"expr"`
	Then []*StepSpec `json:"then"`
	Else []*StepSpec `json:"else,omitempty"`
}

// SequenceSpec describes ordered execution.
type SequenceSpec struct {
	Steps []*StepSpec `json:"steps"`
}

// ParallelSpec describes concurrent execution.
type ParallelSpec struct {
	Steps   []*StepSpec `json:"steps"`
	WaitAll bool        `json:"wait_all,omitempty"`
}

// ForeachSpec describes iteration.
type ForeachSpec struct {
	Items          string      `json:"items"`
	ItemVar        string      `json:"item_var,omitempty"`
	IndexVar       string      `json:"index_var,omitempty"`
	MaxConcurrency int         `json:"max_concurrency,omitempty"`
	Steps          []*StepSpec `json:"steps"`
}

// SignalWaitSpec describes waiting for an external signal.
type SignalWaitSpec struct {
	SignalName string   `json:"signal_name"`
	Timeout    Duration `json:"timeout,omitempty"`
	OutputVar  string   `json:"output_var,omitempty"`
}

// SignalSendSpec describes sending a signal to another workflow.
type SignalSendSpec struct {
	TargetWorkflowID string         `json:"target_workflow_id"`
	SignalName       string         `json:"signal_name"`
	Payload          map[string]any `json:"payload,omitempty"`
}

// RetrySpec configures retry behavior for a step.
type RetrySpec struct {
	MaxAttempts        int     `json:"max_attempts"`
	InitialInterval    string  `json:"initial_interval,omitempty"`
	BackoffCoefficient float64 `json:"backoff_coefficient,omitempty"`
	MaxInterval        string  `json:"max_interval,omitempty"`
}

// ErrorPolicy configures error handling for a step or workflow.
type ErrorPolicy struct {
	Strategy string      `json:"strategy"`
	Fallback []*StepSpec `json:"fallback,omitempty"`
}

// Duration wraps time.Duration with support for Go-style duration strings
// plus day-level durations (e.g. "7d", "30d").
type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("parse duration: %w", err)
	}

	switch val := v.(type) {
	case string:
		if val == "" {
			d.Duration = 0
			return nil
		}

		parsed, err := ParseDuration(val)
		if err != nil {
			return fmt.Errorf("parse duration %q: %w", val, err)
		}

		d.Duration = parsed

		return nil
	case float64:
		d.Duration = time.Duration(int64(val))
		return nil
	default:
		return fmt.Errorf("invalid duration type: %T", v)
	}
}

// ParseDuration parses a duration string supporting Go-style durations plus "d" for days.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Handle day suffix.
	if strings.HasSuffix(s, "d") {
		dayStr := strings.TrimSuffix(s, "d")

		days, err := strconv.Atoi(dayStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration %q: %w", s, err)
		}

		const hoursPerDay = 24

		return time.Duration(days) * hoursPerDay * time.Hour, nil
	}

	return time.ParseDuration(s)
}
