package datastore

import (
	yaml "gopkg.in/yaml.v2"
)

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
