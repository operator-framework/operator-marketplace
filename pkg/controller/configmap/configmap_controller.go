package configmap

import (
	"os"

	mktconfig "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	ca "github.com/operator-framework/operator-marketplace/pkg/certificateauthority"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new ConfigMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new ReconcileConfigMap.
func newReconciler(mgr manager.Manager) *ReconcileConfigMap {
	return &ReconcileConfigMap{}
}

// add adds a new Controller to mgr with r as the ReconcileConfigMap.
func add(mgr manager.Manager, r *ReconcileConfigMap) error {
	if !mktconfig.IsAPIAvailable() {
		return nil
	}

	// Create a new controller
	c, err := controller.New("configmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ConfigMap.
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, getPredicateFunctions())
	if err != nil {
		return err
	}

	return nil
}

// getPredicateFunctions returns the predicate functions used to identify the configmap
// that contains Certificate Authority information.
// True should only be returned when the ConfigMap is updated by the cert-injector-controller.
func getPredicateFunctions() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// If the ConfigMap is ever changed we should kick off an event.
			if e.MetaOld.GetName() == ca.TrustedCaConfigMapName {
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
}

// Reconcile will restart the marketplace operator.
func (r *ReconcileConfigMap) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Check if the CA ConfigMap is in the same namespace that Marketplace is deployed in.
	objectInOtherNamespace, err := shared.IsObjectInOtherNamespace(request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	// If the CA ConfigMap is in the same namespace we should restart marketplace.
	if !objectInOtherNamespace {
		log.Infof("Certificate Authorization ConfigMap %s/%s has been updated, restarting marketplace.", request.Namespace, request.Name)
		os.Exit(0)
	}

	// Otherwise ignore the event.
	return reconcile.Result{}, nil
}
