package datastore

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// operatorSourceRow is what gets stored in datastore after an OperatorSource CR
// is reconciled.
//
// Every reconciled OperatorSource object has a corresponding operatorSourceRow
// in datastore. The Writer interface accepts a raw operator manifest and
// marshals it into this type before writing it to the underlying storage.
type operatorSourceRow struct {
	// We store the Spec associated with a given OperatorSource object. This is
	// so that we can determine whether Spec for an existing operator source
	// has been updated.
	//
	// We compare the Spec of the received OperatorSource object to the one
	// in datastore.
	Spec *v1alpha1.OperatorSourceSpec

	// Operators is the collection of all single-operator manifest(s) associated
	// with the underlying operator source.
	// The package name is used to uniquely identify the operator manifest(s).
	Operators map[string]*SingleOperatorManifest

	// Metadata is the metadata associated with each repository under the given
	// namespace.
	Metadata map[string]*RegistryMetadata
}

// GetPackages returns the list of available package(s) associated with an
// operator source.
// An empty list is returned if there are no package(s).
func (r *operatorSourceRow) GetPackages() []string {
	packages := make([]string, 0)
	for packageID, _ := range r.Operators {
		packages = append(packages, packageID)
	}

	return packages
}

// operatorSourceRowMap is a map that holds a collection of operator source(s)
// represented by operatorSourceRow.
// The UID of an OperatorSource object is used as the key to uniquely identify
// an operator source.
type operatorSourceRowMap map[types.UID]*operatorSourceRow

// GetAllPackages return a list of all package(s) available across all
// operator source(s).
func (m operatorSourceRowMap) GetAllPackages() []string {
	packages := make([]string, 0)
	for _, row := range m {
		packages = append(packages, row.GetPackages()...)
	}

	return packages
}

// GetAllPackagesMap returns a collection of all available package(s) across all
// operator sources in a map. Package name is used as the key to this map.
func (m operatorSourceRowMap) GetAllPackagesMap() map[string]*SingleOperatorManifest {
	packages := map[string]*SingleOperatorManifest{}
	for _, row := range m {
		for packageID, manifest := range row.Operators {
			packages[packageID] = manifest
		}
	}

	return packages
}
