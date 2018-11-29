package operatorsource

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewHandlerWithParams returns a new Handler.
func NewHandlerWithParams(client client.Client, scheme *runtime.Scheme, factory PhaseReconcilerFactory, transitioner phase.Transitioner) Handler {
	return &operatorsourcehandler{
		client:       client,
		scheme:       scheme,
		factory:      factory,
		transitioner: transitioner,
	}
}

func NewHandler(mgr manager.Manager) Handler {
	return &operatorsourcehandler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		factory: &phaseReconcilerFactory{
			registryClientFactory: appregistry.NewClientFactory(),
			datastore:             datastore.Cache,
			client:                mgr.GetClient(),
		},
		transitioner: phase.NewTransitioner(),
	}
}

// Handler is the interface that wraps the Handle method
//
// Handle handles a new event associated with OperatorSource type.
//
// ctx represents the parent context.
// event encapsulates the event fired by operator sdk.
type Handler interface {
	Handle(ctx context.Context, operatorSource *v1alpha1.OperatorSource) error
}

// operatorsourcehandler implements the Handler interface
type operatorsourcehandler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	factory      PhaseReconcilerFactory
	transitioner phase.Transitioner
}

func (h *operatorsourcehandler) Handle(ctx context.Context, in *v1alpha1.OperatorSource) error {
	logger := log.WithFields(log.Fields{
		"type":      in.TypeMeta.Kind,
		"namespace": in.GetNamespace(),
		"name":      in.GetName(),
	})

	reconciler, err := h.factory.GetPhaseReconciler(logger, in)
	if err != nil {
		return err
	}

	out, status, err := reconciler.Reconcile(ctx, in)

	// If reconciliation threw an error, we can't quit just yet. We need to
	// figure out whether the OperatorSource object needs to be updated.

	if !h.transitioner.TransitionInto(&out.Status.CurrentPhase, status) {
		// OperatorSource object has not changed, no need to update. We are done.
		return err
	}

	// OperatorSource object has been changed. At this point, reconciliation has
	// either completed successfully or failed.
	// In either case, we need to update the modified OperatorSource object.
	if updateErr := h.client.Update(ctx, out); updateErr != nil {
		if err == nil {
			// No reconciliation err, but update of object has failed!
			return updateErr
		}

		// Presence of both Reconciliation error and object update error.
		logger.Errorf("Failed to update object - %v", updateErr)

		// TODO: find a way to chain the update error?
		return err
	}

	return err
}
