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

// CleanupScheduler purges old published events, audit events, and terminal
// workflow execution rows to prevent unbounded table growth.
type CleanupScheduler struct {
	eventRepo  repository.EventLogRepository
	auditRepo  repository.AuditEventRepository
	execRepo   repository.WorkflowExecutionRepository
	timerRepo  repository.WorkflowTimerRepository
	signalRepo repository.WorkflowSignalWaitRepository
	cfg        *config.Config
}

// NewCleanupScheduler creates a new CleanupScheduler.
func NewCleanupScheduler(
	eventRepo repository.EventLogRepository,
	auditRepo repository.AuditEventRepository,
	cfg *config.Config,
	opts ...CleanupSchedulerOption,
) *CleanupScheduler {
	s := &CleanupScheduler{
		eventRepo: eventRepo,
		auditRepo: auditRepo,
		cfg:       cfg,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// CleanupSchedulerOption allows optional workflow-row repos to be wired in.
type CleanupSchedulerOption func(*CleanupScheduler)

// WithWorkflowRowRepos enables retention sweeps for terminal execution, timer,
// and signal-wait rows. Pass all three repos together.
func WithWorkflowRowRepos(
	execRepo repository.WorkflowExecutionRepository,
	timerRepo repository.WorkflowTimerRepository,
	signalRepo repository.WorkflowSignalWaitRepository,
) CleanupSchedulerOption {
	return func(s *CleanupScheduler) {
		s.execRepo = execRepo
		s.timerRepo = timerRepo
		s.signalRepo = signalRepo
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

	// Delete old terminal workflow rows when optional repos are wired in.
	if s.execRepo != nil || s.timerRepo != nil || s.signalRepo != nil {
		wfCutoff := time.Now().UTC().Add(-time.Duration(s.cfg.WorkflowRowRetentionHours) * time.Hour)

		if s.execRepo != nil {
			execDeleted, execErr := s.execRepo.DeleteCompletedBefore(ctx, wfCutoff, cleanupBatch)
			if execErr != nil {
				log.WithError(execErr).Error("cleanup scheduler: delete completed executions failed")
			} else {
				totalDeleted += execDeleted
			}
		}

		if s.timerRepo != nil {
			timerDeleted, timerErr := s.timerRepo.DeleteCompletedBefore(ctx, wfCutoff, cleanupBatch)
			if timerErr != nil {
				log.WithError(timerErr).Error("cleanup scheduler: delete fired timers failed")
			} else {
				totalDeleted += timerDeleted
			}
		}

		if s.signalRepo != nil {
			signalDeleted, signalErr := s.signalRepo.DeleteCompletedBefore(ctx, wfCutoff, cleanupBatch)
			if signalErr != nil {
				log.WithError(signalErr).Error("cleanup scheduler: delete terminal signal waits failed")
			} else {
				totalDeleted += signalDeleted
			}
		}
	}

	span.SetAttributes(attribute.Int64("deleted_count", totalDeleted))

	return totalDeleted
}
