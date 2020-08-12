package builders

import (
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogSourceBuilder builds a new CatalogSource object.
type CatalogSourceBuilder struct {
	cs operatorsv1alpha1.CatalogSource
}

// CatalogSource returns a CatalogSource object.
func (b *CatalogSourceBuilder) CatalogSource() *operatorsv1alpha1.CatalogSource {
	return &b.cs
}

// WithTypeMeta sets basic TypeMeta.
func (b *CatalogSourceBuilder) WithTypeMeta() *CatalogSourceBuilder {
	b.cs.TypeMeta = metav1.TypeMeta{
		Kind:       operatorsv1alpha1.CatalogSourceKind,
		APIVersion: operatorsv1alpha1.CatalogSourceCRDAPIVersion,
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *CatalogSourceBuilder) WithMeta(name, namespace string) *CatalogSourceBuilder {
	b.WithTypeMeta()
	objectMeta := b.cs.GetObjectMeta()
	if objectMeta == nil {
		b.cs.ObjectMeta = metav1.ObjectMeta{}
	}
	b.cs.SetName(name)
	b.cs.SetNamespace(namespace)
	return b
}

// WithOLMLabels adds "olm-visibility", "openshift-marketplace" and and all
// label(s) associated with the CatalogSource object specified in cscLabels.
func (b *CatalogSourceBuilder) WithOLMLabels(cscLabels map[string]string) *CatalogSourceBuilder {
	labels := map[string]string{
		"olm-visibility":        "hidden",
		"openshift-marketplace": "true",
	}

	for key, value := range cscLabels {
		labels[key] = value
	}

	for key, value := range b.cs.GetLabels() {
		labels[key] = value
	}

	b.WithTypeMeta()
	objectMeta := b.cs.GetObjectMeta()
	if objectMeta == nil {
		b.cs.ObjectMeta = metav1.ObjectMeta{}
	}
	b.cs.SetLabels(labels)
	return b
}

// WithOpsrcOwnerLabel sets the owner label of the CatalogSource object to the given owner.
func (b *CatalogSourceBuilder) WithOpsrcOwnerLabel(name, namespace string) *CatalogSourceBuilder {
	labels := map[string]string{
		OpsrcOwnerNameLabel:      name,
		OpsrcOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.cs.GetLabels() {
		labels[key] = value
	}

	b.cs.SetLabels(labels)
	return b
}

// WithCscOwnerLabel sets the owner label of the CatalogSource object to the given owner.
func (b *CatalogSourceBuilder) WithCscOwnerLabel(name, namespace string) *CatalogSourceBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      name,
		CscOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.cs.GetLabels() {
		labels[key] = value
	}

	b.cs.SetLabels(labels)
	return b
}

// WithSpec sets Spec with input data.
func (b *CatalogSourceBuilder) WithSpec(csType operatorsv1alpha1.SourceType, address, displayName, publisher string) *CatalogSourceBuilder {
	b.cs.Spec = operatorsv1alpha1.CatalogSourceSpec{
		SourceType:  csType,
		Address:     address,
		DisplayName: displayName,
		Publisher:   publisher,
	}
	return b
}
