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

package schedulers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

const (
	cronMissedFireThreshold = 5 * time.Minute
	cronDefaultBatchSize    = 500
	cronDefaultIntervalSecs = 1
)

// CronScheduler runs the fire loop. It implements the plan side of
// repository.ScheduleRepository.ClaimAndFireBatch — pure Go, no DB access in
// planOne. The repo owns the transaction.
type CronScheduler struct {
	scheduleRepo repository.ScheduleRepository
	cfg          *config.Config
	metrics      *telemetry.Metrics
}

// NewCronScheduler wires the scheduler with its repo, config, and metrics.
// metrics may be nil (tests pass nil).
func NewCronScheduler(scheduleRepo repository.ScheduleRepository, cfg *config.Config, metrics *telemetry.Metrics) *CronScheduler {
	return &CronScheduler{scheduleRepo: scheduleRepo, cfg: cfg, metrics: metrics}
}

// Start runs the sweep loop until ctx is cancelled.
func (s *CronScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)

	interval := s.interval()
	log.Debug("cron scheduler started",
		"interval_seconds", int(interval.Seconds()),
		"batch_size", s.batchSize(),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.RunOnce(ctx)
		case <-ctx.Done():
			log.Debug("cron scheduler stopped")
			return
		}
	}
}

// RunOnce drives one transactional sweep.
func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerCron)
	defer telemetry.EndSpan(span, nil)

	start := time.Now()

	fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, s.planOne, now, s.batchSize())
	dur := time.Since(start)

	if err != nil {
		log.WithError(err).Error("cron scheduler: sweep failed")
		s.metrics.RecordSchedulerCronSweep(ctx, 0, dur, false)
		return 0
	}

	s.metrics.RecordSchedulerCronSweep(ctx, fired, dur, true)
	if fired > 0 {
		log.Debug("cron scheduler swept", "fired", fired)
	}
	return fired
}

// planOne implements repository.SchedulePlanFn. Pure Go, no DB.
func (s *CronScheduler) planOne(
	ctx context.Context,
	sched *models.ScheduleDefinition,
) (*models.EventLog, *time.Time, int, error) {
	log := util.Log(ctx)
	now := time.Now().UTC()

	cronSched, err := dsl.ParseCron(sched.CronExpr)
	if err != nil {
		log.WithError(err).Error("cron scheduler: invalid cron, parking",
			"schedule_id", sched.ID, "cron_expr", sched.CronExpr)
		if s.metrics != nil {
			s.metrics.SchedulerCronInvalid.Add(ctx, 1, metric.WithAttributes(attribute.String("tenant_id", sched.TenantID)))
		}
		return nil, nil, 0, nil // park: no event, clear next_fire_at
	}

	base := now
	if sched.NextFireAt != nil && now.Sub(*sched.NextFireAt) <= cronMissedFireThreshold {
		base = *sched.NextFireAt
	}

	nominal, err := cronSched.NextInZone(base, sched.Timezone)
	if err != nil {
		log.WithError(err).Error("cron scheduler: invalid timezone, parking",
			"schedule_id", sched.ID, "timezone", sched.Timezone)
		if s.metrics != nil {
			s.metrics.SchedulerCronInvalid.Add(ctx, 1, metric.WithAttributes(attribute.String("tenant_id", sched.TenantID)))
		}
		return nil, nil, 0, nil
	}

	jitter := dsl.JitterFor(sched.ID, cronSched, nominal)
	next := nominal.Add(jitter)

	ev := buildEvent(sched, now)
	return ev, &next, int(jitter / time.Second), nil
}

// buildEvent assembles a schedule.fired event_log row via the typed payload
// helper — user input_payload cannot shadow system fields.
func buildEvent(sched *models.ScheduleDefinition, now time.Time) *models.EventLog {
	var input map[string]any
	if sched.InputPayload != "" {
		var tmp map[string]any
		if err := json.Unmarshal([]byte(sched.InputPayload), &tmp); err == nil {
			input = tmp
		}
	}

	payload := events.BuildScheduleFiredPayload(
		sched.ID, sched.Name, now.Format(time.RFC3339), input,
	)
	raw, _ := payload.ToJSON()

	ev := &models.EventLog{
		EventType:      events.ScheduleFiredType,
		Source:         "schedule:" + sched.ID,
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
		Payload:        raw,
	}
	ev.CopyPartitionInfo(&sched.BaseModel)
	return ev
}

func (s *CronScheduler) batchSize() int {
	if s.cfg != nil && s.cfg.CronSchedulerBatchSize > 0 {
		return s.cfg.CronSchedulerBatchSize
	}
	return cronDefaultBatchSize
}

func (s *CronScheduler) interval() time.Duration {
	if s.cfg != nil && s.cfg.CronSchedulerIntervalSecs > 0 {
		return time.Duration(s.cfg.CronSchedulerIntervalSecs) * time.Second
	}
	return cronDefaultIntervalSecs * time.Second
}
