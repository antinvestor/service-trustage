package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// SchemaRegistryRepository manages immutable schema storage.
type SchemaRegistryRepository interface {
	Store(ctx context.Context, schema *models.WorkflowStateSchema) error
	Lookup(
		ctx context.Context,
		tenantID, workflowName string,
		version int,
		state string,
		schemaType models.SchemaType,
	) (*models.WorkflowStateSchema, error)
	LookupByHash(ctx context.Context, tenantID, hash string) (*models.WorkflowStateSchema, error)
}

type schemaRegistryRepository struct {
	pool pool.Pool
}

// NewSchemaRegistryRepository creates a new SchemaRegistryRepository.
func NewSchemaRegistryRepository(dbPool pool.Pool) SchemaRegistryRepository {
	return &schemaRegistryRepository{pool: dbPool}
}

// Store upserts a schema (immutable by hash — if the hash already exists, it's a no-op).
func (r *schemaRegistryRepository) Store(ctx context.Context, schema *models.WorkflowStateSchema) error {
	db := r.pool.DB(ctx, false)

	result := db.Exec(
		`INSERT INTO workflow_state_schemas (id, tenant_id, partition_id, workflow_name, workflow_version, state, schema_type, schema_hash, schema_blob, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		 ON CONFLICT (tenant_id, workflow_name, workflow_version, state, schema_type) DO NOTHING`,
		schema.ID,
		schema.TenantID,
		schema.PartitionID,
		schema.WorkflowName,
		schema.WorkflowVersion,
		schema.State,
		schema.SchemaType,
		schema.SchemaHash,
		schema.SchemaBlob,
	)

	if result.Error != nil {
		return fmt.Errorf("store schema: %w", result.Error)
	}

	return nil
}

func (r *schemaRegistryRepository) Lookup(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
	schemaType models.SchemaType,
) (*models.WorkflowStateSchema, error) {
	db := r.pool.DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where(
		"tenant_id = ? AND workflow_name = ? AND workflow_version = ? AND state = ? AND schema_type = ?",
		tenantID, workflowName, version, state, schemaType,
	).First(&schema)

	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema: %w", result.Error)
	}

	return &schema, nil
}

func (r *schemaRegistryRepository) LookupByHash(
	ctx context.Context,
	tenantID, hash string,
) (*models.WorkflowStateSchema, error) {
	db := r.pool.DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where("tenant_id = ? AND schema_hash = ?", tenantID, hash).First(&schema)
	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema by hash: %w", result.Error)
	}

	return &schema, nil
}
