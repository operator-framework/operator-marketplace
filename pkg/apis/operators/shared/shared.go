package shared

import (
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
)

// IsObjectInOtherNamespace returns true if the namespace is not the watched
// namespace of the operator. An false, error will be returned if there was an
// error getting the watched namespace.
func IsObjectInOtherNamespace(namespace string) (bool, error) {
	watchNamespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return false, err
	}

	if namespace != watchNamespace {
		return true, nil
	}
	return false, nil
}
