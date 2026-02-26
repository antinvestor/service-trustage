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
	authorizer security.Authorizer
}

func NewMiddleware(authorizer security.Authorizer) Middleware {
	return &middleware{authorizer: authorizer}
}

func (m *middleware) CanManageFormDefinition(ctx context.Context) error {
	return m.check(ctx, PermissionManageFormDefinition)
}

func (m *middleware) CanViewFormDefinition(ctx context.Context) error {
	return m.check(ctx, PermissionViewFormDefinition)
}

func (m *middleware) CanSubmitForm(ctx context.Context) error {
	return m.check(ctx, PermissionSubmitForm)
}

func (m *middleware) CanViewSubmission(ctx context.Context) error {
	return m.check(ctx, PermissionViewSubmission)
}

func (m *middleware) CanUpdateSubmission(ctx context.Context) error {
	return m.check(ctx, PermissionUpdateSubmission)
}

func (m *middleware) CanDeleteSubmission(ctx context.Context) error {
	return m.check(ctx, PermissionDeleteSubmission)
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
