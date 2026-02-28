package authz

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
)

type Middleware interface {
	CanQueueManage(ctx context.Context) error
	CanQueueView(ctx context.Context) error
	CanItemEnqueue(ctx context.Context) error
	CanQueueItemView(ctx context.Context) error
	CanCounterManage(ctx context.Context) error
	CanStatsView(ctx context.Context) error
}

type middleware struct {
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanQueueManage(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionQueueManage)
}

func (m *middleware) CanQueueView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionQueueView)
}

func (m *middleware) CanItemEnqueue(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionItemEnqueue)
}

func (m *middleware) CanQueueItemView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionQueueItemView)
}

func (m *middleware) CanCounterManage(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionCounterManage)
}

func (m *middleware) CanStatsView(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionStatsView)
}
