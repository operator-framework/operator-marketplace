package datastore

import (
	"fmt"
	"strings"
)

// New returns a new instance of datastore for Operator Manifest(s).
func New() *memoryDatastore {
	return &memoryDatastore{
		manifests:   map[string]*OperatorManifest{},
		unmarshaler: &blobUnmarshalerImpl{},
	}
}

// Reader is the interface that wraps the Read method.
//
// Read prepares an operator manifest for a given set of package(s)
// uniquely represeneted by the package ID(s) specified in packageIDs. It
// returns an instance of OperatorManifestData that has the required set of
// CRD(s), CSV(s) and package(s).
//
// The manifest returned can be used by the caller to create a ConfigMap object
// for a catalog source (CatalogSource) in OLM.
type Reader interface {
	Read(packageIDs []string) (marshaled *OperatorManifestData, err error)
}

// Writer is an interface that is used to manage the underlying datastore
// for operator manifest.
type Writer interface {
	// GetPackageIDs returns a comma separated list of IDs of
	// all package(s) from underlying datastore.
	GetPackageIDs() string

	// Write stores the list of operator manifest(s) into datastore.s
	Write(packages []*OperatorMetadata) error
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	manifests   map[string]*OperatorManifest
	unmarshaler blobUnmarshaler
}

func (ds *memoryDatastore) Read(packageIDs []string) (*OperatorManifestData, error) {
	data := StructuredOperatorManifestData{}
	for _, packageID := range packageIDs {
		manifest, exists := ds.manifests[packageID]
		if !exists {
			return nil, fmt.Errorf("package [%s] not found", packageID)
		}

		d := manifest.Data

		data.CustomResourceDefinitions = append(data.CustomResourceDefinitions, d.CustomResourceDefinitions...)
		data.ClusterServiceVersions = append(data.ClusterServiceVersions, d.ClusterServiceVersions...)
		data.Packages = append(data.Packages, d.Packages...)
	}

	return ds.unmarshaler.Marshal(&data)
}

func (ds *memoryDatastore) Write(packages []*OperatorMetadata) error {
	for _, pkg := range packages {
		data, err := ds.unmarshaler.Unmarshal(pkg.RawYAML)
		if err != nil {
			return err
		}

		manifest := &OperatorManifest{
			RegistryMetadata: pkg.RegistryMetadata,
			Data:             *data,
		}

		ds.manifests[pkg.ID()] = manifest
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
