package shared

import (
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetWatchNamespace returns the Namespace the operator should be watching for changes
// Note: the marketplace-operator YAML manifest deployed by the CVO specifies the
// $WATCH_NAMESPACE as an environment variable using the downward API.
// Source: https://sdk.operatorframework.io/docs/building-operators/golang/operator-scope/
func GetWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// EnsureFinalizer ensures that the object's finalizer is included
// in the ObjectMeta Finalizers slice. If it already exists, no state change occurs.
// If it doesn't, the finalizer is appended to the slice.
func EnsureFinalizer(objectMeta *metav1.ObjectMeta, expectedFinalizer string) {
	// First check if the finalizer is already included in the object.
	for _, finalizer := range objectMeta.Finalizers {
		if finalizer == expectedFinalizer {
			return
		}
	}

	// If it doesn't exist, append the finalizer to the object meta.
	objectMeta.Finalizers = append(objectMeta.Finalizers, expectedFinalizer)

	return
}

// RemoveFinalizer removes the finalizer from the object's ObjectMeta.
func RemoveFinalizer(objectMeta *metav1.ObjectMeta, deletingFinalizer string) {
	outFinalizers := make([]string, 0)
	for _, finalizer := range objectMeta.Finalizers {
		if finalizer == deletingFinalizer {
			continue
		}
		outFinalizers = append(outFinalizers, finalizer)
	}

	objectMeta.Finalizers = outFinalizers

	return
}

// HasFinalizer checks to see if the finalizer exists in the object's ObjectMeta.
func HasFinalizer(objectMeta *metav1.ObjectMeta, expectedFinalizer string) bool {
	finalizerExists := false
	for _, finalizer := range objectMeta.Finalizers {
		if finalizer == expectedFinalizer {
			finalizerExists = true
			break
		}
	}

	return finalizerExists
}

// IsObjectInOtherNamespace returns true if the namespace is not the watched
// namespace of the operator. An false, error will be returned if there was an
// error getting the watched namespace.
func IsObjectInOtherNamespace(namespace string) (bool, error) {
	watchNamespace, err := GetWatchNamespace()
	if err != nil {
		return false, err
	}

	if namespace != watchNamespace {
		return true, nil
	}
	return false, nil
}
