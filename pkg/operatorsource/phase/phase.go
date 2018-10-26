package phase

import (
	"errors"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
)

const (
	// The prefix to a name we use to create CatalogSourceConfig object.
	catalogSourceConfigPrefix = "opsrc"
)

var (
	// ErrWrongReconcilerInvoked is thrown when a wrong reconciler is invoked.
	ErrWrongReconcilerInvoked = errors.New("Wrong phase reconciler invoked for the given object")
)

// Given a name of OperatorSource object, this function returns the name
// of the corresponding CatalogSourceConfig type object.
func getCatalogSourceConfigName(operatorsourceName string) string {
	return fmt.Sprintf("%s-%s", catalogSourceConfigPrefix, operatorsourceName)
}

func getNextPhase(phase string) *v1alpha1.Phase {
	return v1alpha1.NewPhase(phase, v1alpha1.GetOperatorSourcePhaseMessage(phase))
}

func getNextPhaseWithMessage(phase string, message string) *v1alpha1.Phase {
	return v1alpha1.NewPhase(phase, message)
}
