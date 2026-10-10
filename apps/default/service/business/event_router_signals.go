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

package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pitabwire/util"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// routeSignals delivers the event as a signal along every active signal
// route of its type. A route targets running instances of exactly its
// workflow version: by correlation key (key routes) or by the instance id
// the payload carries (instance routes). Delivery is idempotent per
// (event, instance, signal), so a redelivered event is safe to re-route.
//
// Lookup and delivery failures are returned so the event is redelivered;
// per-route expression failures only skip that route.
func (r *eventRouter) routeSignals(ctx context.Context, event *events.IngestedEventMessage) error {
	if r.routeRepo == nil {
		return nil
	}

	log := util.Log(ctx)

	routes, err := r.routeRepo.FindActiveByEventType(ctx, event.EventType, r.bindingLimit)
	if err != nil {
		return fmt.Errorf("find signal routes: %w", err)
	}

	if len(routes) >= r.bindingLimit {
		log.Warn("event router: signal routes capped",
			"event_type", event.EventType,
			"limit", r.bindingLimit,
		)
	}

	if len(routes) == 0 {
		return nil
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal signal payload: %w", err)
	}

	var firstErr error

	for _, route := range routes {
		targets, targetErr := r.signalTargets(ctx, route, event)
		if targetErr != nil {
			log.WithError(targetErr).Error("event router: signal route lookup failed",
				"route_id", route.ID,
				"event_id", event.EventID,
			)
			if firstErr == nil {
				firstErr = targetErr
			}
			continue
		}

		for _, instanceID := range targets {
			_, deliverErr := r.engine.DeliverEventSignal(ctx, event.EventID, instanceID, route.SignalName, payload)
			if deliverErr != nil {
				log.WithError(deliverErr).Error("event router: signal delivery failed",
					"route_id", route.ID,
					"event_id", event.EventID,
					"instance_id", instanceID,
					"signal_name", route.SignalName,
				)
				if firstErr == nil {
					firstErr = fmt.Errorf("deliver signal %s to %s: %w", route.SignalName, instanceID, deliverErr)
				}
			}
		}
	}

	return firstErr
}

// signalTargets returns the ids of the running instances the route delivers
// this event to. Expression failures are logged and yield no targets.
func (r *eventRouter) signalTargets(
	ctx context.Context,
	route *models.SignalRoute,
	event *events.IngestedEventMessage,
) ([]string, error) {
	log := util.Log(ctx)

	matched, err := evaluateTriggerFilter(route.Filter, event.Payload)
	if err != nil {
		log.WithError(err).Warn("event router: signal route filter evaluation failed",
			"route_id", route.ID,
			"event_id", event.EventID,
		)
		return nil, nil
	}
	if !matched {
		return nil, nil
	}

	if route.InstanceExpr != "" {
		return r.instanceRouteTarget(ctx, route, event)
	}

	key, err := evaluateRoutingValue(route.KeyExpr, event.Payload)
	if err != nil {
		log.WithError(err).Warn("event router: signal route key evaluation failed",
			"route_id", route.ID,
			"event_id", event.EventID,
		)
		return nil, nil
	}

	instances, err := r.instanceRepo.ListRunningByCorrelationKey(
		ctx, route.WorkflowName, route.WorkflowVersion, key, r.bindingLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("find instances by correlation key: %w", err)
	}

	if len(instances) >= r.bindingLimit {
		log.Warn("event router: signal fan-out capped",
			"route_id", route.ID,
			"event_id", event.EventID,
			"limit", r.bindingLimit,
		)
	}

	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, inst.ID)
	}

	return ids, nil
}

func (r *eventRouter) instanceRouteTarget(
	ctx context.Context,
	route *models.SignalRoute,
	event *events.IngestedEventMessage,
) ([]string, error) {
	log := util.Log(ctx)

	instanceID, err := evaluateRoutingValue(route.InstanceExpr, event.Payload)
	if err != nil {
		log.WithError(err).Warn("event router: signal route instance evaluation failed",
			"route_id", route.ID,
			"event_id", event.EventID,
		)
		return nil, nil
	}

	inst, err := r.instanceRepo.GetByID(ctx, instanceID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && inst == nil) {
		// Routine: another tenant's id, or a purged instance.
		log.Debug("event router: signal route instance not found",
			"route_id", route.ID,
			"event_id", event.EventID,
			"instance_id", instanceID,
		)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load signal target instance: %w", err)
	}

	if inst.WorkflowName != route.WorkflowName ||
		inst.WorkflowVersion != route.WorkflowVersion ||
		inst.Status != models.InstanceStatusRunning {
		return nil, nil
	}

	return []string{inst.ID}, nil
}
