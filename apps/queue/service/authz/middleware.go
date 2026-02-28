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
	checker *authorizer.FunctionChecker
}

func NewMiddleware(service security.Authorizer) Middleware {
	return &middleware{checker: authorizer.NewFunctionChecker(service, NamespaceProfile)}
}

func (m *middleware) CanManageQueue(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionManageQueue)
}

func (m *middleware) CanViewQueue(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewQueue)
}

func (m *middleware) CanEnqueueItem(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionEnqueueItem)
}

func (m *middleware) CanViewQueueItem(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewQueueItem)
}

func (m *middleware) CanManageCounter(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionManageCounter)
}

func (m *middleware) CanViewStats(ctx context.Context) error {
	return m.checker.Check(ctx, PermissionViewStats)
}
