package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// WorkflowRuntimeRepository owns cross-table runtime mutations for the workflow engine.
type WorkflowRuntimeRepository interface {
	UpdateExecutionStatus(
		ctx context.Context,
		exec *models.WorkflowStateExecution,
		status models.ExecutionStatus,
	) error
	CommitExecution(ctx context.Context, req *CommitExecutionRequest) error
	ParkExecution(ctx context.Context, req *ParkExecutionRequest) error
	StartSignalWait(ctx context.Context, req *StartSignalWaitRequest) error
	ClaimSignalDelivery(ctx context.Context, req *ClaimSignalDeliveryRequest) (*SignalDeliveryClaim, error)
	FailExecution(
		ctx context.Context,
		req *FailExecutionRequest,
	) error
	CreateRetryExecution(
		ctx context.Context,
		req *CreateRetryExecutionRequest,
	) (*models.WorkflowStateExecution, error)
	StartBranchScope(ctx context.Context, req *StartBranchScopeRequest) error
	UpdateScope(ctx context.Context, req *UpdateScopeRequest) error
}

type workflowRuntimeRepository struct {
	datastore.BaseRepository[*models.WorkflowInstance]
}

// CommitExecutionRequest describes a successful execution commit.
type CommitExecutionRequest struct {
	Execution           *models.WorkflowStateExecution
	Instance            *models.WorkflowInstance
	TokenHash           string
	VerifyToken         bool
	OutputPayload       string
	ExpectedStatus      models.ExecutionStatus
	NextState           string
	NextInputPayload    string
	NextInputSchemaHash string
}

// ParkExecutionRequest describes parking a dispatched execution behind a durable timer.
type ParkExecutionRequest struct {
	Execution  *models.WorkflowStateExecution
	Instance   *models.WorkflowInstance
	TokenHash  string
	FireAt     time.Time
	AuditTrace string
}

// StartSignalWaitRequest describes parking a dispatched execution on a named signal.
type StartSignalWaitRequest struct {
	Execution  *models.WorkflowStateExecution
	Instance   *models.WorkflowInstance
	TokenHash  string
	SignalName string
	OutputVar  string
	TimeoutAt  *time.Time
	AuditTrace string
}

// ClaimSignalDeliveryRequest describes matching the oldest pending signal to the oldest waiting execution.
type ClaimSignalDeliveryRequest struct {
	InstanceID string
	SignalName string
	Owner      string
	LeaseUntil time.Time
}

// SignalDeliveryClaim is the result of attempting to deliver a pending signal.
type SignalDeliveryClaim struct {
	Message *models.WorkflowSignalMessage
	Wait    *models.WorkflowSignalWait
}

// FailExecutionRequest describes a terminal failure mutation for an execution and its instance.
type FailExecutionRequest struct {
	Execution      *models.WorkflowStateExecution
	Instance       *models.WorkflowInstance
	TokenHash      string
	VerifyToken    bool
	ExpectedStatus models.ExecutionStatus
	Status         models.ExecutionStatus
	ErrorClass     string
	ErrorMessage   string
	AuditTrace     string
	AuditPayload   string
}

// CreateRetryExecutionRequest describes creating a retry execution and resetting the instance.
type CreateRetryExecutionRequest struct {
	Execution    *models.WorkflowStateExecution
	Instance     *models.WorkflowInstance
	NewExecution *models.WorkflowStateExecution
}

// ScopedChildRecord carries the durable rows created for a scoped child workflow.
type ScopedChildRecord struct {
	Instance   *models.WorkflowInstance
	Execution  *models.WorkflowStateExecution
	AuditEvent *models.WorkflowAuditEvent
}

// StartBranchScopeRequest describes parking an execution behind a branch scope and launching initial children.
type StartBranchScopeRequest struct {
	Execution      *models.WorkflowStateExecution
	Instance       *models.WorkflowInstance
	TokenHash      string
	Scope          *models.WorkflowScopeRun
	LaunchChildren []*ScopedChildRecord
	AuditTrace     string
}

