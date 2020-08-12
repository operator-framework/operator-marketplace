package e2e

import (
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testgroups"
	"github.com/operator-framework/operator-sdk/pkg/test"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	test.MainEntry(m)
}

// TestMarketplace is the root function that triggers the set of e2e tests.
func TestMarketplace(t *testing.T) {
	initTestingFramework(t)

	// Run Test Groups
	if isConfigAPIPresent, _ := helpers.EnsureConfigAPIIsAvailable(); isConfigAPIPresent == true {
		t.Run("cluster-operator-status-test-group", testgroups.ClusterOperatorTestGroup)
	}
	t.Run("no-setup-test-group", testgroups.NoSetupTestGroup)
}

// initTestingFramework adds the marketplace OperatorSource and CatalogSourceConfig types as well as the
// operatorsv1alpha1 CatalogSource type to the framework scheme.
func initTestingFramework(t *testing.T) {
	// Add (operatorsv1alpha1) CatalogSources to framework scheme.
	catalogSource := &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       operatorsv1alpha1.CatalogSourceKind,
			APIVersion: operatorsv1alpha1.CatalogSourceCRDAPIVersion,
		},
	}
	err := test.AddToFrameworkScheme(operatorsv1alpha1.AddToScheme, catalogSource)
	if err != nil {
		t.Fatalf("failed to add CatalogSource custom resource scheme to framework: %v", err)
	}

	_, err = helpers.EnsureConfigAPIIsAvailable()
	if err != nil {
		t.Logf("failed to add OpenShift config custom resource scheme to framework: %v. config tests will not run.",
			err)
	}

}
