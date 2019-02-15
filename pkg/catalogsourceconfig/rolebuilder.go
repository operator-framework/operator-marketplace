package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBuilder builds a new Role object.
type RoleBuilder struct {
	role rbac.Role
}

// Role returns a Role object.
func (b *RoleBuilder) Role() *rbac.Role {
	return &b.role
}

// WithTypeMeta sets basic TypeMeta.
func (b *RoleBuilder) WithTypeMeta() *RoleBuilder {
	b.role.TypeMeta = meta.TypeMeta{
		Kind:       "Role",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *RoleBuilder) WithMeta(name, namespace string) *RoleBuilder {
	b.WithTypeMeta()
	if b.role.GetObjectMeta() == nil {
		b.role.ObjectMeta = meta.ObjectMeta{}
	}
	b.role.SetName(name)
	b.role.SetNamespace(namespace)
	return b
}

// WithOwner sets the owner of the Role object to the given owner.
func (b *RoleBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *RoleBuilder {
	b.role.SetOwnerReferences(append(b.role.GetOwnerReferences(),
		[]meta.OwnerReference{
			*meta.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithRules sets the rules for the roles
func (b *RoleBuilder) WithRules(rules []rbac.PolicyRule) *RoleBuilder {
	b.role.Rules = rules
	return b
}

// NewRule returns PolicyRule objects
func NewRule(verbs, apiGroups, resources, resourceNames []string) rbac.PolicyRule {
	return rbac.PolicyRule{
		Verbs:         verbs,
		APIGroups:     apiGroups,
		Resources:     resources,
		ResourceNames: resourceNames,
	}
}
