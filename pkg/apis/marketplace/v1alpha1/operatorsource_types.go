package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Only type definitions go into this file.
// All other constructs (constants, variables, receiver functions and such)
// related to OperatorSource type should be added to operatorsource.go file.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OperatorSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []OperatorSource `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OperatorSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              OperatorSourceSpec   `json:"spec"`
	Status            OperatorSourceStatus `json:"status,omitempty"`
}

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

type OperatorSourceStatus struct {
	// Current phase of the OperatorSource object
	CurrentPhase ObjectPhase `json:"currentPhase,omitempty"`
}
