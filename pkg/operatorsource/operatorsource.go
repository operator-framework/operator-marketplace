package operatorsource

import (
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/kube"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource/phase"
)

// NewHandler returns an instance of the Handler interface
// that can be used to reconcile an OperatorSource type object.
func NewHandler() (Handler, datastore.Reader) {
	datastore := datastore.New()
	kubeclient := kube.New()
	transitioner := phase.NewTransitioner()

	phaseReconcilerFactory := &phaseReconcilerFactory{
		registryClientFactory: appregistry.NewClientFactory(),
		datastore:             datastore,
		kubeclient:            kubeclient,
	}

	handler := NewHandlerWithParams(phaseReconcilerFactory, kubeclient, transitioner)

	return handler, datastore
}
