package testsuites

import (
	"fmt"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CscInvalid tests CatalogSourceConfigs created with invalid values
// and checks if the expected failure state is reached
func CscInvalid(t *testing.T) {
	t.Run("object-in-other-namespace", testCscInOtherNamespace)
}

// testCscInOtherNamespace creates a CatalogSourceConfig in the default
// namespace and forces it through all the phases
// Expected result: CatalogSourceConfig always reaches failed state
func testCscInOtherNamespace(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	client := test.Global.Client

	// Create the CatalogSourceConfig in the default namespace
	namespace := "default"
	cscName := "other-namespace-csc"
	otherNamespaceCatalogSourceConfig := &v2.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: v2.CatalogSourceConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cscName,
			Namespace: namespace,
		},
		Spec: v2.CatalogSourceConfigSpec{
			TargetNamespace: namespace,
			Packages:        "camel-k-marketplace-e2e-tests",
		},
	}
	err := helpers.CreateRuntimeObject(client, ctx, otherNamespaceCatalogSourceConfig)
	require.NoError(t, err, "Could not create CatalogSourceConfig")

	expectedPhase := "Failed"
	csc, err := helpers.WaitForCscExpectedPhaseAndMessage(client, cscName, namespace, expectedPhase,
		"Will only reconcile resources in the operator's namespace")
	assert.NoError(t, err, fmt.Sprintf("CatalogSourceConfig never reached expected phase/message, expected %s", expectedPhase))
	require.NotNil(t, csc, "Could not retrieve CatalogSourceConfig")

	// Force the CatalogSourceConfig status into various phases other than "Failed" and "Initial"
	for _, phase := range []string{"Configuring", "Succeeded"} {
		csc.Status = v2.CatalogSourceConfigStatus{
			CurrentPhase: shared.ObjectPhase{
				Phase: shared.Phase{
					Name: phase,
				},
			},
		}
		err = helpers.UpdateRuntimeObject(client, csc)
		require.NoError(t, err, "Could not update CatalogSourceConfig")
		csc, err = helpers.WaitForCscExpectedPhaseAndMessage(client, cscName, namespace, expectedPhase,
			"Will only reconcile resources in the operator's namespace")
		assert.NoError(t, err, fmt.Sprintf("CatalogSourceConfig never reached expected phase/message for inserted phase %s, expected %s", phase, expectedPhase))
	}

}
