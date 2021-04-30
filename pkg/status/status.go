package status

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	operatorhelpers "github.com/openshift/library-go/pkg/operator/v1helpers"
	mktconfig "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	olm "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// coStatusReportInterval is the interval at which the ClusterOperator status is updated
	coStatusReportInterval = 20 * time.Second

	upgradeable = "Marketplace is upgradeable"

	operatorAvailable = "OperatorAvailable"
)

type Reporter interface {
	StartReporting() <-chan struct{}
}

type reporter struct {
	configClient    *configclient.ConfigV1Client
	rawClient       client.Client
	namespace       string
	clusterOperator *configv1.ClusterOperator
	version         string
	// stopCh is used to signal that threads should stop reporting ClusterOperator status
	stopCh <-chan struct{}
	// monitorDoneCh is used to signal that threads are done reporting ClusterOperator status
	monitorDoneCh       chan struct{}
	clusterOperatorName string
	once                sync.Once
}

// ensureClusterOperator ensures that a ClusterOperator CR is present on the
// cluster
func (r *reporter) ensureClusterOperator() error {
	var err error
	r.clusterOperator, err = r.configClient.ClusterOperators().Get(context.TODO(), r.clusterOperatorName, metav1.GetOptions{})

	if err == nil {
		log.Debug("[status] Found existing ClusterOperator")
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("Error %v getting ClusterOperator", err)
	}

	clusterOperator := &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.clusterOperatorName,
			Namespace: r.namespace,
		},
	}
	r.setRelatedObjects()

	r.clusterOperator, err = r.configClient.ClusterOperators().Create(context.TODO(), clusterOperator, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Error %v creating ClusterOperator", err)
	}
	log.Info("[status] Created ClusterOperator")
	return nil
}

// setStatus handles setting all the required fields for the given
// ClusterStatusConditionType
func (r *reporter) setStatus(statusConditions []configv1.ClusterOperatorStatusCondition) error {
	err := r.ensureClusterOperator()
	if err != nil {
		return err
	}
	previousStatus := r.clusterOperator.Status.DeepCopy()
	for _, statusCondition := range statusConditions {
		r.setStatusCondition(statusCondition)
	}

	err = r.updateStatus(previousStatus)
	if err != nil {
		return err
	}
	return nil
}

// setOperandVersion sets the operator version in the ClusterOperator Status
// Per instructions from the CVO team, setOperandVersion should only be called
// when the operator becomes available
func (r *reporter) setOperandVersion() {
	// Report the operator version
	operatorVersion := configv1.OperandVersion{
		Name:    "operator",
		Version: r.version,
	}
	operatorhelpers.SetOperandVersion(&r.clusterOperator.Status.Versions, operatorVersion)
}

// setStatusCondition sets the operator StatusCondition in the ClusterOperator Status
func (r *reporter) setStatusCondition(statusCondition configv1.ClusterOperatorStatusCondition) {
	// Only update the version when the operator becomes available
	if statusCondition.Type == configv1.OperatorAvailable && statusCondition.Status == configv1.ConditionTrue {
		r.setOperandVersion()
	}
	cohelpers.SetStatusCondition(&r.clusterOperator.Status.Conditions, statusCondition)
}

// updateStatus makes the API call to update the ClusterOperator if the status has changed.
func (r *reporter) updateStatus(previousStatus *configv1.ClusterOperatorStatus) error {
	var err error
	if compareClusterOperatorStatusConditionArrays(previousStatus.Conditions, r.clusterOperator.Status.Conditions) {
		log.Infof("[status] Previous and current ClusterOperator Status are the same, the ClusterOperator Status will not be updated.")
	} else {
		log.Debugf("[status] Previous and current ClusterOperator Status are different, attempting to update the ClusterOperator Status.")

		// Check if the ClusterOperator version has changed and log the attempt to upgrade if it has
		previousVersion := operatorhelpers.FindOperandVersion(previousStatus.Versions, "operator")
		currentVersion := operatorhelpers.FindOperandVersion(r.clusterOperator.Status.Versions, "operator")
		if currentVersion != nil {
			if previousVersion == nil {
				log.Infof("[status] Attempting to set ClusterOperator to version %s", currentVersion.Version)
			} else if previousVersion.Version != currentVersion.Version {
				log.Infof("[status] Attempting to upgrade ClusterOperator version from %s to %s", previousVersion.Version, currentVersion.Version)
			}
		}

		// Log Conditions
		log.Infof("[status] Attempting to set the ClusterOperator status conditions to:")
		for _, statusCondition := range r.clusterOperator.Status.Conditions {
			log.Infof("[status] ConditionType: %v ConditionStatus: %v ConditionMessage: %v", statusCondition.Type, statusCondition.Status, statusCondition.Message)
		}

		// Always update RelatedObjects to account for the upgrade case.
		r.setRelatedObjects()

		_, err := r.configClient.ClusterOperators().UpdateStatus(context.TODO(), r.clusterOperator, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("Error %v updating ClusterOperator", err)
		}
		log.Info("[status] ClusterOperator status conditions updated.")
	}
	return err
}

