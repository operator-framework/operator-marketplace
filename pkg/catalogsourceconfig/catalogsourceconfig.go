package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
)

func NewHandler(r datastore.Reader) Handler {
	return &handler{reader: r}
}
