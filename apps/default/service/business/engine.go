package business

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	framecache "github.com/pitabwire/frame/cache"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"

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
	TenantID        string          `json:"tenant_id"`
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
	GetByNameAndVersion(ctx context.Context, tenantID, name string, version int) (*models.WorkflowDefinition, error)
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
}

type stateEngine struct {
	instanceRepo    repository.WorkflowInstanceRepository
	execRepo        repository.WorkflowExecutionRepository
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
		ctx, instance.TenantID, instance.WorkflowName,
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

	tokenHash := cryptoutil.HashToken(rawToken)
	executionID := util.IDString()

	exec := &models.WorkflowStateExecution{
		ExecutionID:     executionID,
		TenantID:        instance.TenantID,
		PartitionID:     instance.PartitionID,
		InstanceID:      instance.ID,
		State:           instance.CurrentState,
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  tokenHash,
		InputSchemaHash: schemaHash,
	}

	if createErr := e.execRepo.Create(ctx, exec); createErr != nil {
		return nil, fmt.Errorf("create execution: %w", createErr)
	}

	// Audit event.
	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		ID:          util.IDString(),
		TenantID:    instance.TenantID,
		PartitionID: instance.PartitionID,
		InstanceID:  instance.ID,
		ExecutionID: executionID,
		EventType:   events.EventStateDispatched,
		State:       instance.CurrentState,
	})

	return &ExecutionCommand{
		ExecutionID:     executionID,
		InstanceID:      instance.ID,
		TenantID:        instance.TenantID,
		Workflow:        instance.WorkflowName,
		WorkflowVersion: instance.WorkflowVersion,
		State:           instance.CurrentState,
		Attempt:         1,
		InputPayload:    inputPayload,
		InputSchemaHash: schemaHash,
		ExecutionToken:  rawToken,
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

	err := e.execRepo.UpdateStatus(ctx, execution.ExecutionID, models.ExecStatusDispatched, map[string]any{
		"started_at":      now,
		"execution_token": tokenHash,
	})
	if err != nil {
		return nil, fmt.Errorf("mark dispatched: %w", err)
	}

	return &ExecutionCommand{
		ExecutionID:     execution.ExecutionID,
		InstanceID:      execution.InstanceID,
		TenantID:        execution.TenantID,
		Workflow:        instance.WorkflowName,
		WorkflowVersion: instance.WorkflowVersion,
		State:           execution.State,
		Attempt:         execution.Attempt,
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
	spec, err := e.loadSpec(ctx, instance.TenantID, instance.WorkflowName, instance.WorkflowVersion)
	if err != nil {
		commitErr = err
		return commitErr
	}

	// Validate output against schema before entering the transaction.
	if validateErr := e.schemaReg.ValidateOutput(
		ctx, exec.TenantID, instance.WorkflowName,
		instance.WorkflowVersion, exec.State, req.Output,
	); validateErr != nil {
		_ = e.execRepo.UpdateStatus(ctx, req.ExecutionID, models.ExecStatusInvalidOutputContract, nil)
		e.metrics.ContractViolationsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String(telemetry.AttrViolationType, "output"),
		))

		commitErr = fmt.Errorf("%w: %w", ErrOutputContractViolation, validateErr)
		return commitErr
	}

	// Resolve the next step, evaluating CEL conditions on transitions if defined.
	var outputData any
	if unmarshalErr := json.Unmarshal(req.Output, &outputData); unmarshalErr != nil {
		commitErr = fmt.Errorf("unmarshal output for transition: %w", unmarshalErr)
		return commitErr
	}

	transitionVars := map[string]any{
		"output": outputData,
	}

	nextStep, resolveErr := dsl.ResolveNextStep(spec, exec.State, transitionVars)
	if resolveErr != nil {
		commitErr = fmt.Errorf("resolve next step: %w", resolveErr)
		return commitErr
	}

	// Evaluate mapping to produce input for the next state.
	mappedInput, mappingErr := e.evaluateMapping(spec, exec.State, nextStep, req.Output)
	if mappingErr != nil {
		commitErr = fmt.Errorf("evaluate mapping: %w", mappingErr)
		return commitErr
	}

	// Run the entire critical section (token verification + state mutation) in one transaction.
	db := e.execRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Verify and consume token INSIDE the transaction — if tx rolls back, token is restored.
		_, tokenErr := e.execRepo.VerifyAndConsumeTokenTx(tx, req.ExecutionID, tokenHash)
		if tokenErr != nil {
			return fmt.Errorf("%w: %w", ErrInvalidToken, tokenErr)
		}

		// Store validated output with correct output schema hash.
		if storeErr := tx.Create(&models.WorkflowStateOutput{
			ID:          util.IDString(),
			TenantID:    exec.TenantID,
			PartitionID: exec.PartitionID,
			ExecutionID: req.ExecutionID,
			InstanceID:  exec.InstanceID,
			State:       exec.State,
			SchemaHash:  computeOutputSchemaHash(req.Output),
			Payload:     string(req.Output),
		}).Error; storeErr != nil {
			return fmt.Errorf("store output: %w", storeErr)
		}

		// Mark execution as completed.
		now := time.Now()
		if updateErr := tx.Model(&models.WorkflowStateExecution{}).
			Where("execution_id = ?", req.ExecutionID).
			Updates(map[string]any{
				"status":      string(models.ExecStatusCompleted),
				"finished_at": now,
			}).Error; updateErr != nil {
			return fmt.Errorf("mark completed: %w", updateErr)
		}

		if nextStep == nil {
			// Terminal state — mark workflow instance as completed.
			result := tx.Exec(
				`UPDATE workflow_instances
				 SET status = ?, revision = revision + 1, modified_at = ?, finished_at = ?
				 WHERE id = ? AND tenant_id = ? AND current_state = ? AND status = 'running'`,
				string(models.InstanceStatusCompleted), now, now,
				exec.InstanceID, exec.TenantID, exec.State,
			)

			if result.Error != nil {
				return fmt.Errorf("mark instance completed: %w", result.Error)
			}

			return nil
		}

		// Non-terminal — CAS transition to next state.
		casResult := tx.Exec(
			`UPDATE workflow_instances
			 SET current_state = ?, revision = revision + 1, modified_at = ?
			 WHERE id = ? AND tenant_id = ? AND current_state = ? AND revision = ? AND status = 'running'`,
			nextStep.ID, now,
			exec.InstanceID, exec.TenantID, exec.State, instance.Revision,
		)

		if casResult.Error != nil {
			return fmt.Errorf("CAS transition: %w", casResult.Error)
		}

		if casResult.RowsAffected == 0 {
			e.metrics.StaleExecutionsTotal.Add(ctx, 1)
			return ErrStaleExecution
		}

		e.metrics.TransitionsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String(telemetry.AttrFromState, exec.State),
			attribute.String(telemetry.AttrToState, nextStep.ID),
		))

		// Validate mapped input against the next state's input schema.
		nextInputSchemaHash := ""
		if mappedInput != nil {
			var schemaErr error
			nextInputSchemaHash, schemaErr = e.schemaReg.ValidateInput(
				ctx, exec.TenantID, instance.WorkflowName,
				instance.WorkflowVersion, nextStep.ID, mappedInput,
			)
			if schemaErr != nil {
				return fmt.Errorf("validate next state input: %w", schemaErr)
			}
		}

		// Create execution for next state.
		rawToken, genErr := cryptoutil.GenerateToken()
		if genErr != nil {
			return fmt.Errorf("generate next token: %w", genErr)
		}

		nextExec := &models.WorkflowStateExecution{
			ExecutionID:     util.IDString(),
			TenantID:        exec.TenantID,
			PartitionID:     exec.PartitionID,
			InstanceID:      exec.InstanceID,
			State:           nextStep.ID,
			Attempt:         1,
			Status:          models.ExecStatusPending,
			ExecutionToken:  cryptoutil.HashToken(rawToken),
			InputSchemaHash: nextInputSchemaHash,
			TraceID:         exec.TraceID,
		}

		if createErr := tx.Create(nextExec).Error; createErr != nil {
			return fmt.Errorf("create next execution: %w", createErr)
		}

		// Audit: transition committed.
		_ = tx.Create(&models.WorkflowAuditEvent{
			ID:          util.IDString(),
			TenantID:    exec.TenantID,
			PartitionID: exec.PartitionID,
			InstanceID:  exec.InstanceID,
			ExecutionID: req.ExecutionID,
			EventType:   events.EventTransitionCommitted,
			State:       nextStep.ID,
			FromState:   exec.State,
			ToState:     nextStep.ID,
		}).Error

		return nil
	})

	if txErr != nil {
		commitErr = txErr
		return commitErr
	}

	// Audit: completion event (outside tx, best-effort).
	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		ID:          util.IDString(),
		TenantID:    exec.TenantID,
		PartitionID: exec.PartitionID,
		InstanceID:  exec.InstanceID,
		ExecutionID: req.ExecutionID,
		EventType:   events.EventStateCompleted,
		State:       exec.State,
	})

	log.Info("execution committed",
		"execution_id", req.ExecutionID,
		"state", exec.State,
		"instance_id", exec.InstanceID,
	)

	return nil
}

