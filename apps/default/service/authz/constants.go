package authz

const (
	NamespaceProfile       = "service_trustage"
	NamespaceTenancyAccess = "tenancy_access"
	NamespaceProfileUser   = "profile/user"
)

const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleMember  = "member"
	RoleService = "service"
)

const (
	PermissionIngestEvent    = "ingest_event"
	PermissionManageWorkflow = "manage_workflow"
	PermissionViewWorkflow   = "view_workflow"
	PermissionViewInstance   = "view_instance"
	PermissionRetryInstance  = "retry_instance"
	PermissionViewExecution  = "view_execution"
	PermissionRetryExecution = "retry_execution"
)

// RolePermissions maps each role to the permissions it grants.
func RolePermissions() map[string][]string {
	return map[string][]string{
		RoleOwner: {
			PermissionIngestEvent, PermissionManageWorkflow, PermissionViewWorkflow,
			PermissionViewInstance, PermissionRetryInstance,
			PermissionViewExecution, PermissionRetryExecution,
		},
		RoleAdmin: {
			PermissionIngestEvent, PermissionManageWorkflow, PermissionViewWorkflow,
			PermissionViewInstance, PermissionRetryInstance,
			PermissionViewExecution, PermissionRetryExecution,
		},
		RoleMember: {
			PermissionIngestEvent, PermissionViewWorkflow,
			PermissionViewInstance, PermissionViewExecution,
		},
		RoleService: {
			PermissionIngestEvent, PermissionManageWorkflow, PermissionViewWorkflow,
			PermissionViewInstance, PermissionRetryInstance,
			PermissionViewExecution, PermissionRetryExecution,
		},
	}
}
