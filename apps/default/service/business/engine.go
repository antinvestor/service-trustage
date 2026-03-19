package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	framecache "github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cacheutil"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// Maximum number of parsed DSL specs to cache in-process.
const maxDSLCacheSize = 200

// dslBlobCacheTTL is the TTL for DSL blobs cached in Valkey.
const dslBlobCacheTTL = 10 * time.Minute

// dslCache caches parsed workflow specs in-process (parsed specs cannot be serialized).
var dslCache = cacheutil.NewBoundedCache[*dsl.WorkflowSpec](maxDSLCacheSize) //nolint:gochecknoglobals // DSL cache

// ExecutionCommand is the immutable instruction sent to a worker.
type ExecutionCommand struct {
	ExecutionID     string          `json:"execution_id"`
	InstanceID      string          `json:"instance_id"`
	Workflow        string          `json:"workflow"`
	WorkflowVersion int             `json:"workflow_version"`
	State           string          `json:"state"`
	StateVersion    int             `json:"state_version"`
	Attempt         int             `json:"attempt"`
	InputPayload    json.RawMessage `json:"input_payload"`
	InputSchemaHash string          `json:"input_schema_hash"`
	ExecutionToken  string          `json:"execution_token"`
	TraceID         string          `json:"trace_id"`
}

// CommitRequest is the worker's commit payload.
type CommitRequest struct {
	ExecutionID    string          `json:"execution_id"`
	ExecutionToken string          `json:"execution_token"`
	Output         json.RawMessage `json:"output,omitempty"`
	Error          *CommitError    `json:"error,omitempty"`
}

