package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// ConfigMapPrefix is the prefix used for the ConfigMap created by the handler
	ConfigMapPrefix = "csc-cm-"
	// CatalogSourcePrefix is the prefix used for the CatalogSource created by the handler
	CatalogSourcePrefix = "csc-cs-"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogSourceConfigList contains a list of CatalogSourceConfig
type CatalogSourceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogSourceConfig `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CatalogSourceConfig is the Schema for the catalogsourceconfigs API
// +k8s:openapi-gen=true
type CatalogSourceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CatalogSourceConfigSpec   `json:"spec,omitempty"`
	Status            CatalogSourceConfigStatus `json:"status,omitempty"`
}

// CatalogSourceConfigSpec defines the desired state of CatalogSourceConfig
type CatalogSourceConfigSpec struct {
	TargetNamespace string `json:"targetNamespace"`
	Packages        string `json:"packages"`
}

// CatalogSourceConfigStatus defines the observed state of CatalogSourceConfig
type CatalogSourceConfigStatus struct {
	// Current phase of the CatalogSourceConfig object.
	CurrentPhase ObjectPhase `json:"currentPhase,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CatalogSourceConfig{}, &CatalogSourceConfigList{})
}

// Set group, version, and kind strings
// from the internal reference that we defined in the v1alpha1 package.
// The object the sdk client returns does not set these
// so we must find the correct values and set them manually.
func (csc *CatalogSourceConfig) EnsureGVK() {
	gvk := schema.GroupVersionKind{
		Group:   SchemeGroupVersion.Group,
		Version: SchemeGroupVersion.Version,
		Kind:    CatalogSourceConfigKind,
	}
	csc.SetGroupVersionKind(gvk)
}
