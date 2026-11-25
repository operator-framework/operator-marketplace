package catalogsource

import (
	"context"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Add creates a new CatalogSource Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ options.ControllerOptions) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileCatalogSource{
		client: client,
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	return builder.ControllerManagedBy(mgr).
		Named("catalogsource-controller").
		For(&olmv1alpha1.CatalogSource{}).
		WithEventFilter(newPredicate(defaults.GetGlobalCatalogSourceDefinitions())).
		Complete(r)
}

// newPredicate returns the event filter for the CatalogSource controller. It
// lets through events for the default CatalogSources and for CatalogSources
// that are no longer defaults and must be removed from the cluster.
func newPredicate(defaultCatalogsources map[string]olmv1alpha1.CatalogSource) predicate.Funcs {
	isDefault := func(obj client.Object) bool {
		_, ok := defaultCatalogsources[obj.GetName()]
		return ok
	}
	isRemoved := func(obj client.Object) bool {
		return defaults.IsRemovedSource(client.ObjectKeyFromObject(obj))
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Create events are also delivered for every CatalogSource that
			// already exists when the informer starts, so this covers removed
			// CatalogSources left over from an upgrade as well as ones that
			// are created later.
			return isRemoved(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isDefault(e.ObjectOld) || isRemoved(e.ObjectOld)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if isDefault(e.Object) {
				// If DeleteStateUnknown is true it implies that the Delete event was missed
				// and we can ignore it.
				if e.DeleteStateUnknown {
					return false
				}
				return true
			}
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isDefault(e.Object) || isRemoved(e.Object)
		},
	}
}

// blank assignment to verify that ReconcileOperatorHub implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCatalogSource{}

// ReconcileCatalogSource reconciles a CatalogSource object
type ReconcileCatalogSource struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
}

func (r *ReconcileCatalogSource) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	if defaults.IsRemovedSource(request.NamespacedName) {
		return reconcile.Result{}, defaults.EnsureRemoved(ctx, r.client, request.NamespacedName)
	}
	defaultCatalogsources := defaults.GetGlobalCatalogSourceDefinitions()
	return reconcile.Result{}, defaults.New(defaultCatalogsources, operatorhub.GetSingleton().Get()).Ensure(ctx, r.client, request.Name)
}
