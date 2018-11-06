package datastore

// Manifest encapsulates operator manifest data.
type Manifest struct {
	// Publisher represents the publisher of this package.
	Publisher string `yaml:"publisher"`

	// Data reflects the content of the package manifest.
	Data OperatorManifestData `yaml:"data"`
}

// OperatorManifestData encapsulates the list of CRD(s), CSV(s) and package(s)
// associated with an operator manifest.
type OperatorManifestData struct {
	// CustomResourceDefinitions is the set of custom resource definition(s)
	// associated with this package manifest.
	CustomResourceDefinitions string `yaml:"customResourceDefinitions"`

	// ClusterServiceVersions is the set of cluster service version(s)
	// associated with this package manifest.
	ClusterServiceVersions string `yaml:"clusterServiceVersions"`

	// Packages is the set of package(s) associated with this operator manifest.
	Packages string `yaml:"packages"`
}
