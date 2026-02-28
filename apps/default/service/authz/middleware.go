package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanEventIngest(ctx context.Context) error
	CanWorkflowManage(ctx context.Context) error
	CanWorkflowView(ctx context.Context) error
	CanInstanceView(ctx context.Context) error
	CanInstanceRetry(ctx context.Context) error
	CanExecutionView(ctx context.Context) error
	CanExecutionRetry(ctx context.Context) error
}

type middleware struct {
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanEventIngest(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionEventIngest)
}

func (m *middleware) CanWorkflowManage(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionWorkflowManage)
}

func (m *middleware) CanWorkflowView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionWorkflowView)
}

func (m *middleware) CanInstanceView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionInstanceView)
}

func (m *middleware) CanInstanceRetry(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionInstanceRetry)
}

func (m *middleware) CanExecutionView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionExecutionView)
}

func (m *middleware) CanExecutionRetry(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionExecutionRetry)
}
