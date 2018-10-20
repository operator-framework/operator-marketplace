package phase

import (
	"context"
	"fmt"
	"net/url"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// NewValidatingReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Validating" phase
func NewValidatingReconciler(logger *log.Entry) Reconciler {
	return &validatingReconciler{
		logger: logger,
	}
}

// validatingReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Validating" phase.
type validatingReconciler struct {
	logger *log.Entry
}

// Reconcile reconciles an OperatorSource object that is in "Validating" phase.
// It ensures that the given object is valid before it is scheduled for download.
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
//
// On success, it returns "Downloading" as the next phase.
// On error, it returns "Failed" as the next phase.
func (r *validatingReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *NextPhase, err error) {
	if in.Status.CurrentPhase.Name != v1alpha1.OperatorSourcePhaseValidating {
		err = ErrWrongReconcilerInvoked
		return
	}

	// No change is being made to the received OperatorSource object.
	out = in

	// Validate that operator source endpoint is a valid URL.
	_, err = url.ParseRequestURI(in.Spec.Endpoint)
	if err != nil {
		nextPhase = getNextPhaseWithMessage(v1alpha1.OperatorSourcePhaseFailed, fmt.Sprintf("Invalid operator source endpoint - %s", err.Error()))
		return
	}

	r.logger.Info("Scheduling for download")

	nextPhase = getNextPhase(v1alpha1.OperatorSourcePhaseDownloading)
	return
}
