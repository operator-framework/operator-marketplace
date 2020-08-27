package testsuites

import (
	"context"
	"io/ioutil"
	"testing"

	olm "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// DefaultCatsrc tests that the default CatalogSources are restored when updated
// or deleted
func DefaultCatsrc(t *testing.T) {
	require.DirExists(t, helpers.DefaultsDir, "Defaults directory was not present")

	fileInfos, err := ioutil.ReadDir(helpers.DefaultsDir)
	require.NoError(t, err, "Error reading default directory")
	require.True(t, len(fileInfos) > 0, "No default OperatorSources present")

	t.Run("delete-default-catalogsource", testDeleteDefaultCatsrc)
	t.Run("update-default-catalogsource", testUpdateDefaultCatsrc)
	t.Run("delete-default-catsrc-while-stopped", testDeleteDefaultCatsrcWhileStopped)
}

func testDeleteDefaultCatsrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	client := test.Global.Client
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get a default CatalogSource definitions from disk")

	deleteCatsrc := *helpers.DefaultSources[0]
	err = helpers.DeleteRuntimeObject(client, &deleteCatsrc)
	require.NoError(t, err, "Default CatalogSource could not be deleted successfully")

	clusterCatSrc := &olm.CatalogSource{}
	err = helpers.WaitForResult(client, clusterCatSrc, namespace, helpers.DefaultSources[0].Name)
	assert.NoError(t, err, "Default CatalogSource was never created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterCatSrc.Spec)
}

// testUpdateDefaultCatsrc changes a default CatalogSource and checks if it has
// been restored correctly.
func testUpdateDefaultCatsrc(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	client := test.Global.Client

	err := helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get a default CatalogSource definition from disk")

	testCatsrc := helpers.DefaultSources[0]
	updateCatsrc := &olm.CatalogSource{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: testCatsrc.Name, Namespace: testCatsrc.Namespace}, updateCatsrc)

	updateCatsrc.Spec.Publisher = "Random"
	err = helpers.UpdateRuntimeObject(client, updateCatsrc)
	require.NoError(t, err, "Default CatalogSource could not be updated successfully")

	err = helpers.WaitForExpectedSpec(client, testCatsrc.Name, testCatsrc.Namespace, testCatsrc)
	assert.NoError(t, err, "Default CatalogSource never reached the expected Spec")
}

// testDeleteDefaultCatsrcWhileStopped turns off the operator, marks a default CatalogSource
// for deletion, then starts the operator back up. When it does it expects that the
// CatalogSource is correctly recreated.
func testDeleteDefaultCatsrcWhileStopped(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	client := test.Global.Client
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: "marketplace-operator", Namespace: namespace}, &apps.Deployment{})
	if err != nil {
		t.Logf("Failed to find deployment operator-marketplace")
		return
	}

	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(0))
	require.NoError(t, err, "Could not scale down marketplace operator")

	err = helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get default CatalogSource definitions from disk")

	deleteCatsrc := *helpers.DefaultSources[0]
	err = helpers.DeleteRuntimeObject(client, &deleteCatsrc)
	require.NoError(t, err, "Default CatalogSource could not be deleted successfully")

	err = helpers.WaitForCatsrcMarkedForDeletion(client, deleteCatsrc.Name, deleteCatsrc.Namespace)
	require.NoError(t, err, "Default CatalogSource was not successfully deleted")

	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(1))
	require.NoError(t, err, "Could not scale marketplace back up")

	clusterCatSrc := &olm.CatalogSource{}
	err = helpers.WaitForResult(client, clusterCatSrc, namespace, helpers.DefaultSources[0].Name)
	assert.NoError(t, err, "Default CatalogSource was never created")

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterCatSrc.Spec)
}
