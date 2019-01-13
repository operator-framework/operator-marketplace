package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Name for the finalizer to allow for deletion reconciliation
	// when a CatalogSourceConfig is deleted.
	CSCFinalizer = "finalizer.catalogsourceconfigs.marketplace.redhat.com"
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

	// DisplayName is passed along to the CatalogSource to be used
	// as a pretty name.
	DisplayName string `json:"csDisplayName,omitempty"`

	// Publisher is passed along to the CatalogSource to be used
	// to define what entity published the artifacts from the OperatorSource.
	Publisher string `json:"csPublisher,omitempty"`
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

// RemoveFinalizer removes the operator source finalizer from the
// CatatalogSourceConfig ObjectMeta.
func (csc *CatalogSourceConfig) RemoveFinalizer() {
	removeFinalizer(&csc.ObjectMeta, CSCFinalizer)
}

// EnsureFinalizer ensures that the CatatalogSourceConfig finalizer is included
// in the ObjectMeta.
func (csc *CatalogSourceConfig) EnsureFinalizer() {
	ensureFinalizer(&csc.ObjectMeta, CSCFinalizer)
}

func (csc *CatalogSourceConfig) EnsureDisplayName() {
	if csc.Spec.DisplayName == "" {
		csc.Spec.DisplayName = "Custom"
	}
}

func (csc *CatalogSourceConfig) EnsurePublisher() {
	if csc.Spec.Publisher == "" {
		csc.Spec.Publisher = "Custom"
	}
}
