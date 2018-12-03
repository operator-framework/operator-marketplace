package datastore_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// In this test we make sure that datastore can successfully process the
// rh-operators.yaml manifest file.
func TestWriteWithRedHatOperatorsYAML(t *testing.T) {
	// The following packages are defined in rh-operators.yaml and we expect
	// datastore to return this list after it processes the manifest.
	packagesWant := []string{
		"amq-streams",
		"etcd",
		"federationv2",
		"prometheus",
		"service-catalog",
	}

	opsrc := &v1alpha1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("123456"),
		},
	}

	metadata := []*datastore.OperatorMetadata{
		helperLoadFromFile(t, "rh-operators.yaml"),
	}

	ds := datastore.New()
	err := ds.Write(opsrc, metadata)
	require.NoError(t, err)

	list := ds.GetPackageIDs()
	packagesGot := strings.Split(list, ",")
	assert.ElementsMatch(t, packagesWant, packagesGot)
}

// Given a list of package ID(s), we expect datastore to return the correspnding
// manifest YAML that is complete and includes all the package(s) CRD(s)
// and CSV(s) that are required.
func TestGetPackageIDsWithRedHatOperatorsYAML(t *testing.T) {
	opsrc := &v1alpha1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("123456"),
		},
	}

	metadata := []*datastore.OperatorMetadata{
		helperLoadFromFile(t, "rh-operators.yaml"),
	}

	ds := datastore.New()
	err := ds.Write(opsrc, metadata)
	require.NoError(t, err)

	dataGot, errGot := ds.Read([]string{"etcd", "prometheus"})
	assert.NoError(t, errGot)

	_, packageParseErrGot := yaml.YAMLToJSON([]byte(dataGot.Packages))
	assert.NoError(t, packageParseErrGot)

	_, crdParseErrGot := yaml.YAMLToJSON([]byte(dataGot.CustomResourceDefinitions))
	assert.NoError(t, crdParseErrGot)

	_, csvParseErrGot := yaml.YAMLToJSON([]byte(dataGot.ClusterServiceVersions))
	assert.NoError(t, csvParseErrGot)
}

func TestGetPackageIDsWithMultipleOperatorSources(t *testing.T) {
	opsrc1 := &v1alpha1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("123456"),
		},
	}
	opsrc2 := &v1alpha1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("987654"),
		},
	}

	// Both 'source-1.yaml' and 'source-2.yaml' have the following packages
	// combined.
	packagesWant := []string{
		"foo-source-1",
		"foo-source-2",
		"bar-source-1",
		"bar-source-2",
		"baz-source-1",
		"baz-source-2",
	}

	metadata1 := helperLoadFromFile(t, "source-1.yaml")
	metadata2 := helperLoadFromFile(t, "source-2.yaml")

	ds := datastore.New()

	errGot1 := ds.Write(opsrc1, []*datastore.OperatorMetadata{metadata1})
	require.NoError(t, errGot1)

	errGot2 := ds.Write(opsrc2, []*datastore.OperatorMetadata{metadata2})
	require.NoError(t, errGot2)

	value := ds.GetPackageIDs()
	packagesGot := strings.Split(value, ",")

	assert.ElementsMatch(t, packagesWant, packagesGot)
}

func helperLoadFromFile(t *testing.T, filename string) *datastore.OperatorMetadata {
	path := filepath.Join("testdata", filename)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return &datastore.OperatorMetadata{
		RegistryMetadata: datastore.RegistryMetadata{
			Namespace:  "operators",
			Repository: "redhat",
		},
		RawYAML: bytes,
	}
}
