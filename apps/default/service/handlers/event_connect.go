package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/security/authorizer"
	"github.com/pitabwire/util"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	eventv1 "github.com/antinvestor/service-trustage/gen/go/event/v1"
	"github.com/antinvestor/service-trustage/gen/go/event/v1/eventv1connect"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

// EventConnectServer exposes event ingest and timeline reads over ConnectRPC.
type EventConnectServer struct {
	eventRepo   repository.EventLogRepository
	auditRepo   repository.AuditEventRepository
	authz       authz.Middleware
	metrics     *telemetry.Metrics
	rateLimiter *RateLimiter

	eventv1connect.UnimplementedEventServiceHandler
}

// NewEventConnectServer creates a new Connect event server.
func NewEventConnectServer(
	eventRepo repository.EventLogRepository,
	auditRepo repository.AuditEventRepository,
	authzMiddleware authz.Middleware,
	metrics *telemetry.Metrics,
	rateLimiter *RateLimiter,
) *EventConnectServer {
	return &EventConnectServer{
		eventRepo:   eventRepo,
		auditRepo:   auditRepo,
		authz:       authzMiddleware,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

func (s *EventConnectServer) IngestEvent(
	ctx context.Context,
	req *connect.Request[eventv1.IngestEventRequest],
) (*connect.Response[eventv1.IngestEventResponse], error) {
	log := util.Log(ctx)

	if err := s.validateIngestEventRequest(ctx, req); err != nil {
		return nil, err
	}

	idempotentResponse, found, err := s.lookupExistingEventResponse(ctx, req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	if found {
		return idempotentResponse, nil
	}

	eventLog, payload, err := buildEventLogFromRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("marshal event payload: %w", err))
	}

	duplicateResponse, duplicated, createErr := s.storeEventWithDuplicateRecovery(ctx, eventLog)
	if createErr != nil {
		log.WithError(createErr).Error("failed to store event")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", createErr))
	}
	if duplicated {
		return duplicateResponse, nil
	}

	return connect.NewResponse(&eventv1.IngestEventResponse{
		Event: eventRecordToProto(
			eventLog.ID,
			eventLog.EventType,
			eventLog.Source,
			eventLog.IdempotencyKey,
			payload,
		),
	}), nil
}

func (s *EventConnectServer) validateIngestEventRequest(
	ctx context.Context,
	req *connect.Request[eventv1.IngestEventRequest],
) error {
	if err := requireConnectAuth(ctx); err != nil {
		return err
	}
	if s.authz != nil {
		if err := s.authz.CanEventIngest(ctx); err != nil {
			return authorizer.ToConnectError(err)
		}
	}
	if s.rateLimiter != nil && !s.rateLimiter.Allow(ctx) {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
	}
	if req.Msg.GetEventType() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("event_type is required"))
	}

	return nil
}

func buildEventLogFromRequest(
	msg *eventv1.IngestEventRequest,
) (*models.EventLog, map[string]any, error) {
	payload := msg.GetPayload().AsMap()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}

	return &models.EventLog{
		EventType:      msg.GetEventType(),
		Source:         msg.GetSource(),
		IdempotencyKey: msg.GetIdempotencyKey(),
		Payload:        string(payloadBytes),
	}, payload, nil
}

func (s *EventConnectServer) lookupExistingEventResponse(
	ctx context.Context,
	idempotencyKey string,
) (*connect.Response[eventv1.IngestEventResponse], bool, error) {
	if idempotencyKey == "" {
		return nil, false, nil
	}

	existing, err := s.eventRepo.FindByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	if existing == nil {
		return nil, false, nil
	}

	return duplicateEventResponse(existing), true, nil
}

func (s *EventConnectServer) storeEventWithDuplicateRecovery(
	ctx context.Context,
	eventLog *models.EventLog,
) (*connect.Response[eventv1.IngestEventResponse], bool, error) {
	if createErr := s.eventRepo.Create(ctx, eventLog); createErr != nil {
		if eventLog.IdempotencyKey != "" && isDuplicateRecordError(createErr) {
			existing, lookupErr := s.eventRepo.FindByIdempotencyKey(ctx, eventLog.IdempotencyKey)
			if lookupErr == nil && existing != nil {
				return duplicateEventResponse(existing), true, nil
			}
		}

		return nil, false, createErr
	}

	return nil, false, nil
}

func duplicateEventResponse(
	existing *models.EventLog,
) *connect.Response[eventv1.IngestEventResponse] {
	var payload map[string]any
	_ = json.Unmarshal([]byte(existing.Payload), &payload)

	return connect.NewResponse(&eventv1.IngestEventResponse{
		Event: eventRecordToProto(
			existing.ID,
			existing.EventType,
			existing.Source,
			existing.IdempotencyKey,
			payload,
		),
		Idempotent: true,
	})
}

func (s *EventConnectServer) GetInstanceTimeline(
	ctx context.Context,
	req *connect.Request[eventv1.GetInstanceTimelineRequest],
) (*connect.Response[eventv1.GetInstanceTimelineResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanInstanceView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}

	auditEvents, err := s.auditRepo.ListByInstance(ctx, req.Msg.GetInstanceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}

	items := make([]*eventv1.TimelineEntry, 0, len(auditEvents))
	for _, auditEvent := range auditEvents {
		items = append(items, timelineEntryToProto(auditEvent))
	}

	return connect.NewResponse(&eventv1.GetInstanceTimelineResponse{
		Items: items,
	}), nil
}
