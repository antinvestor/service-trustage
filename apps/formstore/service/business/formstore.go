package business

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

// FormStoreBusiness manages form definitions and submissions.
type FormStoreBusiness interface {
	// Form definitions.
	CreateDefinition(ctx context.Context, def *models.FormDefinition) error
	GetDefinition(ctx context.Context, id string) (*models.FormDefinition, error)
	GetDefinitionByFormID(ctx context.Context, formID string) (*models.FormDefinition, error)
	ListDefinitions(ctx context.Context, activeOnly bool, limit, offset int) ([]*models.FormDefinition, error)
	UpdateDefinition(ctx context.Context, def *models.FormDefinition) error
	DeleteDefinition(ctx context.Context, id string) error

	// Form submissions.
	CreateSubmission(ctx context.Context, sub *models.FormSubmission) error
	GetSubmission(ctx context.Context, id string) (*models.FormSubmission, error)
	ListSubmissions(ctx context.Context, formID string, limit, offset int) ([]*models.FormSubmission, error)
	UpdateSubmission(ctx context.Context, sub *models.FormSubmission) error
	DeleteSubmission(ctx context.Context, id string) error
}

type formStoreBusiness struct {
	defRepo   repository.FormDefinitionRepository
	subRepo   repository.FormSubmissionRepository
	uploader  *FileUploader
	validator *SchemaValidator
}

// NewFormStoreBusiness creates a new FormStoreBusiness.
func NewFormStoreBusiness(
	defRepo repository.FormDefinitionRepository,
	subRepo repository.FormSubmissionRepository,
	uploader *FileUploader,
) FormStoreBusiness {
	return &formStoreBusiness{
		defRepo:   defRepo,
		subRepo:   subRepo,
		uploader:  uploader,
		validator: NewSchemaValidator(),
	}
}

func (b *formStoreBusiness) CreateDefinition(ctx context.Context, def *models.FormDefinition) error {
	log := util.Log(ctx)

	// Validate schema is parseable if provided.
	if err := b.validator.ValidateSchema(def.JSONSchema); err != nil {
		return err
	}

	// Ensure JSONB fields contain valid JSON for PostgreSQL.
	if def.JSONSchema == "" {
		def.JSONSchema = "{}"
	}

	if err := b.defRepo.Create(ctx, def); err != nil {
		return fmt.Errorf("persist form definition: %w", err)
	}

	definitionCreateCounter.Add(ctx, 1)

	log.Info("form definition created",
		"form_id", def.FormID,
		"name", def.Name,
	)

	return nil
}

func (b *formStoreBusiness) GetDefinition(ctx context.Context, id string) (*models.FormDefinition, error) {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFormDefinitionNotFound, err)
	}

	return def, nil
}

func (b *formStoreBusiness) GetDefinitionByFormID(ctx context.Context, formID string) (*models.FormDefinition, error) {
	def, err := b.defRepo.GetByFormID(ctx, formID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFormDefinitionNotFound, err)
	}

	return def, nil
}

func (b *formStoreBusiness) ListDefinitions(
	ctx context.Context,
	activeOnly bool,
	limit, offset int,
) ([]*models.FormDefinition, error) {
	return b.defRepo.List(ctx, activeOnly, limit, offset)
}

func (b *formStoreBusiness) UpdateDefinition(ctx context.Context, def *models.FormDefinition) error {
	// Validate schema is parseable if provided.
	if err := b.validator.ValidateSchema(def.JSONSchema); err != nil {
		return err
	}

	if err := b.defRepo.Update(ctx, def); err != nil {
		return fmt.Errorf("update form definition: %w", err)
	}

	return nil
}

func (b *formStoreBusiness) DeleteDefinition(ctx context.Context, id string) error {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFormDefinitionNotFound, err)
	}

	if deleteErr := b.defRepo.SoftDelete(ctx, def); deleteErr != nil {
		return fmt.Errorf("delete form definition: %w", deleteErr)
	}

	return nil
}

