package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// FormSubmissionRepository manages form submission persistence.
type FormSubmissionRepository interface {
	Create(ctx context.Context, sub *models.FormSubmission) error
	GetByID(ctx context.Context, id string) (*models.FormSubmission, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*models.FormSubmission, error)
	ListByFormID(ctx context.Context, formID string, limit, offset int) ([]*models.FormSubmission, error)
	Update(ctx context.Context, sub *models.FormSubmission) error
	SoftDelete(ctx context.Context, sub *models.FormSubmission) error
}

type formSubmissionRepository struct {
	datastore.BaseRepository[*models.FormSubmission]
}

// NewFormSubmissionRepository creates a new FormSubmissionRepository.
func NewFormSubmissionRepository(dbPool pool.Pool) FormSubmissionRepository {
	ctx := context.Background()

	return &formSubmissionRepository{
		BaseRepository: datastore.NewBaseRepository[*models.FormSubmission](
			ctx,
			dbPool,
			nil,
			func() *models.FormSubmission { return &models.FormSubmission{} },
		),
	}
}

func (r *formSubmissionRepository) Create(ctx context.Context, sub *models.FormSubmission) error {
	return r.BaseRepository.Create(ctx, sub)
}

func (r *formSubmissionRepository) GetByID(ctx context.Context, id string) (*models.FormSubmission, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *formSubmissionRepository) FindByIdempotencyKey(
	ctx context.Context,
	key string,
) (*models.FormSubmission, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var sub models.FormSubmission

	result := db.Where(
		"idempotency_key = ? AND deleted_at IS NULL",
		key,
	).First(&sub)

	if result.Error != nil {
		return nil, fmt.Errorf("find submission by idempotency key: %w", result.Error)
	}

	return &sub, nil
}

func (r *formSubmissionRepository) ListByFormID(
	ctx context.Context,
	formID string,
	limit, offset int,
) ([]*models.FormSubmission, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 20
	}

	if limit > maxListLimit {
		limit = maxListLimit
	}

	if offset < 0 {
		offset = 0
	}

	var subs []*models.FormSubmission

	result := db.Where(
		"form_id = ? AND deleted_at IS NULL",
		formID,
	).Order("created_at DESC").Limit(limit).Offset(offset).Find(&subs)

	if result.Error != nil {
		return nil, fmt.Errorf("list submissions by form_id: %w", result.Error)
	}

	return subs, nil
}

func (r *formSubmissionRepository) Update(ctx context.Context, sub *models.FormSubmission) error {
	_, err := r.BaseRepository.Update(ctx, sub)
	return err
}

func (r *formSubmissionRepository) SoftDelete(ctx context.Context, sub *models.FormSubmission) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Delete(sub)
	if result.Error != nil {
		return fmt.Errorf("soft delete form submission: %w", result.Error)
	}

	return nil
}
