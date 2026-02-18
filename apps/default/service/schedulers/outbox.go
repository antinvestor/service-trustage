package schedulers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pitabwire/frame"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// OutboxScheduler publishes unpublished events from the event_log table to NATS.
type OutboxScheduler struct {
	eventRepo repository.EventLogRepository
	svc       *frame.Service
	cfg       *config.Config
	metrics   *telemetry.Metrics
}

// NewOutboxScheduler creates a new OutboxScheduler.
func NewOutboxScheduler(
	eventRepo repository.EventLogRepository,
	svc *frame.Service,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *OutboxScheduler {
	return &OutboxScheduler{
		eventRepo: eventRepo,
		svc:       svc,
		cfg:       cfg,
		metrics:   metrics,
	}
}

// Start begins the outbox publisher loop.
func (s *OutboxScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.OutboxIntervalSeconds) * time.Second

	log.Info("outbox scheduler started", "interval_seconds", s.cfg.OutboxIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			published := s.RunOnce(ctx)
			if published > 0 {
				log.Info("outbox scheduler completed", "published", published)
			}
		case <-ctx.Done():
			log.Info("outbox scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single outbox publish sweep within a single transaction
// to ensure FOR UPDATE SKIP LOCKED locks are held across the entire batch.
func (s *OutboxScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerOutbox)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	// Record outbox lag gauge (count from last batch for visibility).
	published, err := s.eventRepo.FindAndProcessUnpublished(ctx, s.cfg.OutboxBatchSize,
		func(event *models.EventLog) error {
			// Build proper IngestedEventMessage for the event router worker.
			var payload map[string]any
			if unmarshalErr := json.Unmarshal([]byte(event.Payload), &payload); unmarshalErr != nil {
				return fmt.Errorf("unmarshal payload: %w", unmarshalErr)
			}

			msg := &events.IngestedEventMessage{
				EventID:   event.ID,
				TenantID:  event.TenantID,
				EventType: event.EventType,
				Source:    event.Source,
				Payload:   payload,
			}

			// Publish to NATS event stream.
			if publishErr := s.svc.QueueManager().Publish(ctx, s.cfg.QueueEventIngestName, msg); publishErr != nil {
				return fmt.Errorf("publish: %w", publishErr)
			}

			return nil
		},
	)
	if err != nil {
		log.WithError(err).Error("outbox scheduler: sweep failed")
		return 0
	}

	s.metrics.SchedulerOutboxGauge.Record(ctx, int64(published))
	span.SetAttributes(attribute.Int("published_count", published))

	return published
}
