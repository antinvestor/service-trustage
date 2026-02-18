package business

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	framecache "github.com/pitabwire/frame/cache"
	"github.com/pitabwire/util"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cacheutil"
)

// Maximum number of compiled JSON schemas to cache in-process.
const maxSchemaCacheSize = 1000

// schemaBlobCacheTTL is the TTL for schema blobs cached in Valkey.
const schemaBlobCacheTTL = 10 * time.Minute

// schemaCache is a bounded in-process cache for compiled JSON schemas.
// Compiled schemas cannot be serialized, so they stay in-process only.
//
//nolint:gochecknoglobals // schema cache
var schemaCache = cacheutil.NewBoundedCache[*jsonschema.Schema](maxSchemaCacheSize)

// SchemaRegistry validates data against registered JSON Schemas.
type SchemaRegistry interface {
	RegisterSchema(
		ctx context.Context,
		tenantID, partitionID, workflowName string,
		version int,
		state string,
		schemaType models.SchemaType,
		schemaBlob json.RawMessage,
	) (string, error)
	ValidateInput(
		ctx context.Context,
		tenantID, workflowName string,
		version int,
		state string,
		data json.RawMessage,
	) (string, error)
	ValidateOutput(
		ctx context.Context,
		tenantID, workflowName string,
		version int,
		state string,
		data json.RawMessage,
	) error
	ValidateError(
		ctx context.Context,
		tenantID, workflowName string,
		version int,
		state string,
		data json.RawMessage,
	) error
}

type schemaRegistry struct {
	repo  repository.SchemaRegistryRepository
	cache framecache.RawCache
}

// NewSchemaRegistry creates a new SchemaRegistry.
// The cache parameter is used for Valkey-backed schema blob caching.
// If nil, schema blobs are loaded from the database on every call.
func NewSchemaRegistry(repo repository.SchemaRegistryRepository, cache framecache.RawCache) SchemaRegistry {
	return &schemaRegistry{repo: repo, cache: cache}
}

func (sr *schemaRegistry) RegisterSchema(
	ctx context.Context,
	tenantID, partitionID, workflowName string,
	version int,
	state string,
	schemaType models.SchemaType,
	schemaBlob json.RawMessage,
) (string, error) {
	hash := computeSchemaHash(schemaBlob)

	schema := &models.WorkflowStateSchema{
		ID:              util.IDString(),
		TenantID:        tenantID,
		PartitionID:     partitionID,
		WorkflowName:    workflowName,
		WorkflowVersion: version,
		State:           state,
		SchemaType:      schemaType,
		SchemaHash:      hash,
		SchemaBlob:      string(schemaBlob),
	}

	if err := sr.repo.Store(ctx, schema); err != nil {
		return "", fmt.Errorf("register schema: %w", err)
	}

	return hash, nil
}

func (sr *schemaRegistry) ValidateInput(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
	data json.RawMessage,
) (string, error) {
	schema, err := sr.lookupSchema(ctx, tenantID, workflowName, version, state, models.SchemaTypeInput)
	if err != nil {
		return "", fmt.Errorf("%w: input schema for %s/%s: %w", ErrSchemaNotFound, workflowName, state, err)
	}

	if validateErr := validateAgainstSchema(schema.SchemaBlob, data); validateErr != nil {
		return "", fmt.Errorf("%w: %w", ErrInputContractViolation, validateErr)
	}

	return schema.SchemaHash, nil
}

func (sr *schemaRegistry) ValidateOutput(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
	data json.RawMessage,
) error {
	schema, err := sr.lookupSchema(ctx, tenantID, workflowName, version, state, models.SchemaTypeOutput)
	if err != nil {
		return fmt.Errorf("%w: output schema for %s/%s: %w", ErrSchemaNotFound, workflowName, state, err)
	}

	if validateErr := validateAgainstSchema(schema.SchemaBlob, data); validateErr != nil {
		return fmt.Errorf("%w: %w", ErrOutputContractViolation, validateErr)
	}

	return nil
}