// setRelatedObjects populates RelatedObjects in the ClusterOperator.Status.
// RelatedObjects are consumed by https://github.com/openshift/must-gather
func (r *reporter) setRelatedObjects() {
	objectReferences := []configv1.ObjectReference{
		// Add the operator's namespace which will result in core resources
		// being gathered
		{
			Resource: "namespaces",
			Name:     r.namespace,
		},
		// Add the non-core resources we care about
		{
			Group:     olm.GroupName,
			Resource:  "catalogsources",
			Namespace: r.namespace,
		},
	}
	r.clusterOperator.Status.RelatedObjects = objectReferences
}

// monitorClusterStatus updates the ClusterOperator's status based on
// the number of successful syncs / total syncs
func (r *reporter) monitorClusterStatus() {
	msg := fmt.Sprintf("Available release version: %s", r.version)
	// Signal to the main channel that we have stopped reporting status.
	defer func() {
		close(r.monitorDoneCh)
	}()
	// Create the ClusterOperator in the available state if it does not exist
	// and it is the first report.
	if r.clusterOperator == nil {
		conditionListBuilder := clusterStatusListBuilder()
		conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionFalse, fmt.Sprintf("Successfully progressed to release version: %s", r.version), operatorAvailable)
		conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionTrue, msg, operatorAvailable)
		conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeable, operatorAvailable)
		statusConditions := conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, msg, operatorAvailable)
		statusErr := r.setStatus(statusConditions)
		if statusErr != nil {
			log.Error("[status] " + statusErr.Error())
		}
	}
	for {
		select {
		case <-r.stopCh:
			log.Info("[status] Operator no longer reporting status")
			return
		// Attempt to update the ClusterOperator status whenever the seconds
		// number of seconds defined by coStatusReportInterval passes.
		case <-time.After(coStatusReportInterval):
			// Log any status update errors after exit.
			var statusErr error
			defer func() {
				if statusErr != nil {
					log.Error("[status] " + statusErr.Error())
				}
			}()
			// Report that marketplace is available
			conditionListBuilder := clusterStatusListBuilder()
			conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionFalse, fmt.Sprintf("Successfully progressed to release version: %s", r.version), operatorAvailable)
			conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, msg, operatorAvailable)
			conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeable, operatorAvailable)
			statusConditions := conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionTrue, msg, operatorAvailable)
			statusErr = r.setStatus(statusConditions)
		}
	}
}

func NewReporter(cfg *rest.Config, mgr manager.Manager, namespace string, name string, version string, stopCh <-chan struct{}) (Reporter, error) {
	if !mktconfig.IsAPIAvailable() {
		return nil, errors.New("[status] ClusterOperator API not present")
	}

	// Client for handling reporting of operator status
	configClient, err := configclient.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create config v1 client: %s", err.Error())
	}
	// Client for listing OperatorSources
	rawClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create raw client: %s", err.Error())
	}

	// If version is an empty string, warn that the operator is not a part of the OpenShift release payload.
	if version == "" {
		version = "OpenShift Independent Version"
	}

	return &reporter{
		configClient:        configClient,
		rawClient:           rawClient,
		namespace:           namespace,
		version:             version,
		stopCh:              stopCh,
		monitorDoneCh:       make(chan struct{}),
		clusterOperatorName: name,
	}, nil
}

// StartReporting ensures that the cluster supports reporting ClusterOperator status
// and returns a channel that reports if it is actively reporting.
func (r *reporter) StartReporting() <-chan struct{} {
	// ensure each goroutine is only started once.
	r.once.Do(func() {
		// start reporting ClusterOperator status
		go r.monitorClusterStatus()
	})
	return r.monitorDoneCh
}

type NoOpReporter struct{}

func (NoOpReporter) SendSyncMessage(err error) {
}

func (NoOpReporter) StartReporting() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
