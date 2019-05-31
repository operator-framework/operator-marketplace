package testsuites

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PackageTests is a test suite that ensures that package behave as intended
func PackageTests(t *testing.T) {
	t.Run("csc-with-non-existing-package", testCscWithNonExistingPackage)
	t.Run("opsrc-with-identical-packages", testOpSrcWithIdenticalPackages)

	t.Run("resolve-missing-source", resolveMissingSource)
	t.Run("unresolved-missing-source", unresolvedMissingSource)
	t.Run("non-existing-source", nonExistingSource)
	t.Run("source-missing-packages", sourceMissingPackages)
}

// testCscWithNonExistingPackage tests that a csc with a non-existing package
// is handled correctly by the marketplace operator and its child resources are not
// created.
func testCscWithNonExistingPackage(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	client := test.Global.Client

	// Get test namespace
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Create a new catalogsourceconfig with a non-existing Package
	nonExistingPackageCSC := &v2.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: v2.CatalogSourceConfigKind,
		}, ObjectMeta: metav1.ObjectMeta{
			Name:      cscName,
			Namespace: namespace,
		},
		Spec: v2.CatalogSourceConfigSpec{
			TargetNamespace: namespace,
			Packages:        nonExistingPackageName,
		}}

	err = helpers.CreateRuntimeObject(client, ctx, nonExistingPackageCSC)
	require.NoError(t, err, "Unable to create test CatalogSourceConfig")

	// Check status is updated correctly then check child resources are not created
	t.Run("configuring-state-when-package-name-does-not-exist", configuringStateWhenPackageNameDoesNotExist)
	t.Run("child-resources-not-created", childResourcesNotCreated)
}

// configuringStateWhenTargetNamespaceDoesNotExist is a test case that creates a CatalogSourceConfig
// with a targetNamespace that doesn't exist and ensures that the status is updated to reflect the
// nonexistent namespace.
func configuringStateWhenPackageNameDoesNotExist(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check that the catalogsourceconfig with an non-existing package eventually reaches the
	// configuring phase with the expected message
	expectedPhase := "Configuring"
	expectedMessage := fmt.Sprintf("Unable to resolve the source - no source contains the requested package(s) [%s]", nonExistingPackageName)
	_, err = helpers.WaitForCscExpectedPhaseAndMessage(test.Global.Client, cscName, namespace, expectedPhase, expectedMessage)
	assert.NoError(t, err, fmt.Sprintf("CatalogSourceConfig never reached expected phase/message, expected %v/%v", expectedPhase, expectedMessage))
}

// childResourcesNotCreated checks that once a CatalogSourceConfig with a wrong package name
// is created that all expected runtime objects are not created.
func childResourcesNotCreated(t *testing.T) {
	// Get test namespace.
	namespace, err := test.NewTestCtx(t).GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check that the CatalogSourceConfig's child resources were not created.
	err = helpers.CheckChildResourcesDeleted(test.Global.Client, cscName, namespace, namespace)
	assert.NoError(t, err, "Child resources of CatalogSourceConfig were unexpectedly created")
}

// testOpSrcWithIdenticalPackages ensures that an OperatorSource and its child resources
// are successfully rolled out even if another OperatorSource contains identical packages.
func testOpSrcWithIdenticalPackages(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// Get global framework variables.
	client := test.Global.Client

	// Get test namespace.
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// The OperatorSource created below will point to the same Application Registry
	// as the OperatorSource created in operatorsourcetests.go and will contain
	// conflicting package names as a result.
	opSrcName := "conflicting-operators"
	err = helpers.CreateRuntimeObject(test.Global.Client, ctx, helpers.CreateOperatorSourceDefinition(opSrcName, namespace))
	assert.NoError(t, err, "Could not create operator source")

	// Check that the child resources were created.
	err = helpers.CheckChildResourcesCreated(client, opSrcName, namespace, namespace, v1.OperatorSourceKind)
	assert.NoError(t, err)

	t.Run("resolved-multiple-sources", resolvedMultipleSources)
}

