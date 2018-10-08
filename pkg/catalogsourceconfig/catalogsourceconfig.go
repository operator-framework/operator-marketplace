package catalogsourceconfig

import (
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
)

func NewHandler(r operatorsource.DatastoreReader) Handler {
	return &handler{reader: r}
}
