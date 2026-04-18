// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package business

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	GetWorkflowWithSchedules(
		ctx context.Context, id string,
	) (*models.WorkflowDefinition, []*models.ScheduleDefinition, error)
	ListWorkflows(ctx context.Context, name string, limit int) ([]*models.WorkflowDefinition, error)
	SearchWorkflows(ctx context.Context, filter WorkflowListFilter) (*WorkflowListPage, error)
	ActivateWorkflow(ctx context.Context, id string) error
	ArchiveWorkflow(ctx context.Context, id string) error
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

	spec, err := dsl.Parse(dslBlob)
	if err != nil {
		return nil, fmt.Errorf("parse DSL: %w", err)
	}
	if res := dsl.Validate(spec); !res.Valid() {
		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, res.Error())
	}
	err = validateExecutableWorkflow(spec)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, err)
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

	err = b.registerStepSchemas(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("register schemas: %w", err)
	}

	// Tx1: workflow row (single-table auto-commit).
	err = b.defRepo.Create(ctx, def)
	if err != nil {
		return nil, fmt.Errorf("persist workflow: %w", err)
	}

	// Tx2: schedule rows (single-table atomic batch). If this fails, the
	// workflow is an orphan DRAFT — harmless because DRAFT doesn't fire;
	// retry is blocked by idx_wd_name_version on the workflow.
	scheds, err := planScheduleRows(def, spec)
	if err != nil {
		return nil, err
	}
	if len(scheds) > 0 {
		err = b.scheduleRepo.CreateBatch(ctx, scheds)
		if err != nil {
			log.WithError(err).Error("schedule materialisation failed; workflow is orphan DRAFT",
				"workflow_id", def.ID, "name", def.Name)
			return nil, fmt.Errorf(
				"materialise schedules (orphan DRAFT at workflow %s; retry blocked by idx_wd_name_version): %w",
				def.ID,
				err,
			)
		}
	}

	log.Info("workflow created", "workflow_id", def.ID, "name", def.Name)
	return def, nil
}

// planScheduleRows builds []*ScheduleDefinition from spec.Schedules for
// CreateBatch. Pure — no DB access.
func planScheduleRows(
	def *models.WorkflowDefinition,
	spec *dsl.WorkflowSpec,
) ([]*models.ScheduleDefinition, error) {
	out := make([]*models.ScheduleDefinition, 0, len(spec.Schedules))
	for _, sspec := range spec.Schedules {
		payloadJSON := "{}"
		if len(sspec.InputPayload) > 0 {
			raw, err := json.Marshal(sspec.InputPayload)
			if err != nil {
				return nil, fmt.Errorf("marshal input_payload for %s: %w", sspec.Name, err)
			}
			payloadJSON = string(raw)
		}

		tz := sspec.Timezone
		if tz == "" {
			tz = "UTC"
		}

		sched := &models.ScheduleDefinition{
			Name:            sspec.Name,
			CronExpr:        sspec.CronExpr,
			Timezone:        tz,
			WorkflowName:    def.Name,
			WorkflowVersion: def.WorkflowVersion,
			InputPayload:    payloadJSON,
			Active:          false,
			NextFireAt:      nil,
			JitterSeconds:   0,
		}
		sched.CopyPartitionInfo(&def.BaseModel)
		out = append(out, sched)
	}
	return out, nil
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

func defaultSchemasForStep(
	step *dsl.StepSpec,
) (json.RawMessage, json.RawMessage, json.RawMessage, bool) {
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

func (b *workflowBusiness) GetWorkflow(
	ctx context.Context,
	id string,
) (*models.WorkflowDefinition, error) {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	return def, nil
}

func (b *workflowBusiness) GetWorkflowWithSchedules(
	ctx context.Context,
	id string,
) (*models.WorkflowDefinition, []*models.ScheduleDefinition, error) {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	scheds, err := b.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("list schedules: %w", err)
	}

	return def, scheds, nil
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
	log := util.Log(ctx)

	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}
	err = def.TransitionTo(models.WorkflowStatusActive)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	// Tx1: workflow status (single-table auto-commit).
	err = b.defRepo.Update(ctx, def)
	if err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}

	// Build fire plans from ListByWorkflow (single-table read, tenancy-scoped).
	myScheds, err := b.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	if err != nil {
		return fmt.Errorf("list schedules: %w", err)
	}

	now := time.Now().UTC()
	fires := make([]repository.ScheduleActivation, 0, len(myScheds))
	for _, sch := range myScheds {
		cronSched, parseErr := dsl.ParseCron(sch.CronExpr)
		if parseErr != nil {
			return fmt.Errorf("parse cron for %s: %w", sch.Name, parseErr)
		}
		nominal, tzErr := cronSched.NextInZone(now, sch.Timezone)
		if tzErr != nil {
			return fmt.Errorf("timezone for %s: %w", sch.Name, tzErr)
		}
		jitter := dsl.JitterFor(sch.ID, cronSched, nominal)
		fires = append(fires, repository.ScheduleActivation{
			ID:            sch.ID,
			NextFireAt:    nominal.Add(jitter),
			JitterSeconds: int(jitter / time.Second),
		})
	}

	// Tx2: deactivate siblings + activate this version (single-table tx).
	err = b.scheduleRepo.ActivateByWorkflow(
		ctx, def.Name, def.WorkflowVersion, def.TenantID, def.PartitionID, fires,
	)
	if err != nil {
		log.WithError(err).
			Error("activate schedules failed; workflow ACTIVE but schedules stale; retry to reconcile",
				"workflow_id", def.ID)
		return fmt.Errorf("activate schedules: %w", err)
	}
	return nil
}

// ArchiveWorkflow transitions a DRAFT|ACTIVE workflow to ARCHIVED.
// Sequence: deactivate schedules FIRST, then update workflow status. The
// reversed order from Activate is deliberate — if the second step fails,
// schedules are already off (safe: no overfire, just a transient status
// mismatch fixable by retry).
func (b *workflowBusiness) ArchiveWorkflow(ctx context.Context, id string) error {
	log := util.Log(ctx)

	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}
	err = def.TransitionTo(models.WorkflowStatusArchived)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	// Tx1: schedules off FIRST (safe failure ordering).
	err = b.scheduleRepo.DeactivateByWorkflow(ctx, def.Name, def.TenantID, def.PartitionID)
	if err != nil {
		return fmt.Errorf("deactivate schedules: %w", err)
	}

	// Tx2: workflow status.
	err = b.defRepo.Update(ctx, def)
	if err != nil {
		log.WithError(err).
			Error("workflow status update failed after schedules deactivated; retry to reconcile",
				"workflow_id", def.ID)
		return fmt.Errorf("update workflow status: %w", err)
	}
	return nil
}
