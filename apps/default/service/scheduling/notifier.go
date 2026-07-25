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

package scheduling

import (
	"context"
	"time"

	"github.com/pitabwire/frame/v2/queue"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// WorkNotifier implements business.WorkNotifier using Frame queue + DelayedPublisher.
type WorkNotifier struct {
	queueMgr queue.Manager
	cfg      *config.Config
	delayed  DelayedPublisher
	engine   business.StateEngine // optional: for dispatch-on-notify
	execLoad PendingLoader
}

// PendingLoader loads a pending execution for notify-dispatch.
type PendingLoader interface {
	GetByID(ctx context.Context, executionID string) (*business.ExecutionCommand, error)
}

// Ensure interface compliance.
var _ business.WorkNotifier = (*WorkNotifier)(nil)

// NewWorkNotifier creates a WorkNotifier. delayed may be NoopDelayedPublisher.
func NewWorkNotifier(
	queueMgr queue.Manager,
	cfg *config.Config,
	delayed DelayedPublisher,
) *WorkNotifier {
	if delayed == nil {
		delayed = NoopDelayedPublisher{}
	}
	return &WorkNotifier{
		queueMgr: queueMgr,
		cfg:      cfg,
		delayed:  delayed,
	}
}

// NotifyPending best-effort publishes a wake to sched-dispatch (reconcile will also pick up).
func (n *WorkNotifier) NotifyPending(ctx context.Context, executionID string) error {
	if n == nil || n.queueMgr == nil || n.cfg == nil {
		return nil
	}
	payload := events.WakeMessage{
		Kind:        events.WakeKindDispatch,
		ExecutionID: executionID,
	}
	if err := n.queueMgr.Publish(ctx, n.cfg.QueueSchedDispatchName, payload); err != nil {
		util.Log(ctx).WithError(err).Debug("work notifier: NotifyPending publish failed",
			"execution_id", executionID,
		)
		return err
	}
	return nil
}

// NotifyExecutionTimeout schedules a delayed wake at the deadline (or no-ops if Noop delayed).
func (n *WorkNotifier) NotifyExecutionTimeout(ctx context.Context, executionID string, at time.Time) error {
	if n == nil || n.delayed == nil {
		return nil
	}
	payload := events.WakeMessage{
		Kind:        events.WakeKindTimeout,
		ExecutionID: executionID,
		DueAt:       at.UTC(),
	}
	return n.delayed.PublishAt(ctx, n.cfg.QueueSchedTimeoutName, at, payload)
}

// NoopNotifier implements WorkNotifier with no side effects.
type NoopNotifier struct{}

func (NoopNotifier) NotifyPending(context.Context, string) error { return nil }
func (NoopNotifier) NotifyExecutionTimeout(context.Context, string, time.Time) error {
	return nil
}
