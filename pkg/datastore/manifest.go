package datastore

// Manifest encapsulates operator manifest data.
type Manifest struct {
	// Publisher represents the publisher of this package.
	Publisher string `yaml:"publisher"`

	// Data reflects the content of the package manifest.
	Data Data `yaml:"data"`
}

// Data encapsulates the list of CRD(s), CV(s) and packages for an operator
// manifest.
type Data struct {
	// CRDs is the list of CRD(s) associated with a package.
	CRDs string `yaml:"customResourceDefinitions"`

	// CSVs is the list of CSV(s) associated with a package.
	CSVs string `yaml:"clusterServiceVersions"`

	// Packages is the list of channel(s) associated with a package.
	Packages string `yaml:"packages"`
}
