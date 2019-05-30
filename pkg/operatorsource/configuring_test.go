package operatorsource_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gomock "github.com/golang/mock/gomock"
	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
)

// Use Case: Registry returns a non-empty list of metadata
// Expected Result: Next phase is set to "Succeeded"
func TestReconcile_ScheduledForConfiguring_Succeeded(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &marketplace.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)
	fakeclient := NewFakeClient()
	refresher := mocks.NewSyncerPackageRefreshNotificationSender(controller)

	reconciler := operatorsource.NewConfiguringReconcilerWithInterfaceClient(helperGetContextLogger(), factory, writer, fakeclient, refresher)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.Configuring)

	opsrcWant := opsrcIn.DeepCopy()
	opsrcWant.Status.Packages = "etcd,prometheus,amqp"

	registryClient := mocks.NewAppRegistryClient(controller)

	optionsWant := appregistry.Options{Source: opsrcIn.Spec.Endpoint}
	factory.EXPECT().New(optionsWant).Return(registryClient, nil).Times(1)

	// We expect the remote registry to return a non-empty list of manifest(s).
	manifestExpected := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Namespace:  "redhat",
			Repository: "myapp",
			Release:    "1.0.0",
			Digest:     "abcdefgh",
		},
	}
	registryClient.EXPECT().ListPackages(opsrcIn.Spec.RegistryNamespace).Return(manifestExpected, nil).Times(1)

	// We expect the datastore to save downloaded manifest(s) returned by the registry.
	writer.EXPECT().Write(opsrcIn, manifestExpected).Return(1, nil)

	// The first time we ask for the packages from the datastore, we expect to get nothing.
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("")

	// Then we expect to call send refresh, because the package list was empty.
	refresher.EXPECT().SendRefresh()

	// We expect datastore to return the specified list of packages.
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(opsrcWant.Status.Packages)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcWant, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Registry returns an empty list of metadata.
// Expected Result: Next phase is set to "Failed".
func TestReconcile_OperatorSourceReturnsEmptyManifestList_Failed(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)
	fakeclient := NewFakeClient()
	refresher := mocks.NewSyncerPackageRefreshNotificationSender(controller)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), factory, writer, fakeclient, refresher)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.Configuring)

	registryClient := mocks.NewAppRegistryClient(controller)

	optionsWant := appregistry.Options{Source: opsrcIn.Spec.Endpoint}
	factory.EXPECT().New(optionsWant).Return(registryClient, nil).Times(1)

	// We expect the registry to return an empty manifest list.
	manifests := []*datastore.RegistryMetadata{}
	registryClient.EXPECT().ListPackages(opsrcIn.Spec.RegistryNamespace).Return(manifests, nil).Times(1)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)
	assert.Error(t, errGot)

	nextPhaseWant := &marketplace.Phase{
		Name:    phase.Failed,
		Message: errGot.Error(),
	}

	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Not configured, CatalogSourceConfig object has not been created yet.
