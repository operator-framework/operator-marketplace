package datastore

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPackageIDs(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	expected := []string{"foo/1", "bar/2", "braz/3"}

	packages := []*OperatorMetadata{
		helperNewOperatorMetadata("foo", "1"),
		helperNewOperatorMetadata("bar", "2"),
		helperNewOperatorMetadata("braz", "3"),
	}

	parser := NewMockManifestYAMLParser(controller)

	ds := &memoryDatastore{
		manifests: map[string]*OperatorManifest{},
		parser:    parser,
	}

	// We expect Unmarshal function to be invoked for each package.
	parser.EXPECT().Unmarshal(gomock.Any()).Return(&StructuredOperatorManifestData{}, nil).Times(len(packages))

	err := ds.Write(packages)
	require.NoError(t, err)

	result := ds.GetPackageIDs()
	actual := strings.Split(result, ",")

	assert.ElementsMatch(t, expected, actual)
}

func helperNewOperatorMetadata(namespace, repository string) *OperatorMetadata {
	return &OperatorMetadata{
		RegistryMetadata: RegistryMetadata{
			Namespace:  namespace,
			Repository: repository,
		},
	}
}
