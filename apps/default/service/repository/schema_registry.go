package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// SchemaRegistryRepository manages immutable schema storage.
type SchemaRegistryRepository interface {
	Store(ctx context.Context, schema *models.WorkflowStateSchema) error
	Lookup(
		ctx context.Context,
		workflowName string,
		version int,
		state string,
		schemaType models.SchemaType,
	) (*models.WorkflowStateSchema, error)
	LookupByHash(ctx context.Context, hash string) (*models.WorkflowStateSchema, error)
}

type schemaRegistryRepository struct {
	datastore.BaseRepository[*models.WorkflowStateSchema]
}

// NewSchemaRegistryRepository creates a new SchemaRegistryRepository.
func NewSchemaRegistryRepository(dbPool pool.Pool) SchemaRegistryRepository {
	ctx := context.Background()
	return &schemaRegistryRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowStateSchema](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowStateSchema { return &models.WorkflowStateSchema{} },
		),
	}
}

// Store upserts a schema (immutable by hash — if the hash already exists, it's a no-op).
func (r *schemaRegistryRepository) Store(ctx context.Context, schema *models.WorkflowStateSchema) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "workflow_name"},
			{Name: "workflow_version"},
			{Name: "state"},
			{Name: "schema_type"},
		},
		DoNothing: true,
	}).Create(schema)

	if result.Error != nil {
		return fmt.Errorf("store schema: %w", result.Error)
	}

	return nil
}

func (r *schemaRegistryRepository) Lookup(
	ctx context.Context,
	workflowName string,
	version int,
	state string,
	schemaType models.SchemaType,
) (*models.WorkflowStateSchema, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where(
		"workflow_name = ? AND workflow_version = ? AND state = ? AND schema_type = ?",
		workflowName, version, state, schemaType,
	).First(&schema)

	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema: %w", result.Error)
	}

	return &schema, nil
}

func (r *schemaRegistryRepository) LookupByHash(
	ctx context.Context,
	hash string,
) (*models.WorkflowStateSchema, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where("schema_hash = ?", hash).First(&schema)
	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema by hash: %w", result.Error)
	}

	return &schema, nil
}
