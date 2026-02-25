package authz

const (
	NamespaceTenant  = "trustage_tenant"
	NamespaceProfile = "profile"
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
