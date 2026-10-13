package v1

import (
	configv1 "github.com/openshift/api/config/v1"
)

func init() {
	SchemeBuilder.Register(&configv1.OperatorHub{}, &configv1.OperatorHubList{})
}
