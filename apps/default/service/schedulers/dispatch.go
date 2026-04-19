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

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// DispatchScheduler picks up pending executions and publishes them to NATS.
type DispatchScheduler struct {
	execRepo repository.WorkflowExecutionRepository
	engine   business.StateEngine
	queueMgr queue.Manager
	cfg      *config.Config
	metrics  *telemetry.Metrics
}

// NewDispatchScheduler creates a new DispatchScheduler.
func NewDispatchScheduler(
	execRepo repository.WorkflowExecutionRepository,
	engine business.StateEngine,
	queueMgr queue.Manager,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *DispatchScheduler {
	return &DispatchScheduler{
		execRepo: execRepo,
		engine:   engine,
		queueMgr: queueMgr,
		cfg:      cfg,
		metrics:  metrics,
	}
}

// Start begins the dispatch scheduler loop. It blocks until context is cancelled.
func (s *DispatchScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.DispatchIntervalSeconds) * time.Second

	log.Debug("dispatch scheduler started", "interval_seconds", s.cfg.DispatchIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dispatched := s.RunUntilDrained(ctx)
			if dispatched > 0 {
				log.Debug("dispatch scheduler completed", "dispatched", dispatched)
			}
		case <-ctx.Done():
			log.Debug("dispatch scheduler stopped")
			return
		}
	}
}

// RunUntilDrained drains multiple dispatch batches in one scheduler wakeup.
func (s *DispatchScheduler) RunUntilDrained(ctx context.Context) int {
	maxBatches := s.cfg.DispatchMaxBatchesPerSweep
	if maxBatches <= 0 {
		maxBatches = 1
	}

	totalDispatched := 0

	for range maxBatches {
		dispatched := s.RunOnce(ctx)
		totalDispatched += dispatched
		if dispatched < s.cfg.DispatchBatchSize {
			break
		}
	}

	return totalDispatched
}

// RunOnce performs a single dispatch sweep.
func (s *DispatchScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerDispatch)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	pending, err := s.execRepo.FindPending(ctx, s.cfg.DispatchBatchSize)
	if err != nil {
		log.WithError(err).Error("dispatch scheduler: failed to find pending")
		return 0
	}

	// Record scheduler lag gauge.
	s.metrics.SchedulerPendingGauge.Record(ctx, int64(len(pending)))

	dispatched := 0

	for _, exec := range pending {
		cmd, dispatchErr := s.engine.Dispatch(ctx, exec)
		if dispatchErr != nil {
			log.WithError(dispatchErr).Error("dispatch scheduler: dispatch failed",
				"execution_id", exec.ID,
			)

			continue
		}

		// Publish full ExecutionCommand to NATS (includes raw token for worker commit).
		publishErr := s.queueMgr.Publish(ctx, s.cfg.QueueExecDispatchName, cmd)
		if publishErr != nil {
			log.WithError(publishErr).Warn("dispatch scheduler: publish failed; reverting execution to pending",
				"execution_id", exec.ID,
			)

			if revertErr := s.engine.RevertDispatch(ctx, exec.ID); revertErr != nil {
				log.WithError(revertErr).Error("dispatch scheduler: revert failed — execution stranded until timeout",
					"execution_id", exec.ID,
					"publish_err", publishErr.Error(),
				)
			}

			continue
		}

		dispatched++
	}

	span.SetAttributes(attribute.Int("dispatched_count", dispatched))

	return dispatched
}
