package apis

import (
	configv1 "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, configv1.SchemeBuilder.AddToScheme)
}
