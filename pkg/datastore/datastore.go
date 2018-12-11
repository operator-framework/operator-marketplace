package datastore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	Cache *memoryDatastore
)

func init() {
	// Cache is the global instance of datastore used by
	// the Marketplace operator.
	Cache = New()
}

// New returns an instance of memoryDatastore.
func New() *memoryDatastore {
	return &memoryDatastore{
		rows:    newOperatorSourceRowMap(),
		parser:  &manifestYAMLParser{},
		walker:  &walker{},
		bundler: &bundler{},
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
	// GetPackageIDs returns a comma separated list of operator ID(s). This list
	// includes operator(s) across all OperatorSource object(s). Each ID
	// returned can be used to retrieve the manifest associated with the
	// operator from underlying datastore.
	GetPackageIDs() string

	// GetPackageIDsByOperatorSource returns a comma separated list of operator
	// ID(s) associated with a given OperatorSource object.
	// Each ID returned can be used to retrieve the manifest associated with the
	// operator from underlying datastore.
	GetPackageIDsByOperatorSource(opsrcUID types.UID) string

	// Write saves the Spec associated with a given OperatorSource object and
	// the downloaded operator manifest(s) into datastore.
	//
	// opsrc represents the given OperatorSource object.
	// rawManifests is the list of raw operator manifest(s) associated with
	// a given operator source.
	Write(opsrc *v1alpha1.OperatorSource, rawManifests []*OperatorMetadata) error

	// RemoveOperatorSource removes everything associated with a given operator
	// source from the underlying datastore.
	//
	// opsrcUID is the unique identifier associated with a given operator source.
	RemoveOperatorSource(opsrcUID types.UID)

	// AddOperatorSource registers a new OperatorSource object with the
	// the underlying datastore.
	AddOperatorSource(opsrc *v1alpha1.OperatorSource)

	// GetOperatorSource returns the Spec of the OperatorSource object
	// associated with the UID specified in opsrcUID.
	//
	// datastore uses the UID of the given OperatorSource object to check if
	// a Spec already exists. If no Spec is found then the function
	// returns (nil, false).
	GetOperatorSource(opsrcUID types.UID) (key *OperatorSourceKey, ok bool)

	// OperatorSourceHasUpdate returns true if the operator source in remote
	// registry specified in metadata has update(s) that need to be pulled.
	//
	// The function returns true if the remote registry has any update(s). The
	// following event(s) indicate that a remote registry has been updated.
	//   - New repositories have been added to the remote registry associated
	//     with the operator source.
	//   - Existing repositories have been removed from the remote registry
	//     associated with the operator source.
	//   - A new release for an existing repository has been pushed to
	//     the registry.
	//
	// Right now we consider remote and local operator source to be same only
	// when the following conditions are true:
	//
	// - Number of repositories in both local and remote are exactly the same.
	// - Each repository in remote has a corresponding local repository with
	//   exactly the same release.
	//
	// The current implementation does not return update information specific
	// to each repository. The lack of granular (per repository) information
	// will force us to reload the entire namespace.
	OperatorSourceHasUpdate(opsrcUID types.UID, metadata []*RegistryMetadata) (bool, error)

	// GetAllOperatorSources returns a list of all OperatorSource objecs(s) that
	// datastore is aware of.
	GetAllOperatorSources() []*OperatorSourceKey
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	rows    *operatorSourceRowMap
	parser  ManifestYAMLParser
	walker  ManifestWalker
	bundler Bundler
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

func (ds *memoryDatastore) Write(opsrc *v1alpha1.OperatorSource, rawManifests []*OperatorMetadata) error {
	if opsrc == nil || rawManifests == nil {
		return errors.New("invalid argument")
	}

	metadata := map[string]*RegistryMetadata{}
	operators := map[string]*SingleOperatorManifest{}
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
			operators[operatorPackage.GetPackageID()] = packages[i]
		}

		// For each repository store the associated registry metadata.
		metadata[rawManifest.RegistryMetadata.Repository] = &rawManifest.RegistryMetadata
	}

	ds.rows.Add(opsrc, metadata, operators)

	return nil
}

func (ds *memoryDatastore) GetPackageIDs() string {
	keys := ds.rows.GetAllPackages()
	return strings.Join(keys, ",")
}

func (ds *memoryDatastore) GetPackageIDsByOperatorSource(opsrcUID types.UID) string {
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		return ""
	}

	packages := row.GetPackages()
	return strings.Join(packages, ",")
}

func (ds *memoryDatastore) AddOperatorSource(opsrc *v1alpha1.OperatorSource) {
	ds.rows.AddEmpty(opsrc)
}

func (ds *memoryDatastore) RemoveOperatorSource(uid types.UID) {
	ds.rows.Remove(uid)
}

func (ds *memoryDatastore) GetOperatorSource(opsrcUID types.UID) (*OperatorSourceKey, bool) {
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		return nil, false
	}

	return &row.OperatorSourceKey, true
}

func (ds *memoryDatastore) OperatorSourceHasUpdate(opsrcUID types.UID, metadata []*RegistryMetadata) (bool, error) {
	// TODO: Return fine grained information that describes repository that
	// was removed, added or has a new release.
	row, exists := ds.rows.GetRow(opsrcUID)
	if !exists {
		return false, fmt.Errorf("datastore has no record of the specified OperatorSource [%s]", opsrcUID)
	}

	if len(row.Metadata) != len(metadata) {
		return true, nil
	}

	for _, remote := range metadata {
		if remote.Release == "" {
			return false, fmt.Errorf("Release not specified for repository [%s]", remote.ID())
		}

		local, exists := row.Metadata[remote.Repository]
		if !exists {
			return true, nil
		}

		if local.Release != remote.Release {
			return true, nil
		}
	}

	return false, nil
}

func (ds *memoryDatastore) GetAllOperatorSources() []*OperatorSourceKey {
	return ds.rows.GetAllRows()
}

// validate ensures that no package is mentioned more than once in the list.
// It also ensures that the package(s) specified in the list has a corresponding
// manifest in the underlying datastore.
func (ds *memoryDatastore) validate(packageIDs []string) ([]*SingleOperatorManifest, error) {
	packages := make([]*SingleOperatorManifest, 0)
	packageMap := map[string]*SingleOperatorManifest{}

	// Get a list of all available package(s) across all operator source(s)
	// in a map.
	existing := ds.rows.GetAllPackagesMap()

	for _, packageID := range packageIDs {
		operatorPackage, exists := existing[packageID]
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
