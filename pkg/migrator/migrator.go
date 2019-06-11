// Package migrator contains upgrade logic that's needed to upgrade a cluster from
// openshift 4.1.x to openshift 4.2.0.
package migrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/builders"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewMigrator returns a migrator that updates/cleans up existing/stale
// resources, when a Openshift 4.1.x cluster is migrated to version 4.2.x.
// In an openshift 4.1.x cluster, CatalogSourceConfigs were used in the
// install flow of an operator, and the Subscriptions created during
// installing operators were referencing the CatalogSources created by the
// CatalogSourceConfigs(named Installed CatalogSources). In an openshift 4.2 cluster,
// the operator installation flow has been changed, and CatalogSourceConfigs
// are no longer used in the install flow. Instead the CatalogSources are
// directly created by the OperatorSources(named DataStore CatalogSources).
// The migrator updates existing Subscriptions that referenced Installed CatalogSources
// during operator installation in an Openshift 4.1.x cluster, to reference
// Datastore CatalogSources instead. The migrator also deletes the stale
// CatalogSourceConfigs that were created during operator installation, and
// the datastore CatalogSourceConfigs created by OperatorSources
// in an Openshift 4.1.x cluster.
func NewMigrator(client client.Client) migrator {
	return migrator{
		logger: log.NewEntry(log.New()),
		client: client,
	}
}

type migrator struct {
	logger *logrus.Entry
	client client.Client
}

// Migrate updates existing Subscriptions, deletes the CSCs installed during
// operator installation as well the stale datastore CSCs when the cluster is
// migrating from Openshift 4.1.x to Openshift 4.2.x
func (m *migrator) Migrate(operatorNamespace string) {
	installedCscs := m.updateSubscriptions()
	m.deleteInstalledCscs(installedCscs, operatorNamespace)
	m.deleteDatastoreCscs(operatorNamespace)
}

// updateSubscriptions updates the existing Subscriptions'
// spec.source and spec.sourcenamespace fields. Existing
// Subscriptions referenced installed CatalogSources, which
// are updated to reference datastore CatalogSources
// instead.
func (m *migrator) updateSubscriptions() []types.NamespacedName {
	var installedCscs []types.NamespacedName
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf(builders.OwnerNameLabel))

	subscriptions := &olm.SubscriptionList{}
	// Get the list of existing Subscriptions that have the label "csc-owner-name"
	err := m.client.List(context.TODO(), options, subscriptions)
	if err != nil {
		m.logger.Errorf(fmt.Sprintf("Client error: %s", err.Error()))
		return []types.NamespacedName{}
	}
	for _, instance := range subscriptions.Items {
		installedCscs = append(installedCscs, types.NamespacedName{Name: instance.GetLabels()[builders.OwnerNameLabel], Namespace: instance.GetLabels()[builders.OwnerNamespaceLabel]})
		// try to infer the datastore CatalogSource from the Subscription
		datastoreCs, err := findCatalogSource(&instance, m.client)
		if k8s_errors.IsNotFound(err) {
			// infer the CatalogSource from the OperatorSource that has the package
			datastoreCs, err = findCsFromOpsrc(&instance, m.client)
			if err != nil {
				m.logger.Errorf("[migration] Could not infer datastore CatalogSource for Subscription %s.", instance.GetName())
				continue
			}
		}
		// update the Subscription to reference the datastore CatalogSource
		instance.Spec.CatalogSource = datastoreCs.GetName()
		instance.Spec.CatalogSourceNamespace = datastoreCs.GetNamespace()
		err = m.client.Update(context.TODO(), &instance)
		if err != nil {
			m.logger.Errorf("[migration] Error updating subscription %s. Error: %s", instance.GetName(), err.Error())
		} else {
			m.logger.Infof("[migration] Successfully updated Subscription %s", instance.GetName())
		}
	}
	return installedCscs
}

// deleteInstalledCscs deletes the CSCs installed during operator installation.
// The child resources of the CSCs are delete by the finalizer.
func (m *migrator) deleteInstalledCscs(cscs []types.NamespacedName, operatorNamespace string) {
	for _, cscInfo := range cscs {
		// If the CatalogSourceConfig namespace information is missing, try and find the
		// CatalogSourceConfig in the marketplace-operator's namespace
		if cscInfo.Namespace == "" {
			cscInfo.Namespace = operatorNamespace
		}
		csc := newCatalogSourceConfig(cscInfo.Namespace, cscInfo.Name)
		err := m.client.Delete(context.TODO(), csc)
		if err != nil {
			m.logger.Errorf("[migration] Failed to delete installed CSC %s with error: ", cscInfo.Name, err.Error())
		} else {
			m.logger.Infof("[migration] Stale CSC %s scheduled for deletion.", cscInfo.Name)
		}
	}
}

