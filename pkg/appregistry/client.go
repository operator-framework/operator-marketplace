package appregistry

import (
	"errors"
	"fmt"
	"strings"
)

// Client exposes the functionality of app registry server
type Client interface {
	// RetrieveAll retrieves all visible packages from the given source
	// When namespace is specified, only package(s) associated with the given namespace are returned.
	// If namespace is empty then visible package(s) across all namespaces are returned.
	RetrieveAll(namespace string) ([]*OperatorMetadata, error)

	// RetrieveOne retrieves a given package from the source
	RetrieveOne(name, release string) (*OperatorMetadata, error)
}

// OperatorMetadata encapsulates operator metadata and manifest assocated with a package
type OperatorMetadata struct {
	// Namespace is the namespace in app registry server under which the package is hosted.
	Namespace string

	// Repository is the repository name for the specified package in app registry
	Repository string

	// Release represents the release or version number of the given package
	Release string

	// Digest is the sha256 hash value that uniquely corresponds to the blob associated with the release
	Digest string

	// Manifest encapsulates operator manifest
	Manifest *Manifest
}

func (om *OperatorMetadata) ID() string {
	return fmt.Sprintf("%s/%s", om.Namespace, om.Repository)
}

type client struct {
	adapter     apprApiAdapter
	decoder     blobDecoder
	unmarshaler blobUnmarshaler
}

func (c *client) RetrieveAll(namespace string) ([]*OperatorMetadata, error) {
	packages, err := c.adapter.ListPackages(namespace)
	if err != nil {
		return nil, err
	}

	list := make([]*OperatorMetadata, len(packages))
	for i, pkg := range packages {
		manifest, err := c.RetrieveOne(pkg.Name, pkg.Default)
		if err != nil {
			return nil, err
		}

		list[i] = manifest
	}

	return list, nil
}

func (c *client) RetrieveOne(name, release string) (*OperatorMetadata, error) {
	namespace, repository, err := split(name)
	if err != nil {
		return nil, err
	}

	metadata, err := c.adapter.GetPackageMetadata(namespace, repository, release)
	if err != nil {
		return nil, err
	}

	digest := metadata.Content.Digest
	blob, err := c.adapter.DownloadOperatorManifest(namespace, repository, digest)
	if err != nil {
		return nil, err
	}

	decoded, err := c.decoder.Decode(blob)
	if err != nil {
		return nil, err
	}

	manifest, err := c.unmarshaler.Unmarshal(decoded)
	if err != nil {
		return nil, err
	}

	om := &OperatorMetadata{
		Namespace:  namespace,
		Repository: repository,
		Release:    release,
		Manifest:   manifest,
		Digest:     digest,
	}

	return om, nil
}

func split(name string) (namespace string, repository string, err error) {
	// we expect package name to comply to this format - {namespace}/{repository}
	split := strings.Split(name, "/")
	if len(split) != 2 {
		return "", "", errors.New(fmt.Sprintf("package name should be specified in this format {namespace}/{repository}"))
	}

	namespace = split[0]
	repository = split[1]

	return namespace, repository, nil
}
