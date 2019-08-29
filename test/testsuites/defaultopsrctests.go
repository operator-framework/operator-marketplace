package testsuites

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// DefaultOpSrc tests that the default OperatorSources are restored when updated
// or deleted
func DefaultOpSrc(t *testing.T) {
	// Check pre-requisites
	require.DirExists(t, helpers.DefaultsDir, "Defaults directory was not present")

	fileInfos, err := ioutil.ReadDir(helpers.DefaultsDir)
	require.NoError(t, err, "Error reading default directory")
	require.True(t, len(fileInfos) > 0, "No default OperatorSources present")

	t.Run("delete-default-operator-source", testDeleteDefaultOpSrc)
	t.Run("update-default-operator-source", testUpdateDefaultOpSrc)
	t.Run("update-default-registry-namespace-operator-source", testUpdateRegistryNamespaceDefaultOpSrc)
	t.Run("delete-default-opsrc-while-stopped", testDeleteDefaultOpsrcWhileStopped)
}

// testDeleteDefaultOpSrc deletes a default OperatorSource and checks if it has
// been restored correctly.
func testDeleteDefaultOpSrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's delete the OperatorSource
	deleteOpSrc := *helpers.DefaultSources[0]
	err = helpers.DeleteRuntimeObject(client, &deleteOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, helpers.DefaultSources[0].Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, helpers.DefaultSources[0].Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterOpSrc.Spec)
}

// testUpdateDefaultSources changes a default OperatorSource and checks if it has
// been restored correctly.
func testUpdateDefaultOpSrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's update the OperatorSource
	updateOpSrc := &v1.OperatorSource{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name: helpers.DefaultSources[0].Name, Namespace: helpers.DefaultSources[0].Namespace},
		updateOpSrc)

	updateOpSrc.Spec.Publisher = "Random"
	err = helpers.UpdateRuntimeObject(client, updateOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, helpers.DefaultSources[0].Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, helpers.DefaultSources[0].Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterOpSrc.Spec)
}

// testUpdateDefaultOpSrc changes a default OperatorSource and checks if it has
// been restored correctly.
func testUpdateRegistryNamespaceDefaultOpSrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's update the OperatorSource
	updateOpSrc := &v1.OperatorSource{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name: helpers.DefaultSources[0].Name, Namespace: helpers.DefaultSources[0].Namespace},
		updateOpSrc)

	updateOpSrc.Spec.RegistryNamespace = "Random"
	err = helpers.UpdateRuntimeObject(client, updateOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, helpers.DefaultSources[0].Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, helpers.DefaultSources[0].Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterOpSrc.Spec)
}

// testDeleteDefaultOpsrcWhileStopped turns off the operator, marks an operator source
// for deletion, then starts the opeator back up. When it does it expects that the
// operator source is correctly recreated.
func testDeleteDefaultOpsrcWhileStopped(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	// We are assuming that the marketplace deployment is available. This is only
	// the case if not running operator-sdk up local.
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: "marketplace-operator", Namespace: namespace}, &apps.Deployment{})
	if err != nil {
		t.Logf("Failed to find deployment operator-marketplace")
		return
	}

	// Scale down marketplace operator
	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(0))
	require.NoError(t, err, "Could not scale down marketplace operator")

	err = helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's delete the OperatorSource
	deleteOpSrc := *helpers.DefaultSources[0]
	err = helpers.DeleteRuntimeObject(client, &deleteOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	err = helpers.WaitForOpsrcMarkedForDeletionWithFinalizer(client, deleteOpSrc.Name, deleteOpSrc.Namespace)
	require.NoError(t, err, "Default OperatorSource was not successfully marked for deletion")

	// Scale the marketplace operator back up
	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(1))
	require.NoError(t, err, "Could not scale marketplace back up")

	// Now check that it came back up successfully
	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, helpers.DefaultSources[0].Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, helpers.DefaultSources[0].Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterOpSrc.Spec)
}