// UpdateScopeRequest describes a durable scope reconciliation update.
type UpdateScopeRequest struct {
	ScopeID               string
	Status                string
	CompletedChildren     int
	FailedChildren        int
	NextChildIndex        int
	ResultsPayload        string
	ReleaseClaim          bool
	LaunchChildren        []*ScopedChildRecord
	CancelRunningChildren bool
	ParentExecutionID     string
}

// NewWorkflowRuntimeRepository creates a new repository for workflow runtime mutations.
func NewWorkflowRuntimeRepository(dbPool pool.Pool) WorkflowRuntimeRepository {
	ctx := context.Background()

	return &workflowRuntimeRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowInstance](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowInstance { return &models.WorkflowInstance{} },
		),
	}
}

func (r *workflowRuntimeRepository) UpdateExecutionStatus(
	ctx context.Context,
	exec *models.WorkflowStateExecution,
	status models.ExecutionStatus,
) error {
	if exec == nil {
		return errors.New("execution is required")
	}

	ctx = security.SkipTenancyChecksOnClaims(ctx)
	now := time.Now()

	result := r.BaseRepository.Pool().DB(ctx, false).
		Model(&models.WorkflowStateExecution{}).
		Where("id = ? AND deleted_at IS NULL", exec.ID).
		UpdateColumns(map[string]any{
			"status":      status,
			"modified_at": now,
			"finished_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("mark execution contract violation: %w", result.Error)
	}

	return nil
}

func (r *workflowRuntimeRepository) CommitExecution(
	ctx context.Context,
	req *CommitExecutionRequest,
) error {
	if req == nil || req.Execution == nil || req.Instance == nil {
		return errors.New("commit execution request requires execution and instance")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	outputSchemaHash := computeOutputSchemaHash(req.OutputPayload)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if err := verifyCommitExecutionLock(tx, req); err != nil {
			return err
		}
		if err := storeCommitExecutionOutput(tx, req, outputSchemaHash); err != nil {
			return err
		}
		if err := markExecutionCommitted(tx, req.Execution.ID, outputSchemaHash); err != nil {
			return err
		}
		if err := lockCommitInstance(tx, req); err != nil {
			return err
		}
		if req.NextState == "" {
			return completeInstance(tx, req.Instance.ID)
		}

		return createCommitNextExecution(tx, req)
	})
}

func verifyCommitExecutionLock(tx *gorm.DB, req *CommitExecutionRequest) error {
	if req.VerifyToken {
		_, err := lockExecutionByToken(tx, req.Execution.ID, req.TokenHash, req.ExpectedStatus)
		return err
	}

	_, err := lockExecutionByStatus(tx, req.Execution.ID, req.ExpectedStatus)
	return err
}

func storeCommitExecutionOutput(
	tx *gorm.DB,
	req *CommitExecutionRequest,
	outputSchemaHash string,
) error {
	output := &models.WorkflowStateOutput{
		ExecutionID: req.Execution.ID,
		InstanceID:  req.Execution.InstanceID,
		State:       req.Execution.State,
		SchemaHash:  outputSchemaHash,
		Payload:     req.OutputPayload,
	}
	if err := tx.Create(output).Error; err != nil {
		return fmt.Errorf("store output: %w", err)
	}

	return nil
}

func markExecutionCommitted(tx *gorm.DB, executionID, outputSchemaHash string) error {
	now := time.Now()
	execUpdates := map[string]any{
		"status":             models.ExecStatusCompleted,
		"output_schema_hash": outputSchemaHash,
		"execution_token":    "",
		"modified_at":        now,
		"finished_at":        now,
	}

	return updateExecutionByID(tx, executionID, execUpdates)
}

func lockCommitInstance(tx *gorm.DB, req *CommitExecutionRequest) error {
	_, err := lockInstanceForMutation(
		tx,
		req.Instance.ID,
		req.Execution.State,
		req.Instance.Revision,
	)

	return err
}

