package datastore

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
)

// ManifestYAMLParser is an interface that is responsible for marshaling raw operator
// manifest into structured representation and vice versa.
type ManifestYAMLParser interface {
	// Unmarshal unmarshals raw operator manifest YAML into structured
	// representation.
	//
	// The function accepts raw yaml specified in rawYAML and converts it into
	// an instance of StructuredOperatorManifestData.
	Unmarshal(rawYAML []byte) (marshaled *StructuredOperatorManifestData, err error)

	// Marshal marshals a structured representation of an operator manifest into
	// raw YAML representation so that it can be used to create a configMap
	// object for a catalog source in OLM.
	//
	// The function accepts a structured representation of an operator manifest
	// specified in marshaled and returns a raw yaml representation of it.
	Marshal(marshaled *MultiOperatorManifest) (*RawOperatorManifestData, error)
}

type manifestYAMLParser struct{}

func (*manifestYAMLParser) Unmarshal(rawYAML []byte) (*StructuredOperatorManifestData, error) {
	var manifestYAML struct {
		Data RawOperatorManifestData `yaml:"data"`
	}

	if err := yaml.Unmarshal(rawYAML, &manifestYAML); err != nil {
		return nil, fmt.Errorf("error parsing raw YAML : %s", err)
	}

	var crds []CustomResourceDefinition
	var csvs []ClusterServiceVersion
	var packages []PackageManifest
	data := manifestYAML.Data

	crdJSONRaw, err := yaml.YAMLToJSON([]byte(data.CustomResourceDefinitions))
	if err != nil {
		return nil, fmt.Errorf("error converting CRD list (YAML) to JSON : %s", err)
	}
	if err := json.Unmarshal(crdJSONRaw, &crds); err != nil {
		return nil, fmt.Errorf("error parsing CRD list (JSON) : %s", err)
	}

	csvJSONRaw, err := yaml.YAMLToJSON([]byte(data.ClusterServiceVersions))
	if err != nil {
		return nil, fmt.Errorf("error converting CSV list (YAML) to JSON : %s", err)
	}
	if err := json.Unmarshal(csvJSONRaw, &csvs); err != nil {
		return nil, fmt.Errorf("error parsing CSV list (JSON) : %s", err)
	}

	packageJSONRaw, err := yaml.YAMLToJSON([]byte(data.Packages))
	if err != nil {
		return nil, fmt.Errorf("error converting package list (JSON) to YAML : %s", err)
	}
	if err := json.Unmarshal(packageJSONRaw, &packages); err != nil {
		return nil, fmt.Errorf("error parsing package list (JSON) : %s", err)
	}

	marshaled := &StructuredOperatorManifestData{
		CustomResourceDefinitions: crds,
		ClusterServiceVersions:    csvs,
		Packages:                  packages,
	}

	return marshaled, nil
}

func (*manifestYAMLParser) Marshal(marshaled *MultiOperatorManifest) (*RawOperatorManifestData, error) {
	crdRaw, err := yaml.Marshal(marshaled.CustomResourceDefinitions)
	if err != nil {
		return nil, fmt.Errorf("error marshaling CRD list into yaml : %s", err)
	}

	csvRaw, err := yaml.Marshal(marshaled.ClusterServiceVersions)
	if err != nil {
		return nil, fmt.Errorf("error marshaling CSV list into YAML : %s", err)
	}

	packageRaw, err := yaml.Marshal(marshaled.Packages)
	if err != nil {
		return nil, fmt.Errorf("error marshaling package list into YAML : %s", err)
	}

	data := &RawOperatorManifestData{
		CustomResourceDefinitions: string(crdRaw),
		ClusterServiceVersions:    string(csvRaw),
		Packages:                  string(packageRaw),
	}

	return data, nil
}
