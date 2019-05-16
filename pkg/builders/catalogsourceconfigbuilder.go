package builders

import (
	"fmt"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogSourceConfigBuilder builds a new CatalogSourceConfig type object.
type CatalogSourceConfigBuilder struct {
	Object marketplace.CatalogSourceConfig
}

// CatalogSourceConfig returns a prepared CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) CatalogSourceConfig() *marketplace.CatalogSourceConfig {
	return &b.Object
}

// WithTypeMeta sets TypeMeta of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithTypeMeta() *CatalogSourceConfigBuilder {
	b.Object.TypeMeta = metav1.TypeMeta{
		APIVersion: fmt.Sprintf("%s/%s",
			marketplace.SchemeGroupVersion.Group, marketplace.SchemeGroupVersion.Version),
		Kind: marketplace.CatalogSourceConfigKind,
	}

	return b
}

// WithNamespacedName sets name and namespace of the CatalogSourceConfig object.
func (b *CatalogSourceConfigBuilder) WithNamespacedName(namespace, name string) *CatalogSourceConfigBuilder {
	b.Object.SetNamespace(namespace)
	b.Object.SetName(name)

	return b
}

// WithLabels sets appropriate labels for the CatalogSourceConfig object. It
// applies all labels associated with an OperatorSource object specified in
// opsrcLabels.
func (b *CatalogSourceConfigBuilder) WithLabels(opsrcLabels map[string]string) *CatalogSourceConfigBuilder {
	labels := map[string]string{
		datastore.DatastoreLabel: "true",
	}

	for key, value := range opsrcLabels {
		labels[key] = value
	}

	for key, value := range b.Object.GetLabels() {
		labels[key] = value
	}

	b.Object.SetLabels(labels)

	return b
}

// WithOwnerLabel sets the owner label of the CatalogSourceConfig object to the given owner.
func (b *CatalogSourceConfigBuilder) WithOwnerLabel(owner *marketplace.OperatorSource) *CatalogSourceConfigBuilder {
	labels := map[string]string{
		OpsrcOwnerNameLabel:      owner.Name,
		OpsrcOwnerNamespaceLabel: owner.Namespace,
	}

	for key, value := range b.Object.GetLabels() {
		labels[key] = value
	}

	b.Object.SetLabels(labels)
	return b
}

// WithSpec sets Spec accordingly.
func (b *CatalogSourceConfigBuilder) WithSpec(targetNamespace, packages, displayName, publisher string) *CatalogSourceConfigBuilder {
	b.Object.Spec = marketplace.CatalogSourceConfigSpec{
		TargetNamespace: targetNamespace,
		Packages:        packages,
		DisplayName:     displayName,
		Publisher:       publisher,
	}

	return b
}
