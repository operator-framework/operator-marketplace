package datastore

import (
	"sync"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

func newOperatorSourceRowMap() *operatorSourceRowMap {
	return &operatorSourceRowMap{
		Sources: map[types.UID]*operatorSourceRow{},
	}
}

// OperatorSourceKey is what datastore uses to relate to an OperatorSource
// object.
type OperatorSourceKey struct {
	// UID is the UID associated with the OperatorSource object.
	UID types.UID

	// Name is the namespaced name of the given OperatorSource object that
	// uniquely identifies it and can be used to query the k8s API server.
	Name types.NamespacedName

	// We store the Spec associated with a given OperatorSource object. This is
	// so that we can determine whether Spec for an existing operator source
	// has been updated.
	//
	// We compare the Spec of the received OperatorSource object to the one
	// in datastore.
	Spec *v1alpha1.OperatorSourceSpec
}

// operatorSourceRow is what gets stored in datastore after an OperatorSource CR
// is reconciled.
//
// Every reconciled OperatorSource object has a corresponding operatorSourceRow
// in datastore. The Writer interface accepts a raw operator manifest and
// marshals it into this type before writing it to the underlying storage.
type operatorSourceRow struct {
	OperatorSourceKey

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
type operatorSourceRowMap struct {
	lock sync.RWMutex

	// Sources is a map of operatorSourceRow where UID of the given
	// OperatorSource object is used as key.
	Sources map[types.UID]*operatorSourceRow
}

// AddEmpty adds a new operator source to the map with an empty set of
// registry metadata and operator manifest(s).
func (m *operatorSourceRowMap) AddEmpty(opsrc *v1alpha1.OperatorSource) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.add(opsrc, map[string]*RegistryMetadata{}, map[string]*SingleOperatorManifest{})
}

// Add adds a new operator source to the map with an the specified set of
// registry metadata and operator manifest(s).
func (m *operatorSourceRowMap) Add(opsrc *v1alpha1.OperatorSource, metadata map[string]*RegistryMetadata, operators map[string]*SingleOperatorManifest) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.add(opsrc, metadata, operators)
}

func (m *operatorSourceRowMap) add(opsrc *v1alpha1.OperatorSource, metadata map[string]*RegistryMetadata, operators map[string]*SingleOperatorManifest) {
	m.Sources[opsrc.GetUID()] = &operatorSourceRow{
		OperatorSourceKey: OperatorSourceKey{
			UID: opsrc.GetUID(),
			Name: types.NamespacedName{
				Namespace: opsrc.GetNamespace(),
				Name:      opsrc.GetName(),
			},
			Spec: &opsrc.Spec,
		},
		Operators: operators,
		Metadata:  metadata,
	}
}

func (m *operatorSourceRowMap) Remove(opsrcUID types.UID) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.Sources, opsrcUID)
}

func (m *operatorSourceRowMap) GetRow(opsrcUID types.UID) (*operatorSourceRow, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	row, ok := m.Sources[opsrcUID]
	return row, ok
}

// GetAllPackages return a list of all package(s) available across all
// operator source(s).
func (m *operatorSourceRowMap) GetAllPackages() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	packages := make([]string, 0)
	for _, row := range m.Sources {
		packages = append(packages, row.GetPackages()...)
	}

	return packages
}

// GetAllPackagesMap returns a collection of all available package(s) across all
// operator sources in a map. Package name is used as the key to this map.
func (m operatorSourceRowMap) GetAllPackagesMap() map[string]*SingleOperatorManifest {
	m.lock.RLock()
	defer m.lock.RUnlock()

	packages := map[string]*SingleOperatorManifest{}
	for _, row := range m.Sources {
		for packageID, manifest := range row.Operators {
			packages[packageID] = manifest
		}
	}

	return packages
}
