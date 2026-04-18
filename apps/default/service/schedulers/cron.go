package schedulers

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/pitabwire/util"
	"gorm.io/gorm"

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

	log.Debug("cron scheduler started", "interval_seconds", int(cronCheckInterval.Seconds()))

	ticker := time.NewTicker(cronCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fired := s.RunOnce(ctx)
			if fired > 0 {
				log.Debug("cron scheduler completed", "fired", fired)
			}
		case <-ctx.Done():
			log.Debug("cron scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single sweep for due schedules using ClaimAndFireBatch.
func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, now, cronSchedulerBatchSize,
		func(innerCtx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error) {
			if fireErr := s.fireSchedule(innerCtx, tx, sched, now); fireErr != nil {
				log.WithError(fireErr).Error("cron scheduler: failed to fire schedule",
					"schedule_id", sched.ID,
					"schedule_name", sched.Name,
				)
				return nil, 0, fireErr
			}

			nextFire := computeNextFire(sched.CronExpr, now)
			return nextFire, sched.JitterSeconds, nil
		})
	if err != nil {
		log.WithError(err).Error("cron scheduler: ClaimAndFireBatch failed")
	}

	return fired
}

// fireSchedule creates an event_log entry for the schedule within the provided tx.
func (s *CronScheduler) fireSchedule(
	ctx context.Context,
	tx *gorm.DB,
	sched *models.ScheduleDefinition,
	now time.Time,
) error {
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

	// Insert the event_log row inside the same tx so it is atomic with the schedule lock.
	return tx.Create(eventLog).Error
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
