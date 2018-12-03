package operatorsource

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewPurgingReconciler returns a Reconciler that reconciles
// an OperatorSource object that is in "Purging" phase.
func NewPurgingReconciler(logger *log.Entry, datastore datastore.Writer, client client.Client) Reconciler {
	return &purgingReconciler{
		logger:    logger,
		datastore: datastore,
		client:    client,
	}
}

// purgingReconciler implements Reconciler interface.
type purgingReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
	client    client.Client
}

// Reconcile reconciles an OperatorSource object that is in "Purging" phase.
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
// In this phase, we purge the current OperatorSource object, drop the Status
// field and trigger reconciliation anew from "Validating" phase.
//
// If the purge fails the OperatorSource object is moved to "Failed" phase.
func (r *purgingReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *v1alpha1.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.OperatorSourcePurging {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in.DeepCopy()

	r.logger.Info("Purging all resource(s)")

	r.datastore.RemoveOperatorSource(in.GetUID())

	builder := &CatalogSourceConfigBuilder{}
	csc := builder.WithMeta(in.Namespace, getCatalogSourceConfigName(in.Name)).CatalogSourceConfig()

	if err = r.client.Delete(ctx, csc); err != nil && !k8s_errors.IsNotFound(err) {
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	// Since all observable states stored in the Status resource might already
	// be stale, we should Reset everything in Status except for 'Current Phase'
	// to their default values.
	// The reason we are not mutating current phase is because it is the
	// responsibility of the caller to set the new phase appropriately.
	tmp := out.Status.CurrentPhase
	out.Status = v1alpha1.OperatorSourceStatus{}
	out.Status.CurrentPhase = tmp

	nextPhase = phase.GetNext(phase.Initial)
	r.logger.Info("Scheduling for reconciliation from 'Initial' phase")

	return
}
