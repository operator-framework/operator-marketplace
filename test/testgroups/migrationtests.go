package testgroups

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
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

	// We are assuming that the marketplace deployment is available. This is only
	// the case if not running operator-sdk up local.
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: "marketplace-operator", Namespace: namespace}, &apps.Deployment{})
	if err != nil {
		t.Logf("Failed to find deployment operator-marketplace")
		return
	}

	// Create the UI Subscription
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateSubscriptionDefinition(helpers.TestUISubscriptionName, namespace, helpers.TestInstalledCscPublisherName, true))
	require.NoError(t, err, "Could not create UI Subscription")

	// Create the User Subscription
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateSubscriptionDefinition(helpers.TestUserCreatedSubscriptionName, namespace, helpers.TestInstalledCscPublisherName, false))
	require.NoError(t, err, "Could not create User Subscription")

	// Create a Subscription that points to a non existent CatalogSourceConfig
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateSubscriptionDefinition(helpers.TestInvalidSubscriptionName, namespace, helpers.TestInvalidCscName, false))
	require.NoError(t, err, "Could not create Invalid Subscription")

	// Create a CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateCatalogSourceConfigDefinition(helpers.TestCatalogSourceConfigName, namespace, namespace))
	require.NoError(t, err, "Could not create CatalogSourceConfig")

	// Create a CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateCatalogSourceConfigDefinition(helpers.TestNoHyphenCatalogSourceConfigName, namespace, namespace))
	require.NoError(t, err, "Could not create CatalogSourceConfig")

	// Create the installed CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateInstalledCscDefinition(namespace))
	require.NoError(t, err, "Could not create installed CatalogSourceConfig")

	// Create the datastore CatalogSourceConfig.
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateDatastoreCscDefinition(helpers.TestDatastoreCscName, namespace))
	require.NoError(t, err, "Could not create datastore CatalogSourceConfig")
	// Wait for the child resources to deploy successfully
	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestDatastoreCscName, namespace, namespace, v2.CatalogSourceConfigKind)
	require.NoError(t, err, fmt.Sprintf("Could not ensure CatalogSourceConfig %s's child resources were created.", helpers.TestDatastoreCscName))

	// Restart marketplace operator
	err = helpers.RestartMarketplace(test.Global.Client, namespace)
	require.NoError(t, err, "Could not restart marketplace operator")

	// Run the test suites.
	t.Run("migration-test-suite", testsuites.MigrationTests)
}
