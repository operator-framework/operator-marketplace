package phase

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
)

// NewDeletedEventReconciler returns a Reconciler that reconciles
// an OperatorSource object that has been deleted.
func NewDeletedEventReconciler(logger *log.Entry, datastore datastore.Writer) Reconciler {
	return &deletedEventReconciler{
		logger:    logger,
		datastore: datastore,
	}
}

// deletedEventReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object that has been deleted.
type deletedEventReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
}

// Reconcile reconciles an OperatorSource object that has been deleted.
//
// in represents the original OperatorSource object received from the sdk
// and before reconciliation has started.
//
// out represents the OperatorSource object after reconciliation has completed
// and could be different from the original. The OperatorSource object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given OperatorSource
// object. If nil is returned, it implies that no phase transition is expected.
func (r *deletedEventReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *NextPhase, err error) {
	r.logger.Info("No action taken, object has been deleted")

	return in, nil, nil
}
