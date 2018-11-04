package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Do not use tabs for indentation as yaml forbids tabs http://yaml.org/faq.html
	rawCRDs = `
data:
  customResourceDefinitions: |-      
    - apiVersion: apiextensions.k8s.io/v1beta1
      kind: CustomResourceDefinition
      metadata:
        name: jbossapps-1.jboss.middleware.redhat.com
    - apiVersion: apiextensions.k8s.io/v1beta1
      kind: CustomResourceDefinition
      metadata:
        name: jbossapps-2.jboss.middleware.redhat.com
`

	crdWant = []OLMObject{
		OLMObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiextensions.k8s.io/v1beta1",
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "jbossapps-1.jboss.middleware.redhat.com",
			},
		},
		OLMObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiextensions.k8s.io/v1beta1",
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "jbossapps-2.jboss.middleware.redhat.com",
			},
		},
	}

	// Do not use tabs for indentation as yaml forbids tabs http://yaml.org/faq.html
	rawPackages = `
data:
  packages: |-
    - #! package-manifest: ./deploy/chart/catalog_resources/rh-operators/etcdoperator.v0.9.2.clusterserviceversion.yaml
      packageName: etcd
      channels:
        - name: alpha
          currentCSV: etcdoperator.v0.9.2
        - name: nightly
          currentCSV: etcdoperator.v0.9.2-nightly
      defaultChannel: alpha
`

	packagesWant = []PackageManifest{
		PackageManifest{
			PackageName:        "etcd",
			DefaultChannelName: "alpha",
			Channels: []PackageChannel{
				PackageChannel{Name: "alpha", CurrentCSVName: "etcdoperator.v0.9.2"},
				PackageChannel{Name: "nightly", CurrentCSVName: "etcdoperator.v0.9.2-nightly"},
			},
		},
	}

	csvWant = []OLMObject{
		OLMObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.coreos.com/v1alpha1",
				Kind:       "ClusterServiceVersion-v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "jbossapp-operator.v0.1.0",
			},
		},
	}
)

// For a given set of CRD(s) in a raw operator manifest YAML document, we
// expect it to get unmarshaled into corresponding structured type.
func TestUnmarshal_ManifestHasCRD_SuccessfullyParsed(t *testing.T) {

	u := manifestYAMLParser{}
	dataGot, err := u.Unmarshal([]byte(rawCRDs))

	assert.NoError(t, err)
	assert.NotNil(t, dataGot)

	crdGot := dataGot.CustomResourceDefinitions

	assert.ElementsMatch(t, crdWant, crdGot)
}

// For a given set of package(s) in a raw operator manifest YAML document, we
// expect it to get unmarshaled into corresponding structured type.
func TestUnmarshal_ManifestHasPackages_SuccessfullyParsed(t *testing.T) {
	u := manifestYAMLParser{}
	dataGot, err := u.Unmarshal([]byte(rawPackages))

	assert.NoError(t, err)
	assert.NotNil(t, dataGot)

	packagesGot := dataGot.Packages

	assert.ElementsMatch(t, packagesWant, packagesGot)
}

// Given a structured representation of an operator manifest we should be able
// to convert it to raw YAML representation so that a ConfigMap object for
// catalog source (CatalogSource) can be created successfully.
func TestMarshal(t *testing.T) {
	marshaled := StructuredOperatorManifestData{
		CustomResourceDefinitions: crdWant,
		ClusterServiceVersions:    csvWant,
		Packages:                  packagesWant,
	}

	u := manifestYAMLParser{}
	rawGot, err := u.Marshal(&marshaled)

	assert.NoError(t, err)
	assert.NotNil(t, rawGot)
	assert.NotEmpty(t, rawGot.Packages)
	assert.NotEmpty(t, rawGot.CustomResourceDefinitions)
	assert.NotEmpty(t, rawGot.ClusterServiceVersions)
}
