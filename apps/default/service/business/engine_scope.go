package business

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
)

func resolveNextStepForInstance(
	spec *dsl.WorkflowSpec,
	instance *models.WorkflowInstance,
	currentState string,
	vars map[string]any,
) (*dsl.StepSpec, error) {
	if instance == nil || instance.ParentExecutionID == "" {
		return dsl.ResolveNextStep(spec, currentState, vars)
	}

	switch instance.ScopeType {
	case string(dsl.StepTypeParallel):
		return dsl.ResolveNextStepInSubtree(spec, instance.ScopeEntryState, currentState, vars)
	case string(dsl.StepTypeForeach):
		return dsl.ResolveNextStepInContainer(spec, instance.ScopeParentState, currentState, vars)
	default:
		return dsl.ResolveNextStep(spec, currentState, vars)
	}
}

func (e *stateEngine) StartBranchScope(
	ctx context.Context,
	cmd *ExecutionCommand,
	step *dsl.StepSpec,
) error {
	exec, instance, err := e.loadBranchScopeParent(ctx, cmd, step)
	if err != nil {
		return err
	}

	childDefs, err := branchChildrenFromStep(step, cmd.InputPayload, instance.ID, exec.ID)
	if err != nil {
		return err
	}

	scope, err := initializeBranchScope(step, exec, instance, childDefs)
	if err != nil {
		return err
	}

	childRecords, err := e.prepareBranchScopeLaunch(ctx, instance, exec, step, scope, childDefs)
	if err != nil {
		return err
	}

	if txErr := e.runtimeRepo.StartBranchScope(ctx, &repository.StartBranchScopeRequest{
		Execution:      exec,
		Instance:       instance,
		TokenHash:      cryptoutil.HashToken(cmd.ExecutionToken),
		Scope:          scope,
		LaunchChildren: childRecords,
		AuditTrace:     exec.TraceID,
	}); txErr != nil {
		if errors.Is(txErr, repository.ErrInvalidExecutionToken) {
			return fmt.Errorf("%w: %w", ErrInvalidToken, txErr)
		}
		if errors.Is(txErr, repository.ErrStaleMutation) {
			return ErrStaleExecution
		}
		return txErr
	}

	if scope.TotalChildren == 0 {
		return e.ResumeWaitingExecution(ctx, exec.ID, emptyScopeOutput(step.Type))
	}

	return nil
}

func (e *stateEngine) loadBranchScopeParent(
	ctx context.Context,
	cmd *ExecutionCommand,
	step *dsl.StepSpec,
) (*models.WorkflowStateExecution, *models.WorkflowInstance, error) {
	if step == nil {
		return nil, nil, errors.New("branch scope step is required")
	}
	if step.Type != dsl.StepTypeParallel && step.Type != dsl.StepTypeForeach {
		return nil, nil, fmt.Errorf("unsupported branch scope type %q", step.Type)
	}

	exec, err := e.execRepo.GetByID(ctx, cmd.ExecutionID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}
	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("load instance: %w", err)
	}

	return exec, instance, nil
}

func initializeBranchScope(
	step *dsl.StepSpec,
	exec *models.WorkflowStateExecution,
	instance *models.WorkflowInstance,
	childDefs []scopedChildDefinition,
) (*models.WorkflowScopeRun, error) {
	scope := &models.WorkflowScopeRun{
		ParentExecutionID: exec.ID,
		ParentInstanceID:  instance.ID,
		ParentState:       exec.State,
		ScopeType:         string(step.Type),
		Status:            "running",
		TotalChildren:     len(childDefs),
	}

	resultsPayload, err := marshalEmptyResults(scope.TotalChildren)
	if err != nil {
		return nil, err
	}
	scope.ResultsPayload = string(resultsPayload)

	switch step.Type {
	case dsl.StepTypeParallel:
		scope.WaitAll = step.Parallel.WaitAll
		scope.MaxConcurrency = len(childDefs)
	case dsl.StepTypeForeach:
		scope.WaitAll = true
		scope.MaxConcurrency = defaultForeachConcurrency(step.Foreach.MaxConcurrency)
		scope.ItemVar = defaultForeachItemVar(step.Foreach.ItemVar)
		scope.IndexVar = defaultForeachIndexVar(step.Foreach.IndexVar)

		itemsPayload, marshalErr := json.Marshal(extractForeachItems(childDefs))
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal foreach items: %w", marshalErr)
		}
		scope.ItemsPayload = string(itemsPayload)
	case dsl.StepTypeCall,
		dsl.StepTypeDelay,
		dsl.StepTypeIf,
		dsl.StepTypeSequence,
		dsl.StepTypeSignalWait,
		dsl.StepTypeSignalSend:
		return nil, fmt.Errorf("unsupported branch scope type %q", step.Type)
	default:
		return nil, fmt.Errorf("unsupported branch scope type %q", step.Type)
	}

	return scope, nil
}

