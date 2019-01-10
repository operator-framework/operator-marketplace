package catalogsourceconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Copied from https://github.com/operator-framework/operator-lifecycle-manager/blob/master/pkg/controller/registry/configmap_loader.go#L18
// TBD: Vendor in the folder once we require more than just these constants from
// the OLM registry code.
const (
	ConfigMapCRDName     = "customResourceDefinitions"
	ConfigMapCSVName     = "clusterServiceVersions"
	ConfigMapPackageName = "packages"
)

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
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	nextPhase = phase.GetNext(phase.Succeeded)

	r.log.Info("The object has been successfully reconciled")
	return
}

// createCatalogData constructs the ConfigMap data by reading the manifest
// information of all packages from the datasource.
func (r *configuringReconciler) createCatalogData(csc *v1alpha1.CatalogSourceConfig) (map[string]string, error) {
	packageIDs := GetPackageIDs(csc.Spec.Packages)
	data := make(map[string]string)
	if len(packageIDs) < 1 {
		return data, fmt.Errorf("No packages specified in CatalogSourceConfig %s/%s", csc.Namespace, csc.Name)
	}

	manifest, err := r.reader.Read(packageIDs)
	if err != nil {
		r.log.Errorf("Error \"%v\" getting manifest", err)
		return nil, err
	}

	r.log.Infof("The following package(s) have been added: [%s]", packageIDs)

	// TBD: Do we create a CatalogSource per package?
	// TODO: Add more error checking
	data[ConfigMapCRDName] = manifest.CustomResourceDefinitions
	data[ConfigMapCSVName] = manifest.ClusterServiceVersions
	data[ConfigMapPackageName] = manifest.Packages

	return data, nil
}

// reconcileCatalogSource ensures a CatalogSource exists with all the
// resources it requires.
func (r *configuringReconciler) reconcileCatalogSource(csc *v1alpha1.CatalogSourceConfig) error {
	// Reconcile the ConfigMap required for the CatalogSource
	err := r.reconcileConfigMap(csc)
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
	configMapName := csc.Name
	if err == nil {
		if catalogSourceGet.Spec.ConfigMap != configMapName {
			catalogSourceGet.Spec.ConfigMap = configMapName
			r.log.Infof("Updating CatalogSource %s", catalogSourceGet.Name)
			err = r.client.Update(context.TODO(), catalogSourceGet)
			if err != nil {
				r.log.Errorf("Failed to update CatalogSource : %v", err)
				return err
			}
			r.log.Infof("Updated CatalogSource %s", catalogSourceGet.Name)
		}
	} else {
		// Create the CatalogSource structure
		catalogSource := newCatalogSource(csc, configMapName)
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

// reconcileConfigMap ensures a ConfigMap exists with all the Operator artifacts
// in its Data section
func (r *configuringReconciler) reconcileConfigMap(csc *v1alpha1.CatalogSourceConfig) error {
	// Construct the operator artifact data to be placed in the ConfigMap data
	// section.
	data, err := r.createCatalogData(csc)
	if err != nil {
		return err
	}

	// Check if the ConfigMap already exists
	configMapName := csc.Name
	configMapGet := new(ConfigMapBuilder).WithTypeMeta().ConfigMap()
	key := client.ObjectKey{
		Name:      configMapName,
		Namespace: csc.Spec.TargetNamespace,
	}
	err = r.client.Get(context.TODO(), key, configMapGet)

	// Update the ConfigMap if it exists else create one.
	if err == nil {
		r.log.Infof("Updating ConfigMap %s", configMapGet.Name)
		configMapGet.Data = data
		err = r.client.Update(context.TODO(), configMapGet)
		if err != nil {
			r.log.Errorf("Failed to update ConfigMap : %v", err)
			return err
		}
		r.log.Infof("Updated ConfigMap %s", configMapGet.Name)
	} else {
		// Create the ConfigMap structure that will be used by the CatalogSource.
		catalogConfigMap := newConfigMap(csc, data)
		r.log.Infof("Creating ConfigMap %s", catalogConfigMap.Name)
		err = r.client.Create(context.TODO(), catalogConfigMap)
		if err != nil && !errors.IsAlreadyExists(err) {
			r.log.Errorf("Failed to create ConfigMap : %v", err)
			return err
		}
		r.log.Infof("Created ConfigMap %s", catalogConfigMap.Name)
	}
	return nil
}

// GetPackageIDs returns a list of IDs from a comma separated string of IDs.
func GetPackageIDs(csIDs string) []string {
	return strings.Split(csIDs, ",")
}

// newConfigMap returns a new ConfigMap object.
func newConfigMap(csc *v1alpha1.CatalogSourceConfig, data map[string]string) *corev1.ConfigMap {
	return new(ConfigMapBuilder).
		WithMeta(csc.Name, csc.Spec.TargetNamespace).
		WithOwner(csc).
		WithData(data).
		ConfigMap()
}

// newCatalogSource returns a CatalogSource object.
func newCatalogSource(csc *v1alpha1.CatalogSourceConfig, configMapName string) *olm.CatalogSource {
	builder := new(CatalogSourceBuilder).
		WithOwner(csc).
		WithMeta(csc.Name, csc.Spec.TargetNamespace).
		// TBD: where do we get display name and publisher from?
		WithSpec("internal", configMapName, csc.Spec.DisplayName, csc.Spec.Publisher)

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
