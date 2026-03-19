package authz_test

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/formstore/service/authz"
)

func TestGrantedRelationAndRolePermissions(t *testing.T) {
	t.Parallel()

	if got := authz.GrantedRelation(authz.PermissionFormSubmit); got != "granted_form_submit" {
		t.Fatalf("GrantedRelation() = %q", got)
	}

	perms := authz.RolePermissions()
	cases := []struct {
		role     string
		mustHave string
	}{
		{role: authz.RoleOwner, mustHave: authz.PermissionSubmissionDelete},
		{role: authz.RoleAdmin, mustHave: authz.PermissionSubmissionUpdate},
		{role: authz.RoleMember, mustHave: authz.PermissionFormSubmit},
		{role: authz.RoleService, mustHave: authz.PermissionFormDefinitionManage},
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

	if got := authz.BuildAccessTuple(tenancyPath, profileID); got.Relation != "member" {
		t.Fatalf("BuildAccessTuple() = %+v", got)
	}

	if got := authz.BuildServiceAccessTuple(tenancyPath, profileID); got.Relation != authz.RoleService {
		t.Fatalf("BuildServiceAccessTuple() = %+v", got)
	}

	if got := authz.BuildPermissionTuple(
		tenancyPath,
		profileID,
		authz.PermissionSubmissionView,
	); got.Relation != "granted_submission_view" {
		t.Fatalf("BuildPermissionTuple() = %+v", got)
	}
}

func TestMiddlewareInstantiation(t *testing.T) {
	t.Parallel()

	if authz.NewMiddleware(&fakeAuthorizer{}) == nil {
		t.Fatal("expected middleware")
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
		{name: "manage_definition", call: mw.CanFormDefinitionManage},
		{name: "view_definition", call: mw.CanFormDefinitionView},
		{name: "submit_form", call: mw.CanFormSubmit},
		{name: "view_submission", call: mw.CanSubmissionView},
		{name: "update_submission", call: mw.CanSubmissionUpdate},
		{name: "delete_submission", call: mw.CanSubmissionDelete},
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
