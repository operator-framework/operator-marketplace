package migrator

import (
	"context"

	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/builders"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	//OpsrcCRDName is the OperatorSource CRD name
	OpsrcCRDName = "operatorsources.operators.coreos.com"

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

	opsrcs := &v1.OperatorSourceList{}
	err := m.client.List(context.TODO(), opsrcs, nil)
	if err != nil {
		return err
	}
	allErrors := []error{}
	for _, opsrc := range opsrcs.Items {
		err = removeOpsrcOwnerRefFromCatalogSource(&opsrc, m.client)
		if err != nil {
			allErrors = append(allErrors, err)
		}
		err = deleteOpsrc(&opsrc, m.client)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}
	err = deleteCRD(OpsrcCRDName, m.client)
	allErrors = append(allErrors, err)
	return utilerrors.NewAggregate(allErrors)
}

func removeOpsrcOwnerRefFromCatalogSource(opsrc *v1.OperatorSource, kubeClient client.Client) error {
	catsrc := olm.CatalogSource{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{
		Name:      opsrc.Name,
		Namespace: opsrc.Namespace},
		&catsrc)
	if err != nil {
		return err
	}
	labels := catsrc.Labels
	delete(labels, builders.OpsrcOwnerNameLabel)
	delete(labels, builders.OpsrcOwnerNamespaceLabel)
	catsrc.Labels = labels
	err = kubeClient.Update(context.TODO(), &catsrc)
	if err != nil {
		return err
	}
	return nil
}

func deleteOpsrc(opsrc *v1.OperatorSource, client client.Client) error {
	opsrc.ObjectMeta.Finalizers = []string{}
	err := client.Update(context.TODO(), opsrc)
	if err != nil {
		return err
	}
	err = client.Delete(context.TODO(), opsrc)
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
