package catalogsourceconfig

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewUpdateReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object that needs to be updated
func NewUpdateReconciler(log *logrus.Entry, client client.Client, cache Cache, targetChanged bool) Reconciler {
	return &updateReconciler{
		cache:         cache,
		client:        client,
		log:           log,
		targetChanged: targetChanged,
	}
}

// updateReconciler is an implementation of Reconciler interface that
// reconciles an CatalogSourceConfig object that needs to be updated.
type updateReconciler struct {
	cache         Cache
	client        client.Client
	log           *logrus.Entry
	targetChanged bool
}

// Reconcile reconciles an CatalogSourceConfig object that needs to be updated.
// It returns "Configuring" as the next desired phase.
func (r *updateReconciler) Reconcile(ctx context.Context, in *v1alpha1.CatalogSourceConfig) (out *v1alpha1.CatalogSourceConfig, nextPhase *v1alpha1.Phase, err error) {
	out = in.DeepCopy()

	// The TargetNamespace of the CatalogSourceConfig object has changed
	if r.targetChanged {
		// Delete the objects in the old TargetNamespace
		err = r.deleteObjects(in)
		if err != nil {
			nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
			return
		}
	}

	// Remove it from the cache so that it does not get picked up during
	// the "Configuring" phase
	r.cache.Evict(in)

	// Drop existing Status field so that reconciliation can start anew.
	out.Status = v1alpha1.CatalogSourceConfigStatus{}
	nextPhase = phase.GetNext(phase.Configuring)

	r.log.Info("Spec has changed, scheduling for configuring")

	return
}

// deleteObjects deletes the CatalogSource and ConfigMap in the old TargetNamespace.
func (r *updateReconciler) deleteObjects(in *v1alpha1.CatalogSourceConfig) error {
	cachedCSCSpec, found := r.cache.Get(in)
	// This should never happen as it is because the cached Spec has changed
	// that we are in the update reconciler.
	if !found {
		return fmt.Errorf("Unexpected cache miss")
	}

	// Delete the ConfigMap
	name := in.Name
	configMap := new(ConfigMapBuilder).
		WithMeta(name, cachedCSCSpec.TargetNamespace).
		ConfigMap()
	err := r.client.Delete(context.TODO(), configMap)
	if err != nil {
		r.log.Errorf("Error %v deleting ConfigMap %s/%s", err, cachedCSCSpec.TargetNamespace, name)
	}
	r.log.Infof("Deleted ConfigMap %s/%s", cachedCSCSpec.TargetNamespace, name)

	// Delete the CatalogSource
	name = in.Name
	catalogSource := new(CatalogSourceBuilder).
		WithMeta(name, cachedCSCSpec.TargetNamespace).
		CatalogSource()
	err = r.client.Delete(context.TODO(), catalogSource)
	if err != nil {
		r.log.Errorf("Error %v deleting CatalogSource %s/%s", err, cachedCSCSpec.TargetNamespace, name)
	}
	r.log.Infof("Deleted CatalogSource %s/%s", cachedCSCSpec.TargetNamespace, name)

	return nil
}
