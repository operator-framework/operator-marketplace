package operatorsource

import (
	"errors"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
)

var (
	ErrManifestNotFound = errors.New("manifest not found")
)

type DatastoreReader interface {
	// Read returns the associated operator manifest given a package ID
	Read(packageID string) (*appregistry.OperatorMetadata, error)
}

func newDatastore() *hashmapDatastore {
	return &hashmapDatastore{
		list: map[string]*appregistry.OperatorMetadata{},
	}
}

type hashmapDatastore struct {
	list map[string]*appregistry.OperatorMetadata
}

func (ds *hashmapDatastore) Read(packageID string) (*appregistry.OperatorMetadata, error) {
	manifest, exists := ds.list[packageID]
	if !exists {
		return nil, ErrManifestNotFound
	}

	return manifest, nil
}

func (ds *hashmapDatastore) Write(packages []*appregistry.OperatorMetadata) error {
	for _, pkg := range packages {
		ds.list[pkg.ID()] = pkg
	}

	return nil
}

// GetPackageIDs returns a comma separated list of IDs of all packages in datastore
func (ds *hashmapDatastore) GetPackageIDs() string {
	keys := make([]string, 0, len(ds.list))
	for key := range ds.list {
		keys = append(keys, key)
	}

	return strings.Join(keys, ",")
}
