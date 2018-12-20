package catalogsourceconfig

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDeletedReconciler returns a Reconciler that reconciles
// a CatalogSourceConfig that has been marked for deletion.
func NewDeletedReconciler(logger *log.Entry, cache Cache, client client.Client) Reconciler {
	return &deletedReconciler{
		logger: logger,
		cache:  cache,
		client: client,
	}
}

// deletedReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object that has been marked for deletion.
type deletedReconciler struct {
	logger *log.Entry
	cache  Cache
	client client.Client
}

// Reconcile reconciles a CatalogSourceConfig object that is marked for deletion.
// In the generic case, this is called when the CatalogSourceConfig has been marked
// for deletion. It removes all data related to this CatalogSourceConfig from the
// datastore, and it removes the CatalogSourceConfig finalizer from the object so
// that it can be cleaned up by the garbage collector.
//
// in represents the original CatalogSourceConfig object received from the sdk
// and before reconciliation has started.
//
// out represents the CatalogSourceConfig object after reconciliation has completed
// and could be different from the original. The CatalogSourceConfig object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given CatalogSourceConfig
// object. If nil is returned, it implies that no phase transition is expected.
func (r *deletedReconciler) Reconcile(ctx context.Context, in *v1alpha1.CatalogSourceConfig) (out *v1alpha1.CatalogSourceConfig, nextPhase *v1alpha1.Phase, err error) {
	out = in

	// Evict the catalogsourceconfig data from the cache.
	r.cache.Evict(out)

	// Remove the csc finalizer from the object.
	out.RemoveFinalizer()

	// Update the client. Since there is no phase shift, the transitioner
	// will not update it automatically like the normal phases.
	err = r.client.Update(context.TODO(), out)
	if err != nil {
		return nil, nil, err
	}

	r.logger.Info("Finalizer removed, now garbage collector will clean it up.")

	return out, nil, nil
}
