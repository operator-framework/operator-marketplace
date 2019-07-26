package testsuites

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var defaultOpSrc *v1.OperatorSource

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

	err = initOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's delete the OperatorSource
	deleteOpSrc := *defaultOpSrc
	err = helpers.DeleteRuntimeObject(client, &deleteOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, defaultOpSrc.Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, defaultOpSrc.Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(defaultOpSrc.Spec, clusterOpSrc.Spec)
}

// testUpdateDefaultOpSrc changes a default OperatorSource and checks if it has
// been restored correctly.
func testUpdateDefaultOpSrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = initOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's update the OperatorSource
	updateOpSrc := &v1.OperatorSource{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name: defaultOpSrc.Name, Namespace: defaultOpSrc.Namespace},
		updateOpSrc)

	updateOpSrc.Spec.Publisher = "Random"
	err = helpers.UpdateRuntimeObject(client, updateOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, defaultOpSrc.Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, defaultOpSrc.Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(defaultOpSrc.Spec, clusterOpSrc.Spec)
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

	err = initOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Now let's update the OperatorSource
	updateOpSrc := &v1.OperatorSource{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name: defaultOpSrc.Name, Namespace: defaultOpSrc.Namespace},
		updateOpSrc)

	updateOpSrc.Spec.RegistryNamespace = "Random"
	err = helpers.UpdateRuntimeObject(client, updateOpSrc)
	require.NoError(t, err, "Default OperatorSource could not be deleted successfully")

	// Ensure the OperatorSource phase is "Succeeded"
	clusterOpSrc, err := helpers.WaitForOpSrcExpectedPhaseAndMessage(client, defaultOpSrc.Name, namespace, "Succeeded",
		"The object has been successfully reconciled")
	assert.NoError(t, err, "Default OperatorSource never reached the succeeded phase")

	// Check for the child resources which implies that the OperatorSource was recreated
	err = helpers.CheckChildResourcesCreated(client, defaultOpSrc.Name, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err, "Could not ensure that child resources were created")

	assert.ObjectsAreEqualValues(defaultOpSrc.Spec, clusterOpSrc.Spec)
}

func initOpSrcDefinition() error {
	if defaultOpSrc != nil {
		return nil
	}

	fileInfos, _ := ioutil.ReadDir(helpers.DefaultsDir)
	fileName := fileInfos[0].Name()

	file, err := os.Open(filepath.Join(helpers.DefaultsDir, fileName))
	if err != nil {
		return err
	}

	defaultOpSrc = &v1.OperatorSource{}
	decoder := yaml.NewYAMLOrJSONDecoder(file, 1024)
	err = decoder.Decode(defaultOpSrc)
	if err != nil {
		defaultOpSrc = nil
		return err
	}
	return nil
}
