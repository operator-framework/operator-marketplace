package catalogsourceconfig

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
)

// NewInitialReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Initial" phase.
func NewInitialReconciler(log *logrus.Entry) Reconciler {
	return &initialReconciler{
		log: log,
	}
}

// initialReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object in the "Initial" phase.
type initialReconciler struct {
	log *logrus.Entry
}

// Reconcile reconciles a CatalogSourceConfig object that is in the "Initial"
// phase. This is the first phase in the reconciliation process.
//
// in represents the original CatalogSourceConfig object received from the sdk
// and before reconciliation has started.
//
// out represents the CatalogSourceConfig object after reconciliation has
// completed and could be different from the original. The CatalogSourceConfig
// object received (in) should be deep copied into (out) before changes are
// made.
//
// nextPhase represents the next desired phase for the CatalogSourceConfig
// object. If nil is returned, it implies that no phase transition is expected.
//
// Upon success, it returns "Validating" as the next desired phase.
func (r *initialReconciler) Reconcile(ctx context.Context, in *v1alpha1.CatalogSourceConfig) (out *v1alpha1.CatalogSourceConfig, nextPhase *v1alpha1.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Initial {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in.DeepCopy()

	r.log.Info("Scheduling for configuring")

	nextPhase = phase.GetNext(phase.Configuring)
	return
}
