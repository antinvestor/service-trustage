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

package queues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pitabwire/frame/v2/queue"
	"github.com/pitabwire/frame/v2/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/scheduling"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// ReconcileWorker handles sched-reconcile / sched-cron / sched-cleanup push messages.
type ReconcileWorker struct {
	runner   *scheduling.MultiSweepRunner
	fixedJob string // if non-empty, always run this job name
	cronOnce func(ctx context.Context) int
	cleanup  func(ctx context.Context) int
}

// NewReconcileWorker creates a general reconcile worker (jobs from payload or all).
func NewReconcileWorker(runner *scheduling.MultiSweepRunner) queue.SubscribeWorker {
	return &ReconcileWorker{runner: runner}
}

// NewCronWorker creates a worker that only runs the cron sweep.
func NewCronWorker(cronOnce func(ctx context.Context) int) queue.SubscribeWorker {
	return &ReconcileWorker{cronOnce: cronOnce, fixedJob: events.WakeKindCron}
}

// NewCleanupWorker creates a worker that only runs cleanup.
func NewCleanupWorker(cleanup func(ctx context.Context) int) queue.SubscribeWorker {
	return &ReconcileWorker{cleanup: cleanup, fixedJob: events.WakeKindCleanup}
}

// Handle processes an external scheduler or Cloud Tasks push.
func (w *ReconcileWorker) Handle(ctx context.Context, _ map[string]string, message []byte) error {
	ctx = security.SkipTenancyChecksOnClaims(ctx)
	log := util.Log(ctx)

	var wake events.WakeMessage
	if len(message) > 0 {
		if err := json.Unmarshal(message, &wake); err != nil {
			return fmt.Errorf("%w: unmarshal wake: %v", queue.ErrNotRetryable, err)
		}
	}

	switch {
	case w.cronOnce != nil || wake.Kind == events.WakeKindCron:
		fn := w.cronOnce
		if fn == nil && w.runner != nil {
			n := w.runner.RunJobsOnce(ctx, []string{"cron"})
			log.Debug("cron reconcile", "processed", n)
			return nil
		}
		if fn != nil {
			n := fn(ctx)
			log.Debug("cron reconcile", "processed", n)
		}
		return nil
	case w.cleanup != nil || wake.Kind == events.WakeKindCleanup:
		fn := w.cleanup
		if fn == nil && w.runner != nil {
			n := w.runner.RunJobsOnce(ctx, []string{"cleanup"})
			log.Debug("cleanup reconcile", "processed", n)
			return nil
		}
		if fn != nil {
			n := fn(ctx)
			log.Debug("cleanup reconcile", "processed", n)
		}
		return nil
	default:
		if w.runner == nil {
			return fmt.Errorf("%w: reconcile runner not configured", queue.ErrNotRetryable)
		}
		jobs := wake.Jobs
		if wake.Kind != "" && wake.Kind != events.WakeKindReconcile && len(jobs) == 0 {
			jobs = []string{wake.Kind}
		}
		n := w.runner.RunJobsOnce(ctx, jobs)
		log.Debug("reconcile sweep", "processed", n, "jobs", jobs)
		return nil
	}
}