func (e *stateEngine) prepareBranchScopeLaunch(
	ctx context.Context,
	instance *models.WorkflowInstance,
	exec *models.WorkflowStateExecution,
	step *dsl.StepSpec,
	scope *models.WorkflowScopeRun,
	childDefs []scopedChildDefinition,
) ([]*repository.ScopedChildRecord, error) {
	launchCount := len(childDefs)
	if step.Type == dsl.StepTypeForeach && launchCount > scope.MaxConcurrency {
		launchCount = scope.MaxConcurrency
	}

	childRecords := make([]*repository.ScopedChildRecord, 0, launchCount)
	for _, child := range childDefs[:launchCount] {
		record, err := e.buildScopedChildRecord(ctx, instance, exec, step, child)
		if err != nil {
			return nil, err
		}
		childRecords = append(childRecords, record)
	}

	scope.NextChildIndex = launchCount
	return childRecords, nil
}

func (e *stateEngine) ReconcileBranchScope(
	ctx context.Context,
	scopeID string,
) error {
	scopeCtx, err := e.loadScopeContext(ctx, scopeID)
	if err != nil {
		return err
	}
	if scopeCtx.scope.Status != "running" {
		return nil
	}

	results, completedCount, failedCount, runningCount, failure, err := e.collectScopeResults(
		ctx,
		scopeCtx.children,
		scopeCtx.scope.TotalChildren,
	)
	if err != nil {
		return err
	}

	launched, err := e.tryLaunchMoreForeachChildren(ctx, scopeCtx, results, completedCount, failedCount, runningCount)
	if err != nil {
		return err
	}
	if launched {
		return nil
	}

	return e.finalizeBranchScope(ctx, scopeCtx, results, completedCount, failedCount, failure)
}

type scopedChildDefinition struct {
	Index      int
	EntryState string
	Input      json.RawMessage
	Item       any
}

func branchChildrenFromStep(
	step *dsl.StepSpec,
	inputPayload json.RawMessage,
	parentInstanceID string,
	parentExecutionID string,
) ([]scopedChildDefinition, error) {
	switch step.Type {
	case dsl.StepTypeParallel:
		return parallelChildrenFromStep(step, inputPayload, parentInstanceID, parentExecutionID)
	case dsl.StepTypeForeach:
		return foreachChildrenFromStep(step, inputPayload, parentInstanceID, parentExecutionID)
	case dsl.StepTypeCall,
		dsl.StepTypeDelay,
		dsl.StepTypeIf,
		dsl.StepTypeSequence,
		dsl.StepTypeSignalWait,
		dsl.StepTypeSignalSend:
		return nil, fmt.Errorf("unsupported scope type %q", step.Type)
	default:
		return nil, fmt.Errorf("unsupported scope type %q", step.Type)
	}
}

type branchScopeContext struct {
	scope          *models.WorkflowScopeRun
	parentExec     *models.WorkflowStateExecution
	parentInstance *models.WorkflowInstance
	parentStep     *dsl.StepSpec
	children       []*models.WorkflowInstance
}

