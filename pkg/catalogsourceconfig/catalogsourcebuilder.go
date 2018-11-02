package catalogsourceconfig

import (
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogSourceBuilder builds a new CatalogSource object.
type CatalogSourceBuilder struct {
	cs olm.CatalogSource
}

// CatalogSource returns a CatalogSource object.
func (b *CatalogSourceBuilder) CatalogSource() *olm.CatalogSource {
	return &b.cs
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *CatalogSourceBuilder) WithMeta(name, namespace string) *CatalogSourceBuilder {
	b.cs.TypeMeta = metav1.TypeMeta{
		Kind:       olm.CatalogSourceKind,
		APIVersion: olm.CatalogSourceCRDAPIVersion,
	}
	b.cs.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// WithOwner sets the owner of the CatalogSource object to the given owner.
func (b *CatalogSourceBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *CatalogSourceBuilder {
	b.cs.SetOwnerReferences(append(b.cs.GetOwnerReferences(),
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithSpec sets Spec with input data.
func (b *CatalogSourceBuilder) WithSpec(csType, cmName, displayName, publisher string) *CatalogSourceBuilder {
	b.cs.Spec = olm.CatalogSourceSpec{
		SourceType:  csType,
		ConfigMap:   cmName,
		DisplayName: displayName,
		Publisher:   publisher,
	}
	return b
}
