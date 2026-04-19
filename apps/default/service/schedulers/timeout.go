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
	"math"
	"math/rand/v2"
	"time"

	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// TimeoutScheduler finds dispatched executions that have exceeded their timeout
// and either schedules a retry (if policy allows) or marks them as fatal.
type TimeoutScheduler struct {
	execRepo        repository.WorkflowExecutionRepository
	instanceRepo    repository.WorkflowInstanceRepository
	retryPolicyRepo repository.RetryPolicyRepository
	auditRepo       repository.AuditEventRepository
	cfg             *config.Config
	metrics         *telemetry.Metrics
}

// NewTimeoutScheduler creates a new TimeoutScheduler.
func NewTimeoutScheduler(
	execRepo repository.WorkflowExecutionRepository,
	instanceRepo repository.WorkflowInstanceRepository,
	retryPolicyRepo repository.RetryPolicyRepository,
	auditRepo repository.AuditEventRepository,
	cfg *config.Config,
	metrics *telemetry.Metrics,
) *TimeoutScheduler {
	return &TimeoutScheduler{
		execRepo:        execRepo,
		instanceRepo:    instanceRepo,
		retryPolicyRepo: retryPolicyRepo,
		auditRepo:       auditRepo,
		cfg:             cfg,
		metrics:         metrics,
	}
}

// Start begins the timeout scheduler loop.
func (s *TimeoutScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.TimeoutIntervalSeconds) * time.Second

	log.Debug("timeout scheduler started", "interval_seconds", s.cfg.TimeoutIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			timedOut := s.RunOnce(ctx)
			if timedOut > 0 {
				log.Debug("timeout scheduler completed", "timed_out", timedOut)
			}
		case <-ctx.Done():
			log.Debug("timeout scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single timeout sweep.
func (s *TimeoutScheduler) RunOnce(ctx context.Context) int {
	ctx, span := telemetry.StartSpan(ctx, telemetry.TracerScheduler, telemetry.SpanSchedulerTimeout)
	defer telemetry.EndSpan(span, nil)

	log := util.Log(ctx)

	overdue, err := s.execRepo.FindTimedOut(ctx, s.cfg.DefaultExecutionTimeoutSeconds, s.cfg.TimeoutBatchSize)
	if err != nil {
		log.WithError(err).Error("timeout scheduler: failed to find timed out")
		return 0
	}

	// Record scheduler lag gauge.
	s.metrics.SchedulerDispatchedGauge.Record(ctx, int64(len(overdue)))

	timedOut := 0

	for _, exec := range overdue {
		// Attempt to schedule a retry — builds the new execution if allowed.
		// The mark-timed-out + create-retry steps are committed atomically to
		// prevent a pod crash between them from leaving a stuck dispatched row.
		if retried := s.scheduleRetryIfAllowed(ctx, exec); retried {
			log.Debug("timeout scheduler: retry scheduled",
				"execution_id", exec.ID,
				"attempt", exec.Attempt,
			)
		} else {
			// No retry possible — mark as timed_out then fatal and fail the instance.
			updateErr := s.execRepo.UpdateStatus(ctx, exec.ID, models.ExecStatusTimedOut, map[string]any{
				"error_class":   "retryable",
				"error_message": "execution timed out",
			})
			if updateErr != nil {
				log.WithError(updateErr).Error("timeout scheduler: mark timed out failed",
					"execution_id", exec.ID,
				)
				continue
			}
			// No retry possible — mark as fatal and fail the instance.
			_ = s.execRepo.UpdateStatus(ctx, exec.ID, models.ExecStatusFatal, map[string]any{
				"error_class":   "retryable",
				"error_message": "execution timed out, retries exhausted",
			})
			_ = s.instanceRepo.UpdateStatus(ctx, exec.InstanceID, models.InstanceStatusFailed)

			_ = s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
				InstanceID:  exec.InstanceID,
				ExecutionID: exec.ID,
				EventType:   events.EventStateFailed,
				State:       exec.State,
			})
		}

		timedOut++
	}

	span.SetAttributes(attribute.Int("timed_out_count", timedOut))

	return timedOut
}

const timeoutExponentialBase = 2

// scheduleRetryIfAllowed checks the retry policy and creates a new pending execution if allowed.
func (s *TimeoutScheduler) scheduleRetryIfAllowed(
	ctx context.Context,
	exec *models.WorkflowStateExecution,
) bool {
	log := util.Log(ctx)

	// Load instance to get workflow info for retry policy lookup.
	instance, err := s.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		log.WithError(err).Error("timeout scheduler: load instance failed",
			"execution_id", exec.ID,
		)
		return false
	}

	policy, policyErr := s.retryPolicyRepo.Lookup(
		ctx, instance.WorkflowName, instance.WorkflowVersion, exec.State,
	)
	if policyErr != nil {
		return false // no retry policy
	}

	if exec.Attempt >= policy.MaxAttempts {
		return false // retries exhausted
	}

	// Compute next retry time with exponential backoff and full jitter.
	delayMs := policy.InitialDelayMs
	if policy.BackoffStrategy == "exponential" {
		delayMs = min(
			int64(float64(policy.InitialDelayMs)*math.Pow(timeoutExponentialBase, float64(exec.Attempt-1))),
			policy.MaxDelayMs,
		)
	}

	// Apply full jitter to prevent thundering herd.
	jitteredMs := rand.Int64N(delayMs + 1) //nolint:gosec // jitter doesn't need crypto random
	nextRetry := time.Now().Add(time.Duration(jitteredMs) * time.Millisecond)

	// Create new pending execution (dispatch scheduler will pick it up).
	rawToken, tokenErr := cryptoutil.GenerateToken()
	if tokenErr != nil {
		log.WithError(tokenErr).Error("timeout scheduler: generate token failed")
		return false
	}

	newExec := &models.WorkflowStateExecution{
		InstanceID:      exec.InstanceID,
		State:           exec.State,
		StateVersion:    exec.StateVersion,
		Attempt:         exec.Attempt + 1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  cryptoutil.HashToken(rawToken),
		InputSchemaHash: exec.InputSchemaHash,
		InputPayload:    exec.InputPayload,
		TraceID:         exec.TraceID,
		NextRetryAt:     &nextRetry,
	}

	// Atomic: mark old execution timed_out AND insert the retry row in one tx.
	// A pod crash between the two would otherwise leave a stuck dispatched row.
	if createErr := s.execRepo.MarkTimedOutAndCreateRetry(ctx, exec.ID, newExec); createErr != nil {
		log.WithError(createErr).Error("timeout scheduler: mark-timed-out+create-retry failed",
			"execution_id", exec.ID,
		)
		return false
	}

	_ = s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  exec.InstanceID,
		ExecutionID: newExec.ID,
		EventType:   events.EventStateRetried,
		State:       exec.State,
	})

	return true
}