// CommitError describes a classified error from a worker.
type CommitError struct {
	Class   string `json:"class"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WorkflowDefinitionLoader loads workflow definitions by name and version.
// Used by the execution worker to load DSL without importing the repository package.
type WorkflowDefinitionLoader interface {
	GetByNameAndVersion(ctx context.Context, name string, version int) (*models.WorkflowDefinition, error)
}

// StateEngine handles dispatch and commit of state executions.
type StateEngine interface {
	CreateInitialExecution(
		ctx context.Context,
		instance *models.WorkflowInstance,
		inputPayload json.RawMessage,
	) (*ExecutionCommand, error)
	Dispatch(ctx context.Context, execution *models.WorkflowStateExecution) (*ExecutionCommand, error)
	Commit(ctx context.Context, req *CommitRequest) error
	ParkExecutionUntil(ctx context.Context, executionID, executionToken string, fireAt time.Time) error
	ResumeWaitingExecution(ctx context.Context, executionID string, output json.RawMessage) error
	FailWaitingExecution(
		ctx context.Context,
		executionID string,
		status models.ExecutionStatus,
		failure *CommitError,
	) error
	StartSignalWait(
		ctx context.Context,
		cmd *ExecutionCommand,
		step *dsl.StepSpec,
	) error
	SendSignal(
		ctx context.Context,
		instanceID string,
		signalName string,
		payload json.RawMessage,
	) (bool, error)
	StartBranchScope(
		ctx context.Context,
		cmd *ExecutionCommand,
		step *dsl.StepSpec,
	) error
	ReconcileBranchScope(ctx context.Context, scopeID string) error
}

type stateEngine struct {
	instanceRepo    repository.WorkflowInstanceRepository
	execRepo        repository.WorkflowExecutionRepository
	runtimeRepo     repository.WorkflowRuntimeRepository
	timerRepo       repository.WorkflowTimerRepository
	scopeRepo       repository.WorkflowScopeRunRepository
	signalWaitRepo  repository.WorkflowSignalWaitRepository
	signalMsgRepo   repository.WorkflowSignalMessageRepository
	outputRepo      repository.WorkflowOutputRepository
	auditRepo       repository.AuditEventRepository
	defRepo         repository.WorkflowDefinitionRepository
	retryPolicyRepo repository.RetryPolicyRepository
	schemaReg       SchemaRegistry
	metrics         *telemetry.Metrics
	cache           framecache.RawCache
}

// NewStateEngine creates a new StateEngine.
func NewStateEngine(
	instanceRepo repository.WorkflowInstanceRepository,
	execRepo repository.WorkflowExecutionRepository,
	runtimeRepo repository.WorkflowRuntimeRepository,
	timerRepo repository.WorkflowTimerRepository,
	scopeRepo repository.WorkflowScopeRunRepository,
	signalWaitRepo repository.WorkflowSignalWaitRepository,
	signalMsgRepo repository.WorkflowSignalMessageRepository,
	outputRepo repository.WorkflowOutputRepository,
	auditRepo repository.AuditEventRepository,
	defRepo repository.WorkflowDefinitionRepository,
	retryPolicyRepo repository.RetryPolicyRepository,
	schemaReg SchemaRegistry,
	metrics *telemetry.Metrics,
	cache framecache.RawCache,
) StateEngine {
	return &stateEngine{
		instanceRepo:    instanceRepo,
		execRepo:        execRepo,
		runtimeRepo:     runtimeRepo,
		timerRepo:       timerRepo,
		scopeRepo:       scopeRepo,
		signalWaitRepo:  signalWaitRepo,
		signalMsgRepo:   signalMsgRepo,
		outputRepo:      outputRepo,
		auditRepo:       auditRepo,
		defRepo:         defRepo,
		retryPolicyRepo: retryPolicyRepo,
		schemaReg:       schemaReg,
		metrics:         metrics,
		cache:           cache,
	}
}

// CreateInitialExecution creates the first execution for a workflow instance.
func (e *stateEngine) CreateInitialExecution(
	ctx context.Context,
	instance *models.WorkflowInstance,
	inputPayload json.RawMessage,
) (*ExecutionCommand, error) {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, "engine.create_initial_execution",
		attribute.String(telemetry.AttrTenantID, instance.TenantID),
		attribute.String(telemetry.AttrWorkflow, instance.WorkflowName),
		attribute.String(telemetry.AttrState, instance.CurrentState),
	)
	defer func() { telemetry.EndSpan(span, nil) }()

	log := util.Log(ctx)

	// Validate input against schema.
	schemaHash, err := e.schemaReg.ValidateInput(
		ctx, instance.WorkflowName,
		instance.WorkflowVersion, instance.CurrentState, inputPayload,
	)
	if err != nil {
		log.WithError(err).Error("initial input validation failed",
			"instance_id", instance.ID,
			"state", instance.CurrentState,
		)

		return nil, fmt.Errorf("validate initial input: %w", err)
	}

	// Generate execution token.
	rawToken, tokenErr := cryptoutil.GenerateToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("generate token: %w", tokenErr)
	}
	traceID, traceErr := cryptoutil.GenerateToken()
	if traceErr != nil {
		return nil, fmt.Errorf("generate trace id: %w", traceErr)
	}

	tokenHash := cryptoutil.HashToken(rawToken)
	exec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           instance.CurrentState,
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  tokenHash,
		InputSchemaHash: schemaHash,
		InputPayload:    string(inputPayload),
		TraceID:         traceID,
	}

	if createErr := e.execRepo.Create(ctx, exec); createErr != nil {
		return nil, fmt.Errorf("create execution: %w", createErr)
	}

	// Audit event.
	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  instance.ID,
		ExecutionID: exec.ID,
		EventType:   events.EventStateDispatched,
		State:       instance.CurrentState,
		TraceID:     traceID,
	})

	return &ExecutionCommand{
		ExecutionID:     exec.ID,
		InstanceID:      instance.ID,
		Workflow:        instance.WorkflowName,
		WorkflowVersion: instance.WorkflowVersion,
		State:           instance.CurrentState,
		Attempt:         1,
		InputPayload:    inputPayload,
		InputSchemaHash: schemaHash,
		ExecutionToken:  rawToken,
		TraceID:         traceID,
	}, nil
}

// Dispatch transitions a pending execution to dispatched and builds the command.
func (e *stateEngine) Dispatch(
	ctx context.Context,
	execution *models.WorkflowStateExecution,
) (*ExecutionCommand, error) {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, telemetry.SpanDispatch,
		attribute.String(telemetry.AttrTenantID, execution.TenantID),
		attribute.String(telemetry.AttrState, execution.State),
	)
	start := time.Now()
	defer func() {
		e.metrics.DispatchLatency.Record(ctx, float64(time.Since(start).Milliseconds()))
		e.metrics.ExecutionsTotal.Add(ctx, 1)
		telemetry.EndSpan(span, nil)
	}()

	// Load instance to populate workflow name/version in the command.
	instance, instanceErr := e.instanceRepo.GetByID(ctx, execution.InstanceID)
	if instanceErr != nil {
		return nil, fmt.Errorf("load instance for dispatch: %w", instanceErr)
	}

	// Generate a new raw token for the worker to use when committing.
	rawToken, tokenErr := cryptoutil.GenerateToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("generate dispatch token: %w", tokenErr)
	}

	// Atomically mark as dispatched and store the hashed token in a single update.
	now := time.Now()
	tokenHash := cryptoutil.HashToken(rawToken)

	err := e.execRepo.UpdateStatus(ctx, execution.ID, models.ExecStatusDispatched, map[string]any{
		"started_at":      now,
		"execution_token": tokenHash,
	})
	if err != nil {
		return nil, fmt.Errorf("mark dispatched: %w", err)
	}

	var inputPayload json.RawMessage
	if execution.InputPayload != "" {
		inputPayload = json.RawMessage(execution.InputPayload)
	}

	return &ExecutionCommand{
		ExecutionID:     execution.ID,
		InstanceID:      execution.InstanceID,
		Workflow:        instance.WorkflowName,
		WorkflowVersion: instance.WorkflowVersion,
		State:           execution.State,
		Attempt:         execution.Attempt,
		InputPayload:    inputPayload,
		InputSchemaHash: execution.InputSchemaHash,
		ExecutionToken:  rawToken,
		TraceID:         execution.TraceID,
	}, nil
}

// Commit processes a worker's result: validates output, stores it, advances state, and creates the next execution.
// The entire operation runs inside a database transaction.
func (e *stateEngine) Commit( //nolint:funlen,gocognit // commit is inherently complex
	ctx context.Context,
	req *CommitRequest,
) error {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, telemetry.SpanCommit)
	start := time.Now()
	var commitErr error
	defer func() {
		e.metrics.CommitLatency.Record(ctx, float64(time.Since(start).Milliseconds()))
		telemetry.EndSpan(span, commitErr)
	}()

	log := util.Log(ctx)

	tokenHash := cryptoutil.HashToken(req.ExecutionToken)

	// Load execution (read-only, no lock) to get metadata for pre-tx validation.
	exec, err := e.execRepo.GetByID(ctx, req.ExecutionID)
	if err != nil {
		commitErr = fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
		return commitErr
	}

	span.SetAttributes(
		attribute.String(telemetry.AttrTenantID, exec.TenantID),
		attribute.String(telemetry.AttrState, exec.State),
	)

	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		commitErr = fmt.Errorf("load instance: %w", err)
		return commitErr
	}

	// Handle error case — token verification happens inside commitError too.
	if req.Error != nil {
		commitErr = e.commitError(ctx, req, exec, instance)
		return commitErr
	}

	// Load and parse workflow DSL (cached to avoid re-parsing on every commit).
	spec, err := e.loadSpec(ctx, instance.WorkflowName, instance.WorkflowVersion)
	if err != nil {
		commitErr = err
		return commitErr
	}

	// Validate output against schema before entering the transaction.
	if validateErr := e.schemaReg.ValidateOutput(
		ctx, instance.WorkflowName,
		instance.WorkflowVersion, exec.State, req.Output,
	); validateErr != nil {
		_ = e.runtimeRepo.UpdateExecutionStatus(
			ctx,
			exec,
			models.ExecStatusInvalidOutputContract,
		)
		e.metrics.ContractViolationsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String(telemetry.AttrViolationType, "output"),
		))

		commitErr = fmt.Errorf("%w: %w", ErrOutputContractViolation, validateErr)
		return commitErr
	}

	// Resolve the next step, evaluating CEL conditions on transitions if defined.
	transitionVars, varsErr := transitionVarsFromOutput(req.Output)
	if varsErr != nil {
		commitErr = varsErr
		return commitErr
	}

	nextStep, resolveErr := resolveNextStepForInstance(spec, instance, exec.State, transitionVars)
	if resolveErr != nil {
		commitErr = fmt.Errorf("resolve next step: %w", resolveErr)
		return commitErr
	}

	// Evaluate mapping to produce input for the next state.
	currentInput := executionInputPayload(exec)
	mappedInput, mappingErr := e.evaluateMapping(spec, exec.State, nextStep, currentInput, req.Output)
	if mappingErr != nil {
		commitErr = fmt.Errorf("evaluate mapping: %w", mappingErr)
		return commitErr
	}

	nextInputSchemaHash, schemaErr := e.validateNextInput(ctx, instance, nextStep, mappedInput)
	if schemaErr != nil {
		commitErr = schemaErr
		return commitErr
	}

	nextState := ""
	nextInputPayload := ""
	if nextStep != nil {
		nextState = nextStep.ID
		nextInputPayload = string(mappedInput)
	}

	commitReq := &repository.CommitExecutionRequest{
		Execution:           exec,
		Instance:            instance,
		TokenHash:           tokenHash,
		VerifyToken:         true,
		OutputPayload:       string(req.Output),
		ExpectedStatus:      models.ExecStatusDispatched,
		NextState:           nextState,
		NextInputPayload:    nextInputPayload,
		NextInputSchemaHash: nextInputSchemaHash,
	}
	if txErr := e.runtimeRepo.CommitExecution(ctx, commitReq); txErr != nil {
		if errors.Is(txErr, repository.ErrInvalidExecutionToken) {
			commitErr = fmt.Errorf("%w: %w", ErrInvalidToken, txErr)
			return commitErr
		}
		if errors.Is(txErr, repository.ErrStaleMutation) {
			commitErr = ErrStaleExecution
			return commitErr
		}
		commitErr = txErr
		return commitErr
	}

	if nextState != "" {
		e.metrics.TransitionsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String(telemetry.AttrFromState, exec.State),
			attribute.String(telemetry.AttrToState, nextState),
		))
	}

	// Audit: completion event (outside tx, best-effort).
	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  exec.InstanceID,
		ExecutionID: req.ExecutionID,
		EventType:   events.EventStateCompleted,
		State:       exec.State,
		TraceID:     exec.TraceID,
		Payload:     string(req.Output),
	})

	log.Info("execution committed",
		"execution_id", req.ExecutionID,
		"state", exec.State,
		"instance_id", exec.InstanceID,
	)

	return nil
}

func (e *stateEngine) ParkExecutionUntil(
	ctx context.Context,
	executionID, executionToken string,
	fireAt time.Time,
) error {
	exec, err := e.execRepo.GetByID(ctx, executionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}

	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return fmt.Errorf("load instance: %w", err)
	}

	tokenHash := cryptoutil.HashToken(executionToken)
	err = e.runtimeRepo.ParkExecution(ctx, &repository.ParkExecutionRequest{
		Execution:  exec,
		Instance:   instance,
		TokenHash:  tokenHash,
		FireAt:     fireAt,
		AuditTrace: exec.TraceID,
	})
	if errors.Is(err, repository.ErrStaleMutation) {
		e.metrics.StaleExecutionsTotal.Add(ctx, 1)
		return ErrStaleExecution
	}
	if errors.Is(err, repository.ErrInvalidExecutionToken) {
		return fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if err != nil {
		return err
	}

	return nil
}

func (e *stateEngine) ResumeWaitingExecution(
	ctx context.Context,
	executionID string,
	output json.RawMessage,
) error {
	exec, err := e.execRepo.GetByID(ctx, executionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}

	if exec.Status != models.ExecStatusWaiting {
		return ErrStaleExecution
	}

	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return fmt.Errorf("load instance: %w", err)
	}

	if instance.Status != models.InstanceStatusRunning {
		_ = e.runtimeRepo.UpdateExecutionStatus(ctx, exec, models.ExecStatusStale)
		return nil
	}

	spec, err := e.loadSpec(ctx, instance.WorkflowName, instance.WorkflowVersion)
	if err != nil {
		return err
	}

	if output == nil {
		output = json.RawMessage(`{}`)
	}

	if validateErr := e.schemaReg.ValidateOutput(
		ctx, instance.WorkflowName,
		instance.WorkflowVersion, exec.State, output,
	); validateErr != nil {
		_ = e.runtimeRepo.UpdateExecutionStatus(
			ctx,
			exec,
			models.ExecStatusInvalidOutputContract,
		)
		e.metrics.ContractViolationsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String(telemetry.AttrViolationType, "output"),
		))

		return fmt.Errorf("%w: %w", ErrOutputContractViolation, validateErr)
	}

	transitionVars, err := transitionVarsFromOutput(output)
	if err != nil {
		return err
	}

	nextStep, err := resolveNextStepForInstance(spec, instance, exec.State, transitionVars)
	if err != nil {
		return fmt.Errorf("resolve next step: %w", err)
	}

	currentInput := executionInputPayload(exec)
	mappedInput, err := e.evaluateMapping(spec, exec.State, nextStep, currentInput, output)
	if err != nil {
		return fmt.Errorf("evaluate mapping: %w", err)
	}

	nextInputSchemaHash, err := e.validateNextInput(ctx, instance, nextStep, mappedInput)
	if err != nil {
		return err
	}

	nextState := ""
	nextInputPayload := ""
	if nextStep != nil {
		nextState = nextStep.ID
		nextInputPayload = string(mappedInput)
	}

	if txErr := e.runtimeRepo.CommitExecution(ctx, &repository.CommitExecutionRequest{
		Execution:           exec,
		Instance:            instance,
		ExpectedStatus:      models.ExecStatusWaiting,
		OutputPayload:       string(output),
		NextState:           nextState,
		NextInputPayload:    nextInputPayload,
		NextInputSchemaHash: nextInputSchemaHash,
	}); txErr != nil {
		if errors.Is(txErr, repository.ErrStaleMutation) {
			return ErrStaleExecution
		}
		return txErr
	}

	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  exec.InstanceID,
		ExecutionID: exec.ID,
		EventType:   events.EventStateCompleted,
		State:       exec.State,
	})

	return nil
}

func executionInputPayload(exec *models.WorkflowStateExecution) json.RawMessage {
	if exec.InputPayload == "" {
		return json.RawMessage(`{}`)
	}

	return json.RawMessage(exec.InputPayload)
}

func transitionVarsFromOutput(output json.RawMessage) (map[string]any, error) {
	var outputData any
	if unmarshalErr := json.Unmarshal(output, &outputData); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal output for transition: %w", unmarshalErr)
	}

	return map[string]any{
		"output": outputData,
	}, nil
}

func (e *stateEngine) validateNextInput(
	ctx context.Context,
	instance *models.WorkflowInstance,
	nextStep *dsl.StepSpec,
	mappedInput json.RawMessage,
) (string, error) {
	if nextStep == nil || mappedInput == nil {
		return "", nil
	}

	nextInputSchemaHash, err := e.schemaReg.ValidateInput(
		ctx,
		instance.WorkflowName,
		instance.WorkflowVersion,
		nextStep.ID,
		mappedInput,
	)
	if err != nil {
		return "", fmt.Errorf("validate next state input: %w", err)
	}

	return nextInputSchemaHash, nil
}

// evaluateMapping evaluates the state mapping from currentState to nextStep.
// If no mapping is defined, passes the raw output through.
func (e *stateEngine) evaluateMapping(
	spec *dsl.WorkflowSpec,
	currentState string,
	nextStep *dsl.StepSpec,
	currentInput json.RawMessage,
	output json.RawMessage,
) (json.RawMessage, error) {
	if nextStep == nil {
		return nil, nil
	}

	// Find the current step to check for output_var mapping.
	currentStep := dsl.FindStep(spec, currentState)
	if currentStep == nil {
		return output, nil
	}

	switch currentStep.Type { //nolint:exhaustive // only explicit control-flow cases override default passthrough
	case dsl.StepTypeSequence, dsl.StepTypeIf, dsl.StepTypeDelay:
		return currentInput, nil
	}

	// Only apply mapping when current step exposes an output_var and next step has call input.
	outputVar := ""
	switch currentStep.Type { //nolint:exhaustive
	case dsl.StepTypeCall:
		if currentStep.Call != nil {
			outputVar = currentStep.Call.OutputVar
		}
	case dsl.StepTypeSignalWait:
		if currentStep.SignalWait != nil {
			outputVar = currentStep.SignalWait.OutputVar
		}
	}
	hasNextInput := nextStep.Type == dsl.StepTypeCall && nextStep.Call != nil && len(nextStep.Call.Input) > 0

	if outputVar == "" || !hasNextInput {
		return output, nil
	}

	var outputData any
	if unmarshalErr := json.Unmarshal(output, &outputData); unmarshalErr != nil {
		return output, nil //nolint:nilerr // pass through if output is not JSON
	}

	vars := map[string]any{
		outputVar: outputData,
	}

	resolved, resolveErr := dsl.ResolveTemplateValue(nextStep.Call.Input, vars)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve mapping templates: %w", resolveErr)
	}

	mappedBytes, marshalErr := json.Marshal(resolved)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal mapped input: %w", marshalErr)
	}

	return mappedBytes, nil
}

// commitError handles the error path in Commit: verifies token, classifies errors, schedules retries with backoff.
func (e *stateEngine) commitError(
	ctx context.Context,
	req *CommitRequest,
	exec *models.WorkflowStateExecution,
	instance *models.WorkflowInstance,
) error {
	log := util.Log(ctx)

	// Validate error payload against registered error schema (best-effort).
	errorPayload, _ := json.Marshal(req.Error)
	if validateErr := e.schemaReg.ValidateError(
		ctx, instance.WorkflowName,
		instance.WorkflowVersion, exec.State, errorPayload,
	); validateErr != nil {
		log.WithError(validateErr).Warn("error payload failed schema validation",
			"execution_id", req.ExecutionID,
		)
	}

	log.Info("execution failed",
		"execution_id", req.ExecutionID,
		"error_class", req.Error.Class,
		"error_code", req.Error.Code,
	)

	// Check if this error class is retryable and retry policy allows it.
	if req.Error.Class == "retryable" || req.Error.Class == "external_dependency" {
		scheduled, schedErr := e.scheduleRetryIfAllowed(ctx, exec, instance, req)
		if schedErr != nil {
			log.WithError(schedErr).Error("failed to schedule retry",
				"execution_id", req.ExecutionID,
			)
		}

		if scheduled {
			return nil
		}
	}

	status := models.ExecStatusFatal
	if req.Error.Class == "timeout" {
		status = models.ExecStatusTimedOut
	}

	tokenHash := cryptoutil.HashToken(req.ExecutionToken)
	failErr := e.runtimeRepo.FailExecution(ctx, &repository.FailExecutionRequest{
		Execution:      exec,
		Instance:       instance,
		TokenHash:      tokenHash,
		VerifyToken:    true,
		ExpectedStatus: models.ExecStatusDispatched,
		Status:         status,
		ErrorClass:     req.Error.Class,
		ErrorMessage:   req.Error.Message,
		AuditTrace:     exec.TraceID,
		AuditPayload:   string(errorPayload),
	})
	if errors.Is(failErr, repository.ErrStaleMutation) {
		return ErrStaleExecution
	}
	if failErr != nil && errors.Is(failErr, repository.ErrInvalidExecutionToken) {
		return fmt.Errorf("%w: %w", ErrInvalidToken, failErr)
	}

	return failErr
}

// scheduleRetryIfAllowed checks retry policy and schedules a retry with backoff.
// Returns true if a retry was scheduled.
func (e *stateEngine) scheduleRetryIfAllowed(
	ctx context.Context,
	exec *models.WorkflowStateExecution,
	instance *models.WorkflowInstance,
	req *CommitRequest,
) (bool, error) {
	policy, err := e.retryPolicyRepo.Lookup(
		ctx, instance.WorkflowName, instance.WorkflowVersion, exec.State,
	)
	if err != nil {
		// No retry policy found — not retryable.
		return false, nil //nolint:nilerr // no policy means no retry, not an error
	}

	if exec.Attempt >= policy.MaxAttempts {
		return false, nil
	}

	// Compute next retry time with exponential backoff.
	nextRetryAt := computeRetryTime(exec.Attempt, policy)

	updateErr := e.execRepo.UpdateStatus(ctx, req.ExecutionID, models.ExecStatusRetryScheduled, map[string]any{
		"error_class":   req.Error.Class,
		"error_message": req.Error.Message,
		"next_retry_at": nextRetryAt,
	})

	if updateErr != nil {
		return false, fmt.Errorf("schedule retry: %w", updateErr)
	}

	e.metrics.RetriesTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String(telemetry.AttrState, exec.State),
	))

	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  exec.InstanceID,
		ExecutionID: req.ExecutionID,
		EventType:   events.EventStateRetried,
		State:       exec.State,
		TraceID:     exec.TraceID,
		Payload: fmt.Sprintf(
			`{"next_retry_at":"%s","attempt":%d,"max_attempts":%d}`,
			nextRetryAt.UTC().Format(time.RFC3339),
			exec.Attempt+1,
			policy.MaxAttempts,
		),
	})

	return true, nil
}

// loadSpec loads and parses a workflow DSL spec.
// Check order: in-process compiled cache → Valkey DSL blob cache → database.
func (e *stateEngine) loadSpec(
	ctx context.Context,
	workflowName string,
	version int,
) (*dsl.WorkflowSpec, error) {
	tenantID := tenantFromContext(ctx)
	cacheKey := fmt.Sprintf("dsl:%s:%s:%d", tenantID, workflowName, version)

	// L1: in-process parsed spec cache.
	if spec, ok := dslCache.Get(cacheKey); ok {
		return spec, nil
	}

	// L2: Valkey DSL blob cache (avoids database round-trip).
	if e.cache != nil {
		blob, found, cacheErr := e.cache.Get(ctx, cacheKey)
		if cacheErr == nil && found {
			spec, parseErr := dsl.Parse(blob)
			if parseErr == nil {
				dslCache.Put(cacheKey, spec)
				return spec, nil
			}
		}
	}

	// L3: database.
	def, err := e.defRepo.GetByNameAndVersion(ctx, workflowName, version)
	if err != nil {
		return nil, fmt.Errorf("load workflow definition: %w", err)
	}

	spec, err := dsl.Parse([]byte(def.DSLBlob))
	if err != nil {
		return nil, fmt.Errorf("parse workflow DSL: %w", err)
	}

	// Populate L1 and L2.
	dslCache.Put(cacheKey, spec)

	if e.cache != nil {
		_ = e.cache.Set(ctx, cacheKey, []byte(def.DSLBlob), dslBlobCacheTTL)
	}

	return spec, nil
}

const exponentialBase = 2

func tenantFromContext(ctx context.Context) string {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		return "unknown"
	}

	tenantID := claims.GetTenantID()
	if tenantID == "" {
		return "unknown"
	}

	return tenantID
}

// computeRetryTime calculates the next retry time using the configured backoff strategy
// with full jitter to prevent thundering herd.
func computeRetryTime(attempt int, policy *models.WorkflowRetryPolicy) time.Time {
	delayMs := policy.InitialDelayMs

	if policy.BackoffStrategy == "exponential" {
		delayMs = int64(float64(policy.InitialDelayMs) * math.Pow(exponentialBase, float64(attempt-1)))
	}

	if delayMs > policy.MaxDelayMs {
		delayMs = policy.MaxDelayMs
	}

	// Apply full jitter: random value in [0, delayMs] to prevent thundering herd.
	jitteredMs := rand.Int64N(delayMs + 1) //nolint:gosec // jitter doesn't need crypto random

	return time.Now().Add(time.Duration(jitteredMs) * time.Millisecond)
}
