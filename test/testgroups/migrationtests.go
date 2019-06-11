package testgroups

import (
	"fmt"
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
)

// MigrationTestGroup creates a Subscription, an installed  CatalogSourceConfig
// and a datastore CatalogSourceConfig, restarts the marketplace operator so that
// the Migrator is run again, and then runs a series of test suites.
func MigrationTestGroup(t *testing.T) {
	// Create a ctx that is used to delete the CatalogSourceConfigs and Subscription at the completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Create the Subscription
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateSubscription(helpers.TestSubscriptionName, namespace))
	require.NoError(t, err, "Could not create Subscription")

	// Create the installed CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateInstalledCsc(namespace))
	require.NoError(t, err, "Could not create installed CatalogSourceConfig")

	// Create the datastore CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateDatastoreCsc(helpers.TestDatastoreCscName, namespace))
	require.NoError(t, err, "Could not create datastore CatalogSourceConfig")
	// Wait for the child resources to deploy successfully
	err = helpers.CheckCscChildResourcesCreated(test.Global.Client, helpers.TestDatastoreCscName, namespace, namespace)
	require.NoError(t, err, fmt.Sprintf("Could not ensure CatalogSourceConfig %s's child resources were created.", helpers.TestDatastoreCscName))

	// Restart marketplace operator
	err = helpers.RestartMarketplace(test.Global.Client, namespace)
	require.NoError(t, err, "Could not restart marketplace operator")

	// Run the test suites.
	t.Run("migration-test-suite", testsuites.MigrationTests)
}
