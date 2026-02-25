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
	authorizer security.Authorizer
}

func NewMiddleware(authorizer security.Authorizer) Middleware {
	return &middleware{authorizer: authorizer}
}

func (m *middleware) CanIngestEvent(ctx context.Context) error {
	return m.check(ctx, PermissionIngestEvent)
}

func (m *middleware) CanManageWorkflow(ctx context.Context) error {
	return m.check(ctx, PermissionManageWorkflow)
}

func (m *middleware) CanViewWorkflow(ctx context.Context) error {
	return m.check(ctx, PermissionViewWorkflow)
}

func (m *middleware) CanViewInstance(ctx context.Context) error {
	return m.check(ctx, PermissionViewInstance)
}

func (m *middleware) CanRetryInstance(ctx context.Context) error {
	return m.check(ctx, PermissionRetryInstance)
}

func (m *middleware) CanViewExecution(ctx context.Context) error {
	return m.check(ctx, PermissionViewExecution)
}

func (m *middleware) CanRetryExecution(ctx context.Context) error {
	return m.check(ctx, PermissionRetryExecution)
}

func (m *middleware) check(ctx context.Context, permission string) error {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		return authorizer.ErrInvalidSubject
	}

	subjectID, err := claims.GetSubject()
	if err != nil || subjectID == "" {
		return authorizer.ErrInvalidSubject
	}

	tenantID := claims.GetTenantID()
	if tenantID == "" {
		return authorizer.ErrInvalidObject
	}

	req := security.CheckRequest{
		Object:     security.ObjectRef{Namespace: NamespaceTenant, ID: tenantID},
		Permission: permission,
		Subject:    security.SubjectRef{Namespace: NamespaceProfile, ID: subjectID},
	}

	result, err := m.authorizer.Check(ctx, req)
	if err != nil {
		return err
	}
	if !result.Allowed {
		return authorizer.NewPermissionDeniedError(req.Object, permission, req.Subject, result.Reason)
	}

	return nil
}
