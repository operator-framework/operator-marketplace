package status

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	operatorhelpers "github.com/openshift/library-go/pkg/operator/v1helpers"
	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const clusterOperatorName = "marketplace-operator"

type status struct {
	configClient    *configclient.ConfigV1Client
	coAPINotPresent bool
	namespace       string
	clusterOperator *configv1.ClusterOperator
	version         string
}

// Status is the interface to report the health of the operator to CVO
type Status interface {
	SetFailing(message string)
	SetAvailable(message string)
	SetProgressing(message string)
	GetVersion() string
}

func (s *status) GetVersion() string {
	return s.version
}

// New returns an initialized Status
func New(cfg *rest.Config, mgr manager.Manager, namespace string, version string) Status {
	// The default client serves read requests from the cache which contains
	// objects only from the namespace the operator is watching. Given we need
	// to query CRDs which are cluster wide, we add the CRDs to the manager's
	// scheme and create our own client.
	if err := v1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	controllerClient, err := client.New(cfg, client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		log.Fatal(err)
	}

	// Check if the ClusterOperator API is present on the cluster. This is so
	// that we can continue to work with Kubernetes clusters that don't have
	// this API.
	key := client.ObjectKey{
		Name: "clusteroperators.config.openshift.io",
	}
	err = controllerClient.Get(context.TODO(), key, (&v1beta1.CustomResourceDefinition{}))
	coAPINotPresent := false
	if err != nil {
		log.Warningf("ClusterOperator API not present: %v", err)
		coAPINotPresent = true
	}

	// Client for handling reporting of operator status
	configClient, err := configclient.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &status{
		configClient:    configClient,
		coAPINotPresent: coAPINotPresent,
		namespace:       namespace,
		version:         version,
	}
}

// SetFailing reports that operator has failed along with the error message
func (s *status) SetFailing(message string) {
	s.setStatus(configv1.OperatorFailing, message)
}

// SetAvailable reports that the operator is available to process events
func (s *status) SetAvailable(message string) {
	s.setStatus(configv1.OperatorAvailable, message)
}

// SetProgressing reports that the operator is being deployed
func (s *status) SetProgressing(message string) {
	s.setStatus(configv1.OperatorProgressing, message)
}

// ensureClusterOperator ensures that a ClusterOperator CR is present on the
// cluster
func (s *status) ensureClusterOperator() {
	var err error
	s.clusterOperator, err = s.configClient.ClusterOperators().Get(clusterOperatorName, v1.GetOptions{})

	if err == nil {
		log.Info("Found existing ClusterOperator")
		return
	}

	if err != nil && !errors.IsNotFound(err) {
		log.Fatalf("Error %v getting ClusterOperator", err)
	}

	s.clusterOperator, err = s.configClient.ClusterOperators().Create(&configv1.ClusterOperator{
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterOperatorName,
			Namespace: s.namespace,
		},
	})
	if err != nil {
		log.Fatalf("Error %v creating ClusterOperator", err)
	}
	log.Info("Created ClusterOperator")
}

// setStatus handles setting all the required fields for the given
// ClusterStatusConditionType
func (s *status) setStatus(condition configv1.ClusterStatusConditionType, message string) {
	if s.coAPINotPresent {
		return
	}
	s.ensureClusterOperator()
	s.setStatusCondition(condition, message)
	s.updateStatus()
}

// setOperandVersion sets the operator version in the ClusterOperator Status
func (s *status) setOperandVersion() {
	// Report the operator version
	operatorVersion := configv1.OperandVersion{
		Name:    "operator",
		Version: s.version,
	}
	operatorhelpers.SetOperandVersion(&s.clusterOperator.Status.Versions, operatorVersion)
}

// setStatusCondition sets the operator StatusCondition in the ClusterOperator Status
func (s *status) setStatusCondition(condition configv1.ClusterStatusConditionType, message string) {
	log.Infof("Setting ClusterOperator condition: %s message: %s", condition, message)

	availableStatus := configv1.ConditionFalse
	failingStatus := configv1.ConditionFalse
	progressingStatus := configv1.ConditionFalse
	availableMessage := ""
	failingMessage := ""
	progressingMessage := ""

	switch condition {
	case configv1.OperatorAvailable:
		availableStatus = configv1.ConditionTrue
		availableMessage = message
		// The operator version should only be set when the operator becomes available
		s.setOperandVersion()

	case configv1.OperatorFailing:
		failingStatus = configv1.ConditionTrue
		failingMessage = message

	case configv1.OperatorProgressing:
		progressingStatus = configv1.ConditionTrue
		progressingMessage = message
	}

	time := v1.Now()
	// https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#conditions
	// implies that all three StatusConditionTypes needs to be set with only
	// the relevant ClusterStatusConditionType's Status set to ConditionTrue
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorProgressing,
		Status:             progressingStatus,
		Message:            progressingMessage,
		LastTransitionTime: time,
	})
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorAvailable,
		Status:             availableStatus,
		Message:            availableMessage,
		LastTransitionTime: time,
	})
	cohelpers.SetStatusCondition(&s.clusterOperator.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorFailing,
		Status:             failingStatus,
		Message:            failingMessage,
		LastTransitionTime: time,
	})
}

// updateStatus makes the API call to update the ClusterOperator
func (s *status) updateStatus() {
	_, err := s.configClient.ClusterOperators().UpdateStatus(s.clusterOperator)
	if err != nil {
		log.Fatalf("Error %v updating ClusterOperator", err)
	}
}
