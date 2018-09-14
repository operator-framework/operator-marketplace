package stub

import (
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createCatalogSource creates a new CatalogSource CR and all the resources it requires
func createCatalogSource(cr *v1alpha1.CatalogSourceConfig) error {
	// Create the ConfigMap that will be used by the CatalogSource
	catalogConfigMap := newConfigMap(cr)
	logrus.Infof("Creating %s ConfigMap in %s namespace", catalogConfigMap.Name, cr.Spec.TargetNamespace)
	err := sdk.Create(catalogConfigMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Failed to create catalog source : %v", err)
		return err
	}

	catalogSource := newCatalogSource(cr, catalogConfigMap.Name)
	logrus.Infof("Creating %s CatalogSource in %s namespace", catalogSource.Name, cr.Spec.TargetNamespace)
	err = sdk.Create(catalogSource)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Failed to create catalog source : %v", err)
		return err
	}
	return nil
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
