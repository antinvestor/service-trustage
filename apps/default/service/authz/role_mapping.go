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

import "github.com/pitabwire/frame/security"

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

// GrantedRelation returns the OPL relation name for a direct grant.
func GrantedRelation(permission string) string {
	return "granted_" + permission
}

// BuildAccessTuple creates a tenancy_access#member tuple for a regular user.
func BuildAccessTuple(tenancyPath, profileID string) security.RelationTuple {
	return security.RelationTuple{
		Object:   security.ObjectRef{Namespace: NamespaceTenancyAccess, ID: tenancyPath},
		Relation: "member",
		Subject:  security.SubjectRef{Namespace: NamespaceProfileUser, ID: profileID},
	}
}

// BuildServiceAccessTuple creates a tenancy_access#service tuple for a service bot.
func BuildServiceAccessTuple(tenancyPath, profileID string) security.RelationTuple {
	return security.RelationTuple{
		Object:   security.ObjectRef{Namespace: NamespaceTenancyAccess, ID: tenancyPath},
		Relation: "service",
		Subject:  security.SubjectRef{Namespace: NamespaceProfileUser, ID: profileID},
	}
}

// BuildPermissionTuple creates a service_trustage#granted_<permission> tuple
// for a direct grant. The relation is prefixed with "granted_" to avoid name
// conflicts with OPL permit functions.
func BuildPermissionTuple(tenancyPath, profileID, permission string) security.RelationTuple {
	return security.RelationTuple{
		Object:   security.ObjectRef{Namespace: NamespaceProfile, ID: tenancyPath},
		Relation: GrantedRelation(permission),
		Subject:  security.SubjectRef{Namespace: NamespaceProfileUser, ID: profileID},
	}
}

// BuildServiceInheritanceTuples creates the bridge tuple that allows service bots
// (who have tenancy_access#service) to inherit functional permissions in
// service_trustage via subject sets.
// Only the bridge tuple is needed — the OPL permits already check the service
// role directly, so explicit granted_* tuples per permission are redundant.
func BuildServiceInheritanceTuples(tenancyPath string) []security.RelationTuple {
	return []security.RelationTuple{{
		Object:   security.ObjectRef{Namespace: NamespaceProfile, ID: tenancyPath},
		Relation: RoleService,
		Subject: security.SubjectRef{
			Namespace: NamespaceTenancyAccess,
			ID:        tenancyPath,
			Relation:  RoleService,
		},
	}}
}
