//go:build ignore
// +build ignore

package business

import (
	"context"
	"fmt"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
)

type InstancesQuery struct {
	Status        string
	WorkflowName  string
	Limit         int
	CreatedBefore *time.Time
}

type ExecutionsQuery struct {
	InstanceID    string
	Status        string
	Limit         int
	CreatedBefore *time.Time
}

type ExecutionDetail struct {
	Execution *models.WorkflowStateExecution
	Output    *models.WorkflowStateOutput
}

// ObservabilityBusiness exposes read and control operations for UI and operators.
type ObservabilityBusiness interface {
	ListInstances(ctx context.Context, tenantID string, q InstancesQuery) ([]*models.WorkflowInstance, error)
	GetInstance(ctx context.Context, tenantID, instanceID string) (*models.WorkflowInstance, error)
	ListExecutions(ctx context.Context, tenantID string, q ExecutionsQuery) ([]*models.WorkflowStateExecution, error)
	GetExecution(ctx context.Context, tenantID, executionID string, includeOutput bool) (*ExecutionDetail, error)
	RetryExecution(ctx context.Context, tenantID, executionID string) (*models.WorkflowStateExecution, error)
	RetryInstanceLastFailure(ctx context.Context, tenantID, instanceID string) (*models.WorkflowStateExecution, error)
}

type observabilityBusiness struct {
	instanceRepo repository.WorkflowInstanceRepository
	execRepo     repository.WorkflowExecutionRepository
	outputRepo   repository.WorkflowOutputRepository
	auditRepo    repository.AuditEventRepository
	metrics      *telemetry.Metrics
}

func NewObservabilityBusiness(
	instanceRepo repository.WorkflowInstanceRepository,
	execRepo repository.WorkflowExecutionRepository,
	outputRepo repository.WorkflowOutputRepository,
	auditRepo repository.AuditEventRepository,
	metrics *telemetry.Metrics,
) ObservabilityBusiness {
	return &observabilityBusiness{
		instanceRepo: instanceRepo,
		execRepo:     execRepo,
		outputRepo:   outputRepo,
		auditRepo:    auditRepo,
		metrics:      metrics,
	}
}

func (b *observabilityBusiness) ListInstances(
	ctx context.Context,
	tenantID string,
	q InstancesQuery,
) ([]*models.WorkflowInstance, error) {
	return b.instanceRepo.ListByTenant(ctx, tenantID, q.Status, q.WorkflowName, q.Limit, q.CreatedBefore)
}

func (b *observabilityBusiness) GetInstance(
	ctx context.Context,
	tenantID, instanceID string,
) (*models.WorkflowInstance, error) {
	inst, err := b.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstanceNotFound, err)
	}
	if inst.TenantID != tenantID {
		return nil, fmt.Errorf("%w: tenant mismatch", ErrInstanceNotFound)
	}
	return inst, nil
}

func (b *observabilityBusiness) ListExecutions(
	ctx context.Context,
	tenantID string,
	q ExecutionsQuery,
) ([]*models.WorkflowStateExecution, error) {
	if q.InstanceID != "" {
		return b.execRepo.ListByInstance(ctx, tenantID, q.InstanceID, q.Limit, q.CreatedBefore)
	}

	return b.execRepo.ListByTenant(ctx, tenantID, q.Status, q.Limit, q.CreatedBefore)
}

func (b *observabilityBusiness) GetExecution(
	ctx context.Context,
	tenantID, executionID string,
	includeOutput bool,
) (*ExecutionDetail, error) {
	exec, err := b.execRepo.GetByIDForTenant(ctx, tenantID, executionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}

	detail := &ExecutionDetail{Execution: exec}
	if includeOutput {
		output, outputErr := b.outputRepo.GetByExecution(ctx, tenantID, executionID)
		if outputErr == nil {
			detail.Output = output
		}
	}

	return detail, nil
}

