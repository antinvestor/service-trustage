package business

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
)

// WorkflowBusiness manages workflow definition lifecycle.
type WorkflowBusiness interface {
	CreateWorkflow(
		ctx context.Context,
		tenantID, partitionID string,
		dslBlob json.RawMessage,
	) (*models.WorkflowDefinition, error)
	GetWorkflow(ctx context.Context, id string) (*models.WorkflowDefinition, error)
	ListWorkflows(ctx context.Context, tenantID, name string) ([]*models.WorkflowDefinition, error)
	ActivateWorkflow(ctx context.Context, id string) error
}

type workflowBusiness struct {
	defRepo   repository.WorkflowDefinitionRepository
	schemaReg SchemaRegistry
}

// NewWorkflowBusiness creates a new WorkflowBusiness.
func NewWorkflowBusiness(
	defRepo repository.WorkflowDefinitionRepository,
	schemaReg SchemaRegistry,
) WorkflowBusiness {
	return &workflowBusiness{
		defRepo:   defRepo,
		schemaReg: schemaReg,
	}
}

// CreateWorkflow parses and validates a DSL document, registers schemas, and persists the definition.
func (b *workflowBusiness) CreateWorkflow(
	ctx context.Context,
	tenantID, partitionID string,
	dslBlob json.RawMessage,
) (*models.WorkflowDefinition, error) {
	log := util.Log(ctx)

	// Parse DSL.
	spec, err := dsl.Parse(dslBlob)
	if err != nil {
		return nil, fmt.Errorf("parse DSL: %w", err)
	}

	// Validate DSL.
	result := dsl.Validate(spec)
	if !result.Valid() {
		log.Error("DSL validation failed",
			"errors", len(result.Errors),
			"workflow", spec.Name,
		)

		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, result.Error())
	}

	def := &models.WorkflowDefinition{
		Name:            spec.Name,
		WorkflowVersion: 1,
		Status:          models.WorkflowStatusDraft,
		DSLBlob:         string(dslBlob),
	}

	def.TenantID = tenantID
	def.PartitionID = partitionID

	if spec.Timeout.Duration > 0 {
		def.TimeoutSeconds = int64(spec.Timeout.Duration.Seconds())
	}

	// Register schemas for each step that has a call action.
	if regErr := b.registerStepSchemas(ctx, tenantID, partitionID, spec); regErr != nil {
		return nil, fmt.Errorf("register schemas: %w", regErr)
	}

	if err = b.defRepo.Create(ctx, def); err != nil {
		return nil, fmt.Errorf("persist workflow: %w", err)
	}

	log.Info("workflow created",
		"workflow_id", def.ID,
		"name", spec.Name,
	)

	return def, nil
}

// registerStepSchemas iterates all steps in the DSL and registers input/output schemas
// for call steps that define them via their adapter schemas.
func (b *workflowBusiness) registerStepSchemas(
	ctx context.Context,
	tenantID, partitionID string,
	spec *dsl.WorkflowSpec,
) error {
	for _, step := range dsl.CollectAllSteps(spec) {
		if step.Type != dsl.StepTypeCall || step.Call == nil {
			continue
		}

		// Register a default input schema for each call step.
		inputSchema := json.RawMessage(`{"type": "object"}`)
		if _, err := b.schemaReg.RegisterSchema(
			ctx, tenantID, partitionID, spec.Name, 1, step.ID,
			models.SchemaTypeInput, inputSchema,
		); err != nil {
			return fmt.Errorf("register input schema for step %s: %w", step.ID, err)
		}

		// Register a default output schema for each call step.
		outputSchema := json.RawMessage(`{"type": "object"}`)
		if _, err := b.schemaReg.RegisterSchema(
			ctx, tenantID, partitionID, spec.Name, 1, step.ID,
			models.SchemaTypeOutput, outputSchema,
		); err != nil {
			return fmt.Errorf("register output schema for step %s: %w", step.ID, err)
		}

		// Register a default error schema for each call step (ARCHITECTURE.md §4.2).
		errorSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"class": {"type": "string"},
				"code": {"type": "string"},
				"message": {"type": "string"}
			},
			"required": ["class", "code", "message"]
		}`)
		if _, err := b.schemaReg.RegisterSchema(
			ctx, tenantID, partitionID, spec.Name, 1, step.ID,
			models.SchemaTypeError, errorSchema,
		); err != nil {
			return fmt.Errorf("register error schema for step %s: %w", step.ID, err)
		}
	}

	return nil
}

func (b *workflowBusiness) GetWorkflow(ctx context.Context, id string) (*models.WorkflowDefinition, error) {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	return def, nil
}

func (b *workflowBusiness) ListWorkflows(
	ctx context.Context,
	tenantID, name string,
) ([]*models.WorkflowDefinition, error) {
	return b.defRepo.ListActiveByName(ctx, tenantID, name)
}

func (b *workflowBusiness) ActivateWorkflow(ctx context.Context, id string) error {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	if err = def.TransitionTo(models.WorkflowStatusActive); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	if err = b.defRepo.Update(ctx, def); err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}

	return nil
}
