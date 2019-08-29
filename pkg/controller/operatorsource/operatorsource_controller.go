package operatorsource

import (
	"context"
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	operatorsourcehandler "github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	"github.com/operator-framework/operator-marketplace/pkg/watches"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new OperatorSource Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new ReconcileOperatorSource
func newReconciler(mgr manager.Manager) *ReconcileOperatorSource {
	client := mgr.GetClient()
	handler := operatorsourcehandler.NewHandler(mgr, client)
	return &ReconcileOperatorSource{
		client:                client,
		OperatorSourceHandler: handler,
	}
}

// add adds a new Controller to mgr with r as the ReconcileOperatorSource
func add(mgr manager.Manager, r *ReconcileOperatorSource) error {
	// Create a new controller
	c, err := controller.New("operatorsource-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OperatorSource
	err = c.Watch(&source.Kind{Type: &v1.OperatorSource{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for child resource deletions
	err = watches.WatchChildResourcesDeletionEvents(c, r.client, v1.OperatorSourceKind)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileOperatorSource{}

// ReconcileOperatorSource reconciles a OperatorSource object
type ReconcileOperatorSource struct {
	OperatorSourceHandler operatorsourcehandler.Handler
	client                client.Client
}

// Reconcile reads that state of the cluster for a OperatorSource object and makes changes based on the state read
// and what is in the OperatorSource.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOperatorSource) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling OperatorSource %s/%s\n", request.Namespace, request.Name)
	// Reconcile kicked off, message Sync Channel with sync event
	status.SendSyncMessage(nil)

	// Fetch the OperatorSource instance
	instance := &v1.OperatorSource{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// If the object is not found but is one of the default operator sources, let's ensure it gets recreated.
			if defaults.IsDefaultSource(request.Name) {
				err = defaults.New(defaults.GetGlobalDefinitions(), operatorhub.GetSingleton().Get()).Ensure(r.client, request.Name)
				if err != nil {
					// If we run into an error when attempting to ensure the default, let's requeue and try again
					return reconcile.Result{}, err
				}
			}

			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - report a failed sync and requeue the request.
		status.SendSyncMessage(err)
		return reconcile.Result{}, err
	}

	// Needed because sdk does not get the gvk
	instance.EnsureGVK()

	requeue, err := r.OperatorSourceHandler.Handle(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	if requeue {
		return reconcile.Result{RequeueAfter: time.Second * 5}, nil
	}

	return reconcile.Result{}, nil
}