func (b *observabilityBusiness) RetryExecution(
	ctx context.Context,
	tenantID, executionID string,
) (*models.WorkflowStateExecution, error) {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, "execution.retry",
		attribute.String(telemetry.AttrTenantID, tenantID),
		attribute.String("execution_id", executionID),
	)
	defer func() { telemetry.EndSpan(span, nil) }()

	exec, err := b.execRepo.GetByIDForTenant(ctx, tenantID, executionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}

	if !isRetryableStatus(exec.Status) {
		return nil, ErrInvalidRetry
	}

	instance, instErr := b.instanceRepo.GetByID(ctx, exec.InstanceID)
	if instErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstanceNotFound, instErr)
	}
	if instance.TenantID != tenantID {
		return nil, fmt.Errorf("%w: tenant mismatch", ErrInstanceNotFound)
	}

	if instance.Status == models.InstanceStatusFailed || instance.Status == models.InstanceStatusCancelled {
		_ = b.instanceRepo.UpdateStatus(ctx, instance.ID, tenantID, models.InstanceStatusRunning)
	}

	return b.createManualRetry(ctx, exec)
}

func (b *observabilityBusiness) RetryInstanceLastFailure(
	ctx context.Context,
	tenantID, instanceID string,
) (*models.WorkflowStateExecution, error) {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, "instance.retry",
		attribute.String(telemetry.AttrTenantID, tenantID),
		attribute.String("instance_id", instanceID),
	)
	defer func() { telemetry.EndSpan(span, nil) }()

	instance, err := b.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstanceNotFound, err)
	}
	if instance.TenantID != tenantID {
		return nil, fmt.Errorf("%w: tenant mismatch", ErrInstanceNotFound)
	}

	exec, execErr := b.execRepo.GetLatestFailureByInstance(ctx, tenantID, instanceID)
	if execErr != nil {
		return nil, ErrInvalidRetry
	}

	if instance.Status == models.InstanceStatusFailed || instance.Status == models.InstanceStatusCancelled {
		_ = b.instanceRepo.UpdateStatus(ctx, instance.ID, tenantID, models.InstanceStatusRunning)
	}

	return b.createManualRetry(ctx, exec)
}

func (b *observabilityBusiness) createManualRetry(
	ctx context.Context,
	exec *models.WorkflowStateExecution,
) (*models.WorkflowStateExecution, error) {
	active, activeErr := b.execRepo.HasActiveForInstanceState(ctx, exec.TenantID, exec.InstanceID, exec.State)
	if activeErr != nil {
		return nil, activeErr
	}
	if active {
		return nil, ErrInvalidRetry
	}

	rawToken, tokenErr := cryptoutil.GenerateToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("generate token: %w", tokenErr)
	}

	newExec := &models.WorkflowStateExecution{
		ExecutionID:     util.IDString(),
		TenantID:        exec.TenantID,
		PartitionID:     exec.PartitionID,
		InstanceID:      exec.InstanceID,
		State:           exec.State,
		StateVersion:    exec.StateVersion,
		Attempt:         exec.Attempt + 1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  cryptoutil.HashToken(rawToken),
		InputSchemaHash: exec.InputSchemaHash,
		InputPayload:    exec.InputPayload,
		TraceID:         exec.TraceID,
	}

	if createErr := b.execRepo.Create(ctx, newExec); createErr != nil {
		return nil, fmt.Errorf("create manual retry execution: %w", createErr)
	}

	_ = b.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		ID:          util.IDString(),
		TenantID:    exec.TenantID,
		PartitionID: exec.PartitionID,
		InstanceID:  exec.InstanceID,
		ExecutionID: newExec.ExecutionID,
		EventType:   events.EventStateRetried,
		State:       exec.State,
	})

	return newExec, nil
}

func isRetryableStatus(status models.ExecutionStatus) bool {
	switch status {
	case models.ExecStatusFailed,
		models.ExecStatusFatal,
		models.ExecStatusTimedOut,
		models.ExecStatusInvalidInputContract,
		models.ExecStatusInvalidOutputContract:
		return true
	default:
		return false
	}
}
