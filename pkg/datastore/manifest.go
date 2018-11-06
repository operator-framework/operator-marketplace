package datastore

// SingleOperatorManifest is a structured representation of a single operator
// manifest.
type SingleOperatorManifest struct {
	// Package holds information about the associated with this operator.
	Package *PackageManifest

	// CustomResourceDefinitions is the list of CRD(s) that this operator owns.
	CustomResourceDefinitions []*CustomResourceDefinition

	// ClusterServiceVersions is the list of CSV(s) that this operator manages.
	ClusterServiceVersions []*ClusterServiceVersion
}

// GetPackageID returns the name that uniquely identifies this operator package.
func (p *SingleOperatorManifest) GetPackageID() string {
	return p.Package.PackageName
}

// MultiOperatorManifest is a structured representation of a manifest that has
// multiple operator(s).
type MultiOperatorManifest struct {
	// Packages is the list of packages each of which uniquely describes a given
	// operator in the manifest.
	Packages []*PackageManifest

	// CustomResourceDefinitions is the list of CRD(s) that managed by the
	// operator(s) specified in the manifest.
	CustomResourceDefinitions []*CustomResourceDefinition

	// ClusterServiceVersions is the list of CSV(s) managed by the operator(s)
	// specified in the manifest.
	ClusterServiceVersions []*ClusterServiceVersion
}

// RawOperatorManifestData encapsulates the list of CRD(s), CSV(s) and
// package(s) associated with an operator manifest. It contains a complete and
// properly indented operator manifest.
//
// The Reader interface returns an object of this type so that it can be used to
// create a ConfigMap object associated with a given operator manifest.
type RawOperatorManifestData struct {
	// CustomResourceDefinitions is the set of custom resource definition(s)
	// associated with this package manifest.
	CustomResourceDefinitions string `yaml:"customResourceDefinitions"`

	// ClusterServiceVersions is the set of cluster service version(s)
	// associated with this package manifest.
	ClusterServiceVersions string `yaml:"clusterServiceVersions"`

	// Packages is the set of package(s) associated with this operator manifest.
	Packages string `yaml:"packages"`
}

// StructuredOperatorManifestData is a structured representation of an operator
// manifest. An operator manifest is a YAML document with the following sections:
// - customResourceDefinitions
// - clusterServiceVersions
// - packages
//
// An operator manifest is unmarshaled into this type so that we can perform
// certain operations like, but not limited to:
// - Construct a new operator manifest object to be used by a CatalogSourceConfig
//   by combining a set of existing operator manifest(s).
// - Construct a new operator manifest object by extracting a certain
//   operator/package from a a given operator manifest.
type StructuredOperatorManifestData struct {
	// CustomResourceDefinitions is the list of custom resource definition(s)
	// associated with this operator manifest.
	CustomResourceDefinitions []CustomResourceDefinition `json:"customResourceDefinitions"`

	// ClusterServiceVersions is the list of cluster service version(s)
	//associated with this operators manifest.
	ClusterServiceVersions []ClusterServiceVersion `json:"clusterServiceVersions"`

	// Packages is the list of package(s) associated with this operator manifest.
	Packages []PackageManifest `json:"packages"`
}

// PackageManifest holds information about a package, which is a reference to
// one (or more) channels under a single package.
//
// The following type has been copied as is from OLM.
// See https://github.com/operator-framework/operator-lifecycle-manager/blob/724b209ccfff33b6208cc5d05283800d6661d441/pkg/controller/registry/types.go#L78:6.
//
// We use it to unmarshal 'packages' element of an operator manifest.
type PackageManifest struct {
	// PackageName is the name of the overall package, ala `etcd`.
	PackageName string `json:"packageName"`

	// Channels are the declared channels for the package,
	// ala `stable` or `alpha`.
	Channels []PackageChannel `json:"channels"`

	// DefaultChannelName is, if specified, the name of the default channel for
	// the package. The default channel will be installed if no other channel is
	// explicitly given. If the package has a single channel, then that
	// channel is implicitly the default.
	DefaultChannelName string `json:"defaultChannel"`
}

// PackageChannel defines a single channel under a package, pointing to a
// version of that package.
//
// The following type has been directly copied as is from OLM.
// See https://github.com/operator-framework/operator-lifecycle-manager/blob/724b209ccfff33b6208cc5d05283800d6661d441/pkg/controller/registry/types.go#L105.
//
// We use it to unmarshal 'packages/package/channels' element of
// an operator manifest.
type PackageChannel struct {
	// Name is the name of the channel, e.g. `alpha` or `stable`.
	Name string `json:"name"`

	// CurrentCSVName defines a reference to the CSV holding the version of
	// this package currently for the channel.
	CurrentCSVName string `json:"currentCSV"`
}
