package datastore

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In this test we make sure that datastore can successfully process the
// rh-operators.yaml manifest file.
func TestWrite(t *testing.T) {
	// The following packages are defined in rh-operators.yaml and we expect
	// datastore to return this list after it processes the manifest.
	packagesWant := []string{
		"amq-streams",
		"etcd",
		"federationv2",
		"prometheus",
		"service-catalog",
	}

	metadata := []*OperatorMetadata{
		helperLoadFromFile(t, "rh-operators.yaml"),
	}

	ds := newDataStore()
	err := ds.Write(metadata)
	require.NoError(t, err)

	list := ds.GetPackageIDs()
	packagesGot := strings.Split(list, ",")
	assert.ElementsMatch(t, packagesWant, packagesGot)
}

// Given a list of package ID(s), we expect datastore to return the correspnding
// manifest YAML that is complete and includes all the package(s) CRD(s)
// and CSV(s) that are required.
func TestGetPackageIDs(t *testing.T) {
	metadata := []*OperatorMetadata{
		helperLoadFromFile(t, "rh-operators.yaml"),
	}

	ds := newDataStore()
	err := ds.Write(metadata)
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

func helperLoadFromFile(t *testing.T, filename string) *OperatorMetadata {
	path := filepath.Join("testdata", filename)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return &OperatorMetadata{
		RegistryMetadata: RegistryMetadata{
			Namespace:  "operators",
			Repository: "redhat",
		},
		RawYAML: bytes,
	}
}