func (e *stateEngine) loadScopeContext(ctx context.Context, scopeID string) (*branchScopeContext, error) {
	scope, err := e.scopeRepo.GetByID(ctx, scopeID)
	if err != nil {
		return nil, fmt.Errorf("load workflow scope: %w", err)
	}
	parentExec, err := e.execRepo.GetByID(ctx, scope.ParentExecutionID)
	if err != nil {
		return nil, fmt.Errorf("load parent execution: %w", err)
	}
	parentInstance, err := e.instanceRepo.GetByID(ctx, scope.ParentInstanceID)
	if err != nil {
		return nil, fmt.Errorf("load parent instance: %w", err)
	}
	spec, err := e.loadSpec(ctx, parentInstance.WorkflowName, parentInstance.WorkflowVersion)
	if err != nil {
		return nil, err
	}
	parentStep := dsl.FindStep(spec, scope.ParentState)
	if parentStep == nil {
		return nil, fmt.Errorf("scope parent step %q not found", scope.ParentState)
	}
	children, err := e.instanceRepo.ListByParentExecutionID(ctx, scope.ParentExecutionID)
	if err != nil {
		return nil, err
	}

	return &branchScopeContext{
		scope:          scope,
		parentExec:     parentExec,
		parentInstance: parentInstance,
		parentStep:     parentStep,
		children:       children,
	}, nil
}

func (e *stateEngine) tryLaunchMoreForeachChildren(
	ctx context.Context,
	scopeCtx *branchScopeContext,
	results []byte,
	completedCount int,
	failedCount int,
	runningCount int,
) (bool, error) {
	if scopeCtx.scope.ScopeType != string(dsl.StepTypeForeach) || failedCount > 0 ||
		scopeCtx.scope.NextChildIndex >= scopeCtx.scope.TotalChildren {
		return false, nil
	}

	capacity := scopeCtx.scope.MaxConcurrency - runningCount
	if capacity <= 0 {
		return false, nil
	}

	launchUntil, childRecords, err := e.buildForeachLaunchRecords(ctx, scopeCtx, capacity)
	if err != nil {
		return false, err
	}
	if len(childRecords) == 0 {
		return false, nil
	}

	if txErr := e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
		ScopeID:           scopeCtx.scope.ID,
		Status:            "running",
		CompletedChildren: completedCount,
		FailedChildren:    failedCount,
		NextChildIndex:    launchUntil,
		ResultsPayload:    string(results),
		ReleaseClaim:      true,
		LaunchChildren:    childRecords,
	}); txErr != nil {
		if errors.Is(txErr, repository.ErrStaleMutation) {
			return false, ErrStaleExecution
		}
		return false, txErr
	}

	return true, nil
}

func (e *stateEngine) buildForeachLaunchRecords(
	ctx context.Context,
	scopeCtx *branchScopeContext,
	capacity int,
) (int, []*repository.ScopedChildRecord, error) {
	childDefs, err := branchChildrenFromStep(
		scopeCtx.parentStep,
		json.RawMessage(scopeCtx.parentExec.InputPayload),
		scopeCtx.parentInstance.ID,
		scopeCtx.parentExec.ID,
	)
	if err != nil {
		return 0, nil, err
	}

	launchUntil := scopeCtx.scope.NextChildIndex + capacity
	if launchUntil > len(childDefs) {
		launchUntil = len(childDefs)
	}

	childRecords := make([]*repository.ScopedChildRecord, 0, launchUntil-scopeCtx.scope.NextChildIndex)
	for _, child := range childDefs[scopeCtx.scope.NextChildIndex:launchUntil] {
		record, buildErr := e.buildScopedChildRecord(
			ctx,
			scopeCtx.parentInstance,
			scopeCtx.parentExec,
			scopeCtx.parentStep,
			child,
		)
		if buildErr != nil {
			return 0, nil, buildErr
		}
		childRecords = append(childRecords, record)
	}

	return launchUntil, childRecords, nil
}

func (e *stateEngine) finalizeBranchScope(
	ctx context.Context,
	scopeCtx *branchScopeContext,
	results []byte,
	completedCount int,
	failedCount int,
	failure *CommitError,
) error {
	successOutput, shouldResume := evaluateScopeSuccess(*scopeCtx.scope, results, completedCount, failedCount)
	if failedCount > 0 {
		return e.failBranchScopeExecution(ctx, scopeCtx, results, completedCount, failedCount, failure)
	}
	if shouldResume {
		return e.completeBranchScopeExecution(ctx, scopeCtx, results, completedCount, failedCount, successOutput)
	}

	return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
		ScopeID:           scopeCtx.scope.ID,
		Status:            "running",
		CompletedChildren: completedCount,
		FailedChildren:    failedCount,
		NextChildIndex:    scopeCtx.scope.NextChildIndex,
		ResultsPayload:    string(results),
		ReleaseClaim:      true,
	})
}

