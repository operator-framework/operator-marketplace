package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountBuilder builds a new ServiceAccount object.
type ServiceAccountBuilder struct {
	sa core.ServiceAccount
}

// ServiceAccount returns a ServiceAccount object.
func (b *ServiceAccountBuilder) ServiceAccount() *core.ServiceAccount {
	return &b.sa
}

// WithTypeMeta sets basic TypeMeta.
func (b *ServiceAccountBuilder) WithTypeMeta() *ServiceAccountBuilder {
	b.sa.TypeMeta = meta.TypeMeta{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *ServiceAccountBuilder) WithMeta(name, namespace string) *ServiceAccountBuilder {
	b.WithTypeMeta()
	if b.sa.GetObjectMeta() == nil {
		b.sa.ObjectMeta = meta.ObjectMeta{}
	}
	b.sa.SetName(name)
	b.sa.SetNamespace(namespace)
	return b
}

// WithOwner sets the owner of the ServiceAccount object to the given owner.
func (b *ServiceAccountBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *ServiceAccountBuilder {
	b.sa.SetOwnerReferences(append(b.sa.GetOwnerReferences(),
		[]meta.OwnerReference{
			*meta.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}
