package schedulers

import (
	"context"
	"errors"
	"time"

	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// TimerScheduler completes durable waiting executions whose timers have fired.
type TimerScheduler struct {
	timerRepo repository.WorkflowTimerRepository
	engine    business.StateEngine
	cfg       *config.Config
	metrics   *telemetry.Metrics
}

// NewTimerScheduler creates a new TimerScheduler.
func NewTimerScheduler(
	timerRepo repository.WorkflowTimerRepository,
	engine business.StateEngine,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *TimerScheduler {
	return &TimerScheduler{
		timerRepo: timerRepo,
		engine:    engine,
		cfg:       cfg,
		metrics:   metrics,
	}
}

// Start begins the timer scheduler loop.
func (s *TimerScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.TimerIntervalSeconds) * time.Second

	log.Info("timer scheduler started", "interval_seconds", s.cfg.TimerIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fired := s.RunOnce(ctx)
			if fired > 0 {
				log.Info("timer scheduler completed", "fired", fired)
			}
		case <-ctx.Done():
			log.Info("timer scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single timer sweep.
func (s *TimerScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, "scheduler.timer")
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)
	now := time.Now()
	owner := "timer-scheduler"
	leaseUntil := now.Add(time.Duration(s.cfg.TimerClaimTTLSeconds) * time.Second)

	timers, err := s.timerRepo.ClaimDue(ctx, now, s.cfg.TimerBatchSize, owner, leaseUntil)
	if err != nil {
		log.WithError(err).Error("timer scheduler: failed to claim due timers")
		return 0
	}

	fired := 0
	for _, timer := range timers {
		resumeErr := s.engine.ResumeWaitingExecution(ctx, timer.ExecutionID, nil)
		if resumeErr != nil && !errors.Is(resumeErr, business.ErrStaleExecution) {
			log.WithError(resumeErr).Error("timer scheduler: resume waiting execution failed",
				"timer_id", timer.ID,
				"execution_id", timer.ExecutionID,
			)
			_ = s.timerRepo.ReleaseClaim(ctx, timer.ID, owner)
			continue
		}

		if markErr := s.timerRepo.MarkFiredByOwner(ctx, timer.ID, owner, time.Now()); markErr != nil {
			log.WithError(markErr).Error("timer scheduler: mark fired failed",
				"timer_id", timer.ID,
				"execution_id", timer.ExecutionID,
			)
			continue
		}

		fired++
	}

	span.SetAttributes(attribute.Int("fired_count", fired))

	return fired
}
