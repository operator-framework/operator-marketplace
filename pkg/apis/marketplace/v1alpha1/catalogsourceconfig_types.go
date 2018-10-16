package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConfigMapPrefix is the prefix used for the ConfigMap created by the handler
	ConfigMapPrefix = "csc-cm-"
	// CatalogSourcePrefix is the prefix used for the CatalogSource created by the handler
	CatalogSourcePrefix = "csc-cs-"
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
	TargetNamespace string `json:"targetNamespace"`
	Packages        string `json:"packages"`
}
type CatalogSourceConfigStatus struct {
	// Fill me
}
