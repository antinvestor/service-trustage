package handlers

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/security/authorizer"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
	"github.com/antinvestor/service-trustage/gen/go/workflow/v1/workflowv1connect"
)

// WorkflowConnectServer exposes workflow management over ConnectRPC.
type WorkflowConnectServer struct {
	workflowBiz business.WorkflowBusiness
	authz       authz.Middleware

	workflowv1connect.UnimplementedWorkflowServiceHandler
}

// NewWorkflowConnectServer creates a new Connect workflow server.
func NewWorkflowConnectServer(
	biz business.WorkflowBusiness,
	authzMiddleware authz.Middleware,
) *WorkflowConnectServer {
	return &WorkflowConnectServer{
		workflowBiz: biz,
		authz:       authzMiddleware,
	}
}

func (s *WorkflowConnectServer) CreateWorkflow(
	ctx context.Context,
	req *connect.Request[workflowv1.CreateWorkflowRequest],
) (*connect.Response[workflowv1.CreateWorkflowResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanWorkflowManage(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetDsl() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dsl is required"))
	}

	dslBlob, err := rawJSONFromStruct(req.Msg.GetDsl())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	def, err := s.workflowBiz.CreateWorkflow(ctx, dslBlob)
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return connect.NewResponse(&workflowv1.CreateWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}), nil
}

func (s *WorkflowConnectServer) GetWorkflow(
	ctx context.Context,
	req *connect.Request[workflowv1.GetWorkflowRequest],
) (*connect.Response[workflowv1.GetWorkflowResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanWorkflowView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	def, err := s.workflowBiz.GetWorkflow(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return connect.NewResponse(&workflowv1.GetWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}), nil
}

func (s *WorkflowConnectServer) ListWorkflows(
	ctx context.Context,
	req *connect.Request[workflowv1.ListWorkflowsRequest],
) (*connect.Response[workflowv1.ListWorkflowsResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanWorkflowView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	_, err := workflowStatusFilter(req.Msg.GetStatus())
	if err != nil {
		return nil, err
	}

	pageLimit := searchLimit(req.Msg.GetSearch(), 50)
	page, err := s.workflowBiz.SearchWorkflows(ctx, business.WorkflowListFilter{
		Name:    req.Msg.GetName(),
		Query:   searchQuery(req.Msg.GetSearch()),
		IDQuery: searchIDQuery(req.Msg.GetSearch()),
		Cursor:  searchPage(req.Msg.GetSearch()),
		Limit:   pageLimit,
	})
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	items := make([]*workflowv1.WorkflowDefinition, 0, len(page.Items))
	for _, def := range page.Items {
		items = append(items, workflowDefinitionToProto(def))
	}

	return connect.NewResponse(&workflowv1.ListWorkflowsResponse{
		Items:      items,
		NextCursor: nextCursorProto(page.NextCursor, pageLimit),
	}), nil
}

func (s *WorkflowConnectServer) ActivateWorkflow(
	ctx context.Context,
	req *connect.Request[workflowv1.ActivateWorkflowRequest],
) (*connect.Response[workflowv1.ActivateWorkflowResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanWorkflowManage(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.workflowBiz.ActivateWorkflow(ctx, req.Msg.GetId()); err != nil {
		return nil, connectErrorForBusiness(err)
	}

	def, err := s.workflowBiz.GetWorkflow(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return connect.NewResponse(&workflowv1.ActivateWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}), nil
}