func (e *stateEngine) failBranchScopeExecution(
	ctx context.Context,
	scopeCtx *branchScopeContext,
	results []byte,
	completedCount int,
	failedCount int,
	failure *CommitError,
) error {
	if failure == nil {
		failure = &CommitError{
			Class:   "fatal",
			Code:    "branch_failed",
			Message: "one or more scoped child workflows failed",
		}
	}

	failErr := e.FailWaitingExecution(ctx, scopeCtx.parentExec.ID, models.ExecStatusFatal, failure)
	if failErr != nil && !errors.Is(failErr, ErrStaleExecution) {
		return failErr
	}

	return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
		ScopeID:               scopeCtx.scope.ID,
		Status:                "failed",
		CompletedChildren:     completedCount,
		FailedChildren:        failedCount,
		NextChildIndex:        scopeCtx.scope.NextChildIndex,
		ResultsPayload:        string(results),
		ReleaseClaim:          true,
		CancelRunningChildren: true,
		ParentExecutionID:     scopeCtx.scope.ParentExecutionID,
	})
}

func (e *stateEngine) completeBranchScopeExecution(
	ctx context.Context,
	scopeCtx *branchScopeContext,
	results []byte,
	completedCount int,
	failedCount int,
	successOutput json.RawMessage,
) error {
	resumeErr := e.ResumeWaitingExecution(ctx, scopeCtx.parentExec.ID, successOutput)
	if resumeErr != nil && !errors.Is(resumeErr, ErrStaleExecution) {
		return resumeErr
	}

	return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
		ScopeID:               scopeCtx.scope.ID,
		Status:                "completed",
		CompletedChildren:     completedCount,
		FailedChildren:        failedCount,
		NextChildIndex:        scopeCtx.scope.NextChildIndex,
		ResultsPayload:        string(results),
		ReleaseClaim:          true,
		CancelRunningChildren: scopeCtx.scope.ScopeType == string(dsl.StepTypeParallel) && !scopeCtx.scope.WaitAll,
		ParentExecutionID:     scopeCtx.scope.ParentExecutionID,
	})
}

func defaultForeachConcurrency(value int) int {
	if value <= 0 {
		return 1
	}

	return value
}

func defaultForeachItemVar(value string) string {
	if value == "" {
		return "item"
	}

	return value
}

func defaultForeachIndexVar(value string) string {
	if value == "" {
		return "index"
	}

	return value
}

func parallelChildrenFromStep(
	step *dsl.StepSpec,
	inputPayload json.RawMessage,
	parentInstanceID string,
	parentExecutionID string,
) ([]scopedChildDefinition, error) {
	if step.Parallel == nil {
		return nil, errors.New("parallel step is missing parallel configuration")
	}

	baseInput, err := augmentInputPayload(inputPayload, map[string]any{
		"parent_instance_id":  parentInstanceID,
		"parent_execution_id": parentExecutionID,
	})
	if err != nil {
		return nil, err
	}

	children := make([]scopedChildDefinition, 0, len(step.Parallel.Steps))
	for index, child := range step.Parallel.Steps {
		children = append(children, scopedChildDefinition{
			Index:      index,
			EntryState: child.ID,
			Input:      baseInput,
		})
	}

	return children, nil
}

