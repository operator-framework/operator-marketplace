package operatorsource_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	gomock "github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// Use Case: Successfully validated and scheduled for download.
// Expected Result: Manifest(s) downloaded and stored successfully and the next
// phase set to "Configuring".
func TestReconcile_ScheduledForDownload_Success(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Configuring,
		Message: phase.GetMessage(phase.Configuring),
	}

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)

	reconciler := operatorsource.NewDownloadingReconciler(helperGetContextLogger(), factory, writer)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.OperatorSourceDownloading)

	registryClient := mocks.NewAppRegistryClient(controller)
	factory.EXPECT().New(opsrcIn.Spec.Type, opsrcIn.Spec.Endpoint).Return(registryClient, nil).Times(1)

	// We expect the remote registry to return a non-empty list of manifest(s).
	manifestExpected := []*datastore.OperatorMetadata{
		&datastore.OperatorMetadata{
			RegistryMetadata: datastore.RegistryMetadata{
				Namespace:  "redhat",
				Repository: "myapp",
				Release:    "1.0.0",
				Digest:     "abcdefgh",
			},
		},
	}
	registryClient.EXPECT().RetrieveAll(opsrcIn.Spec.RegistryNamespace).Return(manifestExpected, nil).Times(1)

	// We expect the datastore to save downloaded manifest(s) returned by the registry.
	writer.EXPECT().Write(manifestExpected).Return(nil)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Registry returns an empty list of manifest(s).
// Expected Result: Next phase is set to "Failed".
func TestReconcile_OperatorSourceReturnsEmptyManifestList_ErrorExpected(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewAppRegistryClientFactory(controller)

	reconciler := operatorsource.NewDownloadingReconciler(helperGetContextLogger(), factory, writer)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.OperatorSourceDownloading)

	registryClient := mocks.NewAppRegistryClient(controller)
	factory.EXPECT().New(opsrcIn.Spec.Type, opsrcIn.Spec.Endpoint).Return(registryClient, nil).Times(1)

	// We expect the registry to return an empty manifest list.
	manifests := []*datastore.OperatorMetadata{}
	registryClient.EXPECT().RetrieveAll(opsrcIn.Spec.RegistryNamespace).Return(manifests, nil).Times(1)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)
	assert.Error(t, errGot)

	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Failed,
		Message: errGot.Error(),
	}

	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}
