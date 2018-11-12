package catalogsourceconfig

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// Handler is the interface that wraps the Handle method
type Handler interface {
	Handle(ctx context.Context, event sdk.Event) error
}

type handler struct {
	factory      PhaseReconcilerFactory
	transitioner phase.Transitioner
	reader       datastore.Reader
}

// Handle handles a new event associated with the CatalogSourceConfig type.
func (h *handler) Handle(ctx context.Context, event sdk.Event) error {
	in, ok := event.Object.(*v1alpha1.CatalogSourceConfig)
	if !ok {
		return fmt.Errorf("Received event for incorrect kind %v", event.Object.GetObjectKind().GroupVersionKind())
	}

	log := getLoggerWithCatalogSourceConfigTypeFields(in)
	reconciler, err := h.factory.GetPhaseReconciler(log, event)
	if err != nil {
		return err
	}

	out, status, err := reconciler.Reconcile(ctx, in)

	// If reconciliation threw an error, we can't quit just yet. We need to
	// figure out whether the CatalogSourceConfig object needs to be updated.
	if !h.transitioner.TransitionInto(&out.Status.CurrentPhase, status) {
		// CatalogSourceConfig object has not changed, no need to update. We are done.
		return err
	}

	// CatalogSourceConfig object has been changed. At this point,
	// reconciliation has either completed successfully or failed. In either
	// case, we need to update the modified CatalogSourceConfig object.
	if updateErr := sdk.Update(out); updateErr != nil {
		if err == nil {
			// No reconciliation err, but update of object has failed!
			return updateErr
		}

		// Presence of both Reconciliation error and object update error.
		log.Errorf("Failed to update object - %v", updateErr)

		// TODO: find a way to chain the update error?
		return err
	}

	return err
}

// getLoggerWithCatalogSourceConfigTypeFields returns a logger entry that can be
// used for consistent logging.
func getLoggerWithCatalogSourceConfigTypeFields(csc *v1alpha1.CatalogSourceConfig) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"type":            csc.TypeMeta.Kind,
		"targetNamespace": csc.Spec.TargetNamespace,
		"name":            csc.GetName(),
	})
}
