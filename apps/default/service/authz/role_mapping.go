package authz

import "github.com/pitabwire/frame/security"

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

// BuildServiceInheritanceTuples creates subject set tuples that bridge
// tenancy_access#service → service_trustage#service so that service bots
// gain functional permissions through the same subject set chain.
func BuildServiceInheritanceTuples(tenancyPath string) []security.RelationTuple {
	permissions := RolePermissions()[RoleService]
	tuples := make([]security.RelationTuple, 0, 1+len(permissions))

	tuples = append(tuples, security.RelationTuple{
		Object:   security.ObjectRef{Namespace: NamespaceProfile, ID: tenancyPath},
		Relation: RoleService,
		Subject:  security.SubjectRef{Namespace: NamespaceProfileUser, ID: tenancyPath},
	})

	for _, perm := range permissions {
		tuples = append(tuples, security.RelationTuple{
			Object:   security.ObjectRef{Namespace: NamespaceProfile, ID: tenancyPath},
			Relation: perm,
			Subject:  security.SubjectRef{Namespace: NamespaceProfileUser, ID: tenancyPath},
		})
	}

	return tuples
}
