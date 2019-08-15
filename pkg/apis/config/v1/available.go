package v1

import (
	"errors"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apidiscovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// isAPIAvailable tracks if the config.openshift.io API is available.
var isAPIAvailable = false

// SetConfigAPIAvailability will discover and set the availability of the
// OpenShift API
func SetConfigAPIAvailability(cfg *rest.Config) error {
	if cfg == nil {
		return errors.New("cfg cannot be nil")
	}

	k8sInterface, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	opStatusGV := schema.GroupVersion{
		Group:   apiconfigv1.GroupName,
		Version: apiconfigv1.GroupVersion.Version,
	}

	err = apidiscovery.ServerSupportsVersion(k8sInterface, opStatusGV)
	if err == nil {
		logrus.Info("Config API is available")
		isAPIAvailable = true
		return nil
	}

	logrus.Warn("Config API is not available")
	return nil
}

// IsAPIAvailable returns whether or not the config API is available.
func IsAPIAvailable() bool {
	return isAPIAvailable
}