// deleteDatastoreCscs deletes the datastore CSCs created by OperatorSources.
// The child resources of the CSCs are deleted by the finalizer.
func (m *migrator) deleteDatastoreCscs(operatorNamespace string) {
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf("opsrc-datastore: \"true\""))
	options.InNamespace(operatorNamespace)
	cscs := &v2.CatalogSourceConfigList{}
	// Get the list of existing cscs that have the label "opsrc-datastore: "true" "
	err := m.client.List(context.TODO(), options, cscs)
	if err != nil {
		m.logger.Errorf("Client error: %s", err.Error())
		return
	}
	for _, csc := range cscs.Items {
		err = m.client.Delete(context.TODO(), &csc)
		if err != nil {
			m.logger.Errorf("[migration] Failed to delete CatalogSourceConfig %s. Error: %s", csc.GetName(), err.Error())
		} else {
			m.logger.Infof("[migration] Datastore CSC %s scheduled for deletion.", csc.GetName())
		}
	}
}

// findCatalogSource infers the datastore CatalogSource created by the OperatorSource
// in an Openshift 4.2.0 cluster. The inferred datastore CatalogSource will then be
// referenced from an existing Subscription.
func findCatalogSource(subscription *olm.Subscription, client client.Client) (*olm.CatalogSource, error) {
	associatedCscName := subscription.GetLabels()[builders.OwnerNameLabel]
	possibleCsName := ExtractCsName(associatedCscName)
	possibleCsNamespace := subscription.GetLabels()[builders.OwnerNamespaceLabel]
	// try and fetch the CatalogSource
	datastoreCs := &olm.CatalogSource{}
	namespacedName := types.NamespacedName{Name: possibleCsName, Namespace: possibleCsNamespace}
	err := client.Get(context.TODO(), namespacedName, datastoreCs)
	if err != nil {
		return nil, err
	}
	return datastoreCs, nil
}

// findCsFromOpsrc extracts the packageName from a Subscription,
// and finds the corresponding CatalogSource that the package belongs to.
func findCsFromOpsrc(subscription *olm.Subscription, kubeClient client.Client) (*olm.CatalogSource, error) {
	packageName := subscription.Spec.Package
	opsrcs := &v1.OperatorSourceList{}
	err := kubeClient.List(context.TODO(), &client.ListOptions{}, opsrcs)
	if err != nil {
		return nil, err
	}
	for _, instance := range opsrcs.Items {
		if !IsPackageInOpsrc(packageName, &instance) {
			continue
		}
		// fetch the CatalogSource with the same name
		datastoreCs := &olm.CatalogSource{}
		namespacedName := types.NamespacedName{Name: instance.GetName(), Namespace: instance.GetNamespace()}
		err = kubeClient.Get(context.TODO(), namespacedName, datastoreCs)
		if err != nil {
			return nil, err
		}
		return datastoreCs, nil
	}
	return nil, nil
}

// IsPackageInOpsrc takes the name of a package, and an OperatorSource
// as input, and returns true if the package is present in the OperatorSource's
// status.Packages field
func IsPackageInOpsrc(packageName string, opsrc *v1.OperatorSource) bool {
	packages := opsrc.GetPackages()
	for _, pkg := range packages {
		if pkg == packageName {
			return true
		}
	}
	return false
}

// ExtractCsName takes the name of a CatalogSourceConfig
// as input, and infers the name of the corresponding
// datastore CatalogSource. Installed CSCs follow the
// following naming pattern: installed-publisher-namespace.
// For example, for a CatalogSourceConfig named
// `installed-community-openshift-operators`, extractCsName
// extracts `community` off the name, appends `operators`
// to it, and returns `community-operators` as output
func ExtractCsName(cscName string) string {
	possibleCsName := strings.Split(cscName, "-")[1]
	return fmt.Sprintf("%s-%s", possibleCsName, "operators")
}

// newCatalogSourceConfig returns a newly built CatalogSourceConfig
func newCatalogSourceConfig(namespace, name string) *v2.CatalogSourceConfig {
	return &v2.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version),
			Kind: v2.CatalogSourceConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
