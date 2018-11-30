package datastore

import (
	"fmt"
	"strings"
)

var (
	Cache *memoryDatastore
)

func init() {
	Cache = newDataStore()
}

func newDataStore() *memoryDatastore {
	return &memoryDatastore{
		rows:     map[string]*operatorSourceRow{},
		packages: map[string]*SingleOperatorManifest{},
		parser:   &manifestYAMLParser{},
		walker:   &walker{},
		bundler:  &bundler{},
	}
}

// Reader is the interface that wraps the Read method.
//
// Read prepares an operator manifest for a given set of package(s)
// uniquely represeneted by the package ID(s) specified in packageIDs. It
// returns an instance of RawOperatorManifestData that has the required set of
// CRD(s), CSV(s) and package(s).
//
// The manifest returned can be used by the caller to create a ConfigMap object
// for a catalog source (CatalogSource) in OLM.
type Reader interface {
	Read(packageIDs []string) (marshaled *RawOperatorManifestData, err error)
}

// Writer is an interface that is used to manage the underlying datastore
// for operator manifest.
type Writer interface {
	// GetPackageIDs returns a comma separated list of operator ID(s). Each ID
	// returned can be used to retrieve the manifest associated with the
	// operator from underlying datastore.
	GetPackageIDs() string

	// Write stores the list of operator manifest(s) into datastore.
	Write(rawManifests []*OperatorMetadata) error
}

// operatorSourceRow is what gets stored in datastore after an OperatorSource CR
// is reconciled.
//
// Every reconciled OperatorSource object has a corresponding operatorSourceRow
// in datastore. The Writer interface accepts a raw operator manifest and
// marshals it into this type before writing it to the underlying storage.
type operatorSourceRow struct {
	// RegistryMetadata uniquely identifies a given operator manifest and
	// points to its source in remote registry.
	RegistryMetadata RegistryMetadata

	// Data is a structured representation of the given operator manifest.
	Data StructuredOperatorManifestData
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	rows     map[string]*operatorSourceRow
	packages map[string]*SingleOperatorManifest
	parser   ManifestYAMLParser
	walker   ManifestWalker
	bundler  Bundler
}

func (ds *memoryDatastore) Read(packageIDs []string) (*RawOperatorManifestData, error) {
	singleOperatorManifests, err := ds.validate(packageIDs)
	if err != nil {
		return nil, err
	}

	multiOperatorManifest, err := ds.bundler.Bundle(singleOperatorManifests)
	if err != nil {
		return nil, fmt.Errorf("error while bundling package(s) into  manifest - %s", err)
	}

	return ds.parser.Marshal(multiOperatorManifest)
}

func (ds *memoryDatastore) Write(rawManifests []*OperatorMetadata) error {
	for _, rawManifest := range rawManifests {
		data, err := ds.parser.Unmarshal(rawManifest.RawYAML)
		if err != nil {
			return err
		}

		decomposer := newDecomposer()
		if err := ds.walker.Walk(data, decomposer); err != nil {
			return err
		}

		packages := decomposer.Packages()
		for i, operatorPackage := range packages {
			ds.packages[operatorPackage.GetPackageID()] = packages[i]
		}

		row := &operatorSourceRow{
			RegistryMetadata: rawManifest.RegistryMetadata,
			Data:             *data,
		}

		ds.rows[rawManifest.ID()] = row
	}

	return nil
}

func (ds *memoryDatastore) GetPackageIDs() string {
	keys := make([]string, 0, len(ds.packages))
	for key := range ds.packages {
		keys = append(keys, key)
	}

	return strings.Join(keys, ",")
}

// validate ensures that no package is mentioned more than once in the list.
// It also ensures that the package(s) specified in the list has a corresponding
// manifest in the underlying datastore.
func (ds *memoryDatastore) validate(packageIDs []string) ([]*SingleOperatorManifest, error) {
	packages := make([]*SingleOperatorManifest, 0)
	packageMap := map[string]*SingleOperatorManifest{}

	for _, packageID := range packageIDs {
		operatorPackage, exists := ds.packages[packageID]
		if !exists {
			return nil, fmt.Errorf("package [%s] not found", packageID)
		}

		if _, exists := packageMap[packageID]; exists {
			return nil, fmt.Errorf("package [%s] has been specified more than once", packageID)
		}

		packageMap[packageID] = operatorPackage
		packages = append(packages, operatorPackage)
	}

	return packages, nil
}
