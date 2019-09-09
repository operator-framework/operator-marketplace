// Package migrator contains upgrade logic that's needed to upgrade a cluster from
// openshift 4.1.x to openshift 4.2.0.
package migrator

import (
	"context"
	"fmt"
	"strings"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/builders"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// retryNumber is the number of times to retry migration before failing
const retryNumber = 3

// NewMigrator returns a migrator that updates/cleans up existing/stale
// resources, when a Openshift 4.1.x cluster is migrated to version 4.2.x.
// In an openshift 4.1.x cluster, CatalogSourceConfigs were used in the
// install flow of an operator, and the Subscriptions created during
// installing operators were referencing the CatalogSources created by the
// CatalogSourceConfigs(named Installed CatalogSources). In an openshift 4.2 cluster,
// the operator installation flow has been changed, and CatalogSourceConfigs
// are no longer used in the install flow. Instead the CatalogSources (named
// DataStore CatalogSources) are directly created by the OperatorSources.
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

// doMigrate updates existing Subscriptions, deletes the CSCs installed during
// operator installation as well the stale datastore CSCs when the cluster is
// migrating from Openshift 4.1.z to Openshift 4.2.z
func (m *migrator) doMigrate(operatorNamespace string) []error {
	installedCscs, errors := m.updateSubscriptions()
	errors = append(errors, m.deleteInstalledCscs(installedCscs, operatorNamespace)...)
	errors = append(errors, m.deleteDatastoreCscs(operatorNamespace)...)
	return errors
}

// Migrate is a wrapper for doMigrate in order to retry the migration
// retryNumber times before reporting an error
func (m *migrator) Migrate(operatorNamespace string) error {
	for i := retryNumber; i > 0; i-- {
		err := m.doMigrate(operatorNamespace)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("[migration] Migration Failed to Complete")
}

// updateSubscriptions updates the existing Subscriptions'
// spec.source and spec.sourcenamespace fields. Existing
// Subscriptions referenced installed CatalogSources, which
// are updated to reference datastore CatalogSources
// instead.
func (m *migrator) updateSubscriptions() ([]types.NamespacedName, []error) {
	var installedCscs []types.NamespacedName
	var errors []error
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf(builders.CscOwnerNameLabel))

	subscriptions := &olm.SubscriptionList{}
	// Get the list of existing Subscriptions that have the label "csc-owner-name"
	// which is a label added to Subscriptions in the 4.1 UI
	err := m.client.List(context.TODO(), options, subscriptions)
	if err != nil {
		msg := fmt.Sprintf("[migration] Client error: %s", err.Error())
		errors = append(errors, fmt.Errorf(msg))
		m.logger.Errorf(msg)
		return []types.NamespacedName{}, errors
	}
	for _, subscription := range subscriptions.Items {
		if builders.HasOwnerLabels(subscription.GetLabels(), v2.CatalogSourceConfigKind) {
			// Try to infer the datastore CatalogSource from the Subscription
			datastoreCs, err := findCatalogSource(&subscription, m.client)
			if err != nil {
				// Infer the CatalogSource from the OperatorSource that has the package
				datastoreCs, err = findCsFromOpsrc(&subscription, m.client)
				if err != nil {
					msg := fmt.Sprintf("[migration] Could not infer datastore CatalogSource for Subscription %s: %v", subscription.GetName(), err)
					m.logger.Warnf(msg)
					continue
				}
			}

			// Update the Subscription to reference the datastore CatalogSource
			subscription.Spec.CatalogSource = datastoreCs.GetName()
			subscription.Spec.CatalogSourceNamespace = datastoreCs.GetNamespace()
			// Get the name and namespace of the InstalledCsc
			installedCscName := subscription.GetLabels()[builders.CscOwnerNameLabel]
			installedCscNamespace := subscription.GetLabels()[builders.CscOwnerNamespaceLabel]
			// Remove the owner labels from the subscription
			subscription.Labels = labelsutil.CloneAndRemoveLabel(
				labelsutil.CloneAndRemoveLabel(subscription.GetLabels(), builders.CscOwnerNameLabel),
				builders.CscOwnerNamespaceLabel)
			err = m.client.Update(context.TODO(), &subscription)
			if err != nil {
				msg := fmt.Sprintf("[migration] Error updating subscription %s. Error: %s", subscription.GetName(), err.Error())
				errors = append(errors, fmt.Errorf(msg))
				m.logger.Errorf(msg)
			} else {
				m.logger.Infof("[migration] Successfully updated Subscription %s", subscription.GetName())
				installedCscs = append(
					installedCscs,
					types.NamespacedName{
						Name:      installedCscName,
						Namespace: installedCscNamespace})
			}
		}
	}
	return installedCscs, errors
}

// deleteInstalledCscs deletes the CSCs installed during operator installation.
// The child resources of the CSCs are delete by the finalizer.
func (m *migrator) deleteInstalledCscs(cscs []types.NamespacedName, operatorNamespace string) []error {
	var errors []error
	for _, cscInfo := range cscs {
		csc := newCatalogSourceConfig(operatorNamespace, cscInfo.Name)
		err := m.client.Delete(context.TODO(), csc)
		if err != nil {
			msg := fmt.Sprintf("[migration] Failed to delete installed CSC %s with error: %s", cscInfo.Name, err.Error())
			errors = append(errors, fmt.Errorf(msg))
			m.logger.Errorf(msg)
		} else {
			m.logger.Infof("[migration] Stale CSC %s scheduled for deletion.", cscInfo.Name)
		}
	}
	return errors
}

// deleteDatastoreCscs deletes the datastore CSCs created by OperatorSources.
// The child resources of the CSCs are deleted by the finalizer.
func (m *migrator) deleteDatastoreCscs(operatorNamespace string) []error {
	var errors []error
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf(datastore.DatastoreLabel))
	options.InNamespace(operatorNamespace)
	cscs := &v2.CatalogSourceConfigList{}
	// Get the list of existing cscs that have the datastore label
	err := m.client.List(context.TODO(), options, cscs)
	if err != nil {
		msg := fmt.Sprintf("[migration] Client error: %s", err.Error())
		errors = append(errors, fmt.Errorf(msg))
		m.logger.Errorf(msg)
		return errors
	}
	for _, csc := range cscs.Items {
		err = m.client.Delete(context.TODO(), &csc)
		if err != nil {
			msg := fmt.Sprintf("[migration] Failed to delete CatalogSourceConfig %s. Error: %s", csc.GetName(), err.Error())
			errors = append(errors, fmt.Errorf(msg))
			m.logger.Errorf(msg)
		} else {
			m.logger.Infof("[migration] Datastore CSC %s scheduled for deletion.", csc.GetName())
		}
	}
	return errors
}

