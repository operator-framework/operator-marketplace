package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/apis"
	operator "github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/test"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5

	GroupLabel string = "opsrc-group"
)

// Test marketplace is the root function that triggers the set of e2e tests
func TestMarketplace(t *testing.T) {
	// Add marketplace types to test framework scheme
	operatorsource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
			APIVersion: fmt.Sprintf("%s/%s",
				operator.SchemeGroupVersion.Group, operator.SchemeGroupVersion.Version),
		},
	}
	catalogsourceconfig := &operator.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.CatalogSourceConfigKind,
			APIVersion: fmt.Sprintf("%s/%s",
				operator.SchemeGroupVersion.Group, operator.SchemeGroupVersion.Version),
		},
	}
	err := test.AddToFrameworkScheme(apis.AddToScheme, operatorsource)
	if err != nil {
		t.Fatalf("failed to add operatorsource custom resource scheme to framework: %v", err)
	}
	err = test.AddToFrameworkScheme(apis.AddToScheme, catalogsourceconfig)
	if err != nil {
		t.Fatalf("failed to add catalogsourceconfig custom resource scheme to framework: %v", err)
	}
	// Add (olm) catalog sources to framework scheme
	catalogsource := &olm.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       olm.CatalogSourceKind,
			APIVersion: olm.CatalogSourceCRDAPIVersion,
		},
	}
	err = test.AddToFrameworkScheme(olm.AddToScheme, catalogsource)
	if err != nil {
		t.Fatalf("failed to add catalogsource custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("marketplace-group", func(t *testing.T) {
		t.Run("Cluster", MarketplaceCluster)
	})
}

// This method initializes the environment and triggers the test
func MarketplaceCluster(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// get global framework variables
	f := test.Global

	if err := defaultCreateTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

// This function runs a basic happy case end to end workflow for marketplace
// First create an operatorsource which points to external app registry on quay
// Check that the catalogsourceconfig was created
// Then check the service and deployment were created and are ready
func defaultCreateTest(t *testing.T, f *test.Framework, ctx *test.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	groupWant := "Community"
	testOperatorSource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operators",
			Namespace: namespace,
			Labels: map[string]string{
				GroupLabel: groupWant,
			},
		},
		Spec: operator.OperatorSourceSpec{
			Type:              "appregistry",
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "marketplace_e2e",
		},
	}

	catalogSourceConfigName := "test-operators"
	catalogSourceName := "test-operators"
	serviceName := "test-operators"
	deploymentName := "test-operators"

	// Create the operatorsource to download the manifests.
	err = f.Client.Create(
		context.TODO(),
		testOperatorSource,
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       cleanupTimeout,
			RetryInterval: cleanupRetryInterval,
		})
	if err != nil {
		return err
	}

	// Check that we created the catalogsourceconfig.
	resultCatalogSourceConfig := &operator.CatalogSourceConfig{}
	err = WaitForResult(t, f, resultCatalogSourceConfig, namespace, catalogSourceConfigName)
	if err != nil {
		return err
	}

	// Then check for the catalog source.
	resultCatalogSource := &olm.CatalogSource{}
	err = WaitForResult(t, f, resultCatalogSource, namespace, catalogSourceName)
	if err != nil {
		return err
	}

	// Then check that the service was created.
	resultService := &corev1.Service{}
	err = WaitForResult(t, f, resultService, namespace, serviceName)
	if err != nil {
		return err
	}

	// Then check that the deployment was created.
	resultDeployment := &apps.Deployment{}
	err = WaitForResult(t, f, resultDeployment, namespace, deploymentName)
	if err != nil {
		return err
	}

	// Now check that the deployment is ready.
	err = WaitForSuccessfulDeployment(t, f, *resultDeployment)
	if err != nil {
		return err
	}

	labels := resultCatalogSource.GetLabels()
	groupGot, ok := labels[GroupLabel]

	if !ok || groupGot != groupWant {
		t.Errorf(
			"The created catalogsource %s does not have the right label[%s] - want=%s got=%s",
			resultCatalogSource.Name,
			GroupLabel,
			groupWant,
			groupGot,
		)
	}

	return nil
}
