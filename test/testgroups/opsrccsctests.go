package testgroups

import (
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
)

// OpSrcTestGroup creates an OperatorSource and then runs a series of
// test suites that rely on these resources.
func OpSrcTestGroup(t *testing.T) {
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

	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestOperatorSourceName, namespace, namespace,
		v1.OperatorSourceKind)
	require.NoError(t, err)

	// Run the test suites.
	if isConfigAPIPresent, _ := helpers.EnsureConfigAPIIsAvailable(); isConfigAPIPresent == true {
		t.Run("proxy-test-suite", testsuites.ProxyTests)
		t.Run("operatorhub-test-suite", testsuites.OperatorHubTests)
	}
	t.Run("opsrc-creation-test-suite", testsuites.OpSrcCreation)
	t.Run("watch-tests", testsuites.WatchTests)
}
