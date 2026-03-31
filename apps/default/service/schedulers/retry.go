package schedulers

import (
	"context"
	"time"

	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// RetryScheduler picks up retry_scheduled executions past their next_retry_at
// and creates new pending execution attempts. The dispatch scheduler picks them up.
type RetryScheduler struct {
	execRepo     repository.WorkflowExecutionRepository
	instanceRepo repository.WorkflowInstanceRepository
	cfg          *config.Config
	metrics      *telemetry.Metrics
}

// NewRetryScheduler creates a new RetryScheduler.
func NewRetryScheduler(
	execRepo repository.WorkflowExecutionRepository,
	instanceRepo repository.WorkflowInstanceRepository,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *RetryScheduler {
	return &RetryScheduler{
		execRepo:     execRepo,
		instanceRepo: instanceRepo,
		cfg:          cfg,
		metrics:      metrics,
	}
}

// Start begins the retry scheduler loop.
func (s *RetryScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.RetryIntervalSeconds) * time.Second

	log.Debug("retry scheduler started", "interval_seconds", s.cfg.RetryIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			retried := s.RunOnce(ctx)
			if retried > 0 {
				log.Debug("retry scheduler completed", "retried", retried)
			}
		case <-ctx.Done():
			log.Debug("retry scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single retry sweep.
func (s *RetryScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerRetry)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	due, err := s.execRepo.FindRetryDue(ctx, s.cfg.RetryBatchSize)
	if err != nil {
		log.WithError(err).Error("retry scheduler: failed to find due retries")
		return 0
	}

	// Record scheduler lag gauge.
	s.metrics.SchedulerRetryDueGauge.Record(ctx, int64(len(due)))

	retried := 0

	for _, exec := range due {
		// Verify instance is still running before scheduling retry.
		instance, instanceErr := s.instanceRepo.GetByID(ctx, exec.InstanceID)
		if instanceErr != nil {
			log.WithError(instanceErr).Error("retry scheduler: load instance failed",
				"execution_id", exec.ID,
			)
			// Mark stale so we don't retry this execution again.
			_ = s.execRepo.MarkStale(ctx, exec.ID)

			continue
		}

		if instance.Status != models.InstanceStatusRunning {
			log.Debug("retry scheduler: skipping retry for non-running instance",
				"execution_id", exec.ID,
				"instance_status", instance.Status,
			)
			_ = s.execRepo.MarkStale(ctx, exec.ID)

			continue
		}

		// Mark old execution as stale.
		if staleErr := s.execRepo.MarkStale(ctx, exec.ID); staleErr != nil {
			log.WithError(staleErr).Error("retry scheduler: mark stale failed",
				"execution_id", exec.ID,
			)

			continue
		}

		// Create new execution with incremented attempt.
		rawToken, tokenErr := cryptoutil.GenerateToken()
		if tokenErr != nil {
			log.WithError(tokenErr).Error("retry scheduler: generate token failed")
			continue
		}

		newExec := &models.WorkflowStateExecution{
			InstanceID:      exec.InstanceID,
			State:           exec.State,
			StateVersion:    exec.StateVersion,
			Attempt:         exec.Attempt + 1,
			Status:          models.ExecStatusPending,
			ExecutionToken:  cryptoutil.HashToken(rawToken),
			InputSchemaHash: exec.InputSchemaHash,
			InputPayload:    exec.InputPayload,
			TraceID:         exec.TraceID,
		}

		if createErr := s.execRepo.Create(ctx, newExec); createErr != nil {
			log.WithError(createErr).Error("retry scheduler: create new execution failed")
			continue
		}

		// The new execution has status=pending, so the dispatch scheduler will pick it up.
		retried++
	}

	span.SetAttributes(attribute.Int("retried_count", retried))

	return retried
}
