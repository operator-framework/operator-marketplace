// Package v1 contains API Schema definitions for the config v1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=config.openshift.io
package v1

import (
	config "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: config.SchemeGroupVersion}
)
