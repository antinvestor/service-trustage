package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowScopeRunRepository manages durable branch scope persistence.
type WorkflowScopeRunRepository interface {
	Create(ctx context.Context, scope *models.WorkflowScopeRun) error
	GetByID(ctx context.Context, id string) (*models.WorkflowScopeRun, error)
	GetByParentExecutionID(ctx context.Context, parentExecutionID string) (*models.WorkflowScopeRun, error)
	ListByInstance(ctx context.Context, parentInstanceID string, limit int) ([]*models.WorkflowScopeRun, error)
	ClaimRunning(ctx context.Context, limit int, owner string, leaseUntil time.Time) ([]*models.WorkflowScopeRun, error)
	ReleaseClaim(ctx context.Context, id, owner string) error
	Pool() pool.Pool
}

type workflowScopeRunRepository struct {
	datastore.BaseRepository[*models.WorkflowScopeRun]
	pool pool.Pool
}

// NewWorkflowScopeRunRepository creates a new WorkflowScopeRunRepository.
func NewWorkflowScopeRunRepository(dbPool pool.Pool) WorkflowScopeRunRepository {
	ctx := context.Background()

	return &workflowScopeRunRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowScopeRun](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowScopeRun { return &models.WorkflowScopeRun{} },
		),
		pool: dbPool,
	}
}

func (r *workflowScopeRunRepository) Pool() pool.Pool {
	return r.pool
}

func (r *workflowScopeRunRepository) Create(ctx context.Context, scope *models.WorkflowScopeRun) error {
	return r.BaseRepository.Create(ctx, scope)
}

func (r *workflowScopeRunRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.WorkflowScopeRun, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *workflowScopeRunRepository) GetByParentExecutionID(
	ctx context.Context,
	parentExecutionID string,
) (*models.WorkflowScopeRun, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var scope models.WorkflowScopeRun
	result := db.Where("parent_execution_id = ? AND deleted_at IS NULL", parentExecutionID).First(&scope)
	if result.Error != nil {
		return nil, fmt.Errorf("get scope by parent execution: %w", result.Error)
	}

	return &scope, nil
}

func (r *workflowScopeRunRepository) ListByInstance(
	ctx context.Context,
	parentInstanceID string,
	limit int,
) ([]*models.WorkflowScopeRun, error) {
	db := r.pool.DB(ctx, true)

	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var scopes []*models.WorkflowScopeRun
	result := db.Where("parent_instance_id = ? AND deleted_at IS NULL", parentInstanceID).
		Order("created_at ASC").
		Limit(limit).
		Find(&scopes)
	if result.Error != nil {
		return nil, fmt.Errorf("list scope runs by instance: %w", result.Error)
	}

	return scopes, nil
}

func (r *workflowScopeRunRepository) ClaimRunning(
	ctx context.Context,
	limit int,
	owner string,
	leaseUntil time.Time,
) ([]*models.WorkflowScopeRun, error) {
	db := r.pool.DB(ctx, false)

	var scopes []*models.WorkflowScopeRun
	txErr := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"status = ? AND deleted_at IS NULL AND (claim_until IS NULL OR claim_until < ?)",
				"running",
				time.Now(),
			).
			Order("created_at ASC").
			Limit(limit).
			Find(&scopes)
		if result.Error != nil {
			return fmt.Errorf("claim running scopes: %w", result.Error)
		}
		if len(scopes) == 0 {
			return nil
		}

		ids := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			ids = append(ids, scope.ID)
		}

		updateResult := tx.Model(&models.WorkflowScopeRun{}).
			Where("id IN ? AND deleted_at IS NULL", ids).
			Updates(map[string]any{
				"claim_owner": owner,
				"claim_until": leaseUntil,
			})
		if updateResult.Error != nil {
			return fmt.Errorf("claim running scopes: %w", updateResult.Error)
		}

		for _, scope := range scopes {
			scope.ClaimOwner = owner
			scope.ClaimUntil = &leaseUntil
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return scopes, nil
}

func (r *workflowScopeRunRepository) ReleaseClaim(ctx context.Context, id, owner string) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowScopeRun{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "running").
		Updates(map[string]any{
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("release scope claim: %w", result.Error)
	}

	return nil
}
