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

// NewUpdatedEventReconciler returns a Reconciler that reconciles
// an OperatorSource object that has been updated.
func NewUpdatedEventReconciler(logger *log.Entry, datastore datastore.Writer, client client.Client) Reconciler {
	return &updatedEventReconciler{
		logger:    logger,
		datastore: datastore,
		client:    client,
	}
}

// updatedEventReconciler implements Reconciler interface.
type updatedEventReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
	client    client.Client
}

// Reconcile reconciles an OperatorSource object whose Spec has been updated.
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
// On an update we purge the current OperatorSource object, drop the Status
// field and trigger reconciliation anew from "Validating" phase.
//
// If the purge fails the OperatorSource object is moved to "Failed" phase.
func (r *updatedEventReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *v1alpha1.Phase, err error) {
	out = in.DeepCopy()

	r.logger.Info("Spec has changed, purging all resource(s) associated with it")

	r.datastore.RemoveOperatorSource(in.GetUID())

	builder := &CatalogSourceConfigBuilder{}
	csc := builder.WithMeta(in.Namespace, getCatalogSourceConfigName(in.Name)).CatalogSourceConfig()

	if err = r.client.Delete(ctx, csc); err != nil && !k8s_errors.IsNotFound(err) {
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	// Drop existing Status field so that reconciliation can start anew.
	out.Status = v1alpha1.OperatorSourceStatus{}

	nextPhase = phase.GetNext(phase.OperatorSourceValidating)
	r.logger.Info("Spec has changed, scheduling for reconciliation from 'Validating' phase")

	return
}
