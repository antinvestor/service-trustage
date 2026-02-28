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
	PermissionManageQueue   = "manage_queue"
	PermissionViewQueue     = "view_queue"
	PermissionEnqueueItem   = "enqueue_item"
	PermissionViewQueueItem = "view_queue_item"
	PermissionManageCounter = "manage_counter"
	PermissionViewStats     = "view_stats"
)

// RolePermissions maps each role to the permissions it grants.
func RolePermissions() map[string][]string {
	return map[string][]string{
		RoleOwner: {
			PermissionManageQueue, PermissionViewQueue,
			PermissionEnqueueItem, PermissionViewQueueItem,
			PermissionManageCounter, PermissionViewStats,
		},
		RoleAdmin: {
			PermissionManageQueue, PermissionViewQueue,
			PermissionEnqueueItem, PermissionViewQueueItem,
			PermissionManageCounter, PermissionViewStats,
		},
		RoleMember: {
			PermissionViewQueue, PermissionEnqueueItem,
			PermissionViewQueueItem, PermissionViewStats,
		},
		RoleService: {
			PermissionManageQueue, PermissionViewQueue,
			PermissionEnqueueItem, PermissionViewQueueItem,
			PermissionManageCounter, PermissionViewStats,
		},
	}
}
