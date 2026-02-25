package queues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// ExecutionWorker handles NATS messages from the execution stream.
// It receives the full ExecutionCommand (including raw token), finds the connector,
// executes it, and commits the result back to the engine.
type ExecutionWorker struct {
	engine   business.StateEngine
	defRepo  business.WorkflowDefinitionLoader
	registry *connector.Registry
}

// NewExecutionWorker creates a new ExecutionWorker.
func NewExecutionWorker(
	engine business.StateEngine,
	defRepo business.WorkflowDefinitionLoader,
	registry *connector.Registry,
) queue.SubscribeWorker {
	return &ExecutionWorker{
		engine:   engine,
		defRepo:  defRepo,
		registry: registry,
	}
}

// Handle processes a single NATS message containing an ExecutionCommand.
func (w *ExecutionWorker) Handle(ctx context.Context, _ map[string]string, message []byte) error {
	// Workers process executions across all tenants; skip tenancy checks on BaseRepository queries.
	ctx = security.SkipTenancyChecksOnClaims(ctx)
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, "worker.execute")
	var handleErr error
	defer func() { telemetry.EndSpan(span, handleErr) }()

	log := util.Log(ctx)

	var cmd business.ExecutionCommand
	if err := json.Unmarshal(message, &cmd); err != nil {
		handleErr = fmt.Errorf("unmarshal execution command: %w", err)
		log.WithError(handleErr).Error("failed to unmarshal execution command")
		return handleErr
	}

	log.Info("processing execution",
		"execution_id", cmd.ExecutionID,
		"state", cmd.State,
		"attempt", cmd.Attempt,
	)

	// Load workflow definition.
	def, err := w.defRepo.GetByNameAndVersion(ctx, cmd.Workflow, cmd.WorkflowVersion)
	if err != nil {
		handleErr = fmt.Errorf("load workflow definition: %w", err)
		w.commitError(ctx, &cmd, "fatal", "definition_not_found", handleErr.Error())
		return nil // don't retry NATS delivery for business errors
	}

	// Parse DSL.
	spec, err := dsl.Parse([]byte(def.DSLBlob))
	if err != nil {
		handleErr = fmt.Errorf("parse workflow DSL: %w", err)
		w.commitError(ctx, &cmd, "fatal", "dsl_parse_error", handleErr.Error())
		return nil
	}

	// Find the current step.
	step := dsl.FindStep(spec, cmd.State)
	if step == nil {
		handleErr = fmt.Errorf("step %q not found in workflow", cmd.State)
		w.commitError(ctx, &cmd, "fatal", "step_not_found", handleErr.Error())
		return nil
	}

	// Only call steps invoke connectors. Other step types commit with empty output.
	if step.Type != dsl.StepTypeCall || step.Call == nil {
		return w.commitSuccess(ctx, &cmd, nil)
	}

	return w.executeCallStep(ctx, &cmd, step)
}

// executeCallStep executes a call step via its connector adapter.
func (w *ExecutionWorker) executeCallStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	log := util.Log(ctx)

	adapter, err := w.registry.Get(step.Call.Action)
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "connector_not_found",
			fmt.Sprintf("connector %q not found: %v", step.Call.Action, err))
		return nil
	}

	// Build connector input from mapping or static definition.
	connectorInput := step.Call.Input
	if len(cmd.InputPayload) > 0 {
		var mappedInput map[string]any
		if unmarshalErr := json.Unmarshal(cmd.InputPayload, &mappedInput); unmarshalErr == nil {
			connectorInput = mappedInput
		}
	}

	if connectorInput == nil {
		connectorInput = map[string]any{}
	}

	// Build execution-scoped idempotency key: execution_id + attempt ensures
	// that NATS redeliveries of the same execution produce the same key,
	// preventing duplicate side effects in connectors that support it.
	idempotencyKey := fmt.Sprintf("%s:%d", cmd.ExecutionID, cmd.Attempt)

	execReq := &connector.ExecuteRequest{
		Input: connectorInput,
		Metadata: map[string]string{
			"execution_id": cmd.ExecutionID,
			"instance_id":  cmd.InstanceID,
			"tenant_id":    tenantID,
			"state":        cmd.State,
		},
		IdempotencyKey: idempotencyKey,
	}

	resp, execErr := adapter.Execute(ctx, execReq)
	if execErr != nil {
		log.Info("connector execution failed",
			"execution_id", cmd.ExecutionID,
			"error_class", string(execErr.Class),
			"error_code", execErr.Code,
		)
		w.commitError(ctx, cmd, string(execErr.Class), execErr.Code, execErr.Message)
		return nil //nolint:nilerr // business error committed to engine; don't retry NATS delivery
	}

	outputBytes, marshalErr := json.Marshal(resp.Output)
	if marshalErr != nil {
		w.commitError(ctx, cmd, "fatal", "output_marshal_error",
			fmt.Sprintf("marshal connector output: %v", marshalErr))
		return nil
	}

	return w.commitSuccess(ctx, cmd, outputBytes)
}

// commitSuccess commits a successful execution result to the engine.
func (w *ExecutionWorker) commitSuccess(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	output json.RawMessage,
) error {
	log := util.Log(ctx)

	if output == nil {
		output = json.RawMessage(`{}`)
	}

	commitReq := &business.CommitRequest{
		ExecutionID:    cmd.ExecutionID,
		ExecutionToken: cmd.ExecutionToken,
		Output:         output,
	}

	if err := w.engine.Commit(ctx, commitReq); err != nil {
		log.WithError(err).Error("commit failed", "execution_id", cmd.ExecutionID)
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// commitError commits an error result to the engine.
func (w *ExecutionWorker) commitError(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	errorClass, errorCode, errorMessage string,
) {
	log := util.Log(ctx)

	commitReq := &business.CommitRequest{
		ExecutionID:    cmd.ExecutionID,
		ExecutionToken: cmd.ExecutionToken,
		Error: &business.CommitError{
			Class:   errorClass,
			Code:    errorCode,
			Message: errorMessage,
		},
	}

	if err := w.engine.Commit(ctx, commitReq); err != nil {
		log.WithError(err).Error("error commit failed",
			"execution_id", cmd.ExecutionID,
			"error_class", errorClass,
		)
	}
}
