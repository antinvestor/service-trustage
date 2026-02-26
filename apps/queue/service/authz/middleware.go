package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanManageQueue(ctx context.Context) error
	CanViewQueue(ctx context.Context) error
	CanEnqueueItem(ctx context.Context) error
	CanViewQueueItem(ctx context.Context) error
	CanManageCounter(ctx context.Context) error
	CanViewStats(ctx context.Context) error
}

type middleware struct {
	authorizer security.Authorizer
}

func NewMiddleware(authorizer security.Authorizer) Middleware {
	return &middleware{authorizer: authorizer}
}

func (m *middleware) CanManageQueue(ctx context.Context) error {
	return m.check(ctx, PermissionManageQueue)
}

func (m *middleware) CanViewQueue(ctx context.Context) error {
	return m.check(ctx, PermissionViewQueue)
}

func (m *middleware) CanEnqueueItem(ctx context.Context) error {
	return m.check(ctx, PermissionEnqueueItem)
}

func (m *middleware) CanViewQueueItem(ctx context.Context) error {
	return m.check(ctx, PermissionViewQueueItem)
}

func (m *middleware) CanManageCounter(ctx context.Context) error {
	return m.check(ctx, PermissionManageCounter)
}

func (m *middleware) CanViewStats(ctx context.Context) error {
	return m.check(ctx, PermissionViewStats)
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
