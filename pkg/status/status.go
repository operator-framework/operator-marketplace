package status

import (
	"errors"
	"fmt"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	operatorhelpers "github.com/openshift/library-go/pkg/operator/v1helpers"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	mktconfig "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	v2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// minSyncsBeforeReporting is the minimum number of syncs we wish to see
	// before reporting that the operator is available
	minSyncsBeforeReporting = 3

	// successRatio is the ratio of successful syncs / total syncs that we
	// want to see in order to report that the marketplace operator is not degraded.
	// This value is low right now because the failed syncs come from invalid CRs.
	// As the status reporting evolves we can tweek this ratio to be a better
	// representation of the operator's health.
	successRatio = 0.3

	// syncsBeforeTruncate is used to prevent the totalSyncs and
	// failedSyncs values from overflowing in a long running operator.
	// Once totalSyncs reaches the maxSyncsBeforeTruncate value, totalSyncs
	// and failedSyncs will be recalculated with the following
	// equation: updatedValue = currentValue % syncTruncateValue.
	syncsBeforeTruncate = 10000
	syncTruncateValue   = 100

	// coStatusReportInterval is the interval at which the ClusterOperator status is updated
	coStatusReportInterval = 20 * time.Second

	// Marketplace is always upgradeable and should include this message in the Upgradeable
	// ClusterOperatorStatus condition.
	upgradeableMessage = "Marketplace is upgradeable"
)

type SyncSender interface {
	SendSyncMessage(err error)
}

type Reporter interface {
	StartReporting() <-chan struct{}
	ReportMigration() error
	SyncSender
}

type reporter struct {
	configClient    *configclient.ConfigV1Client
	namespace       string
	clusterOperator *configv1.ClusterOperator
	version         string
	syncRatio       SyncRatio
	// syncCh is used to report sync events
	syncCh chan error
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
	r.clusterOperator, err = r.configClient.ClusterOperators().Get(r.clusterOperatorName, metav1.GetOptions{})

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

	r.clusterOperator, err = r.configClient.ClusterOperators().Create(clusterOperator)
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
		log.Debugf("[status] Previous and current ClusterOperator Status are the same, the ClusterOperator Status will not be updated.")
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

		_, err := r.configClient.ClusterOperators().UpdateStatus(r.clusterOperator)
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
			Group:     v1.SchemeGroupVersion.Group,
			Resource:  v1.OperatorSourceKind,
			Namespace: r.namespace,
		},
		{
			Group:     v2.SchemeGroupVersion.Group,
			Resource:  v2.CatalogSourceConfigKind,
			Namespace: r.namespace,
		},
		{
			Group:     olm.GroupName,
			Resource:  olm.CatalogSourceKind,
			Namespace: r.namespace,
		},
	}
	r.clusterOperator.Status.RelatedObjects = objectReferences
}

// syncChannelReceiver will listen on the sync channel and update the status
// syncsRatio filed until the stopCh is closed.
func (r *reporter) syncChannelReceiver() {
	log.Info("[status] Starting sync consumer")
	for {
		select {
		case <-r.stopCh:
			return
		case err := <-r.syncCh:
			if err == nil {
				r.syncRatio.ReportSyncEvent()
			} else {
				r.syncRatio.ReportFailedSync()
			}
			failedSyncs, syncs := r.syncRatio.GetSyncs()
			log.Debugf("[status] Faild Syncs / Total Syncs : %d/%d", failedSyncs, syncs)
		}
	}
}

