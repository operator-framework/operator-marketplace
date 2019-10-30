package testsuites

import (
	"context"
	"fmt"
	"testing"
	"time"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// OperatorHubTests is a test suite that tests various configuration combinations wrt
// disabling and enabling default OperatorSource.
func OperatorHubTests(t *testing.T) {
	t.Run("disable-test", testDisable)
	t.Run("disable-all", testDisableAll)
	t.Run("disable-all-enable-one", testDisableAllEnableOne)
	t.Run("disable-two-test", testDisableTwo)
	t.Run("disable-enable-test", testDisableEnable)
	t.Run("disable-non-default", testDisableNonDefault)
	t.Run("disable-all-check-cluster-status", testClusterStatusDefaultsDisabled)
	t.Run("disable-some-check-cluster-status",testSomeClusterStatusDefaultsDisabled)
}

// testDisable tests disabling a default OperatorSource and ensures that it is not present on the cluster
func testDisable(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	// Disable a default OperatorSource
	err = toggle(t, 1, true, false)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	// Check that the OperatorSource and its child resource have been deleted
	err = checkDeleted(1, namespace)
	assert.NoError(t, err, "Default OperatorSource or child resources still present on the cluster")

	// Check the cluster OperatorHub resource
	err = checkClusterOperatorHub(t, 1)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	resetClusterOperatorHub(t, namespace)
}

// testDisableAll tests disabling all the default OperatorSources using DisableAllDefaultSources and ensures that they
// are not present on the cluster
func testDisableAll(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = toggle(t, 0, true, true)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkDeleted(3, namespace)
	assert.NoError(t, err, "All default OperatorSource(s) have not been disabled")

	err = checkClusterOperatorHub(t, 3)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

}

// testDisableAllEnableOne tests if disabled=true in a source present in sources overrides DisableAllDefaultSources
// and ensures the correct number of default OperatorSources are present on the cluster.
func testDisableAllEnableOne(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = toggle(t, 1, false, true)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkOpSrcAndChildrenArePresent(helpers.DefaultSources[0].Name, namespace)
	assert.NoError(t, err, "Expected default OperatorSource is not present")

	err = checkOpSrcAndChildrenAreDeleted(helpers.DefaultSources[1].Name, namespace)
	assert.NoError(t, err, "Default OperatorSource has not been disabled")

	err = checkOpSrcAndChildrenAreDeleted(helpers.DefaultSources[2].Name, namespace)
	assert.NoError(t, err, "Default OperatorSource has not been disabled")

	err = checkClusterOperatorHub(t, 2)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

}

// testDisableTwo tests disabling two default OperatorSource and ensures that they are not present on the cluster
func testDisableTwo(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	// Disable two default OperatorSources
	err = toggle(t, 2, true, false)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	// Check that the OperatorSources and its child resource have been deleted
	err = checkDeleted(2, namespace)
	assert.NoError(t, err, "Default OperatorSource(s) or child resources still present on the cluster")

	// Check the cluster OperatorHub resource
	err = checkClusterOperatorHub(t, 2)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	resetClusterOperatorHub(t, namespace)
}

// testDisableEnable tests disabling a defaults OperatorSource and then enables it. At each step resources on the
// cluster are checked appropriately.
func testDisableEnable(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	// Disable a default OperatorSource
	err = toggle(t, 1, true, false)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkDeleted(1, namespace)
	assert.NoError(t, err, "Default OperatorSource(s) or child resources still present on the cluster")

	// Check the cluster OperatorHub resource
	err = checkClusterOperatorHub(t, 1)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	// Enable the default OperatorSource
	err = toggle(t, 1, false, false)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkCreated(1, namespace)
	assert.NoError(t, err, "Default OperatorSource(s) or child resources were not recreated")

	// Check the cluster OperatorHub resource
	err = checkClusterOperatorHub(t, 0)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	resetClusterOperatorHub(t, namespace)
}

// testDisableNonDefault tests disabling a non-default OperatorSource and ensures no action was taken on it.
func testDisableNonDefault(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace.")

	err = helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	sources := []apiconfigv1.HubSource{
		{
			Name:     helpers.TestOperatorSourceName,
			Disabled: true,
		},
	}

	_ = updateOperatorHub(t, sources, false)

	// Wait for the operatorhub update to complete
	cluster := &apiconfigv1.OperatorHub{}
	err = wait.Poll(time.Second*5, time.Minute*1, func() (done bool, err error) {
		err = test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, cluster)
		if err != nil {
			return false, err
		}
		if len(cluster.Status.Sources) == len(helpers.DefaultSources)+1 {
			return true, nil
		}
		return false, nil
	})

	var testStatus apiconfigv1.HubSourceStatus
	for _, sourceStatus := range cluster.Status.Sources {
		if sourceStatus.Name == helpers.TestOperatorSourceName {
			testStatus = sourceStatus
			break
		}
	}
	assert.True(t, testStatus.Name == helpers.TestOperatorSourceName,
		"HubSourceStatus is missing for non-default OperatorSource")
	assert.True(t, testStatus.Status == "Error",
		"HubSourceStatus is not in error state for non-default OperatorSource")
	assert.True(t, testStatus.Message == "Not present in the default definitions",
		"HubSourceStatus message is incorrect for non-default OperatorSource")

	// Check the OperatorSource and child resources
	err = checkOpSrcAndChildrenArePresent(helpers.TestOperatorSourceName, namespace)
	assert.NoError(t, err, "Non-default OperatorSource resources are not present")
	resetClusterOperatorHub(t, namespace)
}

