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
		dslBlob json.RawMessage,
	) (*models.WorkflowDefinition, error)
	GetWorkflow(ctx context.Context, id string) (*models.WorkflowDefinition, error)
	ListWorkflows(ctx context.Context, name string, limit int) ([]*models.WorkflowDefinition, error)
	SearchWorkflows(ctx context.Context, filter WorkflowListFilter) (*WorkflowListPage, error)
	ActivateWorkflow(ctx context.Context, id string) error
}

type WorkflowListFilter struct {
	Name    string
	Query   string
	IDQuery string
	Cursor  string
	Limit   int
}

type WorkflowListPage struct {
	Items      []*models.WorkflowDefinition
	NextCursor string
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

	if execErr := validateExecutableWorkflow(spec); execErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, execErr)
	}

	def := &models.WorkflowDefinition{
		Name:            spec.Name,
		WorkflowVersion: 1,
		Status:          models.WorkflowStatusDraft,
		DSLBlob:         string(dslBlob),
	}

	if spec.Timeout.Duration > 0 {
		def.TimeoutSeconds = int64(spec.Timeout.Duration.Seconds())
	}

	// Register schemas for each step that has a call action.
	if regErr := b.registerStepSchemas(ctx, spec); regErr != nil {
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

func validateExecutableWorkflow(spec *dsl.WorkflowSpec) error {
	for _, step := range dsl.CollectAllSteps(spec) {
		switch step.Type {
		case dsl.StepTypeCall,
			dsl.StepTypeSequence,
			dsl.StepTypeIf,
			dsl.StepTypeDelay,
			dsl.StepTypeParallel,
			dsl.StepTypeForeach,
			dsl.StepTypeSignalWait,
			dsl.StepTypeSignalSend:
			continue
		default:
			return fmt.Errorf(
				"step %q uses unsupported runtime step type %q",
				step.ID,
				step.Type,
			)
		}
	}

	return nil
}

// registerStepSchemas iterates all steps in the DSL and registers input/output schemas
// for call steps that define them via their adapter schemas.
func (b *workflowBusiness) registerStepSchemas(
	ctx context.Context,
	spec *dsl.WorkflowSpec,
) error {
	for _, step := range dsl.CollectAllSteps(spec) {
		inputSchema, outputSchema, errorSchema, ok := defaultSchemasForStep(step)
		if !ok {
			continue
		}

		if _, err := b.schemaReg.RegisterSchema(
			ctx, spec.Name, 1, step.ID,
			models.SchemaTypeInput, inputSchema,
		); err != nil {
			return fmt.Errorf("register input schema for step %s: %w", step.ID, err)
		}

		if _, err := b.schemaReg.RegisterSchema(
			ctx, spec.Name, 1, step.ID,
			models.SchemaTypeOutput, outputSchema,
		); err != nil {
			return fmt.Errorf("register output schema for step %s: %w", step.ID, err)
		}

		if _, err := b.schemaReg.RegisterSchema(
			ctx, spec.Name, 1, step.ID,
			models.SchemaTypeError, errorSchema,
		); err != nil {
			return fmt.Errorf("register error schema for step %s: %w", step.ID, err)
		}
	}

	return nil
}

func defaultSchemasForStep(step *dsl.StepSpec) (json.RawMessage, json.RawMessage, json.RawMessage, bool) {
	objectSchema := json.RawMessage(`{"type":"object"}`)
	errorSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"class": {"type": "string"},
			"code": {"type": "string"},
			"message": {"type": "string"}
		},
		"required": ["class", "code", "message"]
	}`)

	switch step.Type {
	case dsl.StepTypeCall, dsl.StepTypeDelay, dsl.StepTypeSequence, dsl.StepTypeSignalSend:
		return objectSchema, objectSchema, errorSchema, true
	case dsl.StepTypeParallel:
		return objectSchema, json.RawMessage(`{
			"type":"object",
			"properties":{
				"branches":{"type":"array"}
			},
			"required":["branches"]
		}`), errorSchema, true
	case dsl.StepTypeForeach:
		return objectSchema, json.RawMessage(`{
			"type":"object",
			"properties":{
				"items":{"type":"array"}
			},
			"required":["items"]
		}`), errorSchema, true
	case dsl.StepTypeSignalWait:
		return objectSchema, objectSchema, errorSchema, true
	case dsl.StepTypeIf:
		return objectSchema, json.RawMessage(`{
			"type":"object",
			"properties":{
				"branch":{"type":"string","enum":["then","else"]}
			},
			"required":["branch"]
		}`), errorSchema, true
	default:
		return nil, nil, nil, false
	}
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
	name string,
	limit int,
) ([]*models.WorkflowDefinition, error) {
	return b.defRepo.ListActiveByName(ctx, name, limit)
}

func (b *workflowBusiness) SearchWorkflows(
	ctx context.Context,
	filter WorkflowListFilter,
) (*WorkflowListPage, error) {
	page, err := b.defRepo.ListPage(ctx, repository.WorkflowDefinitionListFilter{
		Name:    filter.Name,
		Query:   filter.Query,
		IDQuery: filter.IDQuery,
		Cursor:  filter.Cursor,
		Limit:   filter.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &WorkflowListPage{
		Items:      page.Items,
		NextCursor: page.NextCursor,
	}, nil
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
