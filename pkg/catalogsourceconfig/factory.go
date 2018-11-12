package catalogsourceconfig

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
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
	GetPhaseReconciler(log *logrus.Entry, event sdk.Event) (Reconciler, error)
}

// phaseReconcilerFactory implements PhaseReconcilerFactory interface.
type phaseReconcilerFactory struct {
	reader datastore.Reader
}

func (f *phaseReconcilerFactory) GetPhaseReconciler(log *logrus.Entry, event sdk.Event) (Reconciler, error) {
	csc := event.Object.(*v1alpha1.CatalogSourceConfig)

	if event.Deleted {
		return NewDeletedEventReconciler(log), nil
	}

	switch csc.Status.CurrentPhase.Name {
	case phase.Initial:
		return NewInitialReconciler(log), nil

	case phase.Configuring:
		return NewConfiguringReconciler(log, f.reader), nil

	case phase.Succeeded:
		return NewSucceededReconciler(log), nil

	case phase.Failed:
		return NewFailedReconciler(log), nil

	default:
		return nil,
			fmt.Errorf("No phase reconciler returned, invalid phase for CatalogSourceConfig type [phase=%s]", csc.Status.CurrentPhase.Name)
	}
}
