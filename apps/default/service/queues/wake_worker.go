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
	"errors"
	"fmt"

	"github.com/pitabwire/frame/v2/queue"
	"github.com/pitabwire/frame/v2/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/apps/default/service/schedulers"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// WakeWorker handles single-item delayed wakes by delegating to RunOnce sweeps
// (CAS-safe) or targeted engine calls. Stale dual-path losers are no-ops.
type WakeWorker struct {
	kind     string
	dispatch *schedulers.DispatchScheduler
	retry    *schedulers.RetryScheduler
	timer    *schedulers.TimerScheduler
	timeout  *schedulers.TimeoutScheduler
	signal   *schedulers.SignalScheduler
	engine   business.StateEngine
	execRepo repository.WorkflowExecutionRepository
	queueMgr queue.Manager
	cfg      *config.Config
}

// NewDispatchWakeWorker creates a sched-dispatch wake handler.
func NewDispatchWakeWorker(
	dispatch *schedulers.DispatchScheduler,
	engine business.StateEngine,
	execRepo repository.WorkflowExecutionRepository,
	queueMgr queue.Manager,
	cfg *config.Config,
) queue.SubscribeWorker {
	return &WakeWorker{
		kind:     events.WakeKindDispatch,
		dispatch: dispatch,
		engine:   engine,
		execRepo: execRepo,
		queueMgr: queueMgr,
		cfg:      cfg,
	}
}

// NewRetryWakeWorker creates a sched-retry wake handler.
func NewRetryWakeWorker(retry *schedulers.RetryScheduler) queue.SubscribeWorker {
	return &WakeWorker{kind: events.WakeKindRetry, retry: retry}
}

// NewTimerWakeWorker creates a sched-timer wake handler.
func NewTimerWakeWorker(timer *schedulers.TimerScheduler) queue.SubscribeWorker {
	return &WakeWorker{kind: events.WakeKindTimer, timer: timer}
}

// NewTimeoutWakeWorker creates a sched-timeout wake handler.
func NewTimeoutWakeWorker(timeout *schedulers.TimeoutScheduler) queue.SubscribeWorker {
	return &WakeWorker{kind: events.WakeKindTimeout, timeout: timeout}
}

// NewSignalWakeWorker creates a sched-signal wake handler.
func NewSignalWakeWorker(signal *schedulers.SignalScheduler) queue.SubscribeWorker {
	return &WakeWorker{kind: events.WakeKindSignal, signal: signal}
}

// Handle processes a wake message. Permanent decode errors use ErrNotRetryable.
func (w *WakeWorker) Handle(ctx context.Context, _ map[string]string, message []byte) error {
	ctx = security.SkipTenancyChecksOnClaims(ctx)
	log := util.Log(ctx)

	wake, err := parseWakeMessage(message, w.kind)
	if err != nil {
		return err
	}

	switch wake.Kind {
	case events.WakeKindDispatch:
		return w.handleDispatch(ctx, log, wake.ExecutionID)
	case events.WakeKindRetry:
		if w.retry != nil {
			w.retry.RunOnce(ctx)
		}
	case events.WakeKindTimer:
		if w.timer != nil {
			w.timer.RunOnce(ctx)
		}
	case events.WakeKindTimeout:
		if w.timeout != nil {
			w.timeout.RunOnce(ctx)
		}
	case events.WakeKindSignal:
		if w.signal != nil {
			w.signal.RunOnce(ctx)
		}
	default:
		return fmt.Errorf("%w: unknown wake kind %q", queue.ErrNotRetryable, wake.Kind)
	}
	return nil
}

func parseWakeMessage(message []byte, defaultKind string) (events.WakeMessage, error) {
	var wake events.WakeMessage
	if len(message) > 0 {
		if err := json.Unmarshal(message, &wake); err != nil {
			return wake, fmt.Errorf("%w: unmarshal wake: %w", queue.ErrNotRetryable, err)
		}
	}
	if wake.Kind == "" {
		wake.Kind = defaultKind
	}
	return wake, nil
}

func (w *WakeWorker) handleDispatch(ctx context.Context, log *util.LogEntry, executionID string) error {
	if executionID != "" && w.canTargetDispatch() {
		return w.dispatchOne(ctx, log, executionID)
	}
	if w.dispatch != nil {
		w.dispatch.RunOnce(ctx)
	}
	return nil
}

func (w *WakeWorker) canTargetDispatch() bool {
	return w.engine != nil && w.execRepo != nil && w.queueMgr != nil && w.cfg != nil
}

func (w *WakeWorker) dispatchOne(ctx context.Context, log *util.LogEntry, executionID string) error {
	exec, err := w.execRepo.GetByID(ctx, executionID)
	if err != nil {
		log.WithError(err).Debug("dispatch wake: load failed", "execution_id", executionID)
		return nil
	}
	cmd, dErr := w.engine.Dispatch(ctx, exec)
	if dErr != nil {
		if errors.Is(dErr, business.ErrAlreadyDispatched) {
			return nil
		}
		return fmt.Errorf("dispatch wake: %w", dErr)
	}
	if pErr := w.queueMgr.Publish(ctx, w.cfg.QueueExecDispatchName, cmd); pErr != nil {
		_ = w.engine.RevertDispatch(ctx, exec.ID)
		return fmt.Errorf("dispatch wake publish: %w", pErr)
	}
	return nil
}
