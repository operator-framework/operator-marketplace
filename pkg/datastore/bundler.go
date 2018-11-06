package datastore

import (
	"fmt"
)

// Bundler is the interface that wraps the Bundle method.
//
// Bundle accepts a set of single-operator manifest(s) and bundles them into a
// multi-operator manifest that contains all the package(s), CRD(s) and CSV(s)
// specified.
//
// An error is thrown if an operator appears more than once in the list.
type Bundler interface {
	Bundle(manifests []*SingleOperatorManifest) (*MultiOperatorManifest, error)
}

// bundler implements the Bundler interface.
type bundler struct{}

func (b *bundler) Bundle(manifests []*SingleOperatorManifest) (*MultiOperatorManifest, error) {
	packageMap := PackageManifestMap{}
	crdMap := CustomResourceDefinitionMap{}
	csvMap := ClusterServiceVersionMap{}

	for i, manifest := range manifests {
		operatorID := manifest.GetPackageID()
		if _, exists := packageMap[operatorID]; exists {
			return nil, fmt.Errorf("operator [%s] appears more than once", operatorID)
		}

		packageMap[operatorID] = manifests[i].Package

		for _, csv := range manifest.ClusterServiceVersions {
			csvMap[csv.Name] = csv
		}

		for _, crd := range manifest.CustomResourceDefinitions {
			crdMap[crd.Key()] = crd
		}
	}

	return &MultiOperatorManifest{
		CustomResourceDefinitions: crdMap.Values(),
		ClusterServiceVersions:    csvMap.Values(),
		Packages:                  packageMap.Values(),
	}, nil
}

// PackageManifestMap is a map of PackageManifest object. It uses the package
// name as the key.
type PackageManifestMap map[string]*PackageManifest

// Values returns a list of all PackageManifest object(s) stored in the map.
func (m PackageManifestMap) Values() []*PackageManifest {
	if len(m) == 0 {
		return nil
	}

	values := make([]*PackageManifest, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}

	return values
}
