package appregistry

import (
	yaml "gopkg.in/yaml.v2"
)

// Manifest encapsulates operator manifest data
type Manifest struct {
	// Publisher represents the publisher of this package
	Publisher string `yaml:"publisher"`

	// Data reflects the content of the package manifest
	Data Data `yaml:"data"`
}

type Data struct {
	// CRDs is the list of CRD associated with a package
	CRDs string `yaml:"customResourceDefinitions"`

	// CSVs is the list of CSV associated with a package
	CSVs string `yaml:"clusterServiceVersions"`

	// Packages is the list of channels associated with a package
	Packages string `yaml:"packages"`
}

type blobUnmarshaler interface {
	// Unmarshal unmarshals package blob into structured representations
	Unmarshal(in []byte) (*Manifest, error)
}

type blobUnmarshalerImpl struct{}

func (*blobUnmarshalerImpl) Unmarshal(in []byte) (*Manifest, error) {
	m := &Manifest{}
	if err := yaml.Unmarshal(in, m); err != nil {
		return nil, err
	}

	return m, nil
}
