package migrator

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/builders"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	//CscCRDName is the CatalogSourceConfig CRD name
	CscCRDName = "catalogsourceconfigs.operators.coreos.com"

	//Context is left blank for the default context in kube config.
	Context = ""
)

type migrator struct {
	client client.Client
}

// Migrator migrates an old instance of Marketplace which had the
// CatalogSourceConfig API to a new instance,
// that does not have the previously defined API
type Migrator interface {
	Migrate() error
}

// New returns a Migrator that migrates an old instance of Marketplace
// to a new one.
func New(client client.Client) Migrator {
	return &migrator{
		client: client,
	}
}
func (m *migrator) Migrate() error {

	cscs := &v2.CatalogSourceConfigList{}
	err := m.client.List(context.TODO(), &client.ListOptions{}, cscs)
	if err != nil {
		return err
	}
	allErrors := []error{}
	for _, csc := range cscs.Items {
		err = removeCSCOwnerRefFromCatalogSource(&csc, m.client)
		if err != nil {
			allErrors = append(allErrors, err)
		}
		err = deleteCSC(&csc, m.client)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}
	err = deleteCRD(CscCRDName, m.client)
	allErrors = append(allErrors, err)
	return utilerrors.NewAggregate(allErrors)
}

func removeCSCOwnerRefFromCatalogSource(csc *v2.CatalogSourceConfig, kubeClient client.Client) error {
	catsrc := olm.CatalogSource{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{
		Name:      csc.Name,
		Namespace: csc.Spec.TargetNamespace},
		&catsrc)
	if err != nil {
		return err
	}
	labels := catsrc.Labels
	delete(labels, builders.CscOwnerNameLabel)
	delete(labels, builders.CscOwnerNamespaceLabel)
	catsrc.Labels = labels
	err = kubeClient.Update(context.TODO(), &catsrc)
	if err != nil {
		return err
	}
	return nil
}

func deleteCSC(csc *v2.CatalogSourceConfig, client client.Client) error {
	csc.ObjectMeta.Finalizers = []string{}
	err := client.Update(context.TODO(), csc)
	if err != nil {
		return err
	}
	err = client.Delete(context.TODO(), csc)
	if err != nil {
		return err
	}
	return nil
}

func deleteCRD(name string, kubeClient client.Client) error {
	crd := &v1beta1.CustomResourceDefinition{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Name: name}, crd)
	if err == nil {
		if err := kubeClient.Delete(context.TODO(), crd); err != nil {
			return err
		}
		return nil
	}
	return err
}
