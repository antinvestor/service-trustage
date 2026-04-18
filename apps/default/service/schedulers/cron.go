package schedulers

import (
	"context"
	"encoding/json"
	"hash/fnv"
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
	cronSchedulerBatchSize  = 50
	cronCheckInterval       = 30 * time.Second
	cronMissedFireThreshold = 5 * time.Minute
	cronMaxJitter           = 30 * time.Second
)

// CronScheduler fires events for schedule_definitions rows whose next_fire_at has passed.
// Uses ScheduleRepository.ClaimAndFireBatch to ensure exactly-once fire under multi-pod deployment.
type CronScheduler struct {
	scheduleRepo repository.ScheduleRepository
	eventRepo    repository.EventLogRepository
	cfg          *config.Config
}

// NewCronScheduler creates a new CronScheduler. The eventRepo argument is retained for
// backwards compatibility with main.go wiring but is unused now that the event_log insert
// happens inside ClaimAndFireBatch's tx (via fireSchedule's tx handle).
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

// RunOnce performs one transactional sweep for due schedules.
func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, now, cronSchedulerBatchSize,
		func(innerCtx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error) {
			return fireOne(innerCtx, tx, sched, now)
		})
	if err != nil {
		log.WithError(err).Error("cron scheduler: ClaimAndFireBatch failed")
	}

	return fired
}

// fireOne emits the event_log row (inside tx) and returns (nextFire, jitterSeconds, error).
// Runs inside the tx that holds the FOR UPDATE SKIP LOCKED lock.
func fireOne(
	ctx context.Context,
	tx *gorm.DB,
	sched *models.ScheduleDefinition,
	now time.Time,
) (*time.Time, int, error) {
	log := util.Log(ctx)

	// Build event payload.
	payload := map[string]any{
		"schedule_id":   sched.ID,
		"schedule_name": sched.Name,
		"fired_at":      now.Format(time.RFC3339),
	}
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
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
		Payload:        string(payloadBytes),
	}
	eventLog.CopyPartitionInfo(&sched.BaseModel)

	if err := tx.Create(eventLog).Error; err != nil {
		return nil, 0, err
	}

	// Compute next fire.
	cronSched, err := dsl.ParseCron(sched.CronExpr)
	if err != nil {
		// Invalid cron — log and park next_fire_at = nil so the row drops out of the partial index.
		log.WithError(err).Error("cron scheduler: invalid cron expression, parking schedule",
			"schedule_id", sched.ID, "cron_expr", sched.CronExpr)
		return nil, 0, nil
	}

	base := now
	if sched.NextFireAt != nil && now.Sub(*sched.NextFireAt) <= cronMissedFireThreshold {
		base = *sched.NextFireAt
	}

	nominal := cronSched.Next(base)
	jitter := jitterFor(sched.ID, cronSched, nominal)
	next := nominal.Add(jitter)

	return &next, int(jitter / time.Second), nil
}

// jitterFor returns a deterministic per-schedule offset to flatten thundering herds.
// Capped at min(period/10, cronMaxJitter).
func jitterFor(scheduleID string, cronSched dsl.CronSchedule, nominal time.Time) time.Duration {
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
