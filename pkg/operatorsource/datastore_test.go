package operatorsource

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
)

func TestGetPackageIDs(t *testing.T) {
	expected := []string{"foo/1", "bar/2", "braz/3"}

	packages := []*appregistry.OperatorMetadata{
		&appregistry.OperatorMetadata{Namespace: "foo", Repository: "1"},
		&appregistry.OperatorMetadata{Namespace: "bar", Repository: "2"},
		&appregistry.OperatorMetadata{Namespace: "braz", Repository: "3"},
	}

	ds := newDatastore()
	err := ds.Write(packages)
	require.NoError(t, err)

	result := ds.GetPackageIDs()
	actual := strings.Split(result, ",")

	assert.EqualValues(t, expected, actual)
}
