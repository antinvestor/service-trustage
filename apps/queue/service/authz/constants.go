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
	PermissionQueueManage   = "queue_manage"
	PermissionQueueView     = "queue_view"
	PermissionItemEnqueue   = "item_enqueue"
	PermissionQueueItemView = "queue_item_view"
	PermissionCounterManage = "counter_manage"
	PermissionStatsView     = "stats_view"
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
			PermissionQueueManage, PermissionQueueView,
			PermissionItemEnqueue, PermissionQueueItemView,
			PermissionCounterManage, PermissionStatsView,
		},
		RoleAdmin: {
			PermissionQueueManage, PermissionQueueView,
			PermissionItemEnqueue, PermissionQueueItemView,
			PermissionCounterManage, PermissionStatsView,
		},
		RoleMember: {
			PermissionQueueView, PermissionItemEnqueue,
			PermissionQueueItemView, PermissionStatsView,
		},
		RoleService: {
			PermissionQueueManage, PermissionQueueView,
			PermissionItemEnqueue, PermissionQueueItemView,
			PermissionCounterManage, PermissionStatsView,
		},
	}
}
