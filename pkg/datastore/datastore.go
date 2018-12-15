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
		rows:    operatorSourceRowMap{},
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
	GetOperatorSource(opsrcUID types.UID) (spec *v1alpha1.OperatorSourceSpec, ok bool)
}

// memoryDatastore is an in-memory implementation of operator manifest datastore.
// TODO: In future, it will be replaced by an indexable persistent datastore.
type memoryDatastore struct {
	rows    operatorSourceRowMap
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
	}

	row := &operatorSourceRow{
		Spec:      &opsrc.Spec,
		Operators: operators,
	}

	ds.rows[opsrc.GetUID()] = row

	return nil
}

func (ds *memoryDatastore) GetPackageIDs() string {
	keys := ds.rows.GetAllPackages()
	return strings.Join(keys, ",")
}

func (ds *memoryDatastore) GetPackageIDsByOperatorSource(opsrcUID types.UID) string {
	row, exists := ds.rows[opsrcUID]
	if !exists {
		return ""
	}

	packages := row.GetPackages()
	return strings.Join(packages, ",")
}

func (ds *memoryDatastore) AddOperatorSource(opsrc *v1alpha1.OperatorSource) {
	ds.rows[opsrc.GetUID()] = &operatorSourceRow{
		Spec:      &opsrc.Spec,
		Operators: map[string]*SingleOperatorManifest{},
	}
}

func (ds *memoryDatastore) RemoveOperatorSource(uid types.UID) {
	delete(ds.rows, uid)
}

func (ds *memoryDatastore) GetOperatorSource(opsrcUID types.UID) (opsrc *v1alpha1.OperatorSourceSpec, ok bool) {
	row, exists := ds.rows[opsrcUID]
	if !exists {
		return nil, false
	}

	return row.Spec, true
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
