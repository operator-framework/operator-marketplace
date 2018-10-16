package phase

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogSourceConfigBuilder builds a new CatalogSourceConfig type object.
type CatalogSourceConfigBuilder struct {
	object v1alpha1.CatalogSourceConfig
}

// CatalogSourceConfig returns a prepared CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) CatalogSourceConfig() *v1alpha1.CatalogSourceConfig {
	return &b.object
}

// WithMeta sets TypeMeta and ObjectMeta accordingly.
func (b *CatalogSourceConfigBuilder) WithMeta(namespace, name string) *CatalogSourceConfigBuilder {
	b.object.TypeMeta = metav1.TypeMeta{
		APIVersion: fmt.Sprintf("%s/%s",
			v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
		Kind: v1alpha1.CatalogSourceConfigKind,
	}

	b.object.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}

	return b
}

// WithOwner sets the owner of the CatalogSourceConfig object to the given owner.
func (b *CatalogSourceConfigBuilder) WithOwner(owner *v1alpha1.OperatorSource) *CatalogSourceConfigBuilder {
	trueVar := true
	ownerReference := metav1.OwnerReference{
		APIVersion: owner.APIVersion,
		Kind:       owner.Kind,
		Name:       owner.Name,
		UID:        owner.UID,
		Controller: &trueVar,
	}
	ownerReferences := append(b.object.GetOwnerReferences(), ownerReference)
	b.object.SetOwnerReferences(ownerReferences)

	return b
}

// WithSpec sets Spec accordingly.
func (b *CatalogSourceConfigBuilder) WithSpec(targetNamespace string, packages string) *CatalogSourceConfigBuilder {
	b.object.Spec = v1alpha1.CatalogSourceConfigSpec{
		TargetNamespace: targetNamespace,
		Packages:        packages,
	}

	return b
}
