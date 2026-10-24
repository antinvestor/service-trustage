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

	"connectrpc.com/connect/v2"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
	"github.com/antinvestor/service-trustage/gen/go/workflow/v1/workflowv1connect"
)

// WorkflowConnectServer exposes workflow management over ConnectRPC.
type WorkflowConnectServer struct {
	workflowBiz business.WorkflowBusiness

	workflowv1connect.UnimplementedWorkflowServiceHandler
}

const defaultWorkflowPageLimit = 50

// NewWorkflowConnectServer creates a new Connect workflow server.
func NewWorkflowConnectServer(
	biz business.WorkflowBusiness,
) *WorkflowConnectServer {
	return &WorkflowConnectServer{
		workflowBiz: biz,
	}
}

func (s *WorkflowConnectServer) CreateWorkflow(
	ctx context.Context,
	req *workflowv1.CreateWorkflowRequest,
) (*workflowv1.CreateWorkflowResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetDsl() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, "dsl is required")
	}

	dslBlob, err := rawJSONFromStruct(req.GetDsl())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	def, err := s.workflowBiz.CreateWorkflow(ctx, dslBlob)
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return &workflowv1.CreateWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}, nil
}

func (s *WorkflowConnectServer) GetWorkflow(
	ctx context.Context,
	req *workflowv1.GetWorkflowRequest,
) (*workflowv1.GetWorkflowResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "id is required")
	}

	def, schedules, err := s.workflowBiz.GetWorkflowWithSchedules(ctx, req.GetId())
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return &workflowv1.GetWorkflowResponse{
		Workflow:  workflowDefinitionToProto(def),
		Schedules: scheduleDefinitionsToProto(schedules),
	}, nil
}

func (s *WorkflowConnectServer) ListWorkflows(
	ctx context.Context,
	req *workflowv1.ListWorkflowsRequest,
) (*workflowv1.ListWorkflowsResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	_, err := workflowStatusFilter(req.GetStatus())
	if err != nil {
		return nil, err
	}

	pageLimit := searchLimit(req.GetSearch(), defaultWorkflowPageLimit)
	page, err := s.workflowBiz.SearchWorkflows(ctx, business.WorkflowListFilter{
		Name:    req.GetName(),
		Query:   searchQuery(req.GetSearch()),
		IDQuery: searchIDQuery(req.GetSearch()),
		Cursor:  searchPage(req.GetSearch()),
		Limit:   pageLimit,
	})
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	items := make([]*workflowv1.WorkflowDefinition, 0, len(page.Items))
	for _, def := range page.Items {
		items = append(items, workflowDefinitionToProto(def))
	}

	return &workflowv1.ListWorkflowsResponse{
		Items:      items,
		NextCursor: nextCursorProto(page.NextCursor, pageLimit),
	}, nil
}

func (s *WorkflowConnectServer) ActivateWorkflow(
	ctx context.Context,
	req *workflowv1.ActivateWorkflowRequest,
) (*workflowv1.ActivateWorkflowResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "id is required")
	}

	if err := s.workflowBiz.ActivateWorkflow(ctx, req.GetId()); err != nil {
		return nil, connectErrorForBusiness(err)
	}

	def, err := s.workflowBiz.GetWorkflow(ctx, req.GetId())
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return &workflowv1.ActivateWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}, nil
}

func (s *WorkflowConnectServer) ArchiveWorkflow(
	ctx context.Context,
	req *workflowv1.ArchiveWorkflowRequest,
) (*workflowv1.ArchiveWorkflowResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "id is required")
	}

	if err := s.workflowBiz.ArchiveWorkflow(ctx, req.GetId()); err != nil {
		return nil, connectErrorForBusiness(err)
	}

	def, err := s.workflowBiz.GetWorkflow(ctx, req.GetId())
	if err != nil {
		return nil, connectErrorForBusiness(err)
	}

	return &workflowv1.ArchiveWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}, nil
}
