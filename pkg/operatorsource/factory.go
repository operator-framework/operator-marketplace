package operatorsource

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/kube"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/phase"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
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
	GetPhaseReconciler(logger *log.Entry, event sdk.Event) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	registryClientFactory appregistry.ClientFactory
	datastore             datastore.Writer
	kubeclient            kube.Client
}

func (s *phaseReconcilerFactory) GetPhaseReconciler(logger *log.Entry, event sdk.Event) (Reconciler, error) {
	opsrc := event.Object.(*v1alpha1.OperatorSource)

	if event.Deleted {
		return NewDeletedEventReconciler(logger, s.datastore), nil
	}

	switch opsrc.Status.CurrentPhase.Name {
	case phase.Initial:
		return NewInitialReconciler(logger), nil

	case phase.OperatorSourceValidating:
		return NewValidatingReconciler(logger), nil

	case phase.OperatorSourceDownloading:
		return NewDownloadingReconciler(logger, s.registryClientFactory, s.datastore), nil

	case phase.Configuring:
		return NewConfiguringReconciler(logger, s.datastore, s.kubeclient), nil

	case phase.Succeeded:
		return NewSucceededReconciler(logger), nil

	case phase.Failed:
		return NewFailedReconciler(logger), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for OperatorSource type [phase=%s]", opsrc.Status.CurrentPhase.Phase)
	}
}
