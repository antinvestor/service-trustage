package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanFormDefinitionManage(ctx context.Context) error
	CanFormDefinitionView(ctx context.Context) error
	CanFormSubmit(ctx context.Context) error
	CanSubmissionView(ctx context.Context) error
	CanSubmissionUpdate(ctx context.Context) error
	CanSubmissionDelete(ctx context.Context) error
}

type middleware struct {
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanFormDefinitionManage(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionFormDefinitionManage)
}

func (m *middleware) CanFormDefinitionView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionFormDefinitionView)
}

func (m *middleware) CanFormSubmit(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionFormSubmit)
}

func (m *middleware) CanSubmissionView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionSubmissionView)
}

func (m *middleware) CanSubmissionUpdate(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionSubmissionUpdate)
}

func (m *middleware) CanSubmissionDelete(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionSubmissionDelete)
}
