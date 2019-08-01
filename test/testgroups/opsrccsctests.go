package testgroups

import (
	"fmt"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
)

// OpSrcCscTestGroup creates an OperatorSource and a CatalogSourceConfig and then runs a series of
// test suites that rely on these resources.
func OpSrcCscTestGroup(t *testing.T) {
	// Create a ctx that is used to delete the OperatorSource and CatalogSouceConfig at the
	// completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Create the OperatorSource.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateOperatorSourceDefinition(helpers.TestOperatorSourceName, namespace))
	require.NoError(t, err, "Could not create OperatorSource")

	// Create the CatalogSourceConfig
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateCatalogSourceConfigDefinition(
		helpers.TestCatalogSourceConfigName, namespace, helpers.TestCatalogSourceConfigTargetNamespace))
	require.NoError(t, err, "Could not create CatalogSourceConfig")

	expectedPhase := "Succeeded"
	csc, err := helpers.WaitForCscExpectedPhaseAndMessage(test.Global.Client, helpers.TestCatalogSourceConfigName, namespace, expectedPhase, "")
	require.NoError(t, err, fmt.Sprintf("CatalogSourceConfig never reached the expected phase %s", expectedPhase))
	require.NotNil(t, csc, "Could not retrieve CatalogSourceConfig")

	// Confirm child resources were created without errors.
	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestCatalogSourceConfigName,
		namespace, helpers.TestCatalogSourceConfigTargetNamespace, v2.CatalogSourceConfigKind)
	require.NoError(t, err, "CatalogSourceConfig child resources were not created")

	// Run the test suites.
	t.Run("opsrc-creation-test-suite", testsuites.OpSrcCreation)
	t.Run("csc-target-namespace-test-suite", testsuites.CscTargetNamespace)
	t.Run("packages-test-suite", testsuites.PackageTests)
	t.Run("csc-invalid-tests", testsuites.CscInvalid)
	t.Run("watch-tests", testsuites.WatchTests)
}
