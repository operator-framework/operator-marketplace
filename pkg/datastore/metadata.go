package datastore

import (
	"fmt"
)

// OperatorMetadata encapsulates operator metadata and manifest
// associated with a package.
type OperatorMetadata struct {
	// Metadata that uniquely identifies the given operator manifest in registry.
	RegistryMetadata RegistryMetadata

	// Operator manifest(s) in raw YAML format that contains a set of CRD(s),
	// CSV(s) and package channel(s).
	RawYAML []byte
}

// RegistryMetadata encapsulates metadata that uniquely describes the source of
// a given operator manifest(s) in a registry.
type RegistryMetadata struct {
	// Namespace is the namespace in app registry server
	// under which the package is hosted.
	Namespace string

	// Repository is the repository name for the specified package
	// in app registry.
	Repository string

	// Release represents the release or version number of the given package.
	Release string

	// Digest is the sha256 hash value that uniquely corresponds to the blob
	// associated with the release.
	Digest string
}

// ID returns the unique identifier associated with this operator manifest.
func (om *OperatorMetadata) ID() string {
	return fmt.Sprintf("%s/%s", om.RegistryMetadata.Namespace, om.RegistryMetadata.Repository)
}
