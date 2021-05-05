package testsuites

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	olm "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	// TODO(tflannag): It would be nice if we could skip this test when the operator
	// is being run locally: `if test.Framework.LocalOperator { t.Skip(...) }``
	t.Run("delete-default-catsrc-while-stopped", testDeleteDefaultCatsrcWhileStopped)
	t.Run("catsrc-behavior-when-disabled", testDefaultCatsrcWhileDisabled)
}

func testDeleteDefaultCatsrc(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	client := test.Global.Client
	namespace, err := ctx.GetNamespace()
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
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	err := helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get a default CatalogSource definition from disk")
	require.NotEmpty(t, helpers.DefaultSources)

	client := test.Global.Client
	testCatsrc := helpers.DefaultSources[0]
	updateCatsrc := &olm.CatalogSource{}

	err = client.Get(context.TODO(), types.NamespacedName{Name: testCatsrc.Name, Namespace: testCatsrc.Namespace}, updateCatsrc)
	require.NoError(t, err, "failed to query for the %s CatalogSource in the %s namespace", testCatsrc.Name, testCatsrc.Namespace)

	updateCatsrc.Spec.Publisher = "Random"
	err = helpers.UpdateCatalogSourceWithRetries(client, updateCatsrc)
	require.NoError(t, err, "Default CatalogSource could not be updated successfully")

	err = helpers.WaitForExpectedSpec(client, testCatsrc.Name, testCatsrc.Namespace, testCatsrc)
	assert.NoError(t, err, "Default CatalogSource never reached the expected Spec")
}

// testDeleteDefaultCatsrcWhileStopped turns off the operator, marks a default CatalogSource
// for deletion, then starts the operator back up. When it does it expects that the
// CatalogSource is correctly recreated.
func testDeleteDefaultCatsrcWhileStopped(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	const marketplaceName = "marketplace-operator"

	client := test.Global.Client
	err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: marketplaceName, Namespace: namespace}, &apps.Deployment{})
	require.NoError(t, err, "Failed to find deployment %s", marketplaceName)

	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(0))
	require.NoError(t, err, "Could not scale down the %s deployment to zero replicas", marketplaceName)

	err = helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get default CatalogSource definitions from disk")

	deleteCatsrc := *helpers.DefaultSources[0]
	err = helpers.DeleteRuntimeObject(client, &deleteCatsrc)
	require.NoError(t, err, "Default %s CatalogSource could not be deleted successfully", deleteCatsrc.Name)

	err = helpers.WaitForCatsrcMarkedForDeletion(client, deleteCatsrc.Name, deleteCatsrc.Namespace)
	require.NoError(t, err, "Default %s CatalogSource was not successfully deleted", deleteCatsrc.Name)

	err = helpers.ScaleMarketplace(test.Global.Client, namespace, int32(1))
	require.NoError(t, err, "Could not scale the %s deployment back up", marketplaceName)

	clusterCatSrc := &olm.CatalogSource{}
	err = helpers.WaitForResult(client, clusterCatSrc, namespace, helpers.DefaultSources[0].Name)
	assert.NoError(t, err, "Default %s CatalogSource was never created", clusterCatSrc.Name)

	assert.ObjectsAreEqualValues(helpers.DefaultSources[0].Spec, clusterCatSrc.Spec)
}

// testDefaultCatsrcWhileDisabled checks that when a default CatalogSources is disabled, the marketplace
// operator allows for the creation/update/deletion of a CatalogSource with the same name as one of the
// default CatalogSources, without reverting the CatalogSource back to default. It also checks that when
// the default CatalogSources are re-enabled, the default specs are restored for the CatalogSources which
// have been re-enabled.
func testDefaultCatsrcWhileDisabled(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetOperatorNamespace()
	require.NoError(t, err, "Could not get the marketplace namespace")

	err = toggle(t, 4, true, false) //Disable all default CatalogSources
	require.NoError(t, err, "Could not disable default CatalogSources")

	err = checkDeleted(4, namespace)
	require.NoError(t, err, "Default CatalogSource was not removed from the cluster")

	catSrcName := "redhat-operators"
	client := test.Global.Client

	customCatsrc := olm.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CatalogSource",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      catSrcName,
			Namespace: namespace,
		},
		Spec: olm.CatalogSourceSpec{
			SourceType:  "grpc",
			Image:       "my-cool-registry/my-namespace/my-cool-index",
			DisplayName: "My Cool Red Hat Operators",
			Publisher:   "Me",
		},
	}
	err = helpers.CreateRuntimeObjectNoCleanup(client, &customCatsrc)
	require.NoError(t, err, "Could not create custom CatalogSource %s", catSrcName)

	customCatsrc, err = checkForCatsrc(catSrcName, namespace)
	require.NoError(t, err, "Custom CatalogSource %s was removed from the cluster", catSrcName)

	customCatsrc.Spec.Image = "my-cool-registry/my-namespace/my-other-cool-index"
	err = helpers.UpdateCatalogSourceWithRetries(client, &customCatsrc)
	require.NoError(t, err, "failed to update the CatalogSource custom resource")

	err = wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
		updatedCatsrc, err := checkForCatsrc(catSrcName, namespace)
		if err != nil || !defaults.AreCatsrcSpecsEqual(&customCatsrc.Spec, &updatedCatsrc.Spec) {
			return false, err
		}
		return true, nil
	})
	require.NoError(t, err, "The update on the custom CatalogSource was reverted back")

	customCatsrc, err = checkForCatsrc(catSrcName, namespace)
	require.NoError(t, err, "Custom CatalogSource %s was removed from the cluster after marketplace was restarted", catSrcName)

	err = toggle(t, 4, false, false) //Re-enable all default CatalogSources
	require.NoError(t, err, "Could not enable default CatalogSources")

	err = checkCreated(4, namespace)
	require.NoError(t, err, "Default CatalogSources were not created properly")

	err = helpers.InitCatSrcDefinition()
	require.NoError(t, err, "Could not get a default CatalogSource definitions from disk")

	for _, catsrcDef := range helpers.DefaultSources {
		if catsrcDef.Name != catSrcName {
			continue
		}
		err := wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
			clusterCatsrc := &olm.CatalogSource{}
			err = client.Get(context.TODO(), types.NamespacedName{Name: catSrcName, Namespace: namespace}, clusterCatsrc)
			if err != nil || !defaults.AreCatsrcSpecsEqual(&clusterCatsrc.Spec, &catsrcDef.Spec) {
				return false, err
			}
			return true, nil
		})
		require.NoError(t, err, "Default CatalogSource was not restored properly")
	}
}

// checkForCatsrc checks if CatalogSource is present, and is not being removed after some time has passed
func checkForCatsrc(name, namespace string) (olm.CatalogSource, error) {
	client := test.Global.Client
	// Wait for a minute
	wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
		return false, nil
	})
	//Check CatalogSource is present in the cluster
	catsrc := olm.CatalogSource{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &catsrc)
	if err != nil {
		return olm.CatalogSource{}, err
	}
	return catsrc, nil
}
