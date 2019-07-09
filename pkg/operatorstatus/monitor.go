package operatorstatus

import (
	"fmt"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/clock"
)

const (
	// clusterOperatorName is the name of the ClusterOperator.
	clusterOperatorName = "marketplace"

	// minimumSyncs represents the minimum number of syncs Marketplace must see
	// prior to reporting that it is available to the ClusterOperator.
	minimumSyncs uint = 2

	// degradedRatio is the ratio of failed transition to syncs that must be reached
	// prior to reporting that the marketplace operator is  degraded.
	degradedRatio = 0.7

	// syncLimit is used to prevent the sync and failed phase transition event counts
	// from overflowing in a long running operator.
	// Once the number of syncs reaches the syncLimit value, syncs
	// and failedTransitions will be recalculated with the following
	// equation: updatedValue = currentValue / truncateValue.
	syncLimit     = 1000
	truncateValue = 10
)

var (
	// statusMon is a singleton.
	statusMon *statusMonitor
	once      sync.Once
)

type statusMonitor struct {
	coWriter     *writer
	namespace    string
	eventTracker eventTracker
	eventCh      chan error
}

// initInstance initializes the statusMonitor singleton named statusMon.
func initInstance() error {
	// Get the watch namespace.
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("failed to get watch namespace: %v", err)
	}

	// Create a new writer.
	coWriter, err := newWriter()
	if err != nil {
		return fmt.Errorf("Error creating ClusterOpreator writer: %v", err)
	}

	// Initialize the statusMon singleton once.
	once.Do(func() {
		// init the statusMon singleton.
		statusMon = &statusMonitor{
			coWriter:  coWriter,
			namespace: namespace,
			eventTracker: eventTracker{
				syncs:             0,
				failedTransitions: 0,
				syncLimit:         syncLimit,
				truncateValue:     truncateValue,
				lock:              sync.Mutex{},
			},
			eventCh: make(chan error, 32),
		}
	})

	return nil
}

// StartReporting initializes the statusMon singleton and starts reporting
// ClusterOperator status.
func StartReporting() error {
	err := initInstance()
	if err != nil {
		return err
	}

	err = statusMon.reportProgressing()
	if err != nil {
		log.Errorf("[status] Error updating status: %v", err)
	}

	go statusMon.eventChannelReceiver()
	go statusMon.monitorEvents()

	return nil
}

// eventChannelReceiver will listen on the event channel and update the
// number of syncs and failed phase transition events appropriately.
func (s *statusMonitor) eventChannelReceiver() {
	log.Info("[status] Starting event receiver")
	for {
		select {
		case err := <-s.eventCh:
			if err == nil {
				s.eventTracker.incrementSyncs()
			} else {
				s.eventTracker.incrementFailedTransitions()
			}
		}
	}
}

// SendEventMessage is used to send events to the eventCh. If the channel is
// busy, the event will be dropped to prevent the controller from stalling.
func SendEventMessage(err error) {
	// If the coAPI is not available do not attempt to send messages to the
	// sync channel
	if statusMon == nil {
		return
	}

	// A missing sync status is better than stalling the controller
	select {
	case statusMon.eventCh <- err:
		break
	default:
		log.Warning("[status] Event channel is busy, not reporting event")
	}
}

// monitorEvents will update the ClusterOpreator to reflect that Marketplace
// is available or degraded based on the events reported.
func (s *statusMonitor) monitorEvents() {
	// reportInterval represents the frequency at which marketplace
	// will attempt to update the ClusterOperator.
	reportInterval := 1 * time.Minute
	for {
		select {
		case <-time.After(reportInterval):
			var statusErr error
			defer func() {
				if statusErr != nil {
					log.Errorf("[status] Error updating status: %v", statusErr)
				}
			}()
			syncs, failures := s.eventTracker.getEvents()

			// Wait for the minimum number of sync events.
			if syncs < minimumSyncs {
				break
			}

			// Check if the ratio is appropriate.
			if syncs > 0 && syncs >= minimumSyncs {
				if float64(failures)/float64(syncs) < degradedRatio {
					statusErr = s.reportAvailable()
				} else {
					statusErr = s.reportDegraded()
				}
				reportInterval = 5 * time.Minute
			}
		}
	}
}

// reportProgressing updates the ClusterOperator to reflect that the
// marketplace operator is progressing towards a new version.
func (s *statusMonitor) reportProgressing() error {
	co := NewBuilder(clock.RealClock{}).
		WithProgressing(configv1.ConditionTrue, fmt.Sprintf("Progressing towards release version: %s", getReleaseVersion())).
		WithAvailable(configv1.ConditionFalse, "Determining status").
		WithDegraded(configv1.ConditionFalse, "").
		WithMarketplaceRelatedObjects(s.namespace).
		GetStatus()
	return s.reportStatus(co)
}

// reportAvailable updates the ClusterOperator to reflect that
// the operator is available.
func (s *statusMonitor) reportAvailable() error {
	co := NewBuilder(clock.RealClock{}).
		WithProgressing(configv1.ConditionFalse, fmt.Sprintf("Successfully progressed to release version: %s", getReleaseVersion())).
		WithAvailable(configv1.ConditionTrue, fmt.Sprintf("Available release version: %s", getReleaseVersion())).
		WithDegraded(configv1.ConditionFalse, "").
		WithMarketplaceRelatedObjects(s.namespace).
		WithMarketplaceVersions().
		GetStatus()
	return s.reportStatus(co)
}

// reportDegraded updates the ClusterOperator to reflect that the
// marketplace operator is in a degraded state.
func (s *statusMonitor) reportDegraded() error {
	msg := "Marketplace is unable to transition operands to expected phases"
	co := NewBuilder(clock.RealClock{}).
		WithProgressing(configv1.ConditionFalse, msg).
		WithAvailable(configv1.ConditionFalse, msg).
		WithDegraded(configv1.ConditionTrue, "phaseTransitionFailures").
		WithMarketplaceRelatedObjects(s.namespace).
		GetStatus()
	return s.reportStatus(co)
}

// reportStatus updates the ClusterOperator status if the ClusterOperator API is available.
func (s *statusMonitor) reportStatus(co *configv1.ClusterOperatorStatus) error {
	// If the ClusterOperator API is not present, do not attempt to report the status.
	_, err := s.coWriter.isAPIAvailable()
	if err != nil {
		return nil
	}

	// Make sure that the Marketplace ClusterOperator already exists.
	existing, err := s.coWriter.ensureExists(clusterOperatorName)
	if err != nil {
		return err
	}

	// If the existing ClusterOperator included a version and the
	// new status does not, do not overwrite the version.
	if len(existing.Status.Versions) > 0 && len(co.Versions) == 0 {
		co.Versions = existing.Status.Versions
	}

	// Update the ClusterOperator
	return s.coWriter.updateStatus(existing, co)
}
