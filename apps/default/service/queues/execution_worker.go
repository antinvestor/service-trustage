package queues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

	log.Debug("processing execution",
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

	switch step.Type {
	case dsl.StepTypeCall:
		return w.executeCallStep(ctx, &cmd, step)
	case dsl.StepTypeIf:
		return w.executeIfStep(ctx, &cmd, step)
	case dsl.StepTypeDelay:
		return w.executeDelayStep(ctx, &cmd, step)
	case dsl.StepTypeSignalWait:
		return w.executeSignalWaitStep(ctx, &cmd, step)
	case dsl.StepTypeSignalSend:
		return w.executeSignalSendStep(ctx, &cmd, step)
	case dsl.StepTypeSequence:
		return w.commitSuccess(ctx, &cmd, nil)
	case dsl.StepTypeParallel, dsl.StepTypeForeach:
		return w.executeBranchScopeStep(ctx, &cmd, step)
	default:
		w.commitError(
			ctx,
			&cmd,
			"fatal",
			"unsupported_step_type",
			fmt.Sprintf("step type %q is not executable by the current runtime", step.Type),
		)
		return nil
	}
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
	connectorInput, err := resolveStepInput(step.Call.Input, cmd.InputPayload)
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "input_resolution_failed", err.Error())
		return nil
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
			"tenant_id":    tenantIDFromContext(ctx),
			"state":        cmd.State,
		},
		IdempotencyKey: idempotencyKey,
	}

	resp, execErr := adapter.Execute(ctx, execReq)
	if execErr != nil {
		log.Debug("connector execution failed",
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

func (w *ExecutionWorker) executeIfStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	if step.If == nil {
		w.commitError(ctx, cmd, "fatal", "invalid_if_step", "if step is missing if configuration")
		return nil
	}

	var payload map[string]any
	if len(cmd.InputPayload) > 0 {
		if err := json.Unmarshal(cmd.InputPayload, &payload); err != nil {
			w.commitError(ctx, cmd, "fatal", "input_unmarshal_error", fmt.Sprintf("unmarshal input payload: %v", err))
			return nil
		}
	}
	if payload == nil {
		payload = map[string]any{}
	}

	env, err := dsl.NewExpressionEnv()
	if err != nil {
		return fmt.Errorf("create CEL env: %w", err)
	}

	ast, err := dsl.CompileExpression(env, step.If.Expr)
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "invalid_if_expression", err.Error())
		return nil
	}

	matched, err := dsl.EvaluateCondition(env, ast, map[string]any{
		"payload": payload,
		"now":     time.Now(),
	})
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "if_evaluation_failed", err.Error())
		return nil
	}

	branch := "else"
	if matched {
		branch = "then"
	}

	outputBytes, err := json.Marshal(map[string]any{"branch": branch})
	if err != nil {
		return fmt.Errorf("marshal if branch output: %w", err)
	}

	return w.commitSuccess(ctx, cmd, outputBytes)
}

func (w *ExecutionWorker) executeDelayStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	fireAt, err := resolveDelayFireAt(step, cmd.InputPayload)
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "delay_resolution_failed", err.Error())
		return nil
	}

	if !fireAt.After(time.Now()) {
		return w.commitSuccess(ctx, cmd, nil)
	}

	if parkErr := w.engine.ParkExecutionUntil(ctx, cmd.ExecutionID, cmd.ExecutionToken, fireAt); parkErr != nil {
		if errors.Is(parkErr, business.ErrInvalidToken) || errors.Is(parkErr, business.ErrStaleExecution) {
			return nil
		}

		return fmt.Errorf("park execution until %s: %w", fireAt.UTC().Format(time.RFC3339), parkErr)
	}

	return nil
}

func (w *ExecutionWorker) executeSignalWaitStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	if err := w.engine.StartSignalWait(ctx, cmd, step); err != nil {
		if errors.Is(err, business.ErrInvalidToken) || errors.Is(err, business.ErrStaleExecution) {
			return nil
		}

		return fmt.Errorf("start signal wait: %w", err)
	}

	return nil
}

func (w *ExecutionWorker) executeSignalSendStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	if step.SignalSend == nil {
		w.commitError(
			ctx,
			cmd,
			"fatal",
			"invalid_signal_send_step",
			"signal_send step is missing signal_send configuration",
		)
		return nil
	}

	targetInstanceID, payload, err := resolveSignalSend(step.SignalSend, cmd.InputPayload)
	if err != nil {
		w.commitError(ctx, cmd, "fatal", "signal_resolution_failed", err.Error())
		return nil
	}

	if _, err := w.engine.SendSignal(ctx, targetInstanceID, step.SignalSend.SignalName, payload); err != nil {
		return fmt.Errorf("send signal: %w", err)
	}

	return w.commitSuccess(ctx, cmd, json.RawMessage(`{}`))
}

