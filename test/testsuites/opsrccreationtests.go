package testsuites

import (
	"context"
	"fmt"
	"testing"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// OpSrcCreation is a test suite that ensures that the expected kubernetets resources are
// created by marketplace after the creation of an OperatorSource.
func OpSrcCreation(t *testing.T) {
	t.Run("operator-source-generates-expected-objects", testOperatorSourceGeneratesExpectedObjects)
	t.Run("registry-deployment-retains-changes", testRegistryDeploymentRetainsChanges)
}

// testOperatorSourceGeneratesExpectedObjects ensures that after creating an OperatorSource that the
// following objects are generated as a result:
// a CatalogSourceConfig
// a CatalogSource with expected labels
// a Service
// a Deployment that has reached a ready state
func testOperatorSourceGeneratesExpectedObjects(t *testing.T) {
	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for child resources.
	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestOperatorSourceName, namespace, namespace, v1.OperatorSourceKind)
	require.NoError(t, err)

	// Check that the CatalogSource has the expected labels.
	resultCatalogSource := &olm.CatalogSource{}
	err = helpers.WaitForResult(test.Global.Client, resultCatalogSource, namespace, helpers.TestOperatorSourceName)
	require.NoError(t, err)
	labels := resultCatalogSource.GetLabels()
	groupGot, ok := labels[helpers.TestOperatorSourceLabelKey]

	assert.True(t, ok)
	assert.Equal(t, helpers.TestOperatorSourceLabelValue, groupGot,
		fmt.Sprintf("The created CatalogSource %s does not have the right label[%s] - want=%s got=%s",
			resultCatalogSource.Name,
			helpers.TestOperatorSourceLabelKey,
			helpers.TestOperatorSourceLabelValue,
			groupGot))
}

// testRegistryDeploymentRetainsChanges ensures that changes likes annotations mades to the pod spec of the registry
// deployment associated with an OperatorSource is reatain across updates.
func testRegistryDeploymentRetainsChanges(t *testing.T) {
	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check for child resources.
	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestOperatorSourceName, namespace, namespace,
		v1.OperatorSourceKind)
	require.NoError(t, err)

	client := test.Global.Client

	// Get the registry deployment of the test OperatorSource
	deployment := getRegistryDeployment(t, helpers.TestOperatorSourceName, namespace)
	require.NotNil(t, deployment)

	// Add an annotation to the pod template and update the deployment
	annotationName := "always-here"
	meta.SetMetaDataAnnotation(&deployment.Spec.Template.ObjectMeta, annotationName, "test")
	err = client.Update(context.TODO(), deployment)
	require.NoError(t, err)

	err = helpers.WaitForSuccessfulDeployment(client, *deployment)
	require.NoError(t, err)

	// Get the test OperatorSource
	testOpSrc := &v1.OperatorSource{}
	namespacedName := types.NamespacedName{Name: helpers.TestOperatorSourceName, Namespace: namespace}
	err = client.Get(context.TODO(), namespacedName, testOpSrc)
	require.NoError(t, err)

	// Force an update
	testOpSrc.Status = v1.OperatorSourceStatus{}
	err = client.Update(context.TODO(), testOpSrc)
	require.NoError(t, err)

	// Check for child resources.
	err = helpers.CheckChildResourcesCreated(test.Global.Client, helpers.TestOperatorSourceName, namespace, namespace, v1.OperatorSourceKind)
	require.NoError(t, err)

	// Get the registry deployment again
	deployment = getRegistryDeployment(t, helpers.TestOperatorSourceName, namespace)
	require.NotNil(t, deployment)

	// Check that the annotation was present after the OperatorSource update
	assert.True(t, meta.HasAnnotation(deployment.Spec.Template.ObjectMeta, annotationName),
		"Annotation was not retained in pod template")

}

// getRegistryDeployment returns the deployment object for the given OperatorSource
func getRegistryDeployment(t *testing.T, name, namespace string) *apps.Deployment {
	// Get the registry deployment of the test OperatorSource
	deployment := &apps.Deployment{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := test.Global.Client.Get(context.TODO(), namespacedName, deployment)
	if err != nil {
		return nil
	}
	return deployment
}
