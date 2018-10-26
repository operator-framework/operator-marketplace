package phase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gomock "github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource/phase"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getExpectedCatalogSourceConfigName(opsrcName string) string {
	return fmt.Sprintf("opsrc-%s", opsrcName)
}

// Use Case: Not configured, CatalogSourceConfig object has not been created yet.
// Expected Result: A properly populated CatalogSourceConfig should get created
// and the next phase should be set to "Succeeded".
func TestReconcile_NotConfigured_NewCatalogConfigSourceObjectCreated(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    v1alpha1.OperatorSourcePhaseSucceeded,
		Message: v1alpha1.GetOperatorSourcePhaseMessage(v1alpha1.OperatorSourcePhaseSucceeded),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	kubeclient := mocks.NewKubeClient(controller)

	reconciler := phase.NewConfiguringReconciler(helperGetContextLogger(), datastore, kubeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSource("marketplace", "foo", v1alpha1.OperatorSourcePhaseConfiguring)

	// We expect that the given CatalogConfigSource object does not exist.
	cscGet := helperNewCatalogSourceConfig(opsrcIn.Namespace, getExpectedCatalogSourceConfigName(opsrcIn.Name))
	kubeClientErr := k8s_errors.NewNotFound(schema.GroupResource{}, "CatalogSourceConfig not found")
	kubeclient.EXPECT().Get(cscGet).Return(kubeClientErr)

	packages := "a,b,c"
	datastore.EXPECT().GetPackageIDs().Return(packages)

	trueVar := true
	cscWant := cscGet.DeepCopy()
	cscWant.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		metav1.OwnerReference{
			APIVersion: opsrcIn.APIVersion,
			Kind:       opsrcIn.Kind,
			Name:       opsrcIn.Name,
			UID:        opsrcIn.UID,
			Controller: &trueVar,
		},
	}
	cscWant.Spec = v1alpha1.CatalogSourceConfigSpec{
		TargetNamespace: opsrcIn.Namespace,
		Packages:        packages,
	}
	kubeclient.EXPECT().Create(cscWant).Return(nil)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}

// Use Case: Already configured, CatalogSourceConfig object already exists.
// Expected Result: No action is taken and the next phase is set to "Succeeded".
func TestReconcile_AlreadyConfigured_NoActionTaken(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    v1alpha1.OperatorSourcePhaseSucceeded,
		Message: v1alpha1.GetOperatorSourcePhaseMessage(v1alpha1.OperatorSourcePhaseSucceeded),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	kubeclient := mocks.NewKubeClient(controller)

	reconciler := phase.NewConfiguringReconciler(helperGetContextLogger(), datastore, kubeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSource("marketplace", "foo", v1alpha1.OperatorSourcePhaseConfiguring)
	cscGet := helperNewCatalogSourceConfig(opsrcIn.Namespace, getExpectedCatalogSourceConfigName(opsrcIn.Name))

	// We expect that the given CatalogConfigSource object already exists.
	kubeclient.EXPECT().Get(cscGet).Return(nil).Times(1)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}
