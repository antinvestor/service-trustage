package authz

const (
	NamespaceProfile       = "service_trustage"
	NamespaceTenancyAccess = "tenancy_access"
	NamespaceProfileUser   = "profile_user"
)

const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleMember  = "member"
	RoleService = "service"
)

const (
	PermissionEventIngest    = "event_ingest"
	PermissionWorkflowManage = "workflow_manage"
	PermissionWorkflowView   = "workflow_view"
	PermissionInstanceView   = "instance_view"
	PermissionInstanceRetry  = "instance_retry"
	PermissionInstanceSignal = "instance_signal"
	PermissionExecutionView  = "execution_view"
	PermissionExecutionRetry = "execution_retry"
)

// GrantedRelation returns the OPL relation name for a direct grant.
// Direct grant relations are prefixed with "granted_" to avoid name conflicts
// with permit functions in Keto OPL (Keto skips permit evaluation when a
// relation shares the same name as a permit function).
func GrantedRelation(permission string) string {
	return "granted_" + permission
}

// RolePermissions maps each role to the permissions it grants.
func RolePermissions() map[string][]string {
	return map[string][]string{
		RoleOwner: {
			PermissionEventIngest, PermissionWorkflowManage, PermissionWorkflowView,
			PermissionInstanceView, PermissionInstanceRetry, PermissionInstanceSignal,
			PermissionExecutionView, PermissionExecutionRetry,
		},
		RoleAdmin: {
			PermissionEventIngest, PermissionWorkflowManage, PermissionWorkflowView,
			PermissionInstanceView, PermissionInstanceRetry, PermissionInstanceSignal,
			PermissionExecutionView, PermissionExecutionRetry,
		},
		RoleMember: {
			PermissionEventIngest, PermissionWorkflowView,
			PermissionInstanceView, PermissionInstanceSignal, PermissionExecutionView,
		},
		RoleService: {
			PermissionEventIngest, PermissionWorkflowManage, PermissionWorkflowView,
			PermissionInstanceView, PermissionInstanceRetry, PermissionInstanceSignal,
			PermissionExecutionView, PermissionExecutionRetry,
		},
	}
}
