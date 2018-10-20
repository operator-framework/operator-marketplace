package operatorsource

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/kube"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource/phase"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	log "github.com/sirupsen/logrus"
)

// NewHandlerWithParams returns a new Handler.
func NewHandlerWithParams(factory PhaseReconcilerFactory, kubeclient kube.Client, transitioner phase.Transitioner) Handler {
	return &handler{
		factory:      factory,
		kubeclient:   kubeclient,
		transitioner: transitioner,
	}
}

// Handler is the interface that wraps the Handle method
//
// Handle handles a new event associated with OperatorSource type.
//
// ctx represents the parent context.
// event encapsulates the event fired by operator sdk.
type Handler interface {
	Handle(ctx context.Context, event sdk.Event) error
}

// handler implements the Handler interface
type handler struct {
	factory      PhaseReconcilerFactory
	kubeclient   kube.Client
	transitioner phase.Transitioner
}

func (h *handler) Handle(ctx context.Context, event sdk.Event) error {
	in, ok := event.Object.(*v1alpha1.OperatorSource)
	if !ok {
		return fmt.Errorf("casting failed, wrong type provided")
	}

	logger := log.WithFields(log.Fields{
		"type":      in.TypeMeta.Kind,
		"namespace": in.GetNamespace(),
		"name":      in.GetName(),
	})

	reconciler, err := h.factory.GetPhaseReconciler(logger, event)
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
	if updateErr := h.kubeclient.Update(out); updateErr != nil {
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