func (sr *schemaRegistry) ValidateError(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
	data json.RawMessage,
) error {
	schema, err := sr.lookupSchema(ctx, tenantID, workflowName, version, state, models.SchemaTypeError)
	if err != nil {
		// Error schema is optional — if not registered, skip validation.
		return nil //nolint:nilerr // missing error schema is not a validation failure
	}

	if validateErr := validateAgainstSchema(schema.SchemaBlob, data); validateErr != nil {
		return fmt.Errorf("error contract violation: %w", validateErr)
	}

	return nil
}

// schemaCacheKey returns a Valkey-safe cache key for a schema lookup.
func schemaCacheKey(tenantID, workflowName string, version int, state string, schemaType models.SchemaType) string {
	return fmt.Sprintf("schema:%s:%s:%d:%s:%s", tenantID, workflowName, version, state, schemaType)
}

// cachedSchema is a JSON-serializable subset of WorkflowStateSchema for Valkey caching.
type cachedSchema struct {
	SchemaHash string `json:"h"`
	SchemaBlob string `json:"b"`
}

// lookupSchema fetches a schema, checking Valkey first (if configured) before falling back to the database.
func (sr *schemaRegistry) lookupSchema(
	ctx context.Context,
	tenantID, workflowName string,
	version int,
	state string,
	schemaType models.SchemaType,
) (*models.WorkflowStateSchema, error) {
	key := schemaCacheKey(tenantID, workflowName, version, state, schemaType)

	// Try Valkey cache first.
	if sr.cache != nil {
		data, found, cacheErr := sr.cache.Get(ctx, key)
		if cacheErr == nil && found {
			var cached cachedSchema
			if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr == nil {
				return &models.WorkflowStateSchema{
					SchemaHash: cached.SchemaHash,
					SchemaBlob: cached.SchemaBlob,
				}, nil
			}
		}
	}

	// Fall back to database.
	schema, err := sr.repo.Lookup(ctx, tenantID, workflowName, version, state, schemaType)
	if err != nil {
		return nil, err
	}

	// Store in Valkey cache (best-effort).
	if sr.cache != nil {
		entry := cachedSchema{SchemaHash: schema.SchemaHash, SchemaBlob: schema.SchemaBlob}
		if blob, marshalErr := json.Marshal(entry); marshalErr == nil {
			_ = sr.cache.Set(ctx, key, blob, schemaBlobCacheTTL)
		}
	}

	return schema, nil
}

func computeSchemaHash(blob json.RawMessage) string {
	hash := sha256.Sum256(blob)
	return hex.EncodeToString(hash[:])
}

func validateAgainstSchema(schemaBlob string, data json.RawMessage) error {
	// Use schema hash as cache key.
	hash := sha256.Sum256([]byte(schemaBlob))
	cacheKey := hex.EncodeToString(hash[:])

	// Check bounded cache.
	compiledSchema, cached := schemaCache.Get(cacheKey)

	if !cached {
		compiler := jsonschema.NewCompiler()

		if err := compiler.AddResource("schema.json", strings.NewReader(schemaBlob)); err != nil {
			return fmt.Errorf("add schema resource: %w", err)
		}

		var compileErr error
		compiledSchema, compileErr = compiler.Compile("schema.json")
		if compileErr != nil {
			return fmt.Errorf("compile schema: %w", compileErr)
		}

		schemaCache.Put(cacheKey, compiledSchema)
	}

	var v any
	if unmarshalErr := json.Unmarshal(data, &v); unmarshalErr != nil {
		return fmt.Errorf("unmarshal data: %w", unmarshalErr)
	}

	if validateErr := compiledSchema.Validate(v); validateErr != nil {
		return fmt.Errorf("validation: %w", validateErr)
	}

	return nil
}
