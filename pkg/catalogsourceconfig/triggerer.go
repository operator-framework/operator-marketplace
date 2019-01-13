package catalogsourceconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTriggerer returns a new instance of Triggerer interface.
func NewTriggerer(client client.Client) Triggerer {
	return &triggerer{
		client:       client,
		transitioner: phase.NewTransitioner(),
	}
}

// Triggerer is an interface that wraps the Trigger method.
//
// Trigger iterates through the list of all CatalogSourceConfig object(s) and
// applies the following logic:
//
// a. Compare the list of package(s) specified in Spec.Packages with the update
//    notification list and determine if the given CatalogSourceConfig specifies
//    a package that has either been removed or has a new version available.
//
// b. If the above is true then update the given CatalogSourceConfig object in
//    order to kick off a new reconciliation. This way it will get the latest
//    package manifest from datastore.
//
// The list call applies the label selector [opsrc-datastore!=true] to exclude
// CatalogSourceConfig object which is used as datastore for marketplace.
type Triggerer interface {
	Trigger(notification datastore.PackageUpdateNotification) error
}

// triggerer implements the Triggerer interface.
type triggerer struct {
	client       client.Client
	transitioner phase.Transitioner
}

func (t *triggerer) Trigger(notification datastore.PackageUpdateNotification) error {
	options := &client.ListOptions{}
	options.SetLabelSelector(fmt.Sprintf("%s!=true", operatorsource.DatastoreLabel))

	cscs := &v1alpha1.CatalogSourceConfigList{}
	if err := t.client.List(context.TODO(), options, cscs); err != nil {
		return err
	}

	allErrors := []error{}
	for _, instance := range cscs.Items {
		// Needed because sdk does not get the gvk.
		instance.EnsureGVK()

		packages, updateNeeded := t.setPackages(&instance, notification)
		if !updateNeeded {
			continue
		}

		if err := t.update(&instance, packages); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return utilerrors.NewAggregate(allErrors)
}

func (t *triggerer) setPackages(instance *v1alpha1.CatalogSourceConfig, notification datastore.PackageUpdateNotification) (packages string, updateNeeded bool) {
	packageList := make([]string, 0)
	for _, pkg := range instance.GetPackageIDs() {
		if notification.IsRemoved(pkg) {
			updateNeeded = true

			// The package specified has been removed from the registry. We will
			// remove it from the spec.
			continue
		}

		packageList = append(packageList, pkg)

		if notification.IsUpdated(pkg) {
			updateNeeded = true
		}
	}

	packages = strings.Join(packageList, ",")
	return
}

func (t *triggerer) update(instance *v1alpha1.CatalogSourceConfig, packages string) error {
	out := instance.DeepCopy()

	// We want to Set the phase to Initial to kick off reconciliation anew.
	nextPhase := &v1alpha1.Phase{
		Name:    phase.Initial,
		Message: "Package(s) have update(s), scheduling for reconciliation",
	}
	t.transitioner.TransitionInto(&out.Status.CurrentPhase, nextPhase)

	out.Spec.Packages = packages

	if err := t.client.Update(context.TODO(), out); err != nil {
		return err
	}

	return nil
}
