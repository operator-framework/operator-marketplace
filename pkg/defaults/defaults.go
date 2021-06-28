package defaults

import (
	"io/ioutil"
	"os"

	olm "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
)

var (
	// Dir is the directory where the default OperatorSource definitions are
	// placed on disk. It will be empty if defaults are not required.
	Dir string

	// globalOpsrcDefinitions is used to keep an in-memory record of default
	// OperatorSources as found on disk. It is a map of OperatorSource name
	// to the OperatorSource definition in the defaults directory. It is
	// populated just once during runtime to prevent new defaults from being
	// injected into the operator image.
	globalOpsrcDefinitions = make(map[string]v1.OperatorSource)

	// globalCatsrcDefinitions is used to keep an in-memory record of default
	// CatalogSources as found on disk. It is a map of CatalogSource name
	// to the CatalogSource definition in the defaults directory. It is
	// populated just once during runtime to prevent new defaults from being
	// injected into the operator image.
	globalCatsrcDefinitions = make(map[string]olm.CatalogSource)

	// defaultConfig is the default configuration for the cluster in the absence
	// of a an OperatorHub config object or if there is one with an empty spec.
	// The default is for all the OperatorSources in the globalDefinitions to be
	// enabled.
	defaultConfig = make(map[string]bool)
)

const (
	defaultCatsrcAnnotationKey   string = "operatorframework.io/managed-by"
	defaultCatsrcAnnotationValue string = "marketplace-operator"
)

// Defaults is the interface that can be used to ensure default OperatorSources
// are always present on the cluster.
type Defaults interface {
	EnsureAll(client wrapper.Client) map[string]error
	Ensure(client wrapper.Client, sourceName string) error
	RestoreSpecIfDefault(in *v1.OperatorSource)
}

type defaults struct {
	opsrcDefinitions  map[string]v1.OperatorSource
	catsrcDefinitions map[string]olm.CatalogSource
	config            map[string]bool
}

// New returns an instance of defaults
func New(opsrcDefinitions map[string]v1.OperatorSource, catsrcDefinitions map[string]olm.CatalogSource, config map[string]bool) Defaults {
	// Doing this to remove the need for checking at calls sites. This can be
	// made to return an error if error checking at calls sites is preferable.
	if opsrcDefinitions == nil || catsrcDefinitions == nil || config == nil {
		panic("Defaults cannot be initialized with nil definitions or config")
	}
	return &defaults{
		opsrcDefinitions:  opsrcDefinitions,
		catsrcDefinitions: catsrcDefinitions,
		config:            config,
	}
}

// RestoreSpecIfDefault takes an operator source and, if it is one of the defaults,
// sets the spec back to the expected spec in order to prevent any changes.
func (d *defaults) RestoreSpecIfDefault(in *v1.OperatorSource) {
	defOpsrc, present := d.opsrcDefinitions[in.Name]
	if !present {
		return
	}

	in.Spec = defOpsrc.Spec

	return
}

// Ensure checks if the given OperatorSource or CatalogSource source is one of the
// defaults and if it is, it ensures it is present or absent on the cluster
// based on the config.
func (d *defaults) Ensure(client wrapper.Client, sourceName string) error {
	opsrc, present := d.opsrcDefinitions[sourceName]
	if !present {
		catsrc, present := d.catsrcDefinitions[sourceName]
		if !present {
			return nil
		}
		return ensureCatsrc(client, d.config, catsrc)
	}
	return ensureOpsrc(client, d.config, opsrc)
}

// EnsureAll processes all the default OperatorSources and Catalogsource and
// ensures they are present or absent on the cluster based on the config.
func (d *defaults) EnsureAll(client wrapper.Client) map[string]error {
	result := make(map[string]error)
	for name := range d.config {
		err := d.Ensure(client, name)
		if err != nil {
			result[name] = err
		}
	}
	return result
}

// GetGlobals returns the global OperatorSource and CatalogSource definitions and the
// default config
func GetGlobals() (map[string]v1.OperatorSource, map[string]olm.CatalogSource, map[string]bool) {
	return globalOpsrcDefinitions, globalCatsrcDefinitions, defaultConfig
}

// GetGlobalDefinitions returns the global OperatorSource and CatalogSource definitions
func GetGlobalDefinitions() (map[string]v1.OperatorSource, map[string]olm.CatalogSource) {
	return globalOpsrcDefinitions, globalCatsrcDefinitions
}

// GetDefaultConfig returns the global OperatorHub configuration
func GetDefaultConfig() map[string]bool {
	return defaultConfig
}

// IsDefaultSource returns true if the given name is one of the default
// OperatorSources or CatalogSources
func IsDefaultSource(name string) bool {
	_, present := defaultConfig[name]
	return present

}

// PopulateGlobals populates the global definitions and default config. If Dir
// is blank, the global definitions and config will be initialized but empty.
func PopulateGlobals() error {
	var err error
	globalOpsrcDefinitions, globalCatsrcDefinitions, defaultConfig, err = populateDefsConfig(Dir)
	return err
}

// populateDefsConfig returns populated OperatorSource and CatalogSource definitions
// from files present in dir and an enabled config. It returns error on the first
// issue it runs into. The function also guarantees to return an empty map on error.
func populateDefsConfig(dir string) (map[string]v1.OperatorSource, map[string]olm.CatalogSource, map[string]bool, error) {
	opsrcDefinitions := make(map[string]v1.OperatorSource)
	catsrcDefinitions := make(map[string]olm.CatalogSource)
	config := make(map[string]bool)
	// Default directory has not been specified
	if dir == "" {
		return opsrcDefinitions, catsrcDefinitions, config, nil
	}

	_, err := os.Stat(Dir)
	if err != nil {
		return opsrcDefinitions, catsrcDefinitions, config, err
	}

	fileInfos, err := ioutil.ReadDir(Dir)
	if err != nil {
		return opsrcDefinitions, catsrcDefinitions, config, err
	}

	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		opsrc, err := getOpSrcDefinition(fileName)
		if err != nil {
			catsrc, err := getCatsrcDefinition(fileName)
			if err != nil {
				// Reinitialize the definitions as we hard error on even one failure
				opsrcDefinitions = make(map[string]v1.OperatorSource)
				catsrcDefinitions = make(map[string]olm.CatalogSource)
				config = make(map[string]bool)
				return opsrcDefinitions, catsrcDefinitions, config, err
			}
			catsrcDefinitions[catsrc.Name] = *catsrc
			config[catsrc.Name] = false
		} else {
			opsrcDefinitions[opsrc.Name] = *opsrc
			config[opsrc.Name] = false
		}
	}
	return opsrcDefinitions, catsrcDefinitions, config, nil
}