// computeOutputSchemaHash returns the SHA-256 hash of the output payload for storage.
func computeOutputSchemaHash(output json.RawMessage) string {
	hash := sha256.Sum256(output)
	return hex.EncodeToString(hash[:])
}

// evaluateMapping evaluates the state mapping from currentState to nextStep.
// If no mapping is defined, passes the raw output through.
func (e *stateEngine) evaluateMapping(
	spec *dsl.WorkflowSpec,
	currentState string,
	nextStep *dsl.StepSpec,
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

	// Only apply mapping when current step has an output_var and next step has call input.
	hasOutputVar := currentStep.Type == dsl.StepTypeCall && currentStep.Call != nil && currentStep.Call.OutputVar != ""
	hasNextInput := nextStep.Type == dsl.StepTypeCall && nextStep.Call != nil && len(nextStep.Call.Input) > 0

	if !hasOutputVar || !hasNextInput {
		return output, nil
	}

	var outputData any
	if unmarshalErr := json.Unmarshal(output, &outputData); unmarshalErr != nil {
		return output, nil //nolint:nilerr // pass through if output is not JSON
	}

	vars := map[string]any{
		currentStep.Call.OutputVar: outputData,
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

	// Verify and consume the execution token to prevent unauthorized error commits.
	tokenHash := cryptoutil.HashToken(req.ExecutionToken)

	_, tokenErr := e.execRepo.VerifyAndConsumeToken(ctx, req.ExecutionID, tokenHash)
	if tokenErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidToken, tokenErr)
	}

	// Validate error payload against registered error schema (best-effort).
	errorPayload, _ := json.Marshal(req.Error)
	if validateErr := e.schemaReg.ValidateError(
		ctx, exec.TenantID, instance.WorkflowName,
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

	// Fatal or exhausted retries — mark as fatal.
	status := models.ExecStatusFatal
	updateErr := e.execRepo.UpdateStatus(ctx, req.ExecutionID, status, map[string]any{
		"error_class":   req.Error.Class,
		"error_message": req.Error.Message,
	})

	if updateErr != nil {
		return fmt.Errorf("update failed status: %w", updateErr)
	}

	// Mark workflow instance as failed.
	_ = e.instanceRepo.UpdateStatus(ctx, exec.InstanceID, exec.TenantID, models.InstanceStatusFailed)

	_ = e.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		ID:          util.IDString(),
		TenantID:    exec.TenantID,
		PartitionID: exec.PartitionID,
		InstanceID:  exec.InstanceID,
		ExecutionID: req.ExecutionID,
		EventType:   events.EventStateFailed,
		State:       exec.State,
	})

	return nil
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
		ctx, exec.TenantID, instance.WorkflowName, instance.WorkflowVersion, exec.State,
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
		ID:          util.IDString(),
		TenantID:    exec.TenantID,
		PartitionID: exec.PartitionID,
		InstanceID:  exec.InstanceID,
		ExecutionID: req.ExecutionID,
		EventType:   events.EventStateRetried,
		State:       exec.State,
	})

	return true, nil
}

// loadSpec loads and parses a workflow DSL spec.
// Check order: in-process compiled cache → Valkey DSL blob cache → database.
func (e *stateEngine) loadSpec(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
) (*dsl.WorkflowSpec, error) {
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
	def, err := e.defRepo.GetByNameAndVersion(ctx, tenantID, workflowName, version)
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
