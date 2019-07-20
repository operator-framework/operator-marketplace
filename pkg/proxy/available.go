package proxy

import (
	"errors"
	"strings"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apidiscovery "k8s.io/client-go/discovery"
)

const (
	// This is the error message thrown by ServerSupportsVersion function
	// when an API version is not supported by the server.
	notSupportedErrorMessage = "server does not support API version"
)

// isAPIAvailable tracks if the proxy API is available.
var isAPIAvailable = false

// SetProxyAvailability will set isAPIAvailable to the correct value if
// no unexpected errors are encountered.
func SetProxyAvailability(discovery apidiscovery.DiscoveryInterface) error {
	if discovery == nil {
		return errors.New("discovery interface cannot be nil")
	}

	opStatusGV := schema.GroupVersion{
		Group:   apiconfigv1.GroupName,
		Version: apiconfigv1.GroupVersion.Version,
	}

	if discoveryErr := apidiscovery.ServerSupportsVersion(discovery, opStatusGV); discoveryErr != nil {
		if strings.Contains(discoveryErr.Error(), notSupportedErrorMessage) {
			return nil
		}

		return discoveryErr
	}

	isAPIAvailable = true
	return nil
}

// IsAPIAvailable returns whether or not the proxy API is available.
func IsAPIAvailable() bool {
	return isAPIAvailable
}
