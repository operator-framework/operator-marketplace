package operatorsource_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	gomock "github.com/golang/mock/gomock"
	"github.com/operator-framework/operator-marketplace/pkg/apis"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	mocks "github.com/operator-framework/operator-marketplace/pkg/mocks/operatorsource_mocks"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Use Case: Not configured, CatalogSourceConfig object has not been created yet.
// Expected Result: A properly populated CatalogSourceConfig should get created
// and the next phase should be set to "Succeeded".
func TestReconcile_NotConfigured_NewCatalogConfigSourceObjectCreated(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	namespace, name := "marketplace", "foo"
	nextPhaseWant := &v1alpha1.Phase{
		Name:    phase.Succeeded,
		Message: phase.GetMessage(phase.Succeeded),
	}

	scheme := runtime.NewScheme()
	apis.AddToScheme(scheme)

	datastore := mocks.NewDatastoreWriter(controller)
	fakeclient := fake.NewFakeClientWithScheme(scheme)

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), datastore, fakeclient)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group": "Community",
	}
	opsrcIn.SetLabels(labelsWant)

	packages := "a,b,c"
	datastore.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)

	// Verify reconciler passed expected CatalogSourceConfig to fakeclient
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

	cscNamespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	cscGot := &v1alpha1.CatalogSourceConfig{}
	fakeclient.Get(ctx, cscNamespacedName, cscGot)

	assert.Equal(t, cscWant, cscGot)
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
	scheme := runtime.NewScheme()
	apis.AddToScheme(scheme)

	datastore := mocks.NewDatastoreWriter(controller)
	fakeclient := fake.NewFakeClientWithScheme(scheme)

	ctx := context.TODO()
	opsrcIn := helperNewOperatorSourceWithPhase(namespace, name, phase.Configuring)

	labelsWant := map[string]string{
		"opsrc-group": "Community",
	}
	opsrcIn.SetLabels(labelsWant)

	packages := "a,b,c"
	datastore.EXPECT().GetPackageIDsByOperatorSource(opsrcIn.GetUID()).Return(packages)

	// The given CatalogConfigSource object already exists.
	cscIn := helperNewCatalogSourceConfig(opsrcIn.Namespace, opsrcIn.Name)
	fakeclient.Create(ctx, cscIn)

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

	reconciler := operatorsource.NewConfiguringReconciler(helperGetContextLogger(), datastore, fakeclient)

	opsrcGot, nextPhaseGot, errGot := reconciler.Reconcile(ctx, opsrcIn)

	assert.NoError(t, errGot)
	assert.Equal(t, opsrcIn, opsrcGot)
	assert.Equal(t, nextPhaseWant, nextPhaseGot)

	cscNamespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	cscGot := &v1alpha1.CatalogSourceConfig{}
	fakeclient.Get(ctx, cscNamespacedName, cscGot)

	assert.Equal(t, cscWant, cscGot)
}
