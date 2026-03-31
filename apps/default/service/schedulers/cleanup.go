package schedulers

import (
	"context"
	"time"

	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

const (
	cleanupSpanName = "scheduler.cleanup"
	cleanupBatch    = 1000
)

// CleanupScheduler purges old published events and audit events
// to prevent unbounded table growth.
type CleanupScheduler struct {
	eventRepo repository.EventLogRepository
	auditRepo repository.AuditEventRepository
	cfg       *config.Config
}

// NewCleanupScheduler creates a new CleanupScheduler.
func NewCleanupScheduler(
	eventRepo repository.EventLogRepository,
	auditRepo repository.AuditEventRepository,
	cfg *config.Config,
) *CleanupScheduler {
	return &CleanupScheduler{
		eventRepo: eventRepo,
		auditRepo: auditRepo,
		cfg:       cfg,
	}
}

// Start begins the cleanup scheduler loop.
func (s *CleanupScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.CleanupIntervalHours) * time.Hour

	log.Debug("cleanup scheduler started",
		"interval_hours", s.cfg.CleanupIntervalHours,
		"retention_days", s.cfg.RetentionDays,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			deleted := s.RunOnce(ctx)
			if deleted > 0 {
				log.Debug("cleanup scheduler completed", "deleted", deleted)
			}
		case <-ctx.Done():
			log.Debug("cleanup scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single cleanup sweep.
func (s *CleanupScheduler) RunOnce(ctx context.Context) int64 {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, cleanupSpanName)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	const hoursPerDay = 24
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionDays) * hoursPerDay * time.Hour)
	var totalDeleted int64

	// Delete old published events.
	eventDeleted, err := s.eventRepo.DeletePublishedBefore(ctx, cutoff, cleanupBatch)
	if err != nil {
		log.WithError(err).Error("cleanup scheduler: delete published events failed")
	} else {
		totalDeleted += eventDeleted
	}

	// Delete old audit events.
	auditDeleted, err := s.auditRepo.DeleteBefore(ctx, cutoff, cleanupBatch)
	if err != nil {
		log.WithError(err).Error("cleanup scheduler: delete old audit events failed")
	} else {
		totalDeleted += auditDeleted
	}

	span.SetAttributes(attribute.Int64("deleted_count", totalDeleted))

	return totalDeleted
}
