package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// RetryPolicyRepository manages retry policy persistence.
type RetryPolicyRepository interface {
	Store(ctx context.Context, policy *models.WorkflowRetryPolicy) error
	Lookup(
		ctx context.Context,
		tenantID, workflowName string,
		version int,
		state string,
	) (*models.WorkflowRetryPolicy, error)
}

type retryPolicyRepository struct {
	datastore.BaseRepository[*models.WorkflowRetryPolicy]
}

// NewRetryPolicyRepository creates a new RetryPolicyRepository.
func NewRetryPolicyRepository(dbPool pool.Pool) RetryPolicyRepository {
	ctx := context.Background()

	return &retryPolicyRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowRetryPolicy](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowRetryPolicy { return &models.WorkflowRetryPolicy{} },
		),
	}
}

func (r *retryPolicyRepository) Store(ctx context.Context, policy *models.WorkflowRetryPolicy) error {
	return r.BaseRepository.Create(ctx, policy)
}

func (r *retryPolicyRepository) Lookup(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
) (*models.WorkflowRetryPolicy, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var policy models.WorkflowRetryPolicy

	result := db.Where(
		"tenant_id = ? AND workflow_name = ? AND workflow_version = ? AND state = ? AND deleted_at IS NULL",
		tenantID, workflowName, version, state,
	).First(&policy)

	if result.Error != nil {
		return nil, fmt.Errorf("lookup retry policy: %w", result.Error)
	}

	return &policy, nil
}
