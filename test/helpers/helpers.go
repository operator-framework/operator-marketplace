package helpers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	v2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/builders"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-sdk/pkg/test"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/util/retry"
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

	// TestOpsrcNameForClusterOperator is the name of an OperatorSource that is created
	// for testing ClusterOperator status on custom OperatorSource creation
	TestOpsrcNameForClusterOperator string = "test-operators-for-co"

	// TestOperatorSourceLabelKey is a label key added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelKey string = "opsrc-group"

	// TestOperatorSourceLabelValue is a label value added to the opeator source returned
	// by the CreateOperatorSource function.
	TestOperatorSourceLabelValue string = "Community"

	// DefaultsDir is the relative path to the defaults directory
	DefaultsDir string = "./defaults"

	// TestUISubscriptionName is the name of a Subscription that is returned by
	// the CreateUISubscriptionDefinition function.
	TestUISubscriptionName string = "test-operators-ui-created"

	// TestUserCreatedSubscriptionName is the name of a Subscription that is returned by
	// the CreateUserSubscriptionDefinition function.
	TestUserCreatedSubscriptionName string = "test-operators"

	// TestInvalidSubscriptionName is a subscription that points to a non-existent catalog source
	TestInvalidSubscriptionName string = "invalid-subscription"
)

var (
	// TestUserCreatedSubscriptionResourceVersion is the resourceversion of the user
	// created subscription. This is set when this subscription is created.
	TestUserCreatedSubscriptionResourceVersion string

	// isConfigAPIPresent keeps track of whether or not the OpenShift config API is available.
	isConfigAPIPresent *bool

	// DefaultSources is the in-memory copy of the default OperatorSource definitions
	// from the defaults directory.
	DefaultSources []*olmv1alpha1.CatalogSource
)

