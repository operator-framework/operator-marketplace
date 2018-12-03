package operatorsource_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gomock "github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
)

// This test verifies the happy path for purge. We expect purge to be successful
// and the next desired phase set to "Initial" so that reconciliation can start
// anew.
func TestReconcileWithPurging(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.TODO()

	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.OperatorSourcePurging)
	opsrcWant := opsrcIn.DeepCopy()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Initial,
		Message: phase.GetMessage(phase.Initial),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	client := mocks.NewKubeClient(controller)
	reconciler := operatorsource.NewPurgingReconciler(helperGetContextLogger(), datastore, client)

	// We expect the operator source to be removed from the datastore.
	csc := helperNewCatalogSourceConfig(opsrcIn.Namespace, getExpectedCatalogSourceConfigName(opsrcIn.Name))
	datastore.EXPECT().RemoveOperatorSource(opsrcIn.GetUID()).Times(1)

	// We expect the associated CatalogConfigSource object to be deleted.
	client.EXPECT().Delete(ctx, csc)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcWant, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// In the event the associated CatalogSourceConfig object is not found while
// purging is in progress, we expect NotFound error to be ignored and the next
// phase set to "Initial" so that reconciliation can start anew.
func TestReconcileWithPurgingWithCatalogSourceConfigNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx := context.TODO()

	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.OperatorSourcePurging)
	opsrcWant := opsrcIn.DeepCopy()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Initial,
		Message: phase.GetMessage(phase.Initial),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	client := mocks.NewKubeClient(controller)
	reconciler := operatorsource.NewPurgingReconciler(helperGetContextLogger(), datastore, client)

	// We expect the operator source to be removed from the datastore.
	csc := helperNewCatalogSourceConfig(opsrcIn.Namespace, getExpectedCatalogSourceConfigName(opsrcIn.Name))
	datastore.EXPECT().RemoveOperatorSource(opsrcIn.GetUID())

	// We expect kube client to throw a NotFound error.
	notFoundErr := k8s_errors.NewNotFound(schema.GroupResource{}, "CatalogSourceConfig not found")
	client.EXPECT().Delete(ctx, csc).Return(notFoundErr)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.Error(t, errGot)
	assert.Equal(t, opsrcGot, opsrcWant)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}
