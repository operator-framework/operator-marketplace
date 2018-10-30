package phase

import (
	"errors"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
)

// The following list is the set of phases a Marketplace object can be in while
// it is going through its reconciliation process.
const (
	// This phase applies to when an object has been created and the Phase
	// attribute is empty.
	Initial = ""

	// In this phase, for OperatorSource objects we ensure that a corresponding
	// CatalogSourceConfig object is created. For CatalogSourceConfig objects,
	// we ensure that a ConfigMap is created and is associated with a
	// CatalogSource.
	Configuring = "Configuring"

	// This phase indicates that the object has been successfully reconciled.
	Succeeded = "Succeeded"

	// This phase indicates that reconciliation of the object has failed.
	Failed = "Failed"
)

// The following list is the set of OperatorSource specific phases
const (
	// In this phase we validate the OperatorSource object.
	OperatorSourceValidating = "Validating"

	// In this phase, we connect to the specified registry, download available
	// manifest(s) and save them to the underlying datastore.
	OperatorSourceDownloading = "Downloading"
)

var (
	// Default descriptive message associated with each phase.
	phaseMessages = map[string]string{
		OperatorSourceValidating:  "Scheduled for validation",
		OperatorSourceDownloading: "Scheduled for download of operator manifest(s)",
		Configuring:               "Scheduled for configuration",
		Succeeded:                 "The object has been successfully reconciled",
		Failed:                    "Reconciliation has failed",
	}
	// ErrWrongReconcilerInvoked is thrown when a wrong reconciler is invoked.
	ErrWrongReconcilerInvoked = errors.New("Wrong phase reconciler invoked for the given object")
)

// GetMessage returns the default message associated with a
// particular phase.
func GetMessage(phaseName string) string {
	return phaseMessages[phaseName]
}

// GetNext returns a Phase object with the given phase name and default message
func GetNext(name string) *v1alpha1.Phase {
	return v1alpha1.NewPhase(name, GetMessage(name))
}

// GetNextWithMessage returns a Phase object with the given phase name and default message
func GetNextWithMessage(name string, message string) *v1alpha1.Phase {
	return v1alpha1.NewPhase(name, message)
}
