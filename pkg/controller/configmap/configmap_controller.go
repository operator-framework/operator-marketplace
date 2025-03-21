package configmap

import (
	"context"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	mktconfig "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	ca "github.com/operator-framework/operator-marketplace/pkg/certificateauthority"
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Add creates a new ConfigMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ options.ControllerOptions) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new ReconcileConfigMap.
func newReconciler(mgr manager.Manager) *ReconcileConfigMap {
	client := mgr.GetClient()
	return &ReconcileConfigMap{
		client:  client,
		handler: ca.NewHandler(client),
	}
}

// add adds a new Controller to mgr with r as the ReconcileConfigMap.
func add(mgr manager.Manager, r *ReconcileConfigMap) error {
	if !mktconfig.IsAPIAvailable() || !isRunningOnPod() {
		log.Printf("[ca] Config API is not available or marketplace is not being ran on a pod, the ConfigMap controller will not be started.")
		return nil
	}

	return builder.ControllerManagedBy(mgr).
		Named("configmap-controller").
		For(&corev1.ConfigMap{}).
		WithEventFilter(getPredicateFunctions()).
		Complete(r)
}

// getPredicateFunctions returns the predicate functions used to identify the configmap
// that contains Certificate Authority information.
// True should only be returned when the ConfigMap is updated by the cert-injector-controller.
func getPredicateFunctions() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// If the ConfigMap is created we should kick off an event.
			if e.Object.GetName() == ca.TrustedCaConfigMapName {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// If the ConfigMap is updated we should kick off an event.
			if e.ObjectOld.GetName() == ca.TrustedCaConfigMapName {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

var _ reconcile.Reconciler = &ReconcileConfigMap{}

// ReconcileConfigMap reconciles a ConfigMap object.
type ReconcileConfigMap struct {
	client  client.Client
	handler ca.Handler
}

// Reconcile will restart the marketplace operator if the Certificate Authority ConfigMap is
// not in sync with the Certificate Authority bundle on disk..
func (r *ReconcileConfigMap) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling ConfigMap %s/%s", request.Namespace, request.Name)

	// Check if the CA ConfigMap is in the same namespace that Marketplace is deployed in.
	isConfigMapInOtherNamespace, err := shared.IsObjectInOtherNamespace(request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if isConfigMapInOtherNamespace {
		return reconcile.Result{}, nil
	}

	// Get configMap object
	caConfigMap := &corev1.ConfigMap{}
	if err := r.client.Get(ctx, request.NamespacedName, caConfigMap); err != nil {
		// Requested object was not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	return reconcile.Result{}, r.handler.Handle(ctx, caConfigMap)
}

// isRunningOnPod returns true if marketplace is being ran on a pod.
func isRunningOnPod() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	return !os.IsNotExist(err)
}
