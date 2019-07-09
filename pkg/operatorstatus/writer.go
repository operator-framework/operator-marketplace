package operatorstatus

import (
	"errors"
	"reflect"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// statusReporter is used to communicate with the kubernetes api
// to update the ClusterOperator.
type writer struct {
	discovery discovery.DiscoveryInterface
	client    *configclient.ConfigV1Client
}

// NewStatusReporter returns a new statusReporter.
func newWriter() (*writer, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := configclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	k8sInterface, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &writer{
		client:    client,
		discovery: k8sInterface.Discovery(),
	}, nil
}

// IsAPIAvailable return true if cluster operator API is present on the cluster.
// Otherwise, exists is set to false.
func (w *writer) isAPIAvailable() (exists bool, err error) {
	opStatusGV := schema.GroupVersion{
		Group:   "config.openshift.io",
		Version: "v1",
	}
	err = discovery.ServerSupportsVersion(w.discovery, opStatusGV)
	if err != nil {
		return
	}

	exists = true
	return
}

// ensureExists ensures that the cluster operator resource exists with a default
// status that reflects expecting status.
func (w *writer) ensureExists(name string) (existing *configv1.ClusterOperator, err error) {
	existing, err = w.client.ClusterOperators().Get(name, metav1.GetOptions{})
	if err == nil {
		return
	}

	if !apierrors.IsNotFound(err) {
		return
	}

	co := &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	existing, err = w.client.ClusterOperators().Create(co)
	return
}

// UpdateStatus updates the clusteroperator object with the new status specified.
func (w *writer) updateStatus(existing *configv1.ClusterOperator, newStatus *configv1.ClusterOperatorStatus) error {
	if newStatus == nil || existing == nil {
		return errors.New("input specified is <nil>")
	}

	existingStatus := existing.Status.DeepCopy()
	if reflect.DeepEqual(existingStatus, newStatus) {
		return nil
	}

	existing.Status = *newStatus
	if _, err := w.client.ClusterOperators().UpdateStatus(existing); err != nil {
		return err
	}

	return nil
}
