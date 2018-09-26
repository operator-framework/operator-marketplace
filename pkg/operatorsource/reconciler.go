package operatorsource

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reconciler reconciles objects of OperatorSource type
type Reconciler interface {
	// IsAlreadyReconciled returns true if the event associated with the object has already been reconciled
	IsAlreadyReconciled(*v1alpha1.OperatorSource) (bool, error)

	// Reconcile reconciles a newly created or updated instance
	Reconcile(*v1alpha1.OperatorSource) error
}

type reconciler struct {
	factory   appregistry.ClientFactory
	datastore *hashmapDatastore
}

// Given a name of an instance of OperatorSource type, this function returns the name of the associated CatalogSourceConfig type object
func getCatalogSourceConfigName(operatorSourceName string) string {
	return fmt.Sprintf("opsrc-%s", operatorSourceName)
}

func (r *reconciler) IsAlreadyReconciled(opsrc *v1alpha1.OperatorSource) (bool, error) {
	into := newCatalogSourceConfigTypeWithMetadata(opsrc.Namespace, getCatalogSourceConfigName(opsrc.Name))

	err := sdk.Get(into)

	if err == nil {
		return true, nil
	}

	if k8s_errors.IsNotFound(err) {
		return false, nil
	}

	return false, err
}

func (r *reconciler) Reconcile(opsrc *v1alpha1.OperatorSource) error {
	registry, err := r.factory.New(opsrc.Spec.Type, opsrc.Spec.Endpoint)
	if err != nil {
		return err
	}

	manifests, err := registry.RetrieveAll()
	if err != nil {
		return err
	}

	if err := r.datastore.Write(manifests); err != nil {
		return err
	}

	list := r.datastore.GetPackageIDs()
	o := newCatalogSourceConfigType(getCatalogSourceConfigName(opsrc.Name), list, opsrc)
	if err := sdk.Create(o); err != nil {
		return err
	}

	return nil
}

func newCatalogSourceConfigType(name string, packages string, opsrc *v1alpha1.OperatorSource) *v1alpha1.CatalogSourceConfig {
	csc := newCatalogSourceConfigTypeWithMetadata(opsrc.Namespace, name)

	csc.Spec = v1alpha1.CatalogSourceConfigSpec{
		TargetNamespace: opsrc.Namespace,
		Packages:        packages,
	}

	owner := asOwner(opsrc)
	ownerReferences := append(csc.GetOwnerReferences(), owner)
	csc.SetOwnerReferences(ownerReferences)

	return csc
}

func newCatalogSourceConfigTypeWithMetadata(namespace, name string) *v1alpha1.CatalogSourceConfig {
	csc := &v1alpha1.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind:       v1alpha1.CatalogSourceConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return csc
}

func asOwner(opsrc *v1alpha1.OperatorSource) metav1.OwnerReference {
	trueVar := true

	return metav1.OwnerReference{
		APIVersion: opsrc.APIVersion,
		Kind:       opsrc.Kind,
		Name:       opsrc.Name,
		UID:        opsrc.UID,
		Controller: &trueVar,
	}
}
