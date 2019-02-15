package catalogsourceconfig

import (
	"context"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultRegistryServerImage is the registry image to be used in the absence of
// the command line parameter.
const DefaultRegistryServerImage = "quay.io/openshift/origin-operator-registry"

// RegistryServerImage is the image used for creating the operator registry pod.
// This gets set in the cmd/manager/main.go.
var RegistryServerImage string

// NewConfiguringReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Configuring" phase.
func NewConfiguringReconciler(log *logrus.Entry, reader datastore.Reader, client client.Client, cache Cache) Reconciler {
	return &configuringReconciler{
		log:    log,
		reader: reader,
		client: client,
		cache:  cache,
	}
}

// configuringReconciler is an implementation of Reconciler interface that
// reconciles a CatalogSourceConfig object in the "Configuring" phase.
type configuringReconciler struct {
	log    *logrus.Entry
	reader datastore.Reader
	client client.Client
	cache  Cache
}

// Reconcile reconciles a CatalogSourceConfig object that is in the
// "Configuring" phase. It ensures that a corresponding CatalogSource object
// exists.
//
// Upon success, it returns "Succeeded" as the next and final desired phase.
// On error, the function returns "Failed" as the next desired phase
// and Message is set to the appropriate error message.
func (r *configuringReconciler) Reconcile(ctx context.Context, in *v1alpha1.CatalogSourceConfig) (out *v1alpha1.CatalogSourceConfig, nextPhase *v1alpha1.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Configuring {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	// Populate the cache before we reconcile to preserve previous data
	// in case of a failure.
	r.cache.Set(out)

	err = r.reconcileCatalogSource(in)
	if err != nil {
		nextPhase = phase.GetNextWithMessage(phase.Configuring, err.Error())
		return
	}

	nextPhase = phase.GetNext(phase.Succeeded)

	r.log.Info("The object has been successfully reconciled")
	return
}

// reconcileCatalogSource ensures a CatalogSource exists with all the
// resources it requires.
func (r *configuringReconciler) reconcileCatalogSource(csc *v1alpha1.CatalogSourceConfig) error {
	// Ensure that at least one package in the spec is available in the datastore
	err := r.checkPackages(csc)
	if err != nil {
		return err
	}

	// Ensure that a registry deployment is available
	registry := NewRegistry(r.log, r.client, r.reader, csc, RegistryServerImage)
	err = registry.Ensure()
	if err != nil {
		return err
	}

	// Check if the CatalogSource already exists
	catalogSourceGet := new(CatalogSourceBuilder).WithTypeMeta().CatalogSource()
	key := client.ObjectKey{
		Name:      csc.Name,
		Namespace: csc.Spec.TargetNamespace,
	}
	err = r.client.Get(context.TODO(), key, catalogSourceGet)

	// Update the CatalogSource if it exists else create one.
	if err == nil {
		catalogSourceGet.Spec.Address = registry.GetAddress()
		r.log.Infof("Updating CatalogSource %s", catalogSourceGet.Name)
		err = r.client.Update(context.TODO(), catalogSourceGet)
		if err != nil {
			r.log.Errorf("Failed to update CatalogSource : %v", err)
			return err
		}
		r.log.Infof("Updated CatalogSource %s", catalogSourceGet.Name)
	} else {
		// Create the CatalogSource structure
		catalogSource := newCatalogSource(csc, registry.GetAddress())
		r.log.Infof("Creating CatalogSource %s", catalogSource.Name)
		err = r.client.Create(context.TODO(), catalogSource)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create CatalogSource : %v", err)
			return err
		}
		r.log.Infof("Created CatalogSource %s", catalogSource.Name)
	}

	return nil
}

// checkPackages returns an error if there no valid packages present in the
// datastore.
func (r *configuringReconciler) checkPackages(csc *v1alpha1.CatalogSourceConfig) error {
	atLeastOneFound := false
	errors := []error{}
	packageIDs := GetPackageIDs(csc.Spec.Packages)
	for _, packageID := range packageIDs {
		if _, err := r.reader.Read(packageID); err != nil {
			errors = append(errors, err)
			continue
		}
		atLeastOneFound = true
	}

	if atLeastOneFound == false {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

// GetPackageIDs returns a list of IDs from a comma separated string of IDs.
func GetPackageIDs(csIDs string) []string {
	return strings.Split(csIDs, ",")
}

// newCatalogSource returns a CatalogSource object.
func newCatalogSource(csc *v1alpha1.CatalogSourceConfig, address string) *olm.CatalogSource {
	builder := new(CatalogSourceBuilder).
		WithOwner(csc).
		WithMeta(csc.Name, csc.Spec.TargetNamespace).
		WithSpec(olm.SourceTypeGrpc, address, csc.Spec.DisplayName, csc.Spec.Publisher)

	// Check if the operatorsource.DatastoreLabel is "true" which indicates that
	// the CatalogSource is the datastore for an OperatorSource. This is a hint
	// for us to set the "olm-visibility" label in the CatalogSource so that it
	// is not visible in the OLM Packages UI. In addition we will set the
	// "openshift-marketplace" label which will be used by the Marketplace UI
	// to filter out global CatalogSources.
	cscLabels := csc.ObjectMeta.GetLabels()
	datastoreLabel, found := cscLabels[operatorsource.DatastoreLabel]
	if found && strings.ToLower(datastoreLabel) == "true" {
		builder.WithOLMLabels(cscLabels)
	}

	return builder.CatalogSource()
}
