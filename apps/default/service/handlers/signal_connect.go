package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	signalv1 "github.com/antinvestor/service-trustage/gen/go/signal/v1"
	"github.com/antinvestor/service-trustage/gen/go/signal/v1/signalv1connect"
)

// SignalConnectServer exposes signal delivery over ConnectRPC.
type SignalConnectServer struct {
	engine business.StateEngine

	signalv1connect.UnimplementedSignalServiceHandler
}

// NewSignalConnectServer creates a new Connect signal server.
func NewSignalConnectServer(engine business.StateEngine) *SignalConnectServer {
	return &SignalConnectServer{
		engine: engine,
	}
}

func (s *SignalConnectServer) SendSignal(
	ctx context.Context,
	req *connect.Request[signalv1.SendSignalRequest],
) (*connect.Response[signalv1.SendSignalResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}
	if req.Msg.GetSignalName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signal_name is required"))
	}

	payloadBytes, err := json.Marshal(req.Msg.GetPayload().AsMap())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("marshal signal payload: %w", err))
	}

	delivered, err := s.engine.SendSignal(ctx, req.Msg.GetInstanceId(), req.Msg.GetSignalName(), payloadBytes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}

	return connect.NewResponse(&signalv1.SendSignalResponse{
		Delivered: delivered,
	}), nil
}