// Expected Result: A properly populated CatalogSourceConfig should get created
// and the next phase should be set to "Succeeded".
func TestReconcile_NotConfigured_NewCatalogConfigSourceObjectCreated(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &marketplace.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)
	registryClient := mocks.NewAppRegistryClient(controller)
	fakeclient := NewFakeClient()
	refresher := mocks.NewSyncerPackageRefreshNotificationSender(controller)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), factory, writer, fakeclient, refresher)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group":                           "Community",
		operatorsource.OpsrcOwnerNameLabel:      "foo",
		operatorsource.OpsrcOwnerNamespaceLabel: "marketplace",
	}
	opsrcIn.SetLabels(labelsWant)

	optionsWant := appregistry.Options{Source: opsrcIn.Spec.Endpoint}
	factory.EXPECT().New(optionsWant).Return(registryClient, nil).Times(1)

	// We expect the remote registry to return a non-empty list of manifest(s).
	manifestExpected := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Namespace:  "redhat",
			Repository: "myapp",
			Release:    "1.0.0",
			Digest:     "abcdefgh",
		},
	}
	registryClient.EXPECT().ListPackages(opsrcIn.Spec.RegistryNamespace).Return(manifestExpected, nil).Times(1)

	// We expect the datastore to save downloaded manifest(s) returned by the registry.
	writer.EXPECT().Write(opsrcIn, manifestExpected).Return(1, nil)

	// The first time we ask for the packages from the datastore, we expect to get nothing.
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("")

	// Then we expect to call send refresh, because the package list was empty.
	refresher.EXPECT().SendRefresh()

	packages := "a,b,c"
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	cscWant := helperNewCatalogSourceConfigWithLabels(opsrcIn.Namespace, opsrcIn.Name, labelsWant)
	cscWant.Spec = marketplace.CatalogSourceConfigSpec{
		TargetNamespace: opsrcIn.Namespace,
		Packages:        packages,
	}

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Not configured, CatalogSourceConfig object already exists due to
// past errors.
// Expected Result: The existing CatalogSourceConfig object should be updated
// accordingly and the next phase should be set to "Succeeded".
func TestReconcile_CatalogSourceConfigAlreadyExists_Updated(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace, name := "marketplace", "foo"
	nextPhaseWant := &marketplace.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)
	registryClient := mocks.NewAppRegistryClient(controller)
	refresher := mocks.NewSyncerPackageRefreshNotificationSender(controller)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group":                           "Community",
		operatorsource.OpsrcOwnerNameLabel:      "foo",
		operatorsource.OpsrcOwnerNamespaceLabel: "marketplace",
	}
	opsrcIn.SetLabels(labelsWant)

	optionsWant := appregistry.Options{Source: opsrcIn.Spec.Endpoint}
	factory.EXPECT().New(optionsWant).Return(registryClient, nil).Times(1)

	// We expect the remote registry to return a non-empty list of manifest(s).
	manifestExpected := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Namespace:  "redhat",
			Repository: "myapp",
			Release:    "1.0.0",
			Digest:     "abcdefgh",
		},
	}
	registryClient.EXPECT().ListPackages(opsrcIn.Spec.RegistryNamespace).Return(manifestExpected, nil).Times(1)

	// We expect the datastore to save downloaded manifest(s) returned by the registry.
	writer.EXPECT().Write(opsrcIn, manifestExpected).Return(1, nil)

	// The first time we ask for the packages from the datastore, we expect to get nothing.
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("")

	// Then we expect to call send refresh, because the package list was empty.
	refresher.EXPECT().SendRefresh()

	packages := "a,b,c"
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	csc := helperNewCatalogSourceConfigWithLabels(namespace, name, labelsWant)
	fakeclient := NewFakeClientWithCSC(csc)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), factory, writer, fakeclient, refresher)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Update of existing CatalogSourceConfig object fails.
// Expected Result: The object is moved to "Failed" phase.
func TestReconcile_UpdateError_MovedToFailedPhase(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace, name := "marketplace", "foo"

	updateError := k8s_errors.NewServerTimeout(schema.GroupResource{}, "operation", 1)
	nextPhaseWant := &marketplace.Phase{
		Name:    phase.Configuring,
		Message: updateError.Error(),
	}

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)
	registryClient := mocks.NewAppRegistryClient(controller)
	refresher := mocks.NewSyncerPackageRefreshNotificationSender(controller)
	kubeclient := mocks.NewClient(controller)

	reconciler := operatorsource.NewConfiguringReconcilerWithInterfaceClient(helperGetContextLogger(), factory, writer, kubeclient, refresher)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	optionsWant := appregistry.Options{Source: opsrcIn.Spec.Endpoint}
	factory.EXPECT().New(optionsWant).Return(registryClient, nil).Times(1)

	// We expect the remote registry to return a non-empty list of manifest(s).
	manifestExpected := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Namespace:  "redhat",
			Repository: "myapp",
			Release:    "1.0.0",
			Digest:     "abcdefgh",
		},
	}
	registryClient.EXPECT().ListPackages(opsrcIn.Spec.RegistryNamespace).Return(manifestExpected, nil).Times(1)

	// We expect the datastore to save downloaded manifest(s) returned by the registry.
	writer.EXPECT().Write(opsrcIn, manifestExpected).Return(1, nil)

	// The first time we ask for the packages from the datastore, we expect to get nothing.
	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("")

	// Then we expect to call send refresh, because the package list was empty.
	refresher.EXPECT().SendRefresh()

	writer.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("a,b,c")

	createErr := k8s_errors.NewAlreadyExists(schema.GroupResource{}, "CatalogSourceConfig already exists")

	kubeclient.EXPECT().Create(context.TODO(), gomock.Any()).Return(createErr)

	kubeclient.EXPECT().Get(context.TODO(), gomock.Any(), gomock.Any()).Return(nil)
	kubeclient.EXPECT().Update(context.TODO(), gomock.Any()).Return(updateError)

	_, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.Error(t, errGot)
	assert.Equal(t, updateError, errGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}