func createCommitNextExecution(tx *gorm.DB, req *CommitExecutionRequest) error {
	if err := transitionInstance(tx, req.Instance.ID, req.NextState); err != nil {
		return err
	}

	nextToken, err := cryptoutil.GenerateToken()
	if err != nil {
		return fmt.Errorf("generate next execution token: %w", err)
	}

	nextExec := &models.WorkflowStateExecution{
		InstanceID:      req.Execution.InstanceID,
		State:           req.NextState,
		Attempt:         1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  cryptoutil.HashToken(nextToken),
		InputSchemaHash: req.NextInputSchemaHash,
		InputPayload:    req.NextInputPayload,
		TraceID:         req.Execution.TraceID,
	}
	if createErr := tx.Create(nextExec).Error; createErr != nil {
		return fmt.Errorf("create next execution: %w", createErr)
	}

	return appendCommitTransitionAudit(tx, req)
}

func appendCommitTransitionAudit(tx *gorm.DB, req *CommitExecutionRequest) error {
	audit := &models.WorkflowAuditEvent{
		InstanceID:  req.Execution.InstanceID,
		ExecutionID: req.Execution.ID,
		EventType:   events.EventTransitionCommitted,
		State:       req.NextState,
		FromState:   req.Execution.State,
		ToState:     req.NextState,
		TraceID:     req.Execution.TraceID,
	}
	if err := tx.Create(audit).Error; err != nil {
		return fmt.Errorf("append transition audit: %w", err)
	}

	return nil
}

func (r *workflowRuntimeRepository) ParkExecution(ctx context.Context, req *ParkExecutionRequest) error {
	if req == nil || req.Execution == nil || req.Instance == nil {
		return errors.New("park execution request requires execution and instance")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if _, err := lockExecutionByToken(
			tx,
			req.Execution.ID,
			req.TokenHash,
			models.ExecStatusDispatched,
		); err != nil {
			return err
		}

		now := time.Now()
		if err := updateExecutionByID(tx, req.Execution.ID, map[string]any{
			"status":          models.ExecStatusWaiting,
			"execution_token": "",
			"modified_at":     now,
		}); err != nil {
			return err
		}

		timer := &models.WorkflowTimer{
			ExecutionID: req.Execution.ID,
			InstanceID:  req.Execution.InstanceID,
			State:       req.Execution.State,
			FiresAt:     req.FireAt,
		}
		if err := tx.Create(timer).Error; err != nil {
			return fmt.Errorf("create workflow timer: %w", err)
		}

		audit := &models.WorkflowAuditEvent{
			InstanceID:  req.Instance.ID,
			ExecutionID: req.Execution.ID,
			EventType:   events.EventStateWaiting,
			State:       req.Execution.State,
			TraceID:     req.AuditTrace,
			Payload:     fmt.Sprintf(`{"wait_type":"delay","fire_at":"%s"}`, req.FireAt.UTC().Format(time.RFC3339)),
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("append delay audit: %w", err)
		}

		return nil
	})
}

func (r *workflowRuntimeRepository) StartSignalWait(ctx context.Context, req *StartSignalWaitRequest) error {
	if req == nil || req.Execution == nil || req.Instance == nil {
		return errors.New("signal wait request requires execution and instance")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if _, err := lockExecutionByToken(
			tx,
			req.Execution.ID,
			req.TokenHash,
			models.ExecStatusDispatched,
		); err != nil {
			return err
		}

		now := time.Now()
		if err := updateExecutionByID(tx, req.Execution.ID, map[string]any{
			"status":          models.ExecStatusWaiting,
			"execution_token": "",
			"modified_at":     now,
		}); err != nil {
			return err
		}

		wait := &models.WorkflowSignalWait{
			ExecutionID: req.Execution.ID,
			InstanceID:  req.Execution.InstanceID,
			State:       req.Execution.State,
			SignalName:  req.SignalName,
			OutputVar:   req.OutputVar,
			Status:      "waiting",
			TimeoutAt:   req.TimeoutAt,
		}
		if err := tx.Create(wait).Error; err != nil {
			return fmt.Errorf("create signal wait: %w", err)
		}

		audit := &models.WorkflowAuditEvent{
			InstanceID:  req.Instance.ID,
			ExecutionID: req.Execution.ID,
			EventType:   events.EventStateWaiting,
			State:       req.Execution.State,
			TraceID:     req.AuditTrace,
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("append signal wait audit: %w", err)
		}

		return nil
	})
}

