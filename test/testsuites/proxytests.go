package testsuites

import (
	"context"
	"fmt"
	"testing"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ProxyTests is a test suite that ensures that marketplace is listening to the global proxy.
func ProxyTests(t *testing.T) {
	t.Run("opsrc-registry-includes-proxy-variables", testOpSrcRegistryIncludesProxyVars)
}

// testOpSrcRegistry ensures that the Operator Registry deployment
// created by an OperatorSource has the appropriate proxy environment
// variables set.
func testOpSrcRegistryIncludesProxyVars(t *testing.T) {
	assert.NoError(t, checkDeploymentIncludesProxyVars(t, helpers.TestOperatorSourceName))
}

// checkDeploymentIncludesProxyVars checks if the deployment has the appropriate
// proxy variables set.
func checkDeploymentIncludesProxyVars(t *testing.T, name string) error {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	// Get test namespace
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Get global framework variables.
	client := test.Global.Client
	// Get the cluster proxy
	clusterProxy := &apiconfigv1.Proxy{}
	clusterProxyKey := types.NamespacedName{Name: "cluster"}

	err = client.Get(context.TODO(), clusterProxyKey, clusterProxy)
	if err != nil && !errors.IsNotFound(err) {
		require.NoError(t, err, "Unexpected error while retrieving cluster proxy")
	}

	// Create the expected proxy EnvVar array.
	expected := []corev1.EnvVar{
		{Name: "HTTP_PROXY", Value: clusterProxy.Status.HTTPProxy},
		{Name: "HTTPS_PROXY", Value: clusterProxy.Status.HTTPSProxy},
		{Name: "NO_PROXY", Value: clusterProxy.Status.NoProxy},
	}

	// Get the Deployment with the given namespacedname.
	deployment := &apps.Deployment{}
	key := types.NamespacedName{Namespace: namespace, Name: name}

	err = client.Get(context.TODO(), key, deployment)
	require.NoError(t, err, fmt.Sprintf("Unexpected error while retrieving the %s/%s deployment", namespace, name))

	actual := deployment.Spec.Template.Spec.Containers[0].Env
	// Check that the the lists match.
	for i := range expected {
		if !contains(actual, expected[i]) {
			return fmt.Errorf("EnvVar list %v does not contain %v", actual, expected[i])
		}
	}
	return nil
}

func contains(envVars []corev1.EnvVar, envVar corev1.EnvVar) bool {
	for i := range envVars {
		if envVar == envVars[i] {
			return true
		}
	}
	return false
}
