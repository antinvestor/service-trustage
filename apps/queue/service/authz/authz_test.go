package authz_test

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/queue/service/authz"
)

func TestGrantedRelationAndRolePermissions(t *testing.T) {
	t.Parallel()

	if got := authz.GrantedRelation(authz.PermissionQueueManage); got != "granted_queue_manage" {
		t.Fatalf("GrantedRelation() = %q", got)
	}

	perms := authz.RolePermissions()
	cases := []struct {
		role     string
		mustHave string
	}{
		{role: authz.RoleOwner, mustHave: authz.PermissionCounterManage},
		{role: authz.RoleAdmin, mustHave: authz.PermissionQueueView},
		{role: authz.RoleMember, mustHave: authz.PermissionItemEnqueue},
		{role: authz.RoleService, mustHave: authz.PermissionStatsView},
	}

	for _, tc := range cases {
		found := false
		for _, perm := range perms[tc.role] {
			if perm == tc.mustHave {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("role %s missing permission %s", tc.role, tc.mustHave)
		}
	}
}

func TestBuildTuples(t *testing.T) {
	t.Parallel()

	tenancyPath := "tenant-a/partition-b"
	profileID := "profile-c"

	access := authz.BuildAccessTuple(tenancyPath, profileID)
	if access.Object.Namespace != authz.NamespaceTenancyAccess ||
		access.Subject.Namespace != authz.NamespaceProfileUser {
		t.Fatalf("BuildAccessTuple() = %+v", access)
	}

	service := authz.BuildServiceAccessTuple(tenancyPath, profileID)
	if service.Relation != authz.RoleService {
		t.Fatalf("BuildServiceAccessTuple() = %+v", service)
	}

	perm := authz.BuildPermissionTuple(tenancyPath, profileID, authz.PermissionQueueItemView)
	if perm.Object != (security.ObjectRef{Namespace: authz.NamespaceProfile, ID: tenancyPath}) {
		t.Fatalf("BuildPermissionTuple() object = %+v", perm.Object)
	}
	if perm.Relation != "granted_queue_item_view" {
		t.Fatalf("BuildPermissionTuple() relation = %q", perm.Relation)
	}
}

func TestMiddlewarePermissionMethods(t *testing.T) {
	t.Parallel()

	mw := authz.NewMiddleware(&fakeAuthorizer{})
	claims := &security.AuthenticationClaims{
		TenantID:    "tenant-a",
		PartitionID: "partition-b",
	}
	claims.Subject = "profile-c"
	ctx := claims.ClaimsToContext(context.Background())

	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{name: "queue_manage", call: mw.CanQueueManage},
		{name: "queue_view", call: mw.CanQueueView},
		{name: "item_enqueue", call: mw.CanItemEnqueue},
		{name: "queue_item_view", call: mw.CanQueueItemView},
		{name: "counter_manage", call: mw.CanCounterManage},
		{name: "stats_view", call: mw.CanStatsView},
	}

	for _, tc := range cases {
		if err := tc.call(ctx); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

type fakeAuthorizer struct{}

func (*fakeAuthorizer) Check(context.Context, security.CheckRequest) (security.CheckResult, error) {
	return security.CheckResult{Allowed: true}, nil
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
