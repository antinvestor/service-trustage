package authz

const (
	NamespaceTenant  = "trustage_tenant"
	NamespaceProfile = "profile"
)

const (
	PermissionManageQueue   = "manage_queue"
	PermissionViewQueue     = "view_queue"
	PermissionEnqueueItem   = "enqueue_item"
	PermissionViewQueueItem = "view_queue_item"
	PermissionManageCounter = "manage_counter"
	PermissionViewStats     = "view_stats"
)

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)
