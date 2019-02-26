package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodTemplateBuilder builds a new CatalogSource object.
type PodTemplateBuilder struct {
	pt core.PodTemplateSpec
}

// PodTemplate returns a PodTemplate object.
func (b *PodTemplateBuilder) PodTemplate() *core.PodTemplateSpec {
	return &b.pt
}

// WithObjectMeta sets ObjectMeta.
func (b *PodTemplateBuilder) WithObjectMeta(name, namespace string) *PodTemplateBuilder {
	b.pt.ObjectMeta = meta.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// WithOwnerLabel sets the owner label of the PodTemplate object to the given owner.
func (b *PodTemplateBuilder) WithOwnerLabel(owner *v1alpha1.CatalogSourceConfig) *PodTemplateBuilder {
	labels := map[string]string{
		CscOwnerNameLabel:      owner.Name,
		CscOwnerNamespaceLabel: owner.Namespace,
	}

	for key, value := range b.pt.GetLabels() {
		labels[key] = value
	}

	b.pt.SetLabels(labels)
	return b
}

// WithPodSpec sets Spec in the PodTemplate object
func (b *PodTemplateBuilder) WithPodSpec(podSpec core.PodSpec) *PodTemplateBuilder {
	b.pt.Spec = podSpec
	return b
}
