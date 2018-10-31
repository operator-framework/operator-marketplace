package catalogsourceconfig

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
)

// NewDeletedEventReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object that has been deleted.
func NewDeletedEventReconciler(log *logrus.Entry) Reconciler {
	return &deletedEventReconciler{
		log: log,
	}
}

// deletedEventReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object that has been deleted.
type deletedEventReconciler struct {
	log *logrus.Entry
}

// Reconcile reconciles a CatalogSourceConfig object that has been deleted.
//
// Given that nil is returned here, it implies that no phase transition is
// expected.
func (r *deletedEventReconciler) Reconcile(ctx context.Context, in *v1alpha1.CatalogSourceConfig) (out *v1alpha1.CatalogSourceConfig, nextPhase *v1alpha1.Phase, err error) {
	r.log.Info("No action taken, object has been deleted")

	return in, nil, nil
}
