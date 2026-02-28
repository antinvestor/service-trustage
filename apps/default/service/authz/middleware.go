package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanIngestEvent(ctx context.Context) error
	CanManageWorkflow(ctx context.Context) error
	CanViewWorkflow(ctx context.Context) error
	CanViewInstance(ctx context.Context) error
	CanRetryInstance(ctx context.Context) error
	CanViewExecution(ctx context.Context) error
	CanRetryExecution(ctx context.Context) error
}

type middleware struct {
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanIngestEvent(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionIngestEvent)
}

func (m *middleware) CanManageWorkflow(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionManageWorkflow)
}

func (m *middleware) CanViewWorkflow(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewWorkflow)
}

func (m *middleware) CanViewInstance(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewInstance)
}

func (m *middleware) CanRetryInstance(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionRetryInstance)
}

func (m *middleware) CanViewExecution(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewExecution)
}

func (m *middleware) CanRetryExecution(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionRetryExecution)
}
