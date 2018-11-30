package datastore

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
)

// CustomResourceDefinitionMap is a map of CustomResourceDefinition object(s).
// CRDKey type is used as the key to uniquely identify CustomResourceDefinition
// object(s).
type CustomResourceDefinitionMap map[CRDKey]*CustomResourceDefinition

// Load iterates over a list of CustomResourceDefinition object(s) and loads
// each item into the specified map.
//
// If two CustomResourceDefinition objects share the same key but they are
// semantically different then the function throws an error. OLM enforces the
// same constraint.
func (m CustomResourceDefinitionMap) Load(crds []CustomResourceDefinition) error {
	for i, crd := range crds {
		key := crd.Key()
		if old, exists := m[key]; exists &&
			!equality.Semantic.DeepEqual(crd.CustomResourceDefinition, old.CustomResourceDefinition) {
			return fmt.Errorf("invalid CRD : definition for CRD [%s] has already been set", key)
		}

		m[key] = &crds[i]
	}

	return nil
}

// Values returns a list of all CustomResourceDefinition object(s) stored in
// the map.
func (m CustomResourceDefinitionMap) Values() []*CustomResourceDefinition {
	if len(m) == 0 {
		return nil
	}

	values := make([]*CustomResourceDefinition, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}

	return values
}
