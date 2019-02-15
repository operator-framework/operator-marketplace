package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceBuilder builds a new CatalogSource object.
type ServiceBuilder struct {
	service core.Service
}

// Service returns a Service object.
func (b *ServiceBuilder) Service() *core.Service {
	return &b.service
}

// WithTypeMeta sets TypeMeta.
func (b *ServiceBuilder) WithTypeMeta() *ServiceBuilder {
	b.service.TypeMeta = metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets TypeMeta and ObjectMeta.
func (b *ServiceBuilder) WithMeta(name, namespace string) *ServiceBuilder {
	b.WithTypeMeta()
	b.service.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// WithOwner sets the owner of the CatalogSource object to the given owner.
func (b *ServiceBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *ServiceBuilder {
	b.service.SetOwnerReferences(append(b.service.GetOwnerReferences(),
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithSpec sets the Data.
func (b *ServiceBuilder) WithSpec(spec core.ServiceSpec) *ServiceBuilder {
	b.service.Spec = spec
	return b
}
