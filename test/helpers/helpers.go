package helpers

import (
	"context"
	"fmt"
	"strings"
	"time"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/builders"
	"github.com/operator-framework/operator-sdk/pkg/test"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// RetryInterval defines the frequency at which we check for updates against the
	// k8s api when waiting for a specific condition to be true.
	RetryInterval = time.Second * 5

	// Timeout defines the amount of time we should spend querying the k8s api
	// when waiting for a specific condition to be true.
	Timeout = time.Minute * 5

	// TestOperatorSourceName is the name of an OperatorSource that is returned by
	// the CreateOperatorSource function.
	TestOperatorSourceName string = "test-operators"

	// TestOperatorSourceLabelKey is a label key added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelKey string = "opsrc-group"

	// TestOperatorSourceLabelValue is a label value added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelValue string = "Community"

	// TestCatalogSourceConfigName is the name of the test CatalogSourceConfig.
	TestCatalogSourceConfigName = "test-csc"

	// TestCatalogSourceConfigTargetNamespace is the target namespace used in the test
	// CatalogSourceConfig.
	TestCatalogSourceConfigTargetNamespace = "default"
)

// WaitForResult polls the cluster for a particular resource name and namespace.
// If the request fails because of an IsNotFound error it retries until the specified timeout.
// If it succeeds it sets the result runtime.Object to the requested object.
func WaitForResult(client test.FrameworkClient, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	return wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// WaitForSuccessfulDeployment checks if a given deployment has readied all of
// its replicas. If it has not, it retries until the deployment is ready or it
// reaches the timeout.
func WaitForSuccessfulDeployment(client test.FrameworkClient, deployment apps.Deployment) error {
	// If deployment is already ready, lets just return.
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return nil
	}

	namespacedName := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
	result := &apps.Deployment{}
	return wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		if *deployment.Spec.Replicas == result.Status.ReadyReplicas {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCscExpectedPhaseAndMessage checks if a CatalogSourceConfig with the given name exists in the namespace
// and makes sure that the phase and message matches the expected values.
// If expectedMessage is an empty string, only the expectedPhase is checked.
func WaitForCscExpectedPhaseAndMessage(client test.FrameworkClient, cscName, namespace, expectedPhase, expectedMessage string) (*v2.CatalogSourceConfig, error) {
	// Check that the CatalogSourceConfig exists.
	resultCatalogSourceConfig := &v2.CatalogSourceConfig{}
	err := wait.PollImmediate(RetryInterval, Timeout, func() (bool, error) {
		err := WaitForResult(client, resultCatalogSourceConfig, namespace, cscName)
		if err != nil {
			return false, err
		}
		// log.Infof("Csc: %+v", resultCatalogSourceConfig)
		// Check for the expected phase
		if resultCatalogSourceConfig.Status.CurrentPhase.Name == expectedPhase {
			// If the expected message is not empty make sure that it contains the actual message
			if expectedMessage == "" || strings.Contains(resultCatalogSourceConfig.Status.CurrentPhase.Message, expectedMessage) {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return resultCatalogSourceConfig, nil
}

// WaitForOpSrcExpectedPhaseAndMessage checks if an OperatorSource with the given name exists in the namespace
// and makes sure that the phase and message matches the expected values. It also returns the OperatorSource object.
func WaitForOpSrcExpectedPhaseAndMessage(client test.FrameworkClient, opSrcName string, namespace string, expectedPhase string, expectedMessage string) (*v1.OperatorSource, error) {
	resultOperatorSource := &v1.OperatorSource{}
	err := wait.Poll(RetryInterval, Timeout, func() (bool, error) {
		err := WaitForResult(client, resultOperatorSource, namespace, opSrcName)
		if err != nil {
			return false, err
		}
		if resultOperatorSource.Status.CurrentPhase.Name == expectedPhase &&
			strings.Contains(resultOperatorSource.Status.CurrentPhase.Message, expectedMessage) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return resultOperatorSource, nil
}

// WaitForNotFound polls the cluster for a particular resource name and namespace
// If the request fails because the runtime object is found it retries until the specified timeout
// If the request returns a IsNotFound error the method will return true
func WaitForNotFound(client test.FrameworkClient, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// CreateRuntimeObject creates a runtime object using the test framework.
func CreateRuntimeObject(client test.FrameworkClient, ctx *test.TestCtx, obj runtime.Object) error {
	return client.Create(
		context.TODO(),
		obj,
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       time.Second * 30,
			RetryInterval: time.Second * 1,
		})
}

// CreateRuntimeObjectNoCleanup creates a runtime object without any cleanup
// options. Using this method to create a runtime object means that the framework
// will not automatically delete the object after test execution, and it must
// be manually deleted.
func CreateRuntimeObjectNoCleanup(client test.FrameworkClient, obj runtime.Object) error {
	return client.Create(
		context.TODO(),
		obj,
		nil,
	)
}

// DeleteRuntimeObject deletes a runtime object using the test framework
func DeleteRuntimeObject(client test.FrameworkClient, obj runtime.Object) error {
	return client.Delete(
		context.TODO(),
		obj)
}

// UpdateRuntimeObject updates a runtime object using the test framework
func UpdateRuntimeObject(client test.FrameworkClient, obj runtime.Object) error {
	return client.Update(
		context.TODO(),
		obj,
	)
}

// CreateOperatorSourceDefinition returns an OperatorSource definition that can be turned into
// a runtime object for tests that rely on an OperatorSource
func CreateOperatorSourceDefinition(name, namespace string) *v1.OperatorSource {
	return &v1.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			Kind: v1.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				TestOperatorSourceLabelKey: TestOperatorSourceLabelValue,
			},
		},
		Spec: v1.OperatorSourceSpec{
			Type:              "appregistry",
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "marketplace_e2e",
		},
	}
}

// CreateCatalogSourceConfigDefinition returns an CatalogSourceConfig definition that can
// be turned into a runtime object for tests that rely on an CatalogSourceConfig
func CreateCatalogSourceConfigDefinition(name, namespace, target string) *v2.CatalogSourceConfig {
	return &v2.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: v2.CatalogSourceConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v2.CatalogSourceConfigSpec{
			TargetNamespace: target,
			Packages:        "camel-k-marketplace-e2e-tests",
		},
	}

}

// checkOwnerLabels verifies that the correct owner labels have been set
func checkOwnerLabels(labels map[string]string, owner string) error {
	switch owner {
	case v1.OperatorSourceKind:
		_, hasNameLabel := labels[builders.OpsrcOwnerNameLabel]
		_, hasNamespaceLabel := labels[builders.OpsrcOwnerNamespaceLabel]
		if !hasNameLabel || !hasNamespaceLabel {
			return fmt.Errorf("Created child resource does not have correct %v owner labels", owner)
		}
	case v2.CatalogSourceConfigKind:
		_, hasNameLabel := labels[builders.CscOwnerNameLabel]
		_, hasNamespaceLabel := labels[builders.CscOwnerNamespaceLabel]
		if !hasNameLabel || !hasNamespaceLabel {
			return fmt.Errorf("Created child resource does not have correct %v owner labels", owner)
		}
	}
	return nil
}

// CheckChildResourcesCreated checks that a CatalogSourceConfig's
// child resources were deployed.
func CheckChildResourcesCreated(client test.FrameworkClient, cscName, namespace, targetNamespace, owner string) error {

	// Check that the CatalogSource was created.
	resultCatalogSource := &olm.CatalogSource{}
	err := WaitForResult(client, resultCatalogSource, targetNamespace, cscName)
	if err != nil {
		return err
	}

	// Check owner labels are correctly set.
	err = checkOwnerLabels(resultCatalogSource.Labels, owner)
	if err != nil {
		return err
	}

	// Check that the Service was created.
	resultService := &corev1.Service{}
	err = WaitForResult(client, resultService, namespace, cscName)
	if err != nil {
		return err
	}

	// Check owner labels are correctly set.
	err = checkOwnerLabels(resultService.Labels, owner)
	if err != nil {
		return err
	}

	// Check that the Deployment was created.
	resultDeployment := &apps.Deployment{}
	err = WaitForResult(client, resultDeployment, namespace, cscName)
	if err != nil {
		return err
	}

	// Check owner labels are correctly set.
	err = checkOwnerLabels(resultDeployment.Labels, owner)
	if err != nil {
		return err
	}

	// Now check that the Deployment is ready.
	err = WaitForSuccessfulDeployment(client, *resultDeployment)
	if err != nil {
		return err
	}
	return nil
}

// CheckChildResourcesDeleted checks that a CatalogSourceConfig's
// child resources were deleted.
func CheckChildResourcesDeleted(client test.FrameworkClient, cscName, namespace, targetNamespace string) error {
	// Check that the CatalogSource was deleted.
	resultCatalogSource := &olm.CatalogSource{}
	err := WaitForNotFound(client, resultCatalogSource, targetNamespace, cscName)
	if err != nil {
		return err
	}

	// Check that the Service was deleted.
	resultService := &corev1.Service{}
	err = WaitForNotFound(client, resultService, namespace, cscName)
	if err != nil {
		return err
	}

	// Check that the Deployment was deleted.
	resultDeployment := &apps.Deployment{}
	err = WaitForNotFound(client, resultDeployment, namespace, cscName)
	if err != nil {
		return err
	}
	return nil
}

// CheckCscSuccessfulCreation checks that a CatalogSourceConfig
// and it's child resources were deployed.
func CheckCscSuccessfulCreation(client test.FrameworkClient, cscName, namespace, targetNamespace string) error {
	// Check that the CatalogSourceConfig was created.
	resultCatalogSourceConfig := &v2.CatalogSourceConfig{}
	err := WaitForResult(client, resultCatalogSourceConfig, namespace, cscName)
	if err != nil {
		return err
	}

	// Check that all child resources were created.
	err = CheckChildResourcesCreated(client, cscName, namespace, targetNamespace, v2.CatalogSourceConfigKind)
	if err != nil {
		return err
	}

	return nil
}

// CheckCscSuccessfulDeletion checks that a CatalogSourceConfig
// and it's child resources were deleted.
func CheckCscSuccessfulDeletion(client test.FrameworkClient, cscName, namespace, targetNamespace string) error {
	// Check that the CatalogSourceConfig was deleted.
	resultCatalogSourceConfig := &v2.CatalogSourceConfig{}
	err := WaitForNotFound(client, resultCatalogSourceConfig, namespace, cscName)
	if err != nil {
		return err
	}

	// Check that all child resources were deleted.
	err = CheckChildResourcesDeleted(client, cscName, namespace, targetNamespace)
	if err != nil {
		return err
	}

	return nil
}
