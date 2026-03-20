package schedulers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

const defaultOutboxLeaseTTL = 30 * time.Second

// OutboxScheduler publishes unpublished events from the event_log table to NATS.
type OutboxScheduler struct {
	eventRepo repository.EventLogRepository
	queueMgr  queue.Manager
	cfg       *config.Config
	metrics   *telemetry.Metrics
	owner     string
}

// NewOutboxScheduler creates a new OutboxScheduler.
func NewOutboxScheduler(
	eventRepo repository.EventLogRepository,
	queueMgr queue.Manager,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *OutboxScheduler {
	return &OutboxScheduler{
		eventRepo: eventRepo,
		queueMgr:  queueMgr,
		cfg:       cfg,
		metrics:   metrics,
		owner:     "outbox-" + util.IDString(),
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
			published := s.RunUntilDrained(ctx)
			if published > 0 {
				log.Info("outbox scheduler completed", "published", published)
			}
		case <-ctx.Done():
			log.Info("outbox scheduler stopped")
			return
		}
	}
}

// RunUntilDrained drains multiple outbox batches in one scheduler wakeup.
func (s *OutboxScheduler) RunUntilDrained(ctx context.Context) int {
	maxBatches := s.cfg.OutboxMaxBatchesPerSweep
	if maxBatches <= 0 {
		maxBatches = 1
	}

	totalPublished := 0

	for range maxBatches {
		published := s.RunOnce(ctx)
		totalPublished += published
		if published < s.cfg.OutboxBatchSize {
			break
		}
	}

	return totalPublished
}

// RunOnce performs a single outbox publish sweep using claim/publish/ack flow.
func (s *OutboxScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerOutbox)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	leaseTTL := time.Duration(s.cfg.OutboxClaimTTLSeconds) * time.Second
	if leaseTTL <= 0 {
		leaseTTL = defaultOutboxLeaseTTL
	}

	claimed, err := s.eventRepo.ClaimUnpublished(ctx, s.cfg.OutboxBatchSize, s.owner, time.Now().Add(leaseTTL))
	if err != nil {
		log.WithError(err).Error("outbox scheduler: claim failed")
		return 0
	}

	s.metrics.SchedulerOutboxGauge.Record(ctx, int64(len(claimed)))

	published := 0

	for _, event := range claimed {
		msg, buildErr := buildIngestedEventMessage(event)
		if buildErr != nil {
			log.WithError(buildErr).Error("outbox scheduler: build message failed", "event_id", event.ID)
			_ = s.eventRepo.ReleaseClaim(ctx, event.ID, s.owner)
			continue
		}

		if publishErr := s.queueMgr.Publish(ctx, s.cfg.QueueEventIngestName, msg); publishErr != nil {
			log.WithError(publishErr).Error("outbox scheduler: publish failed", "event_id", event.ID)
			_ = s.eventRepo.ReleaseClaim(ctx, event.ID, s.owner)
			continue
		}

		if ackErr := s.eventRepo.MarkPublishedByOwner(ctx, event.ID, s.owner, time.Now()); ackErr != nil {
			log.WithError(ackErr).Error("outbox scheduler: mark published failed", "event_id", event.ID)
			continue
		}

		published++
	}

	span.SetAttributes(
		attribute.Int("claimed_count", len(claimed)),
		attribute.Int("published_count", published),
	)

	return published
}

func buildIngestedEventMessage(event *models.EventLog) (*events.IngestedEventMessage, error) {
	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(event.Payload), &payload); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", unmarshalErr)
	}

	return &events.IngestedEventMessage{
		EventID:     event.ID,
		TenantID:    event.TenantID,
		PartitionID: event.PartitionID,
		EventType:   event.EventType,
		Source:      event.Source,
		Payload:     payload,
	}, nil
}
