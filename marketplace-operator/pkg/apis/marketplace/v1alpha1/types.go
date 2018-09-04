package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CatalogSourceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CatalogSourceConfig `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CatalogSourceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              CatalogSourceConfigSpec   `json:"spec"`
	Status            CatalogSourceConfigStatus `json:"status,omitempty"`
}

type CatalogSourceConfigSpec struct {
	// Fill me
}
type CatalogSourceConfigStatus struct {
	// Fill me
}

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
	// Fill me
}
type OperatorSourceStatus struct {
	// Fill me
}
