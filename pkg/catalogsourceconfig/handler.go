package catalogsourceconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/datastore"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Copied from https://github.com/operator-framework/operator-lifecycle-manager/blob/master/pkg/controller/registry/configmap_loader.go#L18
// TBD: Vendor in the folder once we require more than just these constants from the OLM registry code
const (
	ConfigMapCRDName     = "customResourceDefinitions"
	ConfigMapCSVName     = "clusterServiceVersions"
	ConfigMapPackageName = "packages"
)

// Handler is the interface that wraps the Handle method
type Handler interface {
	Handle(ctx context.Context, event sdk.Event) error
}

type handler struct {
	reader datastore.Reader
}

var log *logrus.Entry

// Handle handles a new event associated with CatalogSourceConfig type
func (h *handler) Handle(ctx context.Context, event sdk.Event) error {
	csc := event.Object.(*v1alpha1.CatalogSourceConfig)
	log = getLoggerWithCatalogSourceConfigTypeFields(csc)
	// Ignore the delete event as the garbage collector will clean up the created resources as per the OwnerReference
	if event.Deleted {
		log.Infof("Deleted")
		return nil
	}
	data, err := h.createCatalogData(csc)
	if err != nil {
		return err
	}
	return createCatalogSource(csc, data)
}

// createCatalogData constructs the ConfigMap data by reading the manifest information of all packages
// from the datasource
func (h *handler) createCatalogData(csc *v1alpha1.CatalogSourceConfig) (map[string]string, error) {
	packageIDs := getPackageIDs(csc.Spec.Packages)
	data := make(map[string]string)
	if len(packageIDs) < 1 {
		return data, fmt.Errorf("No packages specified in CatalogSourceConfig %s/%s", csc.Namespace, csc.Name)
	}

	// TBD: Do we create a CatalogSource per package?
	for id := range packageIDs {
		manifest, err := h.reader.Read(packageIDs[id])
		if err != nil {
			log.Errorf("Error \"%v\" getting manifest for package ID %s", err, packageIDs[id])
			continue
		}
		// TODO: Add more error checking
		data[ConfigMapCRDName] += manifest.Manifest.Data.CRDs
		data[ConfigMapCSVName] += manifest.Manifest.Data.CSVs
		data[ConfigMapPackageName] += manifest.Manifest.Data.Packages
	}
	return data, nil
}

// createCatalogSource creates a new CatalogSource CR and all the resources it requires
func createCatalogSource(cr *v1alpha1.CatalogSourceConfig, data map[string]string) error {
	// Create the ConfigMap that will be used by the CatalogSource
	catalogConfigMap := newConfigMap(cr, data)
	log.Infof("Creating %s ConfigMap", catalogConfigMap.Name)
	err := sdk.Create(catalogConfigMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Errorf("Failed to create ConfigMap : %v", err)
		return err
	}

	catalogSource := newCatalogSource(cr, catalogConfigMap.Name)
	err = sdk.Create(catalogSource)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Errorf("Failed to create CatalogSource : %v", err)
		return err
	}
	log.Infof("Created")
	return nil
}

// getLoggerWithCatalogSourceConfigTypeFields returns a logger entry that can be used for consistent logging
func getLoggerWithCatalogSourceConfigTypeFields(csc *v1alpha1.CatalogSourceConfig) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"type":            csc.TypeMeta.Kind,
		"targetNamespace": csc.Spec.TargetNamespace,
		"name":            csc.GetName(),
	})
}

// getPackageIDs returns a list of IDs from a comma separated string of IDs
func getPackageIDs(csIDs string) []string {
	return strings.Split(csIDs, ",")
}

// newCatalogSource returns a CatalogSource object
func newCatalogSource(cr *v1alpha1.CatalogSourceConfig, configMapName string) *olm.CatalogSource {
	name := v1alpha1.CatalogSourcePrefix + cr.Name
	return &olm.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       olm.CatalogSourceKind,
			APIVersion: olm.CatalogSourceCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Spec.TargetNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, cr.GroupVersionKind()),
			},
		},
		Spec: olm.CatalogSourceSpec{
			SourceType:  "internal",
			ConfigMap:   configMapName,
			DisplayName: cr.Name,
			// TBD: Where do we get this information from?
			Publisher: cr.Name,
		},
	}
}

// newConfigMap returns a new ConfigMap object
func newConfigMap(cr *v1alpha1.CatalogSourceConfig, data map[string]string) *corev1.ConfigMap {
	name := v1alpha1.ConfigMapPrefix + cr.Name
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Spec.TargetNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, cr.GroupVersionKind()),
			},
		},
		Data: data,
	}
}
