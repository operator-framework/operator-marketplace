package operatorsource

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatastoreLabel is the label used in a CatalogSourceConfig to indicate that
// the resulting CatalogSource acts as the datastore for the OperatorSource
// if it is set to "true".
const DatastoreLabel string = "opsrc-datastore"

// CatalogSourceConfigBuilder builds a new CatalogSourceConfig type object.
type CatalogSourceConfigBuilder struct {
	object v1alpha1.CatalogSourceConfig
}

// CatalogSourceConfig returns a prepared CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) CatalogSourceConfig() *v1alpha1.CatalogSourceConfig {
	return &b.object
}

// WithTypeMeta sets TypeMeta of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithTypeMeta() *CatalogSourceConfigBuilder {
	b.object.TypeMeta = metav1.TypeMeta{
		APIVersion: fmt.Sprintf("%s/%s",
			v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
		Kind: v1alpha1.CatalogSourceConfigKind,
	}

	return b
}

// WithNamespacedName sets name and namespace of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithNamespacedName(namespace, name string) *CatalogSourceConfigBuilder {
	b.object.SetNamespace(namespace)
	b.object.SetName(name)

	return b
}

// WithLabels sets appropriate labels for the CatalogSourceConfig object. It
// applies all labels associated with an OperatorSource object specified in
// opsrcLabels.
func (b *CatalogSourceConfigBuilder) WithLabels(opsrcLabels map[string]string) *CatalogSourceConfigBuilder {
	labels := map[string]string{
		DatastoreLabel: "true",
	}

	for key, value := range opsrcLabels {
		labels[key] = value
	}

	b.object.SetLabels(labels)

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
func (b *CatalogSourceConfigBuilder) WithSpec(targetNamespace, packages, displayName, publisher string) *CatalogSourceConfigBuilder {
	b.object.Spec = v1alpha1.CatalogSourceConfigSpec{
		TargetNamespace: targetNamespace,
		Packages:        packages,
		DisplayName:     displayName,
		Publisher:       publisher,
	}

	return b
}
