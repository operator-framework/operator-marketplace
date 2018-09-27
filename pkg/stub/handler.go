package stub

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/catalogsourceconfig"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

func NewHandler() sdk.Handler {
	opsrcHandler, _ := operatorsource.NewHandler()
	cscHandler := catalogsourceconfig.NewHandler()
	return &Handler{
		operatorSourceHandler:      opsrcHandler,
		catalogSourceConfigHandler: cscHandler,
	}
}

type Handler struct {
	operatorSourceHandler      operatorsource.Handler
	catalogSourceConfigHandler operatorsource.Handler
}

// Handle function for handling CatalogSourceConfig and OperatorSource CR events
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch event.Object.(type) {
	case *v1alpha1.CatalogSourceConfig:
		if err := h.catalogSourceConfigHandler.Handle(ctx, event); err != nil {
			logrus.Errorf("CatalogSourceConfig reconciliation error: %v", err)
			return err
		}

	case *v1alpha1.OperatorSource:
		if err := h.operatorSourceHandler.Handle(ctx, event); err != nil {
			logrus.Errorf("OperatorSource reconciliation error: %v", err)
			return err
		}
	}

	return nil
}
