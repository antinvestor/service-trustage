package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanManageFormDefinition(ctx context.Context) error
	CanViewFormDefinition(ctx context.Context) error
	CanSubmitForm(ctx context.Context) error
	CanViewSubmission(ctx context.Context) error
	CanUpdateSubmission(ctx context.Context) error
	CanDeleteSubmission(ctx context.Context) error
}

type middleware struct {
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanManageFormDefinition(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionManageFormDefinition)
}

func (m *middleware) CanViewFormDefinition(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewFormDefinition)
}

func (m *middleware) CanSubmitForm(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionSubmitForm)
}

func (m *middleware) CanViewSubmission(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewSubmission)
}

func (m *middleware) CanUpdateSubmission(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionUpdateSubmission)
}

func (m *middleware) CanDeleteSubmission(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionDeleteSubmission)
}