// findCatalogSource infers the datastore CatalogSource created by the OperatorSources.
// The inferred datastore CatalogSource will then be referenced from an existing
// 4.1.z Subscription.
func findCatalogSource(subscription *olm.Subscription, client client.Client) (*olm.CatalogSource, error) {
	associatedCscName := subscription.GetLabels()[builders.CscOwnerNameLabel]
	possibleCsName, err := ExtractCsName(associatedCscName)
	if err != nil {
		return nil, err
	}
	possibleCsNamespace := subscription.GetLabels()[builders.CscOwnerNamespaceLabel]
	// Try and fetch the CatalogSource
	datastoreCs := &olm.CatalogSource{}
	namespacedName := types.NamespacedName{Name: possibleCsName, Namespace: possibleCsNamespace}
	err = client.Get(context.TODO(), namespacedName, datastoreCs)
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
	for _, opsrc := range opsrcs.Items {
		if !IsPackageInOpsrc(packageName, &opsrc) {
			continue
		}
		// Fetch the CatalogSource with the same name
		datastoreCs := &olm.CatalogSource{}
		namespacedName := types.NamespacedName{Name: opsrc.GetName(), Namespace: opsrc.GetNamespace()}
		err = kubeClient.Get(context.TODO(), namespacedName, datastoreCs)
		if err != nil {
			return nil, err
		}
		return datastoreCs, nil
	}

	return nil, fmt.Errorf("Unable to find any CatalogSource to associate to the subscription")
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
func ExtractCsName(cscName string) (string, error) {
	possibleCsName := strings.Split(cscName, "-")
	if len(possibleCsName) > 2 {
		return fmt.Sprintf("%s-%s", possibleCsName[1], "operators"), nil
	}
	return "", fmt.Errorf("CatalogSourceConfig name %s implies it was not created by the UI", cscName) 
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
