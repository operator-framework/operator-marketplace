package datastore

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperatorManifestData encapsulates the list of CRD(s), CSV(s) and package(s)
// associated with an operator manifest. It contains a complete and
// properly indented operator manifest.
//
// The Reader interface returns an object of this type so that it can be used to
// create a ConfigMap object associated with a given operator manifest.
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

// OperatorManifest is a structured representation of an operator manifest.
//
// The Writer interface accepts a raw operator manifest and marshals it into
// this type before writing it to the underlying storage.
type OperatorManifest struct {
	// RegistryMetadata uniquely identifies a given operator manifest and
	// points to its source in remote registry.
	RegistryMetadata RegistryMetadata

	// Data is a structured representation of the given operator manifest.
	Data StructuredOperatorManifestData
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
	CustomResourceDefinitions []OLMObject `json:"customResourceDefinitions"`

	// ClusterServiceVersions is the list of cluster service version(s)
	//associated with this operators manifest.
	ClusterServiceVersions []OLMObject `json:"clusterServiceVersions"`

	// Packages is the list of package(s) associated with this operator manifest.
	Packages []PackageManifest `json:"packages"`
}

// OLMObject is a structured representation of OLM object and is
// used to unmarshal CustomResourceDefinition, ClusterServiceVersion
// from raw operator manifest YAML.
//
// This allows us to achieve loose coupling with OLM type(s). We don't need to
// parse the entire CustomResourceDefinition or ClusterServiceVersion object.
type OLMObject struct {
	// Type metadata.
	metav1.TypeMeta `json:",inline"`

	// Object metadata.
	metav1.ObjectMeta `json:"metadata"`

	// Spec is the raw representation of the 'spec' element of
	// CustomResourceDefinition and ClusterServiceVersion object. Since we are
	// not interested in the content of spec we are not parsing it.
	Spec json.RawMessage `json:"spec"`
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