//nolint:gocognit // submission creation handles validation, idempotency, and uploads
func (b *formStoreBusiness) CreateSubmission(ctx context.Context, sub *models.FormSubmission) error {
	log := util.Log(ctx)
	start := time.Now()
	attrs := metric.WithAttributes(attribute.String("tenant_id", tenantFromContext(ctx)))

	// Ensure JSONB fields contain valid JSON for PostgreSQL.
	if sub.Metadata == "" {
		sub.Metadata = "{}"
	}

	// Validate submission data against form definition schema if one exists.
	if sub.FormID != "" {
		def, defErr := b.defRepo.GetByFormID(ctx, sub.FormID)
		if defErr == nil && def != nil && def.JSONSchema != "" && def.JSONSchema != "{}" {
			schemaValidationCounter.Add(ctx, 1, attrs)

			if valErr := b.validator.Validate(def.JSONSchema, sub.Data); valErr != nil {
				schemaValidationErrors.Add(ctx, 1, attrs)
				return valErr
			}
		}
	}

	// Check idempotency — fast path for obvious duplicates.
	if sub.IdempotencyKey != "" {
		existing, _ := b.subRepo.FindByIdempotencyKey(ctx, sub.IdempotencyKey)
		if existing != nil {
			*sub = *existing
			return nil
		}
	}

	// Process file fields if uploader is available.
	//nolint:nestif // upload flow is intentionally nested for clarity
	if b.uploader != nil {
		var dataMap map[string]any
		if err := json.Unmarshal([]byte(sub.Data), &dataMap); err == nil {
			processed, fileCount, procErr := b.uploader.ProcessFields(dataMap)
			if procErr != nil {
				return fmt.Errorf("process file fields: %w", procErr)
			}

			sub.FileCount = fileCount

			processedBytes, marshalErr := json.Marshal(processed)
			if marshalErr != nil {
				return fmt.Errorf("marshal processed data: %w", marshalErr)
			}

			sub.Data = string(processedBytes)
		}
	}

	if err := b.subRepo.Create(ctx, sub); err != nil {
		// Handle concurrent idempotency race: if the unique index rejects the insert,
		// load and return the existing record instead of surfacing a DB error.
		if sub.IdempotencyKey != "" {
			existing, findErr := b.subRepo.FindByIdempotencyKey(ctx, sub.IdempotencyKey)
			if findErr == nil && existing != nil {
				*sub = *existing
				return nil
			}
		}

		submissionErrorCounter.Add(ctx, 1, attrs)
		return fmt.Errorf("persist form submission: %w", err)
	}

	submissionCreateCounter.Add(ctx, 1, attrs)
	submissionHistogram.Record(ctx, float64(time.Since(start).Milliseconds()), attrs)

	if sub.FileCount > 0 {
		fileUploadCounter.Add(ctx, int64(sub.FileCount), attrs)
	}

	log.Info("form submission created",
		"submission_id", sub.ID,
		"form_id", sub.FormID,
		"file_count", sub.FileCount,
	)

	return nil
}

func (b *formStoreBusiness) GetSubmission(ctx context.Context, id string) (*models.FormSubmission, error) {
	sub, err := b.subRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFormSubmissionNotFound, err)
	}

	return sub, nil
}

func (b *formStoreBusiness) ListSubmissions(
	ctx context.Context,
	formID string,
	limit, offset int,
) ([]*models.FormSubmission, error) {
	return b.subRepo.ListByFormID(ctx, formID, limit, offset)
}

func (b *formStoreBusiness) UpdateSubmission(ctx context.Context, sub *models.FormSubmission) error {
	// Validate status if set.
	if sub.Status != "" {
		if err := sub.Status.ValidateStatus(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidStatus, err)
		}
	}

	// Re-process file fields on update.
	//nolint:nestif // upload flow is intentionally nested for clarity
	if b.uploader != nil {
		var dataMap map[string]any
		if err := json.Unmarshal([]byte(sub.Data), &dataMap); err == nil {
			processed, fileCount, procErr := b.uploader.ProcessFields(dataMap)
			if procErr != nil {
				return fmt.Errorf("process file fields: %w", procErr)
			}

			sub.FileCount = fileCount

			processedBytes, marshalErr := json.Marshal(processed)
			if marshalErr != nil {
				return fmt.Errorf("marshal processed data: %w", marshalErr)
			}

			sub.Data = string(processedBytes)
		}
	}

	if err := b.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("update form submission: %w", err)
	}

	return nil
}

func tenantFromContext(ctx context.Context) string {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil || claims.GetTenantID() == "" {
		return "unknown"
	}

	return claims.GetTenantID()
}

func (b *formStoreBusiness) DeleteSubmission(ctx context.Context, id string) error {
	sub, err := b.subRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFormSubmissionNotFound, err)
	}

	if deleteErr := b.subRepo.SoftDelete(ctx, sub); deleteErr != nil {
		return fmt.Errorf("delete form submission: %w", deleteErr)
	}

	return nil
}
