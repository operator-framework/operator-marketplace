package catalogsourceconfig

import (
	"context"

	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Handler is the interface that wraps the Handle method
type Handler interface {
	Handle(ctx context.Context, event sdk.Event) error
}

type handler struct {
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
	return createCatalogSource(csc)
}

// createCatalogSource creates a new CatalogSource CR and all the resources it requires
func createCatalogSource(cr *v1alpha1.CatalogSourceConfig) error {
	// Create the ConfigMap that will be used by the CatalogSource
	catalogConfigMap := newConfigMap(cr)
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
			SourceType: "internal",
			ConfigMap:  configMapName,
		},
	}
}

// newConfigMap returns a new ConfigMap object
func newConfigMap(cr *v1alpha1.CatalogSourceConfig) *corev1.ConfigMap {
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
		// Dummy placeholder data
		Data: map[string]string{
			"clusterServiceVersions":    "csvs",
			"customResourceDefinitions": "crds",
			"packages":                  "packs",
		},
	}
}
