package e2e

import (
	"fmt"
	"testing"
	"time"

	operator "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-sdk/pkg/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// pollInterval is used by the pollUntilTrue function to define the frequency that
	// the provided function is ran
	pollInterval time.Duration = 5 * time.Second

	// pollTimeout is used by the pollUntilTrue function to define the timeout
	pollTimeout time.Duration = 1 * time.Minute

	// invalidTargetNamespaceCSCName is the name of the catalogsourceconfig that points
	// to a non-existing targetNamespace
	invalidTargetNamespaceCSCName string = "non-existing-namespace-operators"

	// targetNamespace is the non-existing target namespace
	targetNamespace string = "non-existing-namespace"
)

func runCSCWithInvalidTargetNamespace(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables
	f := test.Global
	// Run tests
	if err := cscWithInvalidTargetNamespace(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func cscWithInvalidTargetNamespace(t *testing.T, f *test.Framework, ctx *test.TestCtx) error {
	// Get test namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Could not get namespace: %v", err)
	}

	// Create the operatorsource to download the manifests
	testOperatorSource := &operator.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operators2",
			Namespace: namespace,
		},
		Spec: operator.OperatorSourceSpec{
			Type:              "appregistry",
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "marketplace_e2e",
		},
	}
	err = createRuntimeObject(f, ctx, testOperatorSource)
	if err != nil {
		return err
	}

	// Create a new catalogsourceconfig with a nonexistant targetNamespace
	invalidTargetNamespaceCSC := &operator.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: operator.OperatorSourceKind,
		}, ObjectMeta: metav1.ObjectMeta{
			Name:      invalidTargetNamespaceCSCName,
			Namespace: namespace,
		},
		Spec: operator.CatalogSourceConfigSpec{
			TargetNamespace: targetNamespace,
			Packages:        "descheduler",
		}}
	err = createRuntimeObject(f, ctx, invalidTargetNamespaceCSC)
	if err != nil {
		return err
	}

	// Check that we created the catalogsourceconfig with an invalid targetNamespace
	resultCatalogSourceConfig := &operator.CatalogSourceConfig{}
	err = WaitForResult(t, f, resultCatalogSourceConfig, namespace, invalidTargetNamespaceCSCName)
	if err != nil {
		return err
	}

	// Check if the catalogsourceconfig phase and message are the expected values
	expectedPhase := "Configuring"
	expectedMessage := fmt.Sprintf("namespaces \"%s\" not found", targetNamespace)
	result := pollUntilTrue(pollInterval, pollTimeout, func() bool {
		// catalogsourceconfig should always exist so no wait
		err = WaitForResult(t, f, resultCatalogSourceConfig, namespace, invalidTargetNamespaceCSCName)
		if resultCatalogSourceConfig.Status.CurrentPhase.Name == expectedPhase &&
			resultCatalogSourceConfig.Status.CurrentPhase.Message == expectedMessage {
			return true
		}
		return false
	})

	if !result {
		return fmt.Errorf("CatalogSourceConfig never reached expected phase/message, expected %v/%v", expectedPhase, expectedMessage)
	}

	// Create a namespace based on the targetNamespace string
	targetNamespaceRuntimeObject := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: targetNamespace}}
	err = createRuntimeObject(f, ctx, targetNamespaceRuntimeObject)
	if err != nil {
		return err
	}

	// Check if the catalogsourceconfig phase has been set to "Succeeded"
	expectedPhase = "Succeeded"
	result = pollUntilTrue(pollInterval, pollTimeout, func() bool {
		// catalogsourceconfig should always exist so no wait
		err = WaitForResult(t, f, resultCatalogSourceConfig, namespace, invalidTargetNamespaceCSCName)
		if resultCatalogSourceConfig.Status.CurrentPhase.Name == expectedPhase {
			return true
		}
		return false
	})

	if !result {
		return fmt.Errorf("CatalogSourceConfig never reached expected phase/message, expected %v", expectedPhase)
	}

	return nil
}
