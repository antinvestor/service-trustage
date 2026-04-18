// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
