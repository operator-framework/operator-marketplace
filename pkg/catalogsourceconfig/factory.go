package catalogsourceconfig

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PhaseReconcilerFactory is the interface that wraps GetPhaseReconciler method.
//
// GetPhaseReconciler returns an appropriate phase.Reconciler based on the
// current phase of an CatalogSourceConfig object.
// The following chain shows how an CatalogSourceConfig object progresses through
// a series of transitions from the initial phase to complete reconciled state.
//
//  Initial --> Configuring --> Succeeded
//
// log is the prepared contextual logger that is to be used for logging.
// event represents the event fired by sdk, it is used to return the appropriate
// phase.Reconciler.
//
//  On error, the object is transitioned into "Failed" phase.
type PhaseReconcilerFactory interface {
	GetPhaseReconciler(log *logrus.Entry, currentPhase string) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	reader datastore.Reader
	client client.Client
}

func (f *phaseReconcilerFactory) GetPhaseReconciler(log *logrus.Entry, currentPhase string) (Reconciler, error) {
	switch currentPhase {
	case phase.Initial:
		return NewInitialReconciler(log), nil

	case phase.Configuring:
		return NewConfiguringReconciler(log, f.reader, f.client), nil

	case phase.Succeeded:
		return NewSucceededReconciler(log), nil

	case phase.Failed:
		return NewFailedReconciler(log), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for CatalogSourceConfig type [phase=%s]", currentPhase)
	}
}
