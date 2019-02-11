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

// WithOwner sets the owner of the PodTemplate object to the given owner.
func (b *PodTemplateBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *PodTemplateBuilder {
	b.pt.SetOwnerReferences(append(b.pt.GetOwnerReferences(),
		[]meta.OwnerReference{
			*meta.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithPodSpec sets Spec in the PodTemplate object
func (b *PodTemplateBuilder) WithPodSpec(podSpec core.PodSpec) *PodTemplateBuilder {
	b.pt.Spec = podSpec
	return b
}
