package builders

import (
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

// WithOpsrcOwnerLabel sets the owner label of the ServiceAccount object to the given owner.
func (b *ServiceAccountBuilder) WithOpsrcOwnerLabel(name, namespace string) *ServiceAccountBuilder {
	labels := map[string]string{
		OpsrcOwnerNameLabel:      name,
		OpsrcOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.sa.GetLabels() {
		labels[key] = value
	}

	b.sa.SetLabels(labels)
	return b
}

// WithCscOwnerLabel sets the owner label of the ServiceAccount object to the given owner.
func (b *ServiceAccountBuilder) WithCscOwnerLabel(name, namespace string) *ServiceAccountBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      name,
		CscOwnerNamespaceLabel: namespace,
	}
	for key, value := range b.sa.GetLabels() {
		labels[key] = value
	}

	b.sa.SetLabels(labels)
	return b
}
