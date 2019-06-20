package builders

import (
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

// WithOpsrcOwnerLabel sets the owner label of the Role object to the given owner.
func (b *RoleBuilder) WithOpsrcOwnerLabel(name, namespace string) *RoleBuilder {
	labels := map[string]string{
		OpsrcOwnerNameLabel:      name,
		OpsrcOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.role.GetLabels() {
		labels[key] = value
	}

	b.role.SetLabels(labels)
	return b
}

// WithCscOwnerLabel sets the owner label of the Role object to the given owner.
func (b *RoleBuilder) WithCscOwnerLabel(name, namespace string) *RoleBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      name,
		CscOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.role.GetLabels() {
		labels[key] = value
	}

	b.role.SetLabels(labels)
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