func (r *workflowRuntimeRepository) ClaimSignalDelivery(
	ctx context.Context,
	req *ClaimSignalDeliveryRequest,
) (*SignalDeliveryClaim, error) {
	if req == nil {
		return nil, errors.New("claim signal delivery request is required")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	claim := &SignalDeliveryClaim{}
	txErr := r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		message, foundMessage, err := claimPendingSignalMessage(tx, req, now)
		if err != nil {
			return err
		}
		if !foundMessage {
			return nil
		}

		wait, foundWait, waitErr := loadWaitingSignalWait(tx, req)
		if waitErr != nil {
			return waitErr
		}
		if !foundWait {
			return nil
		}

		if finalizeErr := finalizeSignalDeliveryClaim(tx, req, message, wait, now); finalizeErr != nil {
			return finalizeErr
		}

		claim.Message = message
		claim.Wait = wait
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	if claim.Message == nil {
		return nil, nil //nolint:nilnil // nil claim means there is currently no deliverable signal/message pair.
	}

	return claim, nil
}

func claimPendingSignalMessage(
	tx *gorm.DB,
	req *ClaimSignalDeliveryRequest,
	now time.Time,
) (*models.WorkflowSignalMessage, bool, error) {
	message := &models.WorkflowSignalMessage{}
	result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(
			"target_instance_id = ? AND signal_name = ? AND status = ? AND deleted_at IS NULL",
			req.InstanceID,
			req.SignalName,
			"pending",
		).
		Order("created_at ASC").
		First(message)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if result.Error != nil {
		return nil, false, fmt.Errorf("claim signal message: %w", result.Error)
	}

	messageUpdates := map[string]any{
		"claim_owner": req.Owner,
		"claim_until": req.LeaseUntil,
		"attempts":    gorm.Expr("attempts + ?", 1),
		"modified_at": now,
	}
	if err := tx.Model(&models.WorkflowSignalMessage{}).
		Where("id = ? AND deleted_at IS NULL", message.ID).
		Updates(messageUpdates).Error; err != nil {
		return nil, false, fmt.Errorf("lease signal message: %w", err)
	}

	message.ClaimOwner = req.Owner
	message.ClaimUntil = &req.LeaseUntil
	message.Attempts++
	return message, true, nil
}

func loadWaitingSignalWait(
	tx *gorm.DB,
	req *ClaimSignalDeliveryRequest,
) (*models.WorkflowSignalWait, bool, error) {
	wait := &models.WorkflowSignalWait{}
	result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(
			"instance_id = ? AND signal_name = ? AND status = ? AND deleted_at IS NULL",
			req.InstanceID,
			req.SignalName,
			"waiting",
		).
		Order("created_at ASC").
		First(wait)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if result.Error != nil {
		return nil, false, fmt.Errorf("load signal wait: %w", result.Error)
	}

	return wait, true, nil
}

func finalizeSignalDeliveryClaim(
	tx *gorm.DB,
	req *ClaimSignalDeliveryRequest,
	message *models.WorkflowSignalMessage,
	wait *models.WorkflowSignalWait,
	now time.Time,
) error {
	waitUpdates := map[string]any{
		"status":      "matched",
		"matched_at":  now,
		"message_id":  message.ID,
		"modified_at": now,
	}
	if err := tx.Model(&models.WorkflowSignalWait{}).
		Where("id = ? AND status = ? AND deleted_at IS NULL", wait.ID, "waiting").
		Updates(waitUpdates).Error; err != nil {
		return fmt.Errorf("mark signal wait matched: %w", err)
	}

	deliveredUpdates := map[string]any{
		"status":       "delivered",
		"delivered_at": now,
		"wait_id":      wait.ID,
		"claim_owner":  "",
		"claim_until":  nil,
		"modified_at":  now,
	}
	if err := tx.Model(&models.WorkflowSignalMessage{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", message.ID, req.Owner, "pending").
		Updates(deliveredUpdates).Error; err != nil {
		return fmt.Errorf("mark signal delivered: %w", err)
	}

	audit := &models.WorkflowAuditEvent{
		InstanceID:  wait.InstanceID,
		ExecutionID: wait.ExecutionID,
		EventType:   events.EventSignalReceived,
		State:       wait.State,
		Payload:     message.Payload,
	}
	if err := tx.Create(audit).Error; err != nil {
		return fmt.Errorf("append signal delivery audit: %w", err)
	}

	wait.Status = "matched"
	wait.MatchedAt = &now
	wait.MessageID = message.ID
	message.Status = "delivered"
	message.DeliveredAt = &now
	message.WaitID = wait.ID
	message.ClaimOwner = ""
	message.ClaimUntil = nil
	return nil
}

func (r *workflowRuntimeRepository) FailExecution(ctx context.Context, req *FailExecutionRequest) error {
	if req == nil || req.Execution == nil || req.Instance == nil {
		return errors.New("fail execution request requires execution and instance")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if req.VerifyToken {
			if _, err := lockExecutionByToken(tx, req.Execution.ID, req.TokenHash, req.ExpectedStatus); err != nil {
				return err
			}
		} else {
			if _, err := lockExecutionByStatus(tx, req.Execution.ID, req.ExpectedStatus); err != nil {
				return err
			}
		}

		now := time.Now()
		if err := updateExecutionByID(tx, req.Execution.ID, map[string]any{
			"status":          req.Status,
			"error_class":     req.ErrorClass,
			"error_message":   req.ErrorMessage,
			"execution_token": "",
			"modified_at":     now,
			"finished_at":     now,
		}); err != nil {
			return err
		}

		instanceResult := tx.Model(&models.WorkflowInstance{}).
			Where("id = ? AND status = ? AND deleted_at IS NULL", req.Instance.ID, models.InstanceStatusRunning).
			UpdateColumns(map[string]any{
				"status":      models.InstanceStatusFailed,
				"modified_at": now,
				"finished_at": now,
			})
		if instanceResult.Error != nil {
			return fmt.Errorf("mark instance failed: %w", instanceResult.Error)
		}
		if instanceResult.RowsAffected == 0 {
			return ErrStaleMutation
		}

		eventType := events.EventStateFailed
		if req.Status == models.ExecStatusTimedOut {
			eventType = events.EventStateTimedOut
		}

		audit := &models.WorkflowAuditEvent{
			InstanceID:  req.Instance.ID,
			ExecutionID: req.Execution.ID,
			EventType:   eventType,
			State:       req.Execution.State,
			TraceID:     req.AuditTrace,
			Payload:     req.AuditPayload,
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("append failure audit: %w", err)
		}

		return nil
	})
}

func (r *workflowRuntimeRepository) CreateRetryExecution(
	ctx context.Context,
	req *CreateRetryExecutionRequest,
) (*models.WorkflowStateExecution, error) {
	if req == nil || req.Execution == nil || req.Instance == nil || req.NewExecution == nil {
		return nil, errors.New("create retry execution request requires execution, instance and new execution")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	if err := r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(req.NewExecution).Error; err != nil {
			return fmt.Errorf("create retry execution: %w", err)
		}

		now := time.Now()
		result := tx.Model(&models.WorkflowInstance{}).
			Where("id = ? AND deleted_at IS NULL", req.Instance.ID).
			UpdateColumns(map[string]any{
				"status":        models.InstanceStatusRunning,
				"current_state": req.Execution.State,
				"modified_at":   now,
				"finished_at":   nil,
				"revision":      gorm.Expr("revision + 1"),
			})
		if result.Error != nil {
			return fmt.Errorf("reset instance for retry: %w", result.Error)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return req.NewExecution, nil
}

func (r *workflowRuntimeRepository) StartBranchScope(ctx context.Context, req *StartBranchScopeRequest) error {
	if req == nil || req.Execution == nil || req.Instance == nil || req.Scope == nil {
		return errors.New("start branch scope request requires execution, instance and scope")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		if _, err := lockExecutionByToken(
			tx,
			req.Execution.ID,
			req.TokenHash,
			models.ExecStatusDispatched,
		); err != nil {
			return err
		}

		now := time.Now()
		if err := updateExecutionByID(tx, req.Execution.ID, map[string]any{
			"status":          models.ExecStatusWaiting,
			"execution_token": "",
			"modified_at":     now,
		}); err != nil {
			return err
		}

		if err := tx.Create(req.Scope).Error; err != nil {
			return fmt.Errorf("create workflow scope run: %w", err)
		}

		for _, child := range req.LaunchChildren {
			if err := createScopedChildRecords(tx, child); err != nil {
				return err
			}
		}

		scopeUpdates := map[string]any{
			"next_child_index": req.Scope.NextChildIndex,
			"modified_at":      now,
		}
		if err := tx.Model(&models.WorkflowScopeRun{}).
			Where("id = ? AND deleted_at IS NULL", req.Scope.ID).
			Updates(scopeUpdates).Error; err != nil {
			return fmt.Errorf("update scope launch state: %w", err)
		}

		audit := &models.WorkflowAuditEvent{
			InstanceID:  req.Instance.ID,
			ExecutionID: req.Execution.ID,
			EventType:   events.EventStateWaiting,
			State:       req.Execution.State,
			TraceID:     req.AuditTrace,
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("append scope wait audit: %w", err)
		}

		return nil
	})
}

func (r *workflowRuntimeRepository) UpdateScope(ctx context.Context, req *UpdateScopeRequest) error {
	if req == nil {
		return errors.New("update scope request is required")
	}
	ctx = security.SkipTenancyChecksOnClaims(ctx)

	return r.BaseRepository.Pool().DB(ctx, false).Transaction(func(tx *gorm.DB) error {
		for _, child := range req.LaunchChildren {
			if err := createScopedChildRecords(tx, child); err != nil {
				return err
			}
		}

		updates := map[string]any{
			"status":             req.Status,
			"completed_children": req.CompletedChildren,
			"failed_children":    req.FailedChildren,
			"results_payload":    req.ResultsPayload,
			"next_child_index":   req.NextChildIndex,
			"modified_at":        time.Now(),
		}
		if req.ReleaseClaim {
			updates["claim_owner"] = ""
			updates["claim_until"] = nil
		}

		result := tx.Model(&models.WorkflowScopeRun{}).
			Where("id = ? AND deleted_at IS NULL", req.ScopeID).
			UpdateColumns(updates)
		if result.Error != nil {
			return fmt.Errorf("update workflow scope: %w", result.Error)
		}

		if req.CancelRunningChildren {
			now := time.Now()
			cancelResult := tx.Model(&models.WorkflowInstance{}).
				Where(
					"parent_execution_id = ? AND status = ? AND deleted_at IS NULL",
					req.ParentExecutionID,
					models.InstanceStatusRunning,
				).
				UpdateColumns(map[string]any{
					"status":      models.InstanceStatusCancelled,
					"modified_at": now,
					"finished_at": now,
				})
			if cancelResult.Error != nil {
				return fmt.Errorf("cancel running scoped children: %w", cancelResult.Error)
			}
		}

		return nil
	})
}

func createScopedChildRecords(tx *gorm.DB, child *ScopedChildRecord) error {
	if child == nil || child.Instance == nil || child.Execution == nil {
		return errors.New("scoped child record requires instance and execution")
	}

	if err := tx.Create(child.Instance).Error; err != nil {
		return fmt.Errorf("create scoped child instance: %w", err)
	}
	if err := tx.Create(child.Execution).Error; err != nil {
		return fmt.Errorf("create scoped child execution: %w", err)
	}
	if child.AuditEvent != nil {
		if err := tx.Create(child.AuditEvent).Error; err != nil {
			return fmt.Errorf("append scoped child audit: %w", err)
		}
	}

	return nil
}

func lockExecutionByToken(
	tx *gorm.DB,
	executionID string,
	tokenHash string,
	status models.ExecutionStatus,
) (*models.WorkflowStateExecution, error) {
	return lockExecution(tx, executionID, status, tokenHash, true)
}

func lockExecutionByStatus(
	tx *gorm.DB,
	executionID string,
	status models.ExecutionStatus,
) (*models.WorkflowStateExecution, error) {
	return lockExecution(tx, executionID, status, "", false)
}

func lockExecution(
	tx *gorm.DB,
	executionID string,
	status models.ExecutionStatus,
	tokenHash string,
	requireToken bool,
) (*models.WorkflowStateExecution, error) {
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND status = ? AND deleted_at IS NULL", executionID, status)
	if requireToken {
		query = query.Where("execution_token = ?", tokenHash)
	}

	var exec models.WorkflowStateExecution
	result := query.First(&exec)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if requireToken {
			return nil, ErrInvalidExecutionToken
		}

		return nil, fmt.Errorf("lock execution: %w", ErrStaleMutation)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("lock execution: %w", result.Error)
	}

	return &exec, nil
}

func updateExecutionByID(tx *gorm.DB, executionID string, updates map[string]any) error {
	result := tx.Model(&models.WorkflowStateExecution{}).
		Where("id = ? AND deleted_at IS NULL", executionID).
		UpdateColumns(updates)
	if result.Error != nil {
		return fmt.Errorf("update execution status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update execution: %w", ErrStaleMutation)
	}

	return nil
}

func lockInstanceForMutation(
	tx *gorm.DB,
	instanceID string,
	currentState string,
	revision int64,
) (*models.WorkflowInstance, error) {
	var instance models.WorkflowInstance
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND current_state = ? AND revision = ? AND status = ? AND deleted_at IS NULL",
			instanceID,
			currentState,
			revision,
			models.InstanceStatusRunning,
		).
		First(&instance)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lock instance: %w", ErrStaleMutation)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("lock workflow instance: %w", result.Error)
	}

	return &instance, nil
}