func foreachChildrenFromStep(
	step *dsl.StepSpec,
	inputPayload json.RawMessage,
	parentInstanceID string,
	parentExecutionID string,
) ([]scopedChildDefinition, error) {
	if step.Foreach == nil {
		return nil, errors.New("foreach step is missing foreach configuration")
	}
	if len(step.Foreach.Steps) == 0 {
		return nil, nil
	}

	items, err := evaluateForeachItems(step.Foreach, inputPayload)
	if err != nil {
		return nil, err
	}

	itemVar := defaultForeachItemVar(step.Foreach.ItemVar)
	indexVar := defaultForeachIndexVar(step.Foreach.IndexVar)
	children := make([]scopedChildDefinition, 0, len(items))
	for index, item := range items {
		childInput, inputErr := augmentInputPayload(inputPayload, map[string]any{
			"parent_instance_id":  parentInstanceID,
			"parent_execution_id": parentExecutionID,
			itemVar:               item,
			indexVar:              index,
			"scope_index":         index,
		})
		if inputErr != nil {
			return nil, inputErr
		}

		children = append(children, scopedChildDefinition{
			Index:      index,
			EntryState: step.Foreach.Steps[0].ID,
			Input:      childInput,
			Item:       item,
		})
	}

	return children, nil
}

func (e *stateEngine) buildScopedChildRecord(
	ctx context.Context,
	parent *models.WorkflowInstance,
	parentExec *models.WorkflowStateExecution,
	parentStep *dsl.StepSpec,
	child scopedChildDefinition,
) (*repository.ScopedChildRecord, error) {
	schemaHash, err := e.schemaReg.ValidateInput(
		ctx,
		parent.WorkflowName,
		parent.WorkflowVersion,
		child.EntryState,
		child.Input,
	)
	if err != nil {
		return nil, fmt.Errorf("validate scoped child input: %w", err)
	}

	now := time.Now()
	childInstance := &models.WorkflowInstance{
		WorkflowName:      parent.WorkflowName,
		WorkflowVersion:   parent.WorkflowVersion,
		CurrentState:      child.EntryState,
		Status:            models.InstanceStatusRunning,
		Revision:          1,
		ParentInstanceID:  parent.ID,
		ParentExecutionID: parentExec.ID,
		ScopeType:         string(parentStep.Type),
		ScopeParentState:  parentStep.ID,
		ScopeEntryState:   child.EntryState,
		ScopeIndex:        child.Index,
		StartedAt:         &now,
		Metadata:          "{}",
	}
	childInstance.ID = scopedChildInstanceID(parentExec.ID, parentStep.ID, child.Index, child.EntryState)

	execToken, err := cryptoutil.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("generate scoped child token: %w", err)
	}

	childExec := &models.WorkflowStateExecution{
		InstanceID:      childInstance.ID,
		State:           child.EntryState,
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  cryptoutil.HashToken(execToken),
		InputSchemaHash: schemaHash,
		InputPayload:    string(child.Input),
		TraceID:         parentExec.TraceID,
	}
	audit := &models.WorkflowAuditEvent{
		InstanceID:  childInstance.ID,
		ExecutionID: childExec.ID,
		EventType:   events.EventInstanceCreated,
		State:       child.EntryState,
		TraceID:     parentExec.TraceID,
	}

	return &repository.ScopedChildRecord{
		Instance:   childInstance,
		Execution:  childExec,
		AuditEvent: audit,
	}, nil
}

func augmentInputPayload(base json.RawMessage, extra map[string]any) (json.RawMessage, error) {
	payloadMap := map[string]any{}
	if len(base) > 0 {
		if err := json.Unmarshal(base, &payloadMap); err != nil {
			return nil, fmt.Errorf("unmarshal input payload: %w", err)
		}
	}

	for key, value := range extra {
		payloadMap[key] = value
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("marshal augmented input payload: %w", err)
	}

	return payload, nil
}

func evaluateForeachItems(spec *dsl.ForeachSpec, inputPayload json.RawMessage) ([]any, error) {
	payload := map[string]any{}
	if len(inputPayload) > 0 {
		if err := json.Unmarshal(inputPayload, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal foreach input payload: %w", err)
		}
	}

	env, err := dsl.NewExpressionEnv()
	if err != nil {
		return nil, fmt.Errorf("create CEL env: %w", err)
	}

	ast, err := dsl.CompileExpression(env, spec.Items)
	if err != nil {
		return nil, fmt.Errorf("compile foreach.items: %w", err)
	}

	vars := map[string]any{"payload": payload}
	for key, value := range payload {
		vars[key] = value
	}

	value, err := dsl.EvaluateExpression(env, ast, vars)
	if err != nil {
		return nil, fmt.Errorf("evaluate foreach.items: %w", err)
	}

	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("foreach.items must evaluate to list, got %T", value)
	}

	return items, nil
}

