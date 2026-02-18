package business

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cacheutil"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// Maximum number of compiled CEL ASTs to cache in-process.
const maxCELCacheSize = 500

// CEL compilation cache (shared across event router invocations).
var (
	celCache   = cacheutil.NewBoundedCache[*cel.Ast](maxCELCacheSize) //nolint:gochecknoglobals // CEL cache
	celEnv     *cel.Env                                               //nolint:gochecknoglobals // CEL env
	celEnvOnce sync.Once                                              //nolint:gochecknoglobals // CEL env init
)

// EventRouter evaluates trigger bindings and creates workflow instances.
type EventRouter interface {
	RouteEvent(ctx context.Context, event *events.IngestedEventMessage) (int, error)
}

type eventRouter struct {
	triggerRepo  repository.TriggerBindingRepository
	defRepo      repository.WorkflowDefinitionRepository
	instanceRepo repository.WorkflowInstanceRepository
	auditRepo    repository.AuditEventRepository
	engine       StateEngine
	metrics      *telemetry.Metrics
}

// NewEventRouter creates a new EventRouter.
func NewEventRouter(
	triggerRepo repository.TriggerBindingRepository,
	defRepo repository.WorkflowDefinitionRepository,
	instanceRepo repository.WorkflowInstanceRepository,
	auditRepo repository.AuditEventRepository,
	engine StateEngine,
	metrics *telemetry.Metrics,
) EventRouter {
	return &eventRouter{
		triggerRepo:  triggerRepo,
		defRepo:      defRepo,
		instanceRepo: instanceRepo,
		auditRepo:    auditRepo,
		engine:       engine,
		metrics:      metrics,
	}
}

// RouteEvent finds matching trigger bindings, evaluates CEL filters,
// and creates workflow instances. Returns the number of instances created.
func (r *eventRouter) RouteEvent(ctx context.Context, event *events.IngestedEventMessage) (int, error) {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEvent, telemetry.SpanRouteEvent,
		attribute.String(telemetry.AttrTenantID, event.TenantID),
		attribute.String(telemetry.AttrEventType, event.EventType),
	)
	var routeErr error
	defer func() {
		r.metrics.EventsIngestedTotal.Add(ctx, 1)
		telemetry.EndSpan(span, routeErr)
	}()

	log := util.Log(ctx)

	bindings, err := r.triggerRepo.FindByEventType(ctx, event.TenantID, event.EventType)
	if err != nil {
		routeErr = fmt.Errorf("find triggers: %w", err)
		return 0, routeErr
	}

	if len(bindings) == 0 {
		return 0, nil
	}

	created := 0

	for _, binding := range bindings {
		matched, matchErr := evaluateTriggerFilter(binding.EventFilter, event.Payload)
		if matchErr != nil {
			log.WithError(matchErr).Error("trigger filter evaluation failed",
				"binding_id", binding.ID,
				"event_type", event.EventType,
			)

			continue
		}

		if !matched {
			continue
		}

		if instanceErr := r.createInstance(ctx, binding, event); instanceErr != nil {
			log.WithError(instanceErr).Error("failed to create instance",
				"binding_id", binding.ID,
				"workflow", binding.WorkflowName,
			)

			continue
		}

		created++
	}

	r.metrics.EventsRoutedTotal.Add(ctx, int64(created))

	return created, nil
}

func (r *eventRouter) createInstance(
	ctx context.Context,
	binding *models.TriggerBinding,
	event *events.IngestedEventMessage,
) error {
	// Load workflow definition to get initial state.
	def, err := r.defRepo.GetByNameAndVersion(
		ctx, event.TenantID, binding.WorkflowName, binding.WorkflowVersion,
	)
	if err != nil {
		return fmt.Errorf("load workflow: %w", err)
	}

	// Parse DSL to find initial state.
	spec, err := dsl.Parse([]byte(def.DSLBlob))
	if err != nil {
		return fmt.Errorf("parse DSL: %w", err)
	}

	initialStep := dsl.InitialStep(spec)
	if initialStep == nil {
		return fmt.Errorf("workflow %s has no steps", binding.WorkflowName)
	}

	now := time.Now()

	instance := &models.WorkflowInstance{
		WorkflowName:    binding.WorkflowName,
		WorkflowVersion: binding.WorkflowVersion,
		CurrentState:    initialStep.ID,
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		TriggerEventID:  event.EventID,
		StartedAt:       &now,
	}

	// Set tenant and partition from event.
	instance.TenantID = event.TenantID
	instance.PartitionID = def.PartitionID

	if err = r.instanceRepo.Create(ctx, instance); err != nil {
		return fmt.Errorf("create instance: %w", err)
	}

	// Audit event.
	_ = r.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		ID:          util.IDString(),
		TenantID:    event.TenantID,
		PartitionID: def.PartitionID,
		InstanceID:  instance.ID,
		EventType:   events.EventInstanceCreated,
		State:       initialStep.ID,
	})

	// Create initial execution.
	inputPayload, _ := json.Marshal(event.Payload)

	_, execErr := r.engine.CreateInitialExecution(ctx, instance, inputPayload)
	if execErr != nil {
		return fmt.Errorf("create initial execution: %w", execErr)
	}

	return nil
}

func evaluateTriggerFilter(filter string, payload map[string]any) (bool, error) {
	if filter == "" {
		return true, nil
	}

	// Initialize CEL environment once.
	var envErr error
	celEnvOnce.Do(func() {
		celEnv, envErr = dsl.NewExpressionEnv()
	})
	if envErr != nil {
		return false, fmt.Errorf("create CEL env: %w", envErr)
	}

	// Check bounded cache for compiled AST.
	ast, cached := celCache.Get(filter)

	if !cached {
		var compileErr error
		ast, compileErr = dsl.CompileExpression(celEnv, filter)
		if compileErr != nil {
			return false, fmt.Errorf("compile filter: %w", compileErr)
		}

		celCache.Put(filter, ast)
	}

	vars := map[string]any{
		"payload": payload,
	}

	return dsl.EvaluateCondition(celEnv, ast, vars)
}
