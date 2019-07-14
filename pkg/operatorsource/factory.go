package operatorsource

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// PhaseReconcilerFactory is an interface that wraps the GetPhaseReconciler
// method.
type PhaseReconcilerFactory interface {
	// GetPhaseReconciler returns an appropriate phase.Reconciler based on the
	// current phase of an OperatorSource object.
	// The following chain shows how an OperatorSource object progresses through
	// a series of transitions from the initial phase to complete reconciled state.
	//
	//  Initial --> Validating --> Configuring --> Succeeded
	//     ^
	//     |
	//  Purging
	//
	// logger is the prepared contextual logger that is to be used for logging.
	// opsrc represents the given OperatorSource object
	//
	// On error, the object is transitioned into "Failed" phase.
	GetPhaseReconciler(logger *log.Entry, opsrc *v1.OperatorSource) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	registryClientFactory appregistry.ClientFactory
	datastore             datastore.Writer
	client                client.Client
	refresher             PackageRefreshNotificationSender
}

func (s *phaseReconcilerFactory) GetPhaseReconciler(logger *log.Entry, opsrc *v1.OperatorSource) (Reconciler, error) {
	objectInOtherNamespace, err := shared.IsObjectInOtherNamespace(opsrc.GetNamespace())
	if err != nil {
		return nil, err
	}

	// We will only reconcile objects in the operator's namespace. If the object
	// was created in some other namespace, invoke the other namespace
	// reconciler that will place it in the failed phase.
	if objectInOtherNamespace {
		return NewOtherNamespaceReconciler(logger), nil
	}

	currentPhase := opsrc.GetCurrentPhaseName()

	// If the object has a deletion timestamp, it means it has been marked for
	// deletion. Return a deleted reconciler to remove that opsrc data from
	// the datastore, and remove the finalizer so the garbage collector can
	// clean it up.
	if !opsrc.ObjectMeta.DeletionTimestamp.IsZero() {
		return NewDeletedReconciler(logger, s.datastore, s.client), nil
	}

	switch currentPhase {
	case phase.Initial:
		return NewInitialReconciler(logger, s.datastore), nil

	case phase.OperatorSourceValidating:
		return NewValidatingReconciler(logger, s.datastore), nil

	case phase.Configuring:
		return NewConfiguringReconciler(logger, s.registryClientFactory, s.datastore, datastore.Cache, s.client, s.refresher), nil

	case phase.OperatorSourcePurging:
		return NewPurgingReconciler(logger, s.datastore, s.client), nil

	case phase.Succeeded:
		return NewSucceededReconciler(logger, s.client), nil

	case phase.Failed:
		return NewFailedReconciler(logger), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for OperatorSource type [phase=%s]", currentPhase)
	}
}
