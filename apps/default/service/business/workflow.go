package business

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/pitabwire/util"
	"gorm.io/gorm"

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
	defRepo      repository.WorkflowDefinitionRepository
	scheduleRepo repository.ScheduleRepository
	schemaReg    SchemaRegistry
}

// NewWorkflowBusiness creates a new WorkflowBusiness.
func NewWorkflowBusiness(
	defRepo repository.WorkflowDefinitionRepository,
	scheduleRepo repository.ScheduleRepository,
	schemaReg SchemaRegistry,
) WorkflowBusiness {
	return &workflowBusiness{
		defRepo:      defRepo,
		scheduleRepo: scheduleRepo,
		schemaReg:    schemaReg,
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

	if schedErr := b.materialiseSchedules(ctx, def, spec); schedErr != nil {
		return nil, fmt.Errorf("materialise schedules: %w", schedErr)
	}

	log.Info("workflow created",
		"workflow_id", def.ID,
		"name", spec.Name,
	)

	return def, nil
}

func (b *workflowBusiness) materialiseSchedules(
	ctx context.Context,
	def *models.WorkflowDefinition,
	spec *dsl.WorkflowSpec,
) error {
	for _, sspec := range spec.Schedules {
		payloadJSON := "{}"
		if len(sspec.InputPayload) > 0 {
			raw, err := json.Marshal(sspec.InputPayload)
			if err != nil {
				return fmt.Errorf("marshal input_payload for schedule %s: %w", sspec.Name, err)
			}
			payloadJSON = string(raw)
		}

		sched := &models.ScheduleDefinition{
			Name:            sspec.Name,
			CronExpr:        sspec.CronExpr,
			WorkflowName:    def.Name,
			WorkflowVersion: def.WorkflowVersion,
			InputPayload:    payloadJSON,
			Active:          false, // DRAFT — activated by ActivateWorkflow.
			NextFireAt:      nil,
			JitterSeconds:   0,
		}
		sched.CopyPartitionInfo(&def.BaseModel)

		if err := b.scheduleRepo.Create(ctx, sched); err != nil {
			return fmt.Errorf("create schedule %s: %w", sspec.Name, err)
		}
	}

	return nil
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

	db := b.scheduleRepo.Pool().DB(ctx, false)
	now := time.Now().UTC()

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Persist the workflow status change inside the tx.
		if updErr := tx.Save(def).Error; updErr != nil {
			return fmt.Errorf("update workflow: %w", updErr)
		}

		// Deactivate every version's schedules for this workflow name (wildcard).
		// Use raw Exec to guarantee active=false (boolean zero value) is written.
		if deactErr := tx.Exec(
			`UPDATE schedule_definitions
			    SET active = false, next_fire_at = NULL, modified_at = ?
			  WHERE workflow_name = ? AND deleted_at IS NULL`,
			now, def.Name,
		).Error; deactErr != nil {
			return fmt.Errorf("deactivate prior schedules: %w", deactErr)
		}

		// Activate this version's schedules with seeded next_fire_at.
		myScheds, listErr := listSchedulesTx(tx, def.Name, def.WorkflowVersion)
		if listErr != nil {
			return listErr
		}
		for _, sch := range myScheds {
			cronSched, parseErr := dsl.ParseCron(sch.CronExpr)
			if parseErr != nil {
				return fmt.Errorf("parse cron for schedule %s: %w", sch.Name, parseErr)
			}
			nominal := cronSched.Next(now)
			jitter := jitterForSchedule(sch.ID, cronSched, nominal)
			next := nominal.Add(jitter)

			if updErr := tx.Exec(
				`UPDATE schedule_definitions
				    SET active = true, next_fire_at = ?, jitter_seconds = ?, modified_at = ?
				  WHERE id = ? AND tenant_id = ?`,
				&next, int(jitter/time.Second), now,
				sch.ID, sch.TenantID,
			).Error; updErr != nil {
				return fmt.Errorf("activate schedule %s: %w", sch.ID, updErr)
			}
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	return nil
}

// listSchedulesTx is a tx-bound list used inside ActivateWorkflow. The write path
// must read schedules via the same tx as the subsequent UPDATEs so it sees uncommitted
// row-locking consistency; ScheduleRepository.ListByWorkflow does not accept a tx.
func listSchedulesTx(tx *gorm.DB, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error) {
	var out []*models.ScheduleDefinition
	res := tx.Where("workflow_name = ? AND workflow_version = ? AND deleted_at IS NULL",
		workflowName, workflowVersion).Find(&out)
	if res.Error != nil {
		return nil, fmt.Errorf("list schedules by workflow (tx): %w", res.Error)
	}
	return out, nil
}

// jitterForSchedule duplicates schedulers.jitterFor's algorithm to avoid importing
// the schedulers package from business (layer smell). Deterministic per-schedule
// offset, cap = min(period/10, 30s).
func jitterForSchedule(scheduleID string, cronSched dsl.CronSchedule, nominal time.Time) time.Duration {
	const cronMaxJitter = 30 * time.Second
	following := cronSched.Next(nominal)
	period := following.Sub(nominal)
	if period <= 0 {
		return 0
	}
	maxDur := period / 10
	if maxDur > cronMaxJitter {
		maxDur = cronMaxJitter
	}
	if maxDur <= 0 {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(scheduleID))
	return time.Duration(int64(h.Sum64() % uint64(maxDur)))
}
