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

package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/pitabwire/frame/v2/queue"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/events"
)

// OutboxPublisher best-effort publishes newly ingested events to the event-ingest queue.
// Durable correctness remains with the outbox reconcile path; publish failures leave
// published=false for the outbox scheduler.
type OutboxPublisher struct {
	queueMgr  queue.Manager
	eventRepo repository.EventLogRepository
	queueName string
}

// NewOutboxPublisher creates a publisher. queueMgr may be nil (no-op).
func NewOutboxPublisher(
	queueMgr queue.Manager,
	eventRepo repository.EventLogRepository,
	queueName string,
) *OutboxPublisher {
	return &OutboxPublisher{
		queueMgr:  queueMgr,
		eventRepo: eventRepo,
		queueName: queueName,
	}
}

// PublishAfterCreate publishes the event and conditionally marks it published.
// Never fails the request path — logs and returns on any error.
func (p *OutboxPublisher) PublishAfterCreate(ctx context.Context, event *models.EventLog) {
	if p == nil || p.queueMgr == nil || p.eventRepo == nil || event == nil || event.ID == "" {
		return
	}
	if p.queueName == "" {
		return
	}

	log := util.Log(ctx)

	var payload map[string]any
	if event.Payload != "" {
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			log.WithError(err).Warn("outbox best-effort: bad payload; leave unpublished for reconcile",
				"event_id", event.ID,
			)
			return
		}
	}
	if payload == nil {
		payload = map[string]any{}
	}

	msg := &events.IngestedEventMessage{
		EventID:     event.ID,
		TenantID:    event.TenantID,
		PartitionID: event.PartitionID,
		EventType:   event.EventType,
		Source:      event.Source,
		Payload:     payload,
	}

	if err := p.queueMgr.Publish(ctx, p.queueName, msg); err != nil {
		log.WithError(err).Debug("outbox best-effort: publish failed; outbox reconcile will retry",
			"event_id", event.ID,
			"event_type", event.EventType,
		)
		return
	}

	if err := p.eventRepo.MarkPublishedIfUnpublished(ctx, event.ID); err != nil {
		if errors.Is(err, repository.ErrStaleMutation) {
			// Already published by concurrent outbox path — OK.
			return
		}
		log.WithError(err).Warn("outbox best-effort: mark published failed after successful publish",
			"event_id", event.ID,
		)
	}
}
