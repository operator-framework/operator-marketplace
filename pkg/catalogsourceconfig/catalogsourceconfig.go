package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// NewHandler returns an instance of the Handler interface
// that can be used to reconcile a CatalogSourceConfig object.
func NewHandler(r datastore.Reader) Handler {
	return &handler{
		factory:      &phaseReconcilerFactory{reader: r},
		reader:       r,
		transitioner: phase.NewTransitioner(),
	}
}
