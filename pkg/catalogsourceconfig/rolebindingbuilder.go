package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBindingBuilder builds a new RoleBinding object.
type RoleBindingBuilder struct {
	rb rbac.RoleBinding
}

// RoleBinding returns a RoleBinding object.
func (b *RoleBindingBuilder) RoleBinding() *rbac.RoleBinding {
	return &b.rb
}

// WithTypeMeta sets basic TypeMeta.
func (b *RoleBindingBuilder) WithTypeMeta() *RoleBindingBuilder {
	b.rb.TypeMeta = meta.TypeMeta{
		Kind:       "RoleBinding",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *RoleBindingBuilder) WithMeta(name, namespace string) *RoleBindingBuilder {
	b.WithTypeMeta()
	if b.rb.GetObjectMeta() == nil {
		b.rb.ObjectMeta = meta.ObjectMeta{}
	}
	b.rb.SetName(name)
	b.rb.SetNamespace(namespace)
	return b
}

// WithOwner sets the owner of the RoleBinding object to the given owner.
func (b *RoleBindingBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *RoleBindingBuilder {
	b.rb.SetOwnerReferences(append(b.rb.GetOwnerReferences(),
		[]meta.OwnerReference{
			*meta.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithSubjects sets the Subjects for the RoleBinding
func (b *RoleBindingBuilder) WithSubjects(subjects []rbac.Subject) *RoleBindingBuilder {
	b.rb.Subjects = subjects
	return b
}

// WithRoleRef sets the rules for the RoleBinding
func (b *RoleBindingBuilder) WithRoleRef(roleName string) *RoleBindingBuilder {
	b.rb.RoleRef = NewRoleRef(roleName)
	return b
}

// NewRoleRef returns a new RoleRef object
func NewRoleRef(roleName string) rbac.RoleRef {
	return rbac.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Role",
		Name:     roleName,
	}
}
