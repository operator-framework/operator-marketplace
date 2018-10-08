package operatorsource

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// Handler is the interface that wraps the Handle method
//
// Handle handles a new event associated with OperatorSource type
type Handler interface {
	Handle(ctx context.Context, event sdk.Event) error
}

type handler struct {
	reconciler Reconciler
}

func (h *handler) Handle(ctx context.Context, event sdk.Event) error {
	opsrc := event.Object.(*v1alpha1.OperatorSource)

	if event.Deleted {
		logrus.Infof("No action taken, object has been deleted [type=%s object=%s/%s]",
			opsrc.TypeMeta.Kind, opsrc.ObjectMeta.Namespace, opsrc.ObjectMeta.Name)

		return nil
	}

	reconciled, err := h.reconciler.IsAlreadyReconciled(opsrc)
	if err != nil {
		return err
	}

	if reconciled {
		logrus.Infof("Already reconciled, no action taken [type=%s object=%s/%s]",
			opsrc.TypeMeta.Kind, opsrc.ObjectMeta.Namespace, opsrc.ObjectMeta.Name)

		return nil
	}

	if err := h.reconciler.Reconcile(opsrc); err != nil {
		return err
	}

	logrus.Infof("Reconciliation completed successfully [type=%s object=%s/%s]",
		opsrc.TypeMeta.Kind, opsrc.ObjectMeta.Namespace, opsrc.ObjectMeta.Name)
	return nil
}
