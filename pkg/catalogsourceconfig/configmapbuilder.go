package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapBuilder builds a new CatalogSource object.
type ConfigMapBuilder struct {
	cm corev1.ConfigMap
}

// ConfigMap returns a ConfigMap object.
func (b *ConfigMapBuilder) ConfigMap() *corev1.ConfigMap {
	return &b.cm
}

// WithTypeMeta sets TypeMeta.
func (b *ConfigMapBuilder) WithTypeMeta() *ConfigMapBuilder {
	b.cm.TypeMeta = metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
	return b
}

// WithMeta sets TypeMeta and ObjectMeta.
func (b *ConfigMapBuilder) WithMeta(name, namespace string) *ConfigMapBuilder {
	b.WithTypeMeta()
	b.cm.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// WithOwner sets the owner of the CatalogSource object to the given owner.
func (b *ConfigMapBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *ConfigMapBuilder {
	b.cm.SetOwnerReferences(append(b.cm.GetOwnerReferences(),
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithData sets the Data.
func (b *ConfigMapBuilder) WithData(data map[string]string) *ConfigMapBuilder {
	b.cm.Data = data
	return b
}