func transitionInstance(
	tx *gorm.DB,
	instanceID string,
	nextState string,
) error {
	now := time.Now()
	result := tx.Model(&models.WorkflowInstance{}).
		Where("id = ? AND deleted_at IS NULL", instanceID).
		UpdateColumns(map[string]any{
			"current_state": nextState,
			"revision":      gorm.Expr("revision + 1"),
			"modified_at":   now,
		})
	if result.Error != nil {
		return fmt.Errorf("transition workflow instance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("transition instance: %w", ErrStaleMutation)
	}

	return nil
}

func completeInstance(tx *gorm.DB, instanceID string) error {
	now := time.Now()
	result := tx.Model(&models.WorkflowInstance{}).
		Where("id = ? AND deleted_at IS NULL", instanceID).
		UpdateColumns(map[string]any{
			"status":      models.InstanceStatusCompleted,
			"revision":    gorm.Expr("revision + 1"),
			"modified_at": now,
			"finished_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("complete workflow instance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("complete instance: %w", ErrStaleMutation)
	}

	return nil
}

var ErrStaleMutation = errors.New("stale workflow runtime mutation")
var ErrInvalidExecutionToken = errors.New("invalid execution token or execution not in dispatched state")

func computeOutputSchemaHash(payload string) string {
	return hashString(payload)
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
