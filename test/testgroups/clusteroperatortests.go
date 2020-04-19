package testgroups

import (
	"context"
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ClusterOperatorTestGroup runs test suites that check the status of the Cluster Operator
func ClusterOperatorTestGroup(t *testing.T) {

	// Run start-up test suite
	t.Run("cluster-operator-status-on-startup-test-suite", testsuites.ClusterOperatorStatusOnStartup)

	// Create a ctx that is used to create and eventually delete the OperatorSource and CatalogSourceConfig at the completion of this function.
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

	// Create the OperatorSource.
	opsrcDefinition := helpers.CreateOperatorSourceDefinition(helpers.TestOpsrcNameForClusterOperator, namespace)
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, opsrcDefinition)
	require.NoError(t, err, "Could not create OperatorSource")

	// Run opsrc creation related test suites.
	t.Run("cluster-operator-status-on-custom-opsrc-creation-test-suite", testsuites.ClusterOperatorStatusOnCustomResourceCreation)

	// Delete the OperatorSource.
	err = helpers.DeleteRuntimeObject(test.Global.Client, opsrcDefinition)
	require.NoError(t, err, "Could not delete OperatorSource")

	// Run opsrc deletion releated test suites.
	t.Run("cluster-operator-status-on-custom-opsrc-deletion-test-suite", testsuites.ClusterOperatorStatusOnCustomResourceDeletion)
}