// testClusterStatusDefaultsDisabled tests that, when all default operator sources are disabled,
// the clusterstatus sets Available=True
func testClusterStatusDefaultsDisabled(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	client := test.Global.Client

	// Get namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// First set the OperatorHub to disable all the default operator sources
	err = toggle(t, 3, true, true)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkDeleted(3, namespace)
	assert.NoError(t, err, "All default OperatorSource(s) have not been disabled")

	err = checkClusterOperatorHub(t, 3)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	// Restart marketplace operator
	err = helpers.RestartMarketplace(test.Global.Client, namespace)
	require.NoError(t, err, "Could not restart marketplace operator")

	// Check that the ClusterOperator resource has the correct status
	clusterOperatorName := "marketplace"
	expectedTypeStatus := map[apiconfigv1.ClusterStatusConditionType]apiconfigv1.ConditionStatus{
		apiconfigv1.OperatorUpgradeable: apiconfigv1.ConditionTrue,
		apiconfigv1.OperatorProgressing: apiconfigv1.ConditionFalse,
		apiconfigv1.OperatorAvailable:   apiconfigv1.ConditionTrue,
		apiconfigv1.OperatorDegraded:    apiconfigv1.ConditionFalse}

	// Poll to ensure ClusterOperator is present and has the correct status
	// i.e. ConditionType has a ConditionStatus matching expectedTypeStatus
	namespacedName := types.NamespacedName{Name: clusterOperatorName, Namespace: namespace}
	result := &apiconfigv1.ClusterOperator{}
	RetryInterval := time.Second * 5
	Timeout := time.Minute * 5
	err = wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		for _, condition := range result.Status.Conditions {
			if expectedTypeStatus[condition.Type] != condition.Status {
				return false, nil
			}
		}
		return true, nil
	})
	assert.NoError(t, err, "ClusterOperator never reached expected status")

	resetClusterOperatorHub(t, namespace)
}

// testSomeClusterStatusDefaultsDisabled tests that, when some default operator sources are disabled,
// the clusterstatus sets Available=false
func testSomeClusterStatusDefaultsDisabled(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	client := test.Global.Client

	// Get namespace
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// First set the OperatorHub to disable first two default operator sources
	err = toggle(t, 2, true, false)
	require.NoError(t, err, "Error updating cluster OperatorHub")

	err = checkDeleted(2, namespace)
	assert.NoError(t, err, "First two default OperatorSource(s) have not been disabled")

	err = checkClusterOperatorHub(t, 2)
	assert.NoError(t, err, "Incorrect cluster OperatorHub")

	// Restart marketplace operator
	err = helpers.RestartMarketplace(test.Global.Client, namespace)
	require.NoError(t, err, "Could not restart marketplace operator")

	// Check that the ClusterOperator resource has the correct status
	clusterOperatorName := "marketplace"
	expectedTypeStatus := map[apiconfigv1.ClusterStatusConditionType]apiconfigv1.ConditionStatus{
		apiconfigv1.OperatorUpgradeable: apiconfigv1.ConditionTrue,
		apiconfigv1.OperatorProgressing: apiconfigv1.ConditionFalse,
		apiconfigv1.OperatorAvailable:   apiconfigv1.ConditionFalse,
		apiconfigv1.OperatorDegraded:    apiconfigv1.ConditionFalse}

	// Poll to ensure ClusterOperator is present and has the correct status
	// i.e. ConditionType has a ConditionStatus matching expectedTypeStatus
	namespacedName := types.NamespacedName{Name: clusterOperatorName, Namespace: namespace}
	result := &apiconfigv1.ClusterOperator{}
	RetryInterval := time.Second * 5
	Timeout := time.Minute * 5
	err = wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		for _, condition := range result.Status.Conditions {
			if expectedTypeStatus[condition.Type] != condition.Status {
				return false, nil
			}
		}
		return true, nil
	})
	assert.NoError(t, err, "ClusterOperator never reached expected status")

	resetClusterOperatorHub(t, namespace)
}

