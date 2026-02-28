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
	PermissionFormDefinitionManage = "form_definition_manage"
	PermissionFormDefinitionView   = "form_definition_view"
	PermissionFormSubmit           = "form_submit"
	PermissionSubmissionView       = "submission_view"
	PermissionSubmissionUpdate     = "submission_update"
	PermissionSubmissionDelete     = "submission_delete"
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
			PermissionFormDefinitionManage, PermissionFormDefinitionView,
			PermissionFormSubmit, PermissionSubmissionView,
			PermissionSubmissionUpdate, PermissionSubmissionDelete,
		},
		RoleAdmin: {
			PermissionFormDefinitionManage, PermissionFormDefinitionView,
			PermissionFormSubmit, PermissionSubmissionView,
			PermissionSubmissionUpdate, PermissionSubmissionDelete,
		},
		RoleMember: {
			PermissionFormDefinitionView, PermissionFormSubmit,
			PermissionSubmissionView,
		},
		RoleService: {
			PermissionFormDefinitionManage, PermissionFormDefinitionView,
			PermissionFormSubmit, PermissionSubmissionView,
			PermissionSubmissionUpdate, PermissionSubmissionDelete,
		},
	}
}