func extractForeachItems(children []scopedChildDefinition) []any {
	items := make([]any, 0, len(children))
	for _, child := range children {
		items = append(items, child.Item)
	}

	return items
}

func marshalEmptyResults(total int) ([]byte, error) {
	results := make([]any, total)
	return json.Marshal(results)
}

func emptyScopeOutput(stepType dsl.StepType) json.RawMessage {
	switch stepType {
	case dsl.StepTypeForeach:
		return json.RawMessage(`{"items":[]}`)
	case dsl.StepTypeParallel:
		return json.RawMessage(`{"branches":[]}`)
	case dsl.StepTypeCall,
		dsl.StepTypeDelay,
		dsl.StepTypeIf,
		dsl.StepTypeSequence,
		dsl.StepTypeSignalWait,
		dsl.StepTypeSignalSend:
		return json.RawMessage(`{}`)
	default:
		return json.RawMessage(`{"branches":[]}`)
	}
}

func scopedChildInstanceID(parentExecutionID, parentState string, index int, entryState string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", parentExecutionID, parentState, index, entryState)))
	return "wfi_" + hex.EncodeToString(hash[:])[:40]
}

func (e *stateEngine) collectScopeResults(
	ctx context.Context,
	children []*models.WorkflowInstance,
	totalChildren int,
) ([]byte, int, int, int, *CommitError, error) {
	results := make([]any, totalChildren)

	completedCount := 0
	failedCount := 0
	runningCount := 0
	var firstFailure *CommitError

	for _, child := range children {
		switch child.Status {
		case models.InstanceStatusCompleted:
			completedCount++
			output, err := e.outputRepo.GetByInstanceAndState(ctx, child.ID, child.CurrentState)
			if err == nil && output != nil && output.Payload != "" && child.ScopeIndex < len(results) {
				var payload any
				if unmarshalErr := json.Unmarshal([]byte(output.Payload), &payload); unmarshalErr == nil {
					results[child.ScopeIndex] = payload
				}
			}
		case models.InstanceStatusFailed, models.InstanceStatusCancelled:
			failedCount++
			if firstFailure == nil {
				latestExec, err := e.execRepo.GetLatestByInstance(ctx, child.ID)
				if err == nil && latestExec != nil {
					firstFailure = &CommitError{
						Class:   latestExec.ErrorClass,
						Code:    "branch_failed",
						Message: latestExec.ErrorMessage,
					}
				}
			}
		case models.InstanceStatusRunning, models.InstanceStatusSuspended:
			runningCount++
		default:
			runningCount++
		}
	}

	payload, err := json.Marshal(results)
	if err != nil {
		return nil, 0, 0, 0, nil, fmt.Errorf("marshal scope results: %w", err)
	}

	return payload, completedCount, failedCount, runningCount, firstFailure, nil
}

func evaluateScopeSuccess(
	scope models.WorkflowScopeRun,
	resultsPayload []byte,
	completedCount int,
	failedCount int,
) (json.RawMessage, bool) {
	if failedCount > 0 {
		return nil, false
	}

	switch scope.ScopeType {
	case string(dsl.StepTypeParallel):
		if !scope.WaitAll {
			if completedCount == 0 {
				return nil, false
			}

			var results []any
			_ = json.Unmarshal(resultsPayload, &results)
			for index, result := range results {
				if result == nil {
					continue
				}

				output, _ := json.Marshal(map[string]any{
					"winner_index": index,
					"winner":       result,
					"branches":     results,
				})
				return output, true
			}

			return nil, false
		}

		if completedCount != scope.TotalChildren {
			return nil, false
		}

		output, _ := json.Marshal(map[string]any{
			"branches": json.RawMessage(resultsPayload),
		})
		return output, true
	case string(dsl.StepTypeForeach):
		if completedCount != scope.TotalChildren {
			return nil, false
		}

		output, _ := json.Marshal(map[string]any{
			"items": json.RawMessage(resultsPayload),
		})
		return output, true
	default:
		return nil, false
	}
}
