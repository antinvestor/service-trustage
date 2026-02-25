package schedulers

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
)

const (
	cronSchedulerBatchSize = 50
	cronCheckInterval      = 30 * time.Second
)

// CronScheduler fires events for schedule definitions whose next_fire_at has passed.
// Single purpose: check for due schedules and create event_log entries to trigger workflows.
type CronScheduler struct {
	scheduleRepo repository.ScheduleRepository
	eventRepo    repository.EventLogRepository
	cfg          *config.Config
}

// NewCronScheduler creates a new CronScheduler.
func NewCronScheduler(
	scheduleRepo repository.ScheduleRepository,
	eventRepo repository.EventLogRepository,
	cfg *config.Config,
) *CronScheduler {
	return &CronScheduler{
		scheduleRepo: scheduleRepo,
		eventRepo:    eventRepo,
		cfg:          cfg,
	}
}

// Start begins the cron scheduler loop. It blocks until context is cancelled.
func (s *CronScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)

	log.Info("cron scheduler started", "interval_seconds", int(cronCheckInterval.Seconds()))

	ticker := time.NewTicker(cronCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fired := s.RunOnce(ctx)
			if fired > 0 {
				log.Info("cron scheduler completed", "fired", fired)
			}
		case <-ctx.Done():
			log.Info("cron scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single sweep for due schedules.
func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	due, err := s.scheduleRepo.FindDue(ctx, now, cronSchedulerBatchSize)
	if err != nil {
		log.WithError(err).Error("cron scheduler: failed to find due schedules")
		return 0
	}

	fired := 0

	for _, sched := range due {
		if fireErr := s.fireSchedule(ctx, sched, now); fireErr != nil {
			log.WithError(fireErr).Error("cron scheduler: failed to fire schedule",
				"schedule_id", sched.ID,
				"schedule_name", sched.Name,
			)

			continue
		}

		fired++
	}

	return fired
}

// fireSchedule creates an event for the schedule and updates fire times.
func (s *CronScheduler) fireSchedule(
	ctx context.Context,
	sched *models.ScheduleDefinition,
	now time.Time,
) error {
	log := util.Log(ctx)

	// Build event payload.
	payload := map[string]any{
		"schedule_id":   sched.ID,
		"schedule_name": sched.Name,
		"fired_at":      now.Format(time.RFC3339),
	}

	// Merge input payload if present.
	if sched.InputPayload != "" {
		var inputData map[string]any
		if err := json.Unmarshal([]byte(sched.InputPayload), &inputData); err == nil {
			maps.Copy(payload, inputData)
		}
	}

	payloadBytes, _ := json.Marshal(payload)

	eventLog := &models.EventLog{
		EventType:      "schedule.fired",
		Source:         "schedule:" + sched.ID,
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339),
		Payload:        string(payloadBytes),
	}
	eventLog.CopyPartitionInfo(&sched.BaseModel)

	if err := s.eventRepo.Create(ctx, eventLog); err != nil {
		return err
	}

	// Compute next fire time from cron expression (treated as a Go duration).
	nextFire := computeNextFire(sched.CronExpr, now)

	if updateErr := s.scheduleRepo.UpdateFireTimes(ctx, sched.ID, now, nextFire); updateErr != nil {
		log.WithError(updateErr).Error("cron scheduler: failed to update fire times",
			"schedule_id", sched.ID,
		)
	}

	return nil
}

// computeNextFire parses the cron expression as a Go duration (e.g., "1h", "30m", "24h", "7d")
// and adds it to the current time. Returns nil if the expression is invalid.
func computeNextFire(cronExpr string, now time.Time) *time.Time {
	d, err := dsl.ParseDuration(cronExpr)
	if err != nil || d <= 0 {
		return nil
	}

	next := now.Add(d)

	return &next
}
