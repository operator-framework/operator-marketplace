package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentBuilder builds a new Deployment object.
type DeploymentBuilder struct {
	deployment apps.Deployment
}

// Deployment returns a Deployment object.
func (b *DeploymentBuilder) Deployment() *apps.Deployment {
	return &b.deployment
}

// WithTypeMeta sets basic TypeMeta.
func (b *DeploymentBuilder) WithTypeMeta() *DeploymentBuilder {
	b.deployment.TypeMeta = meta.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}
	return b
}

// WithMeta sets basic TypeMeta and ObjectMeta.
func (b *DeploymentBuilder) WithMeta(name, namespace string) *DeploymentBuilder {
	b.WithTypeMeta()
	if b.deployment.GetObjectMeta() == nil {
		b.deployment.ObjectMeta = meta.ObjectMeta{}
	}
	b.deployment.SetName(name)
	b.deployment.SetNamespace(namespace)
	return b
}

// WithOwner sets the owner of the Deployment object to the given owner.
func (b *DeploymentBuilder) WithOwner(owner *v1alpha1.CatalogSourceConfig) *DeploymentBuilder {
	b.deployment.SetOwnerReferences(append(b.deployment.GetOwnerReferences(),
		[]meta.OwnerReference{
			*meta.NewControllerRef(owner, owner.GroupVersionKind()),
		}[0]))
	return b
}

// WithSpec sets the Deployment spec in the object
func (b *DeploymentBuilder) WithSpec(replicas int32, labels map[string]string, podTemplateSpec core.PodTemplateSpec) *DeploymentBuilder {
	b.deployment.Spec = apps.DeploymentSpec{
		Replicas: &replicas,
		Selector: &meta.LabelSelector{
			MatchLabels: labels,
		},
		Template: podTemplateSpec,
	}
	return b
}
