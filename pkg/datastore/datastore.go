package datastore

import (
	"errors"
	"strings"
)

var (
	ErrManifestNotFound = errors.New("manifest not found")
)

// New returns a new instance of datastore for Operator Manifest(s)
func New() *memoryDatastore {
	return &memoryDatastore{
		manifests: map[string]*OperatorMetadata{},
	}
}

// Reader is the interface that wraps the Read method
//
// Read returns the associated operator manifest given a package ID
type Reader interface {
	Read(packageID string) (*OperatorMetadata, error)
}

// Writer is an interface that is used to manage the underlying datastore
// for operator manifest.
type Writer interface {
	// GetPackageIDs returns a comma separated list of IDs of
	// all package(s) from underlying datastore.
	GetPackageIDs() string

	// Write stores the list of operator manifest(s) into datastore
	Write(packages []*OperatorMetadata) error
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	manifests map[string]*OperatorMetadata
}

func (ds *memoryDatastore) Read(packageID string) (*OperatorMetadata, error) {
	manifest, exists := ds.manifests[packageID]
	if !exists {
		return nil, ErrManifestNotFound
	}

	return manifest, nil
}

func (ds *memoryDatastore) Write(packages []*OperatorMetadata) error {
	for _, pkg := range packages {
		ds.manifests[pkg.ID()] = pkg
	}

	return nil
}

func (ds *memoryDatastore) GetPackageIDs() string {
	keys := make([]string, 0, len(ds.manifests))
	for key := range ds.manifests {
		keys = append(keys, key)
	}

	return strings.Join(keys, ",")
}