func (w *ExecutionWorker) executeBranchScopeStep(
	ctx context.Context,
	cmd *business.ExecutionCommand,
	step *dsl.StepSpec,
) error {
	if err := w.engine.StartBranchScope(ctx, cmd, step); err != nil {
		if errors.Is(err, business.ErrInvalidToken) || errors.Is(err, business.ErrStaleExecution) {
			return nil
		}

		return fmt.Errorf("start branch scope: %w", err)
	}

	return nil
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
		if errors.Is(err, business.ErrInvalidToken) || errors.Is(err, business.ErrStaleExecution) {
			log.Debug("commit became stale or was already consumed",
				"execution_id", cmd.ExecutionID,
			)
			return nil
		}

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
		if errors.Is(err, business.ErrInvalidToken) || errors.Is(err, business.ErrStaleExecution) {
			log.Debug("error commit became stale or was already consumed",
				"execution_id", cmd.ExecutionID,
				"error_class", errorClass,
			)
			return
		}

		log.WithError(err).Error("error commit failed",
			"execution_id", cmd.ExecutionID,
			"error_class", errorClass,
		)
	}
}

func tenantIDFromContext(ctx context.Context) string {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		return ""
	}

	return claims.GetTenantID()
}

func resolveDelayFireAt(step *dsl.StepSpec, inputPayload json.RawMessage) (time.Time, error) {
	if step.Delay == nil {
		return time.Time{}, errors.New("delay step is missing delay configuration")
	}

	now := time.Now()
	if step.Delay.Duration.Duration > 0 {
		return now.Add(step.Delay.Duration.Duration), nil
	}

	var payload map[string]any
	if len(inputPayload) > 0 {
		if err := json.Unmarshal(inputPayload, &payload); err != nil {
			return time.Time{}, fmt.Errorf("unmarshal delay input payload: %w", err)
		}
	}
	if payload == nil {
		payload = map[string]any{}
	}

	env, err := dsl.NewExpressionEnv()
	if err != nil {
		return time.Time{}, fmt.Errorf("create CEL env: %w", err)
	}

	ast, err := dsl.CompileExpression(env, step.Delay.Until)
	if err != nil {
		return time.Time{}, fmt.Errorf("compile delay.until: %w", err)
	}

	value, err := dsl.EvaluateExpression(env, ast, map[string]any{
		"payload": payload,
		"now":     now,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("evaluate delay.until: %w", err)
	}

	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		timestamp, parseErr := time.Parse(time.RFC3339, v)
		if parseErr != nil {
			return time.Time{}, fmt.Errorf("parse delay timestamp %q: %w", v, parseErr)
		}
		return timestamp, nil
	default:
		return time.Time{}, fmt.Errorf("delay.until must evaluate to RFC3339 string or timestamp, got %T", value)
	}
}

func resolveStepInput(stepInput map[string]any, inputPayload json.RawMessage) (map[string]any, error) {
	payloadMap, err := payloadVars(inputPayload)
	if err != nil {
		return nil, err
	}

	if len(stepInput) == 0 {
		return payloadMap["payload"].(map[string]any), nil
	}

	resolved, err := dsl.ResolveTemplateValue(stepInput, payloadMap)
	if err != nil {
		return nil, fmt.Errorf("resolve step input templates: %w", err)
	}

	resolvedMap, ok := resolved.(map[string]any)
	if !ok {
		return nil, errors.New("resolved step input was not an object")
	}

	return resolvedMap, nil
}

func resolveSignalSend(
	spec *dsl.SignalSendSpec,
	inputPayload json.RawMessage,
) (string, json.RawMessage, error) {
	vars, err := payloadVars(inputPayload)
	if err != nil {
		return "", nil, err
	}

	targetInstanceID, err := dsl.ResolveTemplate(spec.TargetWorkflowID, vars)
	if err != nil {
		return "", nil, fmt.Errorf("resolve signal target: %w", err)
	}

	payloadValue, err := dsl.ResolveTemplateValue(spec.Payload, vars)
	if err != nil {
		return "", nil, fmt.Errorf("resolve signal payload: %w", err)
	}

	if payloadValue == nil {
		return targetInstanceID, json.RawMessage(`{}`), nil
	}

	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return "", nil, fmt.Errorf("marshal signal payload: %w", err)
	}

	return targetInstanceID, payload, nil
}

func payloadVars(inputPayload json.RawMessage) (map[string]any, error) {
	payload := map[string]any{}
	if len(inputPayload) > 0 {
		if err := json.Unmarshal(inputPayload, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal input payload: %w", err)
		}
	}

	vars := map[string]any{
		"payload": payload,
	}
	for key, value := range payload {
		vars[key] = value
	}

	return vars, nil
}
