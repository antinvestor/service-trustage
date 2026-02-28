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
	PermissionManageFormDefinition = "manage_form_definition"
	PermissionViewFormDefinition   = "view_form_definition"
	PermissionSubmitForm           = "submit_form"
	PermissionViewSubmission       = "view_submission"
	PermissionUpdateSubmission     = "update_submission"
	PermissionDeleteSubmission     = "delete_submission"
)

// RolePermissions maps each role to the permissions it grants.
func RolePermissions() map[string][]string {
	return map[string][]string{
		RoleOwner: {
			PermissionManageFormDefinition, PermissionViewFormDefinition,
			PermissionSubmitForm, PermissionViewSubmission,
			PermissionUpdateSubmission, PermissionDeleteSubmission,
		},
		RoleAdmin: {
			PermissionManageFormDefinition, PermissionViewFormDefinition,
			PermissionSubmitForm, PermissionViewSubmission,
			PermissionUpdateSubmission, PermissionDeleteSubmission,
		},
		RoleMember: {
			PermissionViewFormDefinition, PermissionSubmitForm,
			PermissionViewSubmission,
		},
		RoleService: {
			PermissionManageFormDefinition, PermissionViewFormDefinition,
			PermissionSubmitForm, PermissionViewSubmission,
			PermissionUpdateSubmission, PermissionDeleteSubmission,
		},
	}
}
