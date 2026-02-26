package tests_test

import (
	"context"

	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
)

type fakeAuthorizer struct {
	allow   bool
	err     error
	lastReq security.CheckRequest
}

func (f *fakeAuthorizer) Check(_ context.Context, req security.CheckRequest) (security.CheckResult, error) {
	f.lastReq = req
	if f.err != nil {
		return security.CheckResult{}, f.err
	}
	return security.CheckResult{Allowed: f.allow, Reason: "test"}, nil
}
func (f *fakeAuthorizer) BatchCheck(_ context.Context, _ []security.CheckRequest) ([]security.CheckResult, error) {
	return nil, nil
}
func (f *fakeAuthorizer) WriteTuple(_ context.Context, _ security.RelationTuple) error    { return nil }
func (f *fakeAuthorizer) WriteTuples(_ context.Context, _ []security.RelationTuple) error { return nil }
func (f *fakeAuthorizer) DeleteTuple(_ context.Context, _ security.RelationTuple) error   { return nil }
func (f *fakeAuthorizer) DeleteTuples(_ context.Context, _ []security.RelationTuple) error {
	return nil
}
func (f *fakeAuthorizer) ListRelations(_ context.Context, _ security.ObjectRef) ([]security.RelationTuple, error) {
	return nil, nil
}

func (f *fakeAuthorizer) ListSubjectRelations(
	_ context.Context,
	_ security.SubjectRef,
	_ string,
) ([]security.RelationTuple, error) {
	return nil, nil
}
func (f *fakeAuthorizer) Expand(_ context.Context, _ security.ObjectRef, _ string) ([]security.SubjectRef, error) {
	return nil, nil
}

func (s *DefaultServiceSuite) TestAuthzMiddleware_AllowsAndDenies() {
	ctx := s.tenantCtx()

	authorizerClient := &fakeAuthorizer{allow: true}
	mw := authz.NewMiddleware(authorizerClient)

	err := mw.CanViewWorkflow(ctx)
	s.Require().NoError(err)
	s.Equal(authz.PermissionViewWorkflow, authorizerClient.lastReq.Permission)

	s.Require().NoError(mw.CanIngestEvent(ctx))
	s.Require().NoError(mw.CanManageWorkflow(ctx))
	s.Require().NoError(mw.CanViewInstance(ctx))
	s.Require().NoError(mw.CanRetryInstance(ctx))
	s.Require().NoError(mw.CanViewExecution(ctx))
	s.Require().NoError(mw.CanRetryExecution(ctx))

	denyClient := &fakeAuthorizer{allow: false}
	mw = authz.NewMiddleware(denyClient)
	err = mw.CanViewWorkflow(ctx)
	s.Require().Error(err)
	s.ErrorIs(err, authorizer.ErrPermissionDenied)
}

func (s *DefaultServiceSuite) TestAuthzMiddleware_InvalidClaims() {
	authorizerClient := &fakeAuthorizer{allow: true}
	mw := authz.NewMiddleware(authorizerClient)

	// Missing claims should fail with invalid subject.
	err := mw.CanViewWorkflow(context.Background())
	s.Require().Error(err)
	s.Require().ErrorIs(err, authorizer.ErrInvalidSubject)

	// Missing tenant should fail with invalid object.
	claims := &security.AuthenticationClaims{}
	claims.Subject = "user-1"
	ctx := claims.ClaimsToContext(context.Background())

	err = mw.CanViewWorkflow(ctx)
	s.Require().Error(err)
	s.Require().ErrorIs(err, authorizer.ErrInvalidObject)
}
