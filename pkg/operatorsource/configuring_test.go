package operatorsource_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gomock "github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Use Case: Not configured, CatalogSourceConfig object has not been created yet.
// Expected Result: A properly populated CatalogSourceConfig should get created
// and the next phase should be set to "Succeeded".
func TestReconcile_NotConfigured_NewCatalogConfigSourceObjectCreated(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	kubeclient := mocks.NewKubeClient(controller)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), datastore, kubeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase("marketplace", "foo", phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group": "Community",
	}
	opsrcIn.SetLabels(labelsWant)

	packages := "a,b,c"
	datastore.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	trueVar := true
	cscWant := helperNewCatalogSourceConfigWithLabels(opsrcIn.Namespace, opsrcIn.Name, labelsWant)
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
	kubeclient.EXPECT().Create(context.TODO(), cscWant).Return(nil)

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
	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	kubeclient := mocks.NewKubeClient(controller)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), datastore, kubeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group": "Community",
	}
	opsrcIn.SetLabels(labelsWant)

	packages := "a,b,c"
	datastore.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	createErr := k8s_errors.NewAlreadyExists(schema.GroupResource{}, "CatalogSourceConfig already exists")
	kubeclient.EXPECT().Create(context.TODO(), gomock.Any()).Return(createErr)

	// We expect Get to return the given CatalogSourceConfig successfully.
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	cscGet := v1alpha1.CatalogSourceConfig{}
	kubeclient.EXPECT().Get(context.TODO(), namespacedName, &cscGet).Return(nil)

	trueVar := true
	cscWant := helperNewCatalogSourceConfigWithLabels("", "", labelsWant)
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
	kubeclient.EXPECT().Update(context.TODO(), cscWant).Return(nil)

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
	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Failed,
		Message: updateError.Error(),
	}

	datastore := mocks.NewDatastoreWriter(controller)
	kubeclient := mocks.NewKubeClient(controller)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), datastore, kubeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	datastore.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return("a,b,c")

	createErr := k8s_errors.NewAlreadyExists(schema.GroupResource{}, "CatalogSourceConfig already exists")
	kubeclient.EXPECT().Create(context.TODO(), gomock.Any()).Return(createErr)

	kubeclient.EXPECT().Get(context.TODO(), gomock.Any(), gomock.Any()).Return(nil)
	kubeclient.EXPECT().Update(context.TODO(), gomock.Any()).Return(updateError)

	_, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.Error(t, errGot)
	assert.Equal(t, updateError, errGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)
}
