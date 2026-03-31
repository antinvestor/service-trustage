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
