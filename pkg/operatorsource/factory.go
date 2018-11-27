package operatorsource

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// PhaseReconcilerFactory is the interface that wraps GetPhaseReconciler method.
//
// GetPhaseReconciler returns an appropriate phase.Reconciler based on the
// current phase of an OperatorSource object.
// The following chain shows how an OperatorSource object progresses through
// a series of transitions from the initial phase to complete reconciled state.
//
//  Initial --> Validating --> Downloading --> Configuring --> Succeeded
//
// logger is the prepared contextual logger that is to be used for logging.
// event represents the event fired by sdk, it is used to return the appropriate
// phase.Reconciler.
//
//  On error, the object is transitioned into "Failed" phase.
type PhaseReconcilerFactory interface {
	GetPhaseReconciler(logger *log.Entry, currentPhase string) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	registryClientFactory appregistry.ClientFactory
	datastore             datastore.Writer
	client                client.Client
}

func (s *phaseReconcilerFactory) GetPhaseReconciler(logger *log.Entry, currentPhase string) (Reconciler, error) {
	switch currentPhase {
	case phase.Initial:
		return NewInitialReconciler(logger), nil

	case phase.OperatorSourceValidating:
		return NewValidatingReconciler(logger), nil

	case phase.OperatorSourceDownloading:
		return NewDownloadingReconciler(logger, s.registryClientFactory, s.datastore), nil

	case phase.Configuring:
		return NewConfiguringReconciler(logger, s.datastore, s.client), nil

	case phase.Succeeded:
		return NewSucceededReconciler(logger), nil

	case phase.Failed:
		return NewFailedReconciler(logger), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for OperatorSource type [phase=%s]", currentPhase)
	}
}