// getClusterOperatorHub gets the "cluster" OperatorHub resource
func getClusterOperatorHub(t *testing.T) *apiconfigv1.OperatorHub {
	cluster := &apiconfigv1.OperatorHub{}
	err := test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, cluster)
	require.NoError(t, err, "Unable to get cluster OperatorHub")
	return cluster
}

// resetClusterOperatorHub resets the "cluster" OperatorHub resource to its default value
func resetClusterOperatorHub(t *testing.T, namespace string) {
	cluster := getClusterOperatorHub(t)
	cluster.Spec = apiconfigv1.OperatorHubSpec{}
	err := helpers.UpdateRuntimeObject(test.Global.Client, cluster)
	require.NoError(t, err, "Error resetting cluster OperatorHub")

	err = checkCreated(3, namespace)
	require.NoError(t, err, "Error restoring default OperatorSources")

	err = checkClusterOperatorHub(t, 0)
	require.NoError(t, err, "Incorrect cluster OperatorHub")
}

// updateOperatorHub updates the "cluster" OperatorHub resource
func updateOperatorHub(t *testing.T, sources []apiconfigv1.HubSource, disableAll bool) error {
	cluster := getClusterOperatorHub(t)

	client := test.Global.Client

	// Disable / enable the default OperatorSource
	if sources != nil {
		cluster.Spec = apiconfigv1.OperatorHubSpec{Sources: sources}
	}

	if disableAll {
		cluster.Spec.DisableAllDefaultSources = true
	}

	return helpers.UpdateRuntimeObject(client, cluster)
}

// toggle sets the config for nr default OperatorSources based on disabled and disableAll. For example, if nr=2, then
// the first two defaults in helpers.DefaultSources are marked to be disabled.
func toggle(t *testing.T, nr int, disabled, disableAll bool) error {
	err := helpers.InitOpSrcDefinition()
	require.NoError(t, err, "Could not get a default OperatorSource definition from disk")

	// Construct the list of HubSources
	var sources []apiconfigv1.HubSource
	if nr > 0 {
		sources = make([]apiconfigv1.HubSource, nr)
		for n := 0; n < nr; n++ {
			sources[n] = apiconfigv1.HubSource{
				Name:     helpers.DefaultSources[n].Name,
				Disabled: disabled,
			}
		}
	}

	err = updateOperatorHub(t, sources, disableAll)
	if err != nil {
		return err
	}

	return nil
}

// checkDeleted checks if nr default OperatorSources and its child resources have been removed from the cluster. For
// example, nr=2 checks if the first 2 defaults helpers.DefaultSources have been deleted.
func checkDeleted(nr int, namespace string) error {
	for n := 0; n < nr; n++ {
		err := checkOpSrcAndChildrenAreDeleted(helpers.DefaultSources[n].Name, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkCreated checks if nr default OperatorSources and its child resources are present on the cluster. For example,
// nr=2 checks if the first 2 defaults helpers.DefaultSources are present.
func checkCreated(nr int, namespace string) error {
	for n := 0; n < nr; n++ {
		err := checkOpSrcAndChildrenArePresent(helpers.DefaultSources[n].Name, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkOpSrcAndChildrenArePresent checks if the OperatorSource and its child resources are present.
func checkOpSrcAndChildrenArePresent(name, namespace string) error {
	client := test.Global.Client
	err := helpers.CheckChildResourcesCreated(client, name, namespace, namespace, v1.OperatorSourceKind)
	if err != nil {
		return err
	}

	// Check if the OperatorSource is present
	err = client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &v1.OperatorSource{})
	if err != nil {
		return err
	}
	return nil
}

// checkOpSrcAndChildrenAreDeleted checks if the OperatorSource and its child resources have been deleted.
func checkOpSrcAndChildrenAreDeleted(name, namespace string) error {
	client := test.Global.Client
	err := helpers.CheckChildResourcesDeleted(client, name, namespace, namespace)
	if err != nil {
		return err
	}

	def := &v1.OperatorSource{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace},
		def)
	if !errors.IsNotFound(err) || def.Name != "" {
		return fmt.Errorf("default OperatorSource still present on the cluster")
	}
	return nil
}

// checkClusterOperatorHub checks if the cluster OperatorHub resource is in the expected state
func checkClusterOperatorHub(t *testing.T, nrExpectedDisabled int) error {
	cluster := getClusterOperatorHub(t)
	assert.True(t, len(cluster.Status.Sources) == len(helpers.DefaultSources),
		"Spurious elements in HubSourceStatus")

	var nrDisabled int
	for _, status := range cluster.Status.Sources {
		if status.Disabled {
			nrDisabled++
		}
	}

	if nrDisabled != nrExpectedDisabled {
		return fmt.Errorf("incorrect disabled default OperatorSources")
	}
	return nil
}
