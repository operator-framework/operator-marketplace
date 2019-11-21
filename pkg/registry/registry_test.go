package registry

import (
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

// TestRegistryDeploymentNodeSelector confirms the PodTemplate used in the registry deployment contains a NodeSelector
// that limits registry pods to be scheduled to linux nodes only.
func TestRegistryDeploymentNodeSelector(t *testing.T) {
	reg := NewRegistry(nil, nil, nil, types.NamespacedName{Name: "hi", Namespace: "hins"},
		"", "", "", "")
	r := reg.(*registry)

	podTemplate := r.newPodTemplateSpec(nil, false)

	if podTemplate.Spec.NodeSelector[linuxNodeSelectorKey] != linuxNodeSelectorValue {
		t.Error("linux node selectors not found on registry pod spec")
	}
}
