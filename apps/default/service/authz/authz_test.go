package authz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
)

type fakeAuthorizer struct {
	lastRequest security.CheckRequest
	allowed     bool
	err         error
}

func (f *fakeAuthorizer) Check(_ context.Context, req security.CheckRequest) (security.CheckResult, error) {
	f.lastRequest = req
	if f.err != nil {
		return security.CheckResult{}, f.err
	}

	return security.CheckResult{Allowed: f.allowed}, nil
}

func (*fakeAuthorizer) BatchCheck(context.Context, []security.CheckRequest) ([]security.CheckResult, error) {
	return nil, nil
}
func (*fakeAuthorizer) WriteTuple(context.Context, security.RelationTuple) error { return nil }
func (*fakeAuthorizer) WriteTuples(context.Context, []security.RelationTuple) error {
	return nil
}
func (*fakeAuthorizer) DeleteTuple(context.Context, security.RelationTuple) error { return nil }
func (*fakeAuthorizer) DeleteTuples(context.Context, []security.RelationTuple) error {
	return nil
}
func (*fakeAuthorizer) ListRelations(context.Context, security.ObjectRef) ([]security.RelationTuple, error) {
	return nil, nil
}

func (*fakeAuthorizer) ListSubjectRelations(
	context.Context,
	security.SubjectRef,
	string,
) ([]security.RelationTuple, error) {
	return nil, nil
}
func (*fakeAuthorizer) Expand(context.Context, security.ObjectRef, string) ([]security.SubjectRef, error) {
	return nil, nil
}

func authCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "tenant-a",
		PartitionID: "partition-b",
	}
	claims.Subject = "profile-c"
	return claims.ClaimsToContext(context.Background())
}

func TestGrantedRelationAndRolePermissions(t *testing.T) {
	t.Parallel()

	t.Run("granted relation prefixes permission", func(t *testing.T) {
		t.Parallel()
		if got := authz.GrantedRelation(authz.PermissionExecutionRetry); got != "granted_execution_retry" {
			t.Fatalf("GrantedRelation() = %q", got)
		}
	})

	t.Run("all roles expose expected permissions", func(t *testing.T) {
		t.Parallel()

		perms := authz.RolePermissions()
		cases := []struct {
			role       string
			minimumLen int
			mustHave   string
		}{
			{role: authz.RoleOwner, minimumLen: 8, mustHave: authz.PermissionWorkflowManage},
			{role: authz.RoleAdmin, minimumLen: 8, mustHave: authz.PermissionInstanceRetry},
			{role: authz.RoleMember, minimumLen: 5, mustHave: authz.PermissionInstanceSignal},
			{role: authz.RoleService, minimumLen: 8, mustHave: authz.PermissionExecutionView},
		}

		for _, tc := range cases {

			t.Run(tc.role, func(t *testing.T) {
				t.Parallel()
				rolePerms := perms[tc.role]
				if len(rolePerms) < tc.minimumLen {
					t.Fatalf("role %s permissions too short: %v", tc.role, rolePerms)
				}
				found := false
				for _, perm := range rolePerms {
					if perm == tc.mustHave {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("role %s missing permission %s", tc.role, tc.mustHave)
				}
			})
		}
	})
}

func TestBuildTuples(t *testing.T) {
	t.Parallel()

	tenancyPath := "tenant-a/partition-b"
	profileID := "profile-c"

	access := authz.BuildAccessTuple(tenancyPath, profileID)
	if access.Object.Namespace != authz.NamespaceTenancyAccess || access.Relation != "member" {
		t.Fatalf("unexpected access tuple: %+v", access)
	}

	service := authz.BuildServiceAccessTuple(tenancyPath, profileID)
	if service.Relation != authz.RoleService {
		t.Fatalf("unexpected service tuple: %+v", service)
	}

	perm := authz.BuildPermissionTuple(tenancyPath, profileID, authz.PermissionWorkflowView)
	if perm.Object.Namespace != authz.NamespaceProfile || perm.Relation != "granted_workflow_view" {
		t.Fatalf("unexpected permission tuple: %+v", perm)
	}

	inherit := authz.BuildServiceInheritanceTuples(tenancyPath)
	if len(inherit) != 1 || inherit[0].Subject.Relation != authz.RoleService {
		t.Fatalf("unexpected inheritance tuples: %+v", inherit)
	}
}

func TestMiddlewareChecks(t *testing.T) {
	t.Parallel()

	type checkerCall struct {
		name       string
		call       func(authz.Middleware, context.Context) error
		permission string
	}

	cases := []checkerCall{
		{
			name:       "event ingest",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanEventIngest(ctx) },
			permission: authz.PermissionEventIngest,
		},
		{
			name:       "workflow manage",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanWorkflowManage(ctx) },
			permission: authz.PermissionWorkflowManage,
		},
		{
			name:       "workflow view",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanWorkflowView(ctx) },
			permission: authz.PermissionWorkflowView,
		},
		{
			name:       "instance view",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanInstanceView(ctx) },
			permission: authz.PermissionInstanceView,
		},
		{
			name:       "instance retry",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanInstanceRetry(ctx) },
			permission: authz.PermissionInstanceRetry,
		},
		{
			name:       "instance signal",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanInstanceSignal(ctx) },
			permission: authz.PermissionInstanceSignal,
		},
		{
			name:       "execution view",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanExecutionView(ctx) },
			permission: authz.PermissionExecutionView,
		},
		{
			name:       "execution retry",
			call:       func(m authz.Middleware, ctx context.Context) error { return m.CanExecutionRetry(ctx) },
			permission: authz.PermissionExecutionRetry,
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeAuthorizer{allowed: true}
			middleware := authz.NewMiddleware(fake)
			if err := tc.call(middleware, authCtx()); err != nil {
				t.Fatalf("check failed: %v", err)
			}

			if fake.lastRequest.Permission != tc.permission {
				t.Fatalf("permission = %q want %q", fake.lastRequest.Permission, tc.permission)
			}
			if fake.lastRequest.Object.Namespace != authz.NamespaceProfile {
				t.Fatalf("object namespace = %q", fake.lastRequest.Object.Namespace)
			}
		})
	}
}

func TestMiddlewareBubblesErrors(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("backend down")
	middleware := authz.NewMiddleware(&fakeAuthorizer{err: backendErr})
	if err := middleware.CanWorkflowView(authCtx()); !errors.Is(err, backendErr) {
		t.Fatalf("CanWorkflowView() error = %v", err)
	}
}

func TestMiddlewareRejectsMissingClaims(t *testing.T) {
	t.Parallel()

	middleware := authz.NewMiddleware(&fakeAuthorizer{allowed: true})
	if err := middleware.CanEventIngest(context.Background()); err == nil {
		t.Fatal("expected missing claims error")
	}
}
