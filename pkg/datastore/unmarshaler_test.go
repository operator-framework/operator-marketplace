package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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

	crdWant = []CustomResourceDefinition{
		CustomResourceDefinition{
			v1beta1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1beta1",
					Kind:       "CustomResourceDefinition",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "jbossapps-1.jboss.middleware.redhat.com",
				},
			},
		},
		CustomResourceDefinition{
			v1beta1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1beta1",
					Kind:       "CustomResourceDefinition",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "jbossapps-2.jboss.middleware.redhat.com",
				},
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

	// Do not use tabs for indentation as yaml forbids tabs http://yaml.org/faq.html
	rawCSVs = `
data:
  clusterServiceVersions: |-
    - apiVersion: app.coreos.com/v1alpha1
      kind: ClusterServiceVersion-v1
      metadata:
        name: jbossapp-operator.v0.1.0
      spec:
        replaces: foo
        customresourcedefinitions:
          owned:
          - name: bar
            version: v1
            kind: JBossApp
          required:
          - name: baz
            version: v1
            kind: BazApp
`

	csvWant = &ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "app.coreos.com/v1alpha1",
			Kind:       "ClusterServiceVersion-v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "jbossapp-operator.v0.1.0",
		},
		Spec: []byte{},
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
func TestUnmarshal_ManifestHasCSV_SuccessfullyParsed(t *testing.T) {
	u := manifestYAMLParser{}
	dataGot, err := u.Unmarshal([]byte(rawCSVs))

	assert.NoError(t, err)
	assert.NotNil(t, dataGot)

	assert.Equal(t, 1, len(dataGot.ClusterServiceVersions))
	csvGot := dataGot.ClusterServiceVersions[0]

	assert.Equal(t, csvWant.TypeMeta, csvGot.TypeMeta)
	assert.Equal(t, csvWant.ObjectMeta, csvGot.ObjectMeta)
	assert.NotEmpty(t, csvGot.Spec)
}
