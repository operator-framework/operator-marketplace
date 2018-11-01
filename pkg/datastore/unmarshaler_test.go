package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	// Do not use tabs for indentation as yaml forbids tabs http://yaml.org/faq.html
	data := `
publisher: redhat
data:
  customResourceDefinitions: "my crds"
  clusterServiceVersions: "my csvs"
  packages: "my packages"
`

	u := blobUnmarshalerImpl{}
	manifest, err := u.Unmarshal([]byte(data))

	require.NoError(t, err)

	assert.Equal(t, "redhat", manifest.Publisher)
	assert.Equal(t, "my crds", manifest.Data.CRDs)
	assert.Equal(t, "my csvs", manifest.Data.CSVs)
	assert.Equal(t, "my packages", manifest.Data.Packages)
}