// resolvedMultipleSources checks that a CatalogSourceConfig with an unspecified source
// is resolved if its packages exist within two OperatorSources.
func resolvedMultipleSources(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	assert.NoError(t, err)

	// Check that a CSC with an unspecified source is resolved if its
	// packages exist in two OperatorSources.
	err = runSourceTest(namespace,
		"",
		"camel-k-marketplace-e2e-tests",
		"Succeeded",
		"The object has been successfully reconciled",
	)
	assert.NoError(t, err)
}

// resolveMissingSource ensures that if no source is given, the CatalogSourceConfig is
// placed in the Succeeded phase if a source exists that contains the provided packages.
func resolveMissingSource(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	assert.NoError(t, err)

	err = runSourceTest(namespace,
		"",
		"camel-k-marketplace-e2e-tests",
		"Succeeded",
		"The object has been successfully reconciled",
	)
	assert.NoError(t, err)
}

// unresolvedMissingSource ensures that if no source is given, the CatalogSourceConfig is
// placed in the Configuring phase if marketplace cannot identify a source that contains the
// provided packages.
func unresolvedMissingSource(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	assert.NoError(t, err)

	packages := "camel-k-marketplace-e2e-tests-k,missing-package"
	err = runSourceTest(namespace,
		"",
		packages,
		"Configuring",
		fmt.Sprintf("Unable to resolve the source - no source contains the requested package(s) [%s]", strings.Replace(packages, ",", " ", -1)),
	)
	assert.NoError(t, err)
}

// nonExistingSource ensures that if the provided source does not exist the CatalogSourceConfig
// will be placed in the Configuring phase with the expected messages.
func nonExistingSource(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	assert.NoError(t, err)

	source := "bad-source"
	err = runSourceTest(namespace,
		source,
		"camel-k-marketplace-e2e-tests",
		"Configuring",
		fmt.Sprintf("Provided source (%s) does not exist", source),
	)
	assert.NoError(t, err)
}

// sourceMissingPackages ensures that if the provided source does not contain the expected
// packages the CatalogSourceConfig will be placed in the Configuring phase with the expected
// messages.
func sourceMissingPackages(t *testing.T) {
	namespace, err := test.NewTestCtx(t).GetNamespace()
	assert.NoError(t, err)

	err = runSourceTest(namespace,
		"test-operators",
		"camel-k-marketplace-e2e-tests,missing-package",
		"Configuring",
		"Still resolving package(s) - missing-package. Please make sure these are valid packages within the test-operators OperatorSource.",
	)
	assert.NoError(t, err)
}

func runSourceTest(namespace, source, packages, expectedPhase, expectedMessage string) error {
	// Get global framework variables.
	client := test.Global.Client

	// Create a new CatalogSourceConfig.
	csc := &v2.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: v2.CatalogSourceConfigKind,
		}, ObjectMeta: metav1.ObjectMeta{
			Name:      cscName,
			Namespace: namespace,
		},
		Spec: v2.CatalogSourceConfigSpec{
			Source:          source,
			TargetNamespace: namespace,
			Packages:        packages,
		}}

	// Create the CatalogSourceConfig and if an error occurs do not run tests that
	// rely on the existence of the CatalogSourceConfig.
	// The CatalogSourceConfig is created with nil ctx and must be deleted manually before test suite exits.
	err := helpers.CreateRuntimeObject(client, nil, csc)
	if err != nil {
		return err
	}

	// Check that the CatalogSourceConfig with an non-existing targetNamespace eventually reaches the
	// configuring phase with the expected message.
	_, err = helpers.WaitForCscExpectedPhaseAndMessage(test.Global.Client, cscName, namespace, expectedPhase, expectedMessage)
	if err != nil {
		return err
	}

	// Delete the CatalogSourceConfig.
	err = helpers.DeleteRuntimeObject(client, csc)
	if err != nil {
		return err
	}

	// Wait for the CatalogSourceConfig to be deleted.
	return helpers.WaitForNotFound(client, csc, namespace, cscName)
}