// WaitForResult polls the cluster for a particular resource name and namespace.
// If the request fails because of an IsNotFound error it retries until the specified timeout.
// If it succeeds it sets the result runtime.Object to the requested object.
func WaitForResult(client test.FrameworkClient, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	return wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if apierrors.IsNotFound(err) {
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
func WaitForSuccessfulDeployment(client test.FrameworkClient, deployment appsv1.Deployment) error {
	// If deployment is already ready, lets just return.
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return nil
	}

	namespacedName := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
	result := &appsv1.Deployment{}
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

// WaitForExpectedSpec compares two CatalogSources and return true if their Specs are equal.
func WaitForExpectedSpec(client test.FrameworkClient, name string, namespace string, expected *olmv1alpha1.CatalogSource) error {
	err := wait.Poll(RetryInterval, Timeout, func() (bool, error) {
		actual := &olmv1alpha1.CatalogSource{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, actual); err != nil {
			return false, err
		}
		if defaults.AreCatsrcSpecsEqual(&expected.Spec, &actual.Spec) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// WaitForNotFound polls the cluster for a particular resource name and namespace
// If the request fails because the runtime object is found it retries until the specified timeout
// If the request returns a IsNotFound error the method will return true
func WaitForNotFound(client test.FrameworkClient, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if apierrors.IsNotFound(err) {
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
func CreateRuntimeObject(client test.FrameworkClient, ctx *test.Context, obj runtime.Object) error {
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

func UpdateCatalogSourceWithRetries(client test.FrameworkClient, obj *olmv1alpha1.CatalogSource) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := client.Update(context.TODO(), obj)
		if apierrors.IsConflict(err) {
			clone := obj.DeepCopyObject().(*olmv1alpha1.CatalogSource)
			key := types.NamespacedName{
				Name:      obj.Name,
				Namespace: obj.Namespace,
			}
			err = client.Get(context.TODO(), key, clone)
			if err != nil {
				return err
			}

			obj.SetResourceVersion(clone.GetResourceVersion())
		}
		return err
	})
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

// checkOwnerLabels verifies that the correct owner labels have been set
func checkOwnerLabels(labels map[string]string, owner string) error {
	switch owner {
	case v1.OperatorSourceKind:
		_, hasNameLabel := labels[builders.OpsrcOwnerNameLabel]
		_, hasNamespaceLabel := labels[builders.OpsrcOwnerNamespaceLabel]
		if !hasNameLabel || !hasNamespaceLabel {
			return fmt.Errorf("created child resource does not have correct %v owner labels", owner)
		}
	case v2.CatalogSourceConfigKind:
		_, hasNameLabel := labels[builders.CscOwnerNameLabel]
		_, hasNamespaceLabel := labels[builders.CscOwnerNamespaceLabel]
		if !hasNameLabel || !hasNamespaceLabel {
			return fmt.Errorf("created child resource does not have correct %v owner labels", owner)
		}
	}
	return nil
}

// CheckChildResourcesCreated checks that an OperatorSource's
// child resources were deployed.
func CheckChildResourcesCreated(client test.FrameworkClient, opsrcName, namespace, targetNamespace, owner string) error {

	// Check that the CatalogSource was created.
	resultCatalogSource := &olmv1alpha1.CatalogSource{}
	err := WaitForResult(client, resultCatalogSource, targetNamespace, opsrcName)
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
	err = WaitForResult(client, resultService, namespace, opsrcName)
	if err != nil {
		return err
	}

	// Check owner labels are correctly set.
	err = checkOwnerLabels(resultService.Labels, owner)
	if err != nil {
		return err
	}

	// Check that the Deployment was created.
	resultDeployment := &appsv1.Deployment{}
	err = WaitForResult(client, resultDeployment, namespace, opsrcName)
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

// CheckChildResourcesDeleted checks that an OperatorSource's
// child resources were deleted.
func CheckChildResourcesDeleted(client test.FrameworkClient, opsrcName, namespace, targetNamespace string) error {
	// Check that the CatalogSource was deleted.
	resultCatalogSource := &olmv1alpha1.CatalogSource{}
	err := WaitForNotFound(client, resultCatalogSource, targetNamespace, opsrcName)
	if err != nil {
		return err
	}

	// Check that the Service was deleted.
	resultService := &corev1.Service{}
	err = WaitForNotFound(client, resultService, namespace, opsrcName)
	if err != nil {
		return err
	}

	// Check that the Deployment was deleted.
	resultDeployment := &appsv1.Deployment{}
	err = WaitForNotFound(client, resultDeployment, namespace, opsrcName)
	if err != nil {
		return err
	}
	return nil
}

// WaitForOpsrcMarkedForDeletionWithFinalizer waits until an object with a finalizer is marked for deletion
// but the finalizer has not yet been removed. This method should only be used in the case where
// the finalizer will not be removed automatically, otherwise it will return an error in the case of
// a race condition.
func WaitForOpsrcMarkedForDeletionWithFinalizer(client test.FrameworkClient, name, namespace string) error {
	resultOperatorSource := &v1.OperatorSource{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	return wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, resultOperatorSource)
		if err != nil {
			return false, err
		}
		if resultOperatorSource.DeletionTimestamp != nil && len(resultOperatorSource.Finalizers) > 0 {
			return true, nil
		}
		return false, nil
	})
}

// WaitForCatsrcMarkedForDeletion waits until a CatalogSource is either deleted or is marked for deletion
func WaitForCatsrcMarkedForDeletion(client test.FrameworkClient, name, namespace string) error {
	resultCatalogSource := &olmv1alpha1.CatalogSource{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}

	return wait.Poll(RetryInterval, 1*time.Minute, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, resultCatalogSource)
		if resultCatalogSource.DeletionTimestamp != nil || apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// WaitForDeploymentScaled waits Timeout amount of time for the given deployment to be updated
// with the specified number of replicas.
func WaitForDeploymentScaled(client test.FrameworkClient, name, namespace string, replicas int32) error {
	result := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(RetryInterval, 1*time.Minute, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		if *result.Spec.Replicas == replicas {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ScaleMarketplace scales the marketplace deployment to the specified replica scale size
func ScaleMarketplace(client test.FrameworkClient, namespace string, scale int32) error {
	// TODO(tflannag): This should be a parameter
	const name = "marketplace-operator"

	marketplace := &appsv1.Deployment{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, marketplace)
	if err != nil {
		return err
	}
	marketplace.Spec.Replicas = &scale
	err = client.Update(context.TODO(), marketplace)
	if err != nil {
		return err
	}
	// Wait for deployment to scale
	err = WaitForDeploymentScaled(client, name, namespace, scale)
	if err != nil {
		return err
	}

	return nil
}

// CreateSubscriptionDefinition returns a newly built Subscription with the labels
// `csc-owner-name` and `csc-owner-namespace` based on the catalogsourceconfig and the expected name
func CreateSubscriptionDefinition(name, namespace, cscName string, isCreatedByUI bool) *v1alpha1.Subscription {
	labels := make(map[string]string)
	specSource := fmt.Sprintf("%s-%s", cscName, namespace)

	if isCreatedByUI {
		labels[builders.CscOwnerNameLabel] = specSource
		labels[builders.CscOwnerNamespaceLabel] = namespace
	}

	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind: v1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          specSource,
			CatalogSourceNamespace: namespace,
			Channel:                "alpha",
		},
	}
}

// CheckSubscriptionNotUpdated checks that a user created subscription
// was not updated during migration.
func CheckSubscriptionNotUpdated(client test.FrameworkClient, namespace, subscriptionName, installedCscName string) error {
	subscription := &v1alpha1.Subscription{}
	specSource := fmt.Sprintf("%s-%s", installedCscName, namespace)
	err := client.Get(context.TODO(), types.NamespacedName{Name: subscriptionName, Namespace: namespace}, subscription)
	if err != nil {
		return err
	}
	if subscription.Spec.CatalogSource != specSource {
		return fmt.Errorf("user created Subscription %s Spec.CatalogSource has changed. Spec.CatalogSource was %s and is now %s", subscription.GetName(), subscription.Spec.CatalogSource, specSource)
	}
	return nil
}

// EnsureConfigAPIIsAvailable will make a single attempt to add the config
// APIs to the FrameworkScheme. If an error is encountered in this first call,
// it will be returned. Subsequent calls will always return whether or not the
// config CRDs were added to the FrameworkScheme and nil. The boolean
// returned by this method can be used to identify if the tests are  being run
// on an OpenShift cluster. Please note that if either of the config CRDs cannot
// be added none of the associated config tests will run.
// TBD: Separate out the ClusterOperator and OperatorHub CRD availability
// checking.
func EnsureConfigAPIIsAvailable() (bool, error) {
	var err error
	if isConfigAPIPresent == nil {
		// present is used to allocate space for the isConfigAPIPresent pointer.
		present := false

		// Add (configv1) ClusterOperator to framework scheme
		clusterOperator := &configv1.ClusterOperator{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterOperator",
				APIVersion: fmt.Sprintf("%s/%s",
					configv1.SchemeGroupVersion.Group, configv1.SchemeGroupVersion.Version),
			},
		}

		err = test.AddToFrameworkScheme(configv1.Install, clusterOperator)
		if err == nil {
			present = true
		}

		// Add (configv1) OperatorHub to framework scheme
		operatorHub := &configv1.OperatorHubList{
			TypeMeta: metav1.TypeMeta{
				Kind: "OperatorHub",
				APIVersion: fmt.Sprintf("%s/%s",
					configv1.SchemeGroupVersion.Group, configv1.SchemeGroupVersion.Version),
			},
		}
		err = test.AddToFrameworkScheme(configv1.Install, operatorHub)
		if err != nil {
			present = false
		}

		isConfigAPIPresent = &present
	}

	return *isConfigAPIPresent, err
}

// InitCatSrcDefinition reads a default CatalogSource definition from the default directory
// and initializes DefaultSources
func InitCatSrcDefinition() error {
	if DefaultSources != nil {
		return nil
	}

	fileInfos, err := ioutil.ReadDir(DefaultsDir)
	if err != nil {
		return err
	}

	DefaultSources = make([]*olmv1alpha1.CatalogSource, len(fileInfos))

	for i, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		file, err := os.Open(filepath.Join(DefaultsDir, fileName))
		if err != nil {
			DefaultSources = nil
			return err
		}

		DefaultSources[i] = &olmv1alpha1.CatalogSource{}
		decoder := yaml.NewYAMLOrJSONDecoder(file, 1024)
		err = decoder.Decode(DefaultSources[i])
		if err != nil {
			DefaultSources = nil
			return err
		}
	}
	return nil
}

// GetRegistryDeployment returns the deployment object for the given OperatorSource
func GetRegistryDeployment(client test.FrameworkClient, name, namespace string) *appsv1.Deployment {
	// Get the registry deployment of the test OperatorSource
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := client.Get(context.TODO(), namespacedName, deployment)
	if err != nil {
		return nil
	}
	return deployment
}
