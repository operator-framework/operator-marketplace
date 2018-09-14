package operatorsource

import (
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
)

func NewHandler() (Handler, DatastoreReader) {
	datastore := newDatastore()
	r := &reconciler{
		factory:   appregistry.NewClientFactory(),
		datastore: datastore,
	}

	handler := &handler{reconciler: r}
	return handler, datastore
}
