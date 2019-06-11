package testsuites

import (
	"context"
	"fmt"
	"testing"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

// MigrationTests is a test suite that ensures the following:
// * All stale datastore CatalogSourceConfigs created by marketplace after the creation of
// OperatorSources in a 4.1.x cluster are deleted when the cluster is migrated to openshift 4.2.0.
// * Installed CatalogSourceConfigs created during operator installation in a 4.1 cluster are deleted.
// * Existing Subscriptions are updated to reference the datastore CatalogSources instead
// of installed CatalogSources created during operator installation in a openshift 4.1.x cluster.
func MigrationTests(t *testing.T) {
	t.Run("catalogsourceconfigs-are-cleaned-up", testCatalogSourceConfigsCleanedUp)
	t.Run("subscriptions-are-updated", testSubscriptionsUpdated)
}

// testCatalogSourceConfigsCleanedUp ensures that after a cluster is migrated
// from openshift 4.1.x to openshift 4.2.0, the following stale objects are cleaned up:
// Stale CatalogSourceConfigs created during operator installation in a openshift 4.1.x cluster.
// Datastore CatalogSourceConfigs created by OperatorSources in a openshift 4.1.x cluster.
func testCatalogSourceConfigsCleanedUp(t *testing.T) {
	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for successful deletion of datastore CSC.
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, helpers.TestDatastoreCscName, namespace, namespace)
	require.NoError(t, err)

	// Check for successful deletion of installed CSC.
	installedCscName := helpers.TestInstalledCscPublisherName + namespace
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, installedCscName, namespace, namespace)
	require.NoError(t, err)
}

// testSubscriptionsUpdated ensures that Subscriptions created during operator installation
// in a openshift 4.1.x cluster are updated to reference the datastore CatalogSources.
func testSubscriptionsUpdated(t *testing.T) {
	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check that the Subscription has been successfully updated on cluster upgrade
	// from openshift 4.1.x to openshift 4.2.0 to reference the datastore CatalogSource,
	// which has the same name as the datastore CatalogSourceConfig, instead of
	// the Installed CatalogSource.
	subscription := &olm.Subscription{}
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: helpers.TestSubscriptionName, Namespace: namespace}, subscription)
	require.NoError(t, err, fmt.Sprintf("Could not get Subscription %s", helpers.TestSubscriptionName))
	require.Equal(t, helpers.TestDatastoreCscName, subscription.Spec.CatalogSource)
}
