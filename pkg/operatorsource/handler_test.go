package operatorsource_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Use Case: Happy path, sdk passes an event with a valid object, reconciliation
// is successful and produces change(s) to the OperatorSource object.
// Expected: Handled successfully and the object is updated.
func TestHandle_PhaseHasChanged_UpdateExpected(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	kubeclient := mocks.NewKubeClient(controller)
	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewMockPhaseReconcilerFactory(controller)
	transitioner := mocks.NewPhaseTransitioner(controller)

	cacheReconciler := mocks.NewPhaseReconciler(controller)
	newCacheReconcilerFunc := func(logger *log.Entry, writer datastore.Writer, client client.Client) operatorsource.Reconciler {
		return cacheReconciler
	}

	handler := operatorsource.NewHandlerWithParams(kubeclient, writer, factory, transitioner, newCacheReconcilerFunc)

	ctx := context.TODO()

	// Making two OperatorSource objects that are not equal to simulate a change.
	opsrcIn, opsrcOut := helperNewOperatorSourceWithEndpoint("marketplace", "foo", "remote"), helperNewOperatorSourceWithEndpoint("marketplace", "foo", "local")

	phaseReconciler := mocks.NewPhaseReconciler(controller)
	factory.EXPECT().GetPhaseReconciler(gomock.Any(), opsrcIn).Return(phaseReconciler, nil).Times(1)

	// We expect the pre-phase reconciler to return no next phase
	cacheReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nil, nil)

	// We expect the phase reconciler to successfully reconcile the object inside event.
	nextPhaseExpected := &v1alpha1.Phase{
		Name:    "validating",
		Message: "validation is in progress",
	}
	phaseReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nextPhaseExpected, nil).Times(1)

	// We expect the transitioner to indicate that the object has changed and needs update.
	transitioner.EXPECT().TransitionInto(&opsrcOut.Status.CurrentPhase, nextPhaseExpected).Return(true).Times(1)

	// We expect the object to be updated successfully.
	kubeclient.EXPECT().Update(ctx, opsrcOut).Return(nil).Times(1)

	errGot := handler.Handle(ctx, opsrcIn)

	assert.NoError(t, errGot)
}

// Use Case: sdk passes an event with a valid object and reconciliation is
// successful and produces no change(s) to object.
// Expected: Handled successfully and the object is not updated.
func TestHandle_PhaseHasNotChanged_NoUpdateExpected(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	kubeclient := mocks.NewKubeClient(controller)
	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewMockPhaseReconcilerFactory(controller)
	transitioner := mocks.NewPhaseTransitioner(controller)

	cacheReconciler := mocks.NewPhaseReconciler(controller)
	newCacheReconcilerFunc := func(logger *log.Entry, writer datastore.Writer, client client.Client) operatorsource.Reconciler {
		return cacheReconciler
	}

	handler := operatorsource.NewHandlerWithParams(kubeclient, writer, factory, transitioner, newCacheReconcilerFunc)

	ctx := context.TODO()

	// Making two OperatorSource objects that are not equal to simulate a change.
	opsrcIn, opsrcOut := helperNewOperatorSourceWithEndpoint("namespace", "foo", "local"), helperNewOperatorSourceWithEndpoint("namespace", "foo", "remote")

	phaseReconciler := mocks.NewPhaseReconciler(controller)
	factory.EXPECT().GetPhaseReconciler(gomock.Any(), opsrcIn).Return(phaseReconciler, nil).Times(1)

	// We expect the pre-phase reconciler to return no next phase
	cacheReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nil, nil)

	// We expect the phase reconcile to be successful.
	phaseReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nil, nil).Times(1)

	// We expect transitioner to indicate that the object has not been changed.
	transitioner.EXPECT().TransitionInto(&opsrcOut.Status.CurrentPhase, nil).Return(false).Times(1)

	errGot := handler.Handle(ctx, opsrcIn)

	assert.NoError(t, errGot)
}

// Use Case: sdk passes an event with a valid object, reconciliation is not
// successful and update of given OperatorSource object fails.
// Expected: Reconciliation error is re-thrown.
func TestHandle_UpdateError_ReconciliationErrorReturned(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	kubeclient := mocks.NewKubeClient(controller)
	writer := mocks.NewDatastoreWriter(controller)
	factory := mocks.NewMockPhaseReconcilerFactory(controller)
	transitioner := mocks.NewPhaseTransitioner(controller)

	cacheReconciler := mocks.NewPhaseReconciler(controller)
	newCacheReconcilerFunc := func(logger *log.Entry, writer datastore.Writer, client client.Client) operatorsource.Reconciler {
		return cacheReconciler
	}

	handler := operatorsource.NewHandlerWithParams(kubeclient, writer, factory, transitioner, newCacheReconcilerFunc)

	ctx := context.TODO()

	opsrcIn, opsrcOut := helperNewOperatorSourceWithEndpoint("namespace", "foo", "local"), helperNewOperatorSourceWithEndpoint("namespace", "foo", "remote")

	phaseReconciler := mocks.NewPhaseReconciler(controller)
	factory.EXPECT().GetPhaseReconciler(gomock.Any(), opsrcIn).Return(phaseReconciler, nil).Times(1)

	// We expect the pre-phase reconciler to return no next phase
	cacheReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nil, nil)

	// We expect the phase reconciler to throw an error.
	reconcileErrorExpected := errors.New("reconciliation error")
	nextPhaseExpected := &v1alpha1.Phase{
		Name:    "Failed",
		Message: "Reconciliation has failed",
	}
	phaseReconciler.EXPECT().Reconcile(ctx, opsrcIn).Return(opsrcOut, nextPhaseExpected, reconcileErrorExpected).Times(1)

	// We expect transitioner to indicate that the object has been changed.
	transitioner.EXPECT().TransitionInto(&opsrcOut.Status.CurrentPhase, nextPhaseExpected).Return(true).Times(1)

	// We expect the object to be updated
	updateErrorExpected := errors.New("object update error")
	kubeclient.EXPECT().Update(ctx, opsrcOut).Return(updateErrorExpected).Times(1)

	errGot := handler.Handle(ctx, opsrcIn)

	assert.Error(t, errGot)
	assert.Equal(t, reconcileErrorExpected, errGot)
}
