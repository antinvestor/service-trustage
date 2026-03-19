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
	if step == nil {
		return errors.New("branch scope step is required")
	}
	if step.Type != dsl.StepTypeParallel && step.Type != dsl.StepTypeForeach {
		return fmt.Errorf("unsupported branch scope type %q", step.Type)
	}

	exec, err := e.execRepo.GetByID(ctx, cmd.ExecutionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}
	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return fmt.Errorf("load instance: %w", err)
	}

	scope := &models.WorkflowScopeRun{
		ParentExecutionID: exec.ID,
		ParentInstanceID:  instance.ID,
		ParentState:       exec.State,
		ScopeType:         string(step.Type),
		Status:            "running",
	}

	childDefs, err := branchChildrenFromStep(step, cmd.InputPayload, instance.ID, exec.ID)
	if err != nil {
		return err
	}

	scope.TotalChildren = len(childDefs)
	resultsPayload, err := marshalEmptyResults(scope.TotalChildren)
	if err != nil {
		return err
	}
	scope.ResultsPayload = string(resultsPayload)

	switch step.Type {
	case dsl.StepTypeParallel:
		scope.WaitAll = step.Parallel.WaitAll
		scope.MaxConcurrency = len(childDefs)
	case dsl.StepTypeForeach:
		scope.WaitAll = true
		scope.MaxConcurrency = step.Foreach.MaxConcurrency
		if scope.MaxConcurrency <= 0 {
			scope.MaxConcurrency = 1
		}

		scope.ItemVar = step.Foreach.ItemVar
		if scope.ItemVar == "" {
			scope.ItemVar = "item"
		}
		scope.IndexVar = step.Foreach.IndexVar
		if scope.IndexVar == "" {
			scope.IndexVar = "index"
		}

		itemsPayload, marshalErr := json.Marshal(extractForeachItems(childDefs))
		if marshalErr != nil {
			return fmt.Errorf("marshal foreach items: %w", marshalErr)
		}
		scope.ItemsPayload = string(itemsPayload)
	}

	tokenHash := cryptoutil.HashToken(cmd.ExecutionToken)
	launchCount := len(childDefs)
	if step.Type == dsl.StepTypeForeach && launchCount > scope.MaxConcurrency {
		launchCount = scope.MaxConcurrency
	}

	childRecords := make([]*repository.ScopedChildRecord, 0, launchCount)
	for i := range launchCount {
		record, recordErr := e.buildScopedChildRecord(ctx, instance, exec, step, childDefs[i])
		if recordErr != nil {
			return recordErr
		}
		childRecords = append(childRecords, record)
	}

	scope.NextChildIndex = launchCount
	if txErr := e.runtimeRepo.StartBranchScope(ctx, &repository.StartBranchScopeRequest{
		Execution:      exec,
		Instance:       instance,
		TokenHash:      tokenHash,
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

func (e *stateEngine) ReconcileBranchScope(ctx context.Context, scopeID string) error {
	scope, err := e.scopeRepo.GetByID(ctx, scopeID)
	if err != nil {
		return fmt.Errorf("load workflow scope: %w", err)
	}
	if scope.Status != "running" {
		return nil
	}

	parentExec, err := e.execRepo.GetByID(ctx, scope.ParentExecutionID)
	if err != nil {
		return fmt.Errorf("load parent execution: %w", err)
	}
	parentInstance, err := e.instanceRepo.GetByID(ctx, scope.ParentInstanceID)
	if err != nil {
		return fmt.Errorf("load parent instance: %w", err)
	}

	spec, err := e.loadSpec(ctx, parentInstance.WorkflowName, parentInstance.WorkflowVersion)
	if err != nil {
		return err
	}

	parentStep := dsl.FindStep(spec, scope.ParentState)
	if parentStep == nil {
		return fmt.Errorf("scope parent step %q not found", scope.ParentState)
	}

	children, err := e.instanceRepo.ListByParentExecutionID(ctx, scope.ParentExecutionID)
	if err != nil {
		return err
	}

	results, completedCount, failedCount, runningCount, failure, err := e.collectScopeResults(
		ctx,
		children,
		scope.TotalChildren,
	)
	if err != nil {
		return err
	}

	nextChildIndex := scope.NextChildIndex
	if scope.ScopeType == string(dsl.StepTypeForeach) && failedCount == 0 && nextChildIndex < scope.TotalChildren {
		capacity := scope.MaxConcurrency - runningCount
		if capacity < 0 {
			capacity = 0
		}
		if capacity > 0 {
			childDefs, childErr := branchChildrenFromStep(
				parentStep,
				json.RawMessage(parentExec.InputPayload),
				parentInstance.ID,
				parentExec.ID,
			)
			if childErr != nil {
				return childErr
			}

			launchUntil := nextChildIndex + capacity
			if launchUntil > len(childDefs) {
				launchUntil = len(childDefs)
			}

			childRecords := make([]*repository.ScopedChildRecord, 0, launchUntil-nextChildIndex)
			for _, child := range childDefs[nextChildIndex:launchUntil] {
				record, recordErr := e.buildScopedChildRecord(ctx, parentInstance, parentExec, parentStep, child)
				if recordErr != nil {
					return recordErr
				}
				childRecords = append(childRecords, record)
			}

			if txErr := e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
				ScopeID:           scope.ID,
				Status:            "running",
				CompletedChildren: completedCount,
				FailedChildren:    failedCount,
				NextChildIndex:    launchUntil,
				ResultsPayload:    string(results),
				ReleaseClaim:      true,
				LaunchChildren:    childRecords,
			}); txErr != nil {
				if errors.Is(txErr, repository.ErrStaleMutation) {
					return ErrStaleExecution
				}
				return txErr
			}

			return nil
		}
	}

	successOutput, shouldResume := evaluateScopeSuccess(*scope, results, completedCount, failedCount)
	shouldFail := failedCount > 0

	if shouldFail {
		if failure == nil {
			failure = &CommitError{
				Class:   "fatal",
				Code:    "branch_failed",
				Message: "one or more scoped child workflows failed",
			}
		}

		failErr := e.FailWaitingExecution(ctx, parentExec.ID, models.ExecStatusFatal, failure)
		if failErr != nil && !errors.Is(failErr, ErrStaleExecution) {
			return failErr
		}

		return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
			ScopeID:               scope.ID,
			Status:                "failed",
			CompletedChildren:     completedCount,
			FailedChildren:        failedCount,
			NextChildIndex:        scope.NextChildIndex,
			ResultsPayload:        string(results),
			ReleaseClaim:          true,
			CancelRunningChildren: true,
			ParentExecutionID:     scope.ParentExecutionID,
		})
	}

	if shouldResume {
		resumeErr := e.ResumeWaitingExecution(ctx, parentExec.ID, successOutput)
		if resumeErr != nil && !errors.Is(resumeErr, ErrStaleExecution) {
			return resumeErr
		}

		return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
			ScopeID:               scope.ID,
			Status:                "completed",
			CompletedChildren:     completedCount,
			FailedChildren:        failedCount,
			NextChildIndex:        scope.NextChildIndex,
			ResultsPayload:        string(results),
			ReleaseClaim:          true,
			CancelRunningChildren: scope.ScopeType == string(dsl.StepTypeParallel) && !scope.WaitAll,
			ParentExecutionID:     scope.ParentExecutionID,
		})
	}

	return e.runtimeRepo.UpdateScope(ctx, &repository.UpdateScopeRequest{
		ScopeID:           scope.ID,
		Status:            "running",
		CompletedChildren: completedCount,
		FailedChildren:    failedCount,
		NextChildIndex:    scope.NextChildIndex,
		ResultsPayload:    string(results),
		ReleaseClaim:      true,
	})
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
	case dsl.StepTypeForeach:
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

		itemVar := step.Foreach.ItemVar
		if itemVar == "" {
			itemVar = "item"
		}
		indexVar := step.Foreach.IndexVar
		if indexVar == "" {
			indexVar = "index"
		}

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
	default:
		return nil, fmt.Errorf("unsupported scope type %q", step.Type)
	}
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
