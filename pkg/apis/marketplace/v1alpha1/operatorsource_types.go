package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Only type definitions go into this file.
// All other constructs (constants, variables, receiver functions and such)
// related to OperatorSource type should be added to operatorsource.go file.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorSourceList contains a list of OperatorSource
type OperatorSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorSource `json:"items"`
}

// OperatorSource is the Schema for the operatorsources API
// +k8s:openapi-gen=true
type OperatorSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OperatorSourceSpec   `json:"spec,omitempty"`
	Status            OperatorSourceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorSourceSpec defines the desired state of OperatorSource
type OperatorSourceSpec struct {
	// Type of operator source.
	Type string `json:"type,omitempty"`

	// Endpoint points to the remote app registry server from
	// where operator manifests can be fetched.
	Endpoint string `json:"endpoint,omitempty"`

	// RegistryNamespace refers to the namespace in app registry. Only operator
	// manifests under this namespace will be visible.
	// Please note that this is not a k8s namespace.
	RegistryNamespace string `json:"registryNamespace,omitempty"`
}

// OperatorSourceStatus defines the observed state of OperatorSource
type OperatorSourceStatus struct {
	// Current phase of the OperatorSource object
	CurrentPhase ObjectPhase `json:"currentPhase,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OperatorSource{}, &OperatorSourceList{})
}

// Set group, version, and kind strings
// from the internal reference that we defined in the v1alpha1 package.
// The object the sdk client returns does not set these
// so we must find the correct values and set them manually.
func (opsrc *OperatorSource) EnsureGVK() {
	gvk := schema.GroupVersionKind{
		Group:   SchemeGroupVersion.Group,
		Version: SchemeGroupVersion.Version,
		Kind:    OperatorSourceKind,
	}
	opsrc.SetGroupVersionKind(gvk)
}
