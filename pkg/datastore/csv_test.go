package datastore

import (
	"encoding/json"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// Do not use tabs for indentation as yaml forbids
	// tabs http://yaml.org/faq.html.
	rawCSVWithReplacesAndCustomResourceDefinitions = `
apiVersion: app.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: myapp-operator.v0.2.0
spec:
  replaces: myapp-operator.v0.1.0
  customresourcedefinitions:
    owned:
    - name: foo.redhat.com
      version: v1
      kind: FooApp
    - name: bar.redhat.com
      version: v1
      kind: BarApp
    required:
    - name: baz.redhat.com
      version: v1
      kind: BazApp
`

	rawCSVWithNoReplaces = `
apiVersion: app.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: myapp-operator.v0.2.0
spec:
  displayName: jboss
`

	rawCSVWithNoCustomResourceDefinitions = `
apiVersion: app.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: myapp-operator.v0.2.0
spec:
  customresourcedefinitions:
`
)

// We expect GetReplaces to return the name of the older ClusterServiceVersion
// object that this ClusterServiceVersion object replaces.
func TestGetReplaces(t *testing.T) {
	replacesWant := "myapp-operator.v0.1.0"

	jsonRaw, err := yaml.YAMLToJSON([]byte(rawCSVWithReplacesAndCustomResourceDefinitions))
	require.NoError(t, err)

	var csv ClusterServiceVersion
	err = json.Unmarshal(jsonRaw, &csv)
	require.NoError(t, err)

	replacesGot, errGot := csv.GetReplaces()

	assert.NoError(t, errGot)
	assert.Equal(t, replacesWant, replacesGot)
}

// When a ClusterServiceVersion object does not have a `replaces` attribute
// defined, we expect GetReplaces to return an empty string.
func TestGetReplaces_NotSpecified_EmptyStringExpected(t *testing.T) {
	replacesWant := ""

	jsonRaw, err := yaml.YAMLToJSON([]byte(rawCSVWithNoReplaces))
	require.NoError(t, err)

	var csv ClusterServiceVersion
	err = json.Unmarshal(jsonRaw, &csv)
	require.NoError(t, err)

	replacesGot, errGot := csv.GetReplaces()

	assert.NoError(t, errGot)
	assert.Equal(t, replacesWant, replacesGot)
}

// We expect GetCustomResourceDefintions to return the list of owned and
// required CustomResourceDefinition object(s) specified inside
// the 'customresourcedefinitions' section of a CSV spec.
func TestGetCustomResourceDefintions(t *testing.T) {
	ownedWant := []*CRDKey{
		&CRDKey{
			Kind: "FooApp", Version: "v1", Name: "foo.redhat.com",
		},
		&CRDKey{
			Kind: "BarApp", Version: "v1", Name: "bar.redhat.com",
		},
	}

	requiredWant := []*CRDKey{
		&CRDKey{
			Kind: "BazApp", Version: "v1", Name: "baz.redhat.com",
		},
	}

	jsonRaw, err := yaml.YAMLToJSON([]byte(rawCSVWithReplacesAndCustomResourceDefinitions))
	require.NoError(t, err)

	var csv ClusterServiceVersion
	err = json.Unmarshal(jsonRaw, &csv)
	require.NoError(t, err)

	ownedGot, requiredGot, errGot := csv.GetCustomResourceDefintions()

	assert.NoError(t, errGot)
	assert.ElementsMatch(t, ownedWant, ownedGot)
	assert.ElementsMatch(t, requiredWant, requiredGot)
}

// When no CRD is specified inside the 'customresourcedefinitions' section of a
// CSV spec, we expect GetCustomResourceDefintions to return empty
// list for both owned and required.
func TestGetCustomResourceDefintions_NoCRDSpecified_EmptyListExpected(t *testing.T) {
	jsonRaw, err := yaml.YAMLToJSON([]byte(rawCSVWithNoCustomResourceDefinitions))
	require.NoError(t, err)

	var csv ClusterServiceVersion
	err = json.Unmarshal(jsonRaw, &csv)
	require.NoError(t, err)

	ownedGot, requiredGot, errGot := csv.GetCustomResourceDefintions()

	assert.NoError(t, errGot)
	assert.Nil(t, ownedGot)
	assert.Nil(t, requiredGot)
}
