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

	"github.com/pitabwire/frame/queue"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// EventRouterWorker handles NATS messages from the event ingestion stream.
// It deserializes events and delegates to the EventRouter for workflow instantiation.
type EventRouterWorker struct {
	router business.EventRouter
}

// NewEventRouterWorker creates a new EventRouterWorker.
func NewEventRouterWorker(router business.EventRouter) queue.SubscribeWorker {
	return &EventRouterWorker{router: router}
}

// Handle processes a single NATS message from the event stream.
func (w *EventRouterWorker) Handle(ctx context.Context, _ map[string]string, message []byte) error {
	log := util.Log(ctx)

	var event events.IngestedEventMessage
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	log.Debug("routing event",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"tenant_id", event.TenantID,
	)

	claims := &security.AuthenticationClaims{
		TenantID:    event.TenantID,
		PartitionID: event.PartitionID,
	}
	claims.Subject = "system:event_router_worker"
	ctx = claims.ClaimsToContext(ctx)

	created, err := w.router.RouteEvent(ctx, &event)
	if err != nil {
		return fmt.Errorf("route event %s: %w", event.EventID, err)
	}

	if created > 0 {
		log.Debug("event routed",
			"event_id", event.EventID,
			"instances_created", created,
		)
	}

	return nil
}