// monitorClusterStatus updates the ClusterOperator's status based on
// the number of successful syncs / total syncs
func (r *reporter) monitorClusterStatus() {
	// Signal to the main channel that we have stopped reporting status.
	defer func() {
		close(r.monitorDoneCh)
	}()
	for {
		select {
		case <-r.stopCh:
			// If the stopCh is closed, set all ClusterOperatorStatus conditions to false.
			reason := "OperatorExited"
			msg := "The operator has exited"
			conditionListBuilder := clusterStatusListBuilder()
			conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionFalse, msg, reason)
			conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionFalse, msg, reason)
			conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeableMessage, reason)
			statusConditions := conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, msg, reason)
			statusErr := r.setStatus(statusConditions)
			if statusErr != nil {
				log.Error("[status] " + statusErr.Error())
			}
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

			// Create the ClusterOperator in the progressing state if it does not exist
			// or if it is the first report.
			if r.clusterOperator == nil {
				reason := "OperatorStarting"
				conditionListBuilder := clusterStatusListBuilder()
				conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionTrue, fmt.Sprintf("Progressing towards release version: %s", r.version), reason)
				conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeableMessage, reason)
				msg := fmt.Sprintf("Determining status")
				conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionFalse, msg, reason)
				statusConditions := conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, msg, reason)
				statusErr = r.setStatus(statusConditions)
				break
			}

			_, syncEvents := r.syncRatio.GetSyncs()

			// no default operator sources are present, so assume we are in a good state
			if operatorhub.GetSingleton().Disabled() && syncEvents == 0 {
				reason := "NoDefaultOpSrcEnabled"
				conditionListBuilder := clusterStatusListBuilder()
				conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionFalse, fmt.Sprintf("Successfully progressed to release version: %s", r.version), reason)
				conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeableMessage, reason)
				statusConditions := conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionTrue, fmt.Sprintf("Available release version: %s", r.version), reason)
				statusErr = r.setStatus(statusConditions)
				break
			}

			// Wait until the operator has reconciled the minimum number of syncs.
			if syncEvents < minSyncsBeforeReporting {
				log.Debugf("[status] Waiting to observe %d additional sync(s)", minSyncsBeforeReporting-syncEvents)
				break
			}

			// Report that marketplace is available after meeting minimal syncs.
			if cohelpers.IsStatusConditionFalse(r.clusterOperator.Status.Conditions, configv1.OperatorAvailable) {
				reason := "OperatorAvailable"
				conditionListBuilder := clusterStatusListBuilder()
				conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionFalse, fmt.Sprintf("Successfully progressed to release version: %s", r.version), reason)
				conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeableMessage, reason)
				statusConditions := conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionTrue, fmt.Sprintf("Available release version: %s", r.version), reason)
				statusErr = r.setStatus(statusConditions)
				break
			}

			// Update the status with the appropriate state.
			isSucceeding, ratio := r.syncRatio.IsSucceeding()
			if ratio != nil {
				var statusConditions []configv1.ClusterOperatorStatusCondition
				conditionListBuilder := clusterStatusListBuilder()
				conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionTrue, upgradeableMessage, "OperatorAvailable")
				if isSucceeding {
					statusConditions = conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, fmt.Sprintf("Current CR sync ratio (%g) meets the expected success ratio (%g)", *ratio, successRatio), "OperandTransitionsSucceeding")
				} else {
					statusConditions = conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionTrue, fmt.Sprintf("Current CR sync ratio (%g) does not meet the expected success ratio (%g)", *ratio, successRatio), "OperandTransitionsFailing")
				}
				statusErr = r.setStatus(statusConditions)
				break
			}
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

	syncRatio, err := NewSyncRatio(successRatio, syncsBeforeTruncate, syncTruncateValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create status sync ratio: %s", err.Error())
	}

	// If version is an empty string, warn that the operator is not a part of the OpenShift release payload.
	if version == "" {
		version = "OpenShift Independent Version"
	}

	return &reporter{
		configClient: configClient,
		namespace:    namespace,
		version:      version,
		syncRatio:    syncRatio,
		// Add a buffer to prevent dropping syncs
		syncCh:              make(chan error, 25),
		stopCh:              stopCh,
		monitorDoneCh:       make(chan struct{}),
		clusterOperatorName: name,
	}, nil
}

// ReportMigration sets the clusterOperator status to signal that migration logic is in progress,
// while upgrading from openshift 4.1.z to openshift 4.2.z
func (r *reporter) ReportMigration() error {
	conditionListBuilder := clusterStatusListBuilder()
	conditionListBuilder(configv1.OperatorProgressing, configv1.ConditionTrue, fmt.Sprintf("Performing migration logic to progress towards release version: %s", r.version), "Upgrading")
	msg := fmt.Sprintf("Determining status")
	conditionListBuilder(configv1.OperatorAvailable, configv1.ConditionFalse, msg, "Upgrading")
	conditionListBuilder(configv1.OperatorUpgradeable, configv1.ConditionFalse, msg, "Upgrading")
	statusConditions := conditionListBuilder(configv1.OperatorDegraded, configv1.ConditionFalse, msg, "Upgrading")
	return r.setStatus(statusConditions)
}

// StartReporting ensures that the cluster supports reporting ClusterOperator status
// and returns a channel that reports if it is actively reporting.
func (r *reporter) StartReporting() <-chan struct{} {
	// ensure each goroutine is only started once.
	r.once.Do(func() {
		// start consuming messages on the sync channel
		go r.syncChannelReceiver()

		// start reporting ClusterOperator status
		go r.monitorClusterStatus()
	})
	return r.monitorDoneCh
}

// SendSyncMessage is used to send messages to the syncCh. If the channel is
// busy, the sync will be dropped to prevent the controller from stalling.
func (r *reporter) SendSyncMessage(err error) {
	// A missing sync status is better than stalling the controller
	select {
	case r.syncCh <- err:
		log.Debugf("[status] Sent message to the sync channel")
		break
	default:
		log.Debugf("[status] Sync channel is busy, not reporting sync")
	}
}

type NoOpReporter struct{}

func (NoOpReporter) SendSyncMessage(err error) {
}

func (NoOpReporter) StartReporting() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (NoOpReporter) ReportMigration() error {
	return nil
}
