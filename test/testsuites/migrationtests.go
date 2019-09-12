package testsuites

import (
	"context"
	"fmt"
	"testing"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
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
	t.Run("datastore-and-installed-catalogsourceconfigs-are-cleaned-up", testDatastoreAndInstalledCatalogSourceConfigsCleanedUp)
	t.Run("subscriptions-are-updated", testSubscriptionsUpdated)
	t.Run("user-created-subscriptions-are-not-updated", testUserCreatedSubscriptions)
	t.Run("catalogsourceconfigs-not-deleted", testCatalogSourceConfigsNotDeleted)
	t.Run("bad-subscription-ignored", testCatalogSourceConfigsNotDeleted)
	t.Run("bad-catalogsourceconfig-name-resolved", testBadCatalogSourceConfigNameResolved)
}

// testDatastoreAndInstalledCatalogSourceConfigsCleanedUp ensures that after a cluster is migrated
// from openshift 4.1.x to openshift 4.2.0, the following stale objects are cleaned up:
// Stale CatalogSourceConfigs created during operator installation in a openshift 4.1.x cluster.
// Datastore CatalogSourceConfigs created by OperatorSources in a openshift 4.1.x cluster.
func testDatastoreAndInstalledCatalogSourceConfigsCleanedUp(t *testing.T) {
	// Create a ctx that is used to delete the CatalogSourceConfigs and Subscription at the completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for successful deletion of datastore CSC.
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, helpers.TestDatastoreCscName, namespace, namespace)
	require.NoError(t, err)

	// Check for successful deletion of installed CSC.
	installedCscName := fmt.Sprintf("%s-%s", helpers.TestInstalledCscPublisherName, namespace)
	err = helpers.CheckCscSuccessfulDeletion(test.Global.Client, installedCscName, namespace, namespace)
	require.NoError(t, err)
}

// testSubscriptionsUpdated ensures that Subscriptions created during operator installation
// in a openshift 4.1.x cluster are updated to reference the datastore CatalogSources.
func testSubscriptionsUpdated(t *testing.T) {
	// Create a ctx that is used to delete the CatalogSourceConfigs and Subscription at the completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check that the Subscription has been successfully updated on cluster upgrade
	// from openshift 4.1.x to openshift 4.2.0 to reference the datastore CatalogSource,
	// which has the same name as the datastore CatalogSourceConfig, instead of
	// the Installed CatalogSource.
	subscription := &olm.Subscription{}
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: helpers.TestUISubscriptionName, Namespace: namespace}, subscription)
	require.NoError(t, err, fmt.Sprintf("Could not get Subscription %s", helpers.TestUISubscriptionName))
	require.Equal(t, helpers.TestDatastoreCscName, subscription.Spec.CatalogSource)
}

// testUserCreatedSubscriptions ensures that after a cluster is migrated
// from openshift 4.1.x to openshift 4.2.0, the subscriptions not created by
// the UI (i.e. without the owner labels) are not updated.
func testUserCreatedSubscriptions(t *testing.T) {
	// Create a ctx that is used to delete the CatalogSourceConfigs and Subscription at the completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for user created subscription was not updated after migration
	err = helpers.CheckSubscriptionNotUpdated(test.Global.Client, namespace, helpers.TestUserCreatedSubscriptionName, helpers.TestInstalledCscPublisherName)
	require.NoError(t, err, fmt.Sprintf("User-created Subscription %s was updated after migration", helpers.TestUserCreatedSubscriptionName))
}

// testCatalogSourceConfigsNotDeleted ensures that after a cluster is migrated
// from openshift 4.1.x to openshift 4.2.0, the CatalogSourceConfigs not created by
// the OperatorSources or the UI are not deleted.
func testCatalogSourceConfigsNotDeleted(t *testing.T) {
	// Create a ctx that is used to delete the CatalogSourceConfigs and Subscription at the completion of this function.
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for CatalogSourceConfig is still there after migration
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: helpers.TestCatalogSourceConfigName, Namespace: namespace}, &v2.CatalogSourceConfig{})
	require.NoError(t, err, fmt.Sprintf("Non datastore or non UI CatalogSourceConfig %s was deleted after migration", helpers.TestCatalogSourceConfigName))
}

// testInvalidSubscriptionIgnored ensures that after a cluster is migrated
// any existing subscription that could not be migrated (because it points
// to a non existent package, for example) is ignored
func testInvalidSubscriptionIgnored(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	err = helpers.CheckSubscriptionNotUpdated(test.Global.Client, namespace, helpers.TestInvalidSubscriptionName, helpers.TestInvalidCscName)
	require.NoError(t, err, fmt.Sprintf("User-created Subscription %s was updated after migration", helpers.TestInvalidSubscriptionName))
}

// testBadCatalogSourceConfigNameResolved ensures that a catalogsource with a 
// non-hyphenated name does not crash the pod during migration 
func testBadCatalogSourceConfigNameResolved(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check non-hyphenated CatalogSourceConfig is still there after migration
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: helpers.TestNoHyphenCatalogSourceConfigName, Namespace: namespace}, &v2.CatalogSourceConfig{})
	require.NoError(t, err, fmt.Sprintf("Non-hyphenated CatalogSourceConfig %s was deleted after migration", helpers.TestNoHyphenCatalogSourceConfigName))
}
