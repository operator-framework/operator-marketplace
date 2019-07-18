package operatorsource

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/operator-framework/operator-marketplace/pkg/watches"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewSucceededReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Succeeded" phase.
func NewSucceededReconciler(logger *log.Entry, client client.Client) Reconciler {
	return &succeededReconciler{
		logger: logger,
		client: client,
	}
}

// succeededReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Succeeded" phase.
type succeededReconciler struct {
	logger *log.Entry
	client client.Client
}

// Reconcile reconciles an OperatorSource object that is in "Succeeded" phase.
// Since this phase indicates that the object has been successfully reconciled,
// no further action is taken.
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
func (r *succeededReconciler) Reconcile(ctx context.Context, in *v1.OperatorSource) (out *v1.OperatorSource, nextPhase *shared.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Succeeded {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	// No change is being made, so return the OperatorSource object that was specified as is.
	out = in

	msg := "No action taken, the object has already been reconciled"

	secretIsPresent := r.isSecretPresent(in)

	if watches.CheckChildResources(r.client, in.Name, in.Namespace, in.Namespace, secretIsPresent) {
		// A child has been deleted. Drop the existing Status field so that reconciliation can start anew.
		out.Status = v1.OperatorSourceStatus{}
		nextPhase = phase.GetNext(phase.Configuring)
		msg = "Child resource(s) have been deleted, scheduling for configuring"
	}

	r.logger.Info(msg)

	return
}

// isSecretPresent checks if the OperatorSource contains an authorization token and returns true if it does.
func (r *succeededReconciler) isSecretPresent(opsrc *v1.OperatorSource) bool {
	if opsrc.Spec.AuthorizationToken.SecretName != "" {
		return true
	}
	return false
}
