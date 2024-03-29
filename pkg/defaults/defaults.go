package defaults

import (
	"context"
	"io/ioutil"
	"os"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
)

var (
	// Dir is the directory where the default CatalogSources definitions are
	// placed on disk. It will be empty if defaults are not required.
	Dir string

	// globalCatsrcDefinitions is used to keep an in-memory record of default
	// CatalogSources as found on disk. It is a map of CatalogSource name
	// to the CatalogSource definition in the defaults directory. It is
	// populated just once during runtime to prevent new defaults from being
	// injected into the operator image.
	globalCatsrcDefinitions = make(map[string]olmv1alpha1.CatalogSource)

	// defaultConfig is the default configuration for the cluster in the absence
	// of a an OperatorHub config object or if there is one with an empty spec.
	// The default is for all the CatalogSources in the globalDefinitions to be
	// enabled.
	defaultConfig = make(map[string]bool)
)

const (
	defaultCatsrcAnnotationKey   string = "operatorframework.io/managed-by"
	defaultCatsrcAnnotationValue string = "marketplace-operator"
)

// Defaults is the interface that can be used to ensure the default set
// of CatalogSource resources are always present on cluster.
type Defaults interface {
	EnsureAll(ctx context.Context, client wrapper.Client) map[string]error
	Ensure(ctx context.Context, client wrapper.Client, sourceName string) error
}

type defaults struct {
	catsrcDefinitions map[string]olmv1alpha1.CatalogSource
	config            map[string]bool
}

// New returns an instance of defaults
func New(catsrcDefinitions map[string]olmv1alpha1.CatalogSource, config map[string]bool) Defaults {
	// Doing this to remove the need for checking at calls sites. This can be
	// made to return an error if error checking at calls sites is preferable.
	if catsrcDefinitions == nil || config == nil {
		panic("Defaults cannot be initialized with nil definitions or config")
	}
	return &defaults{
		catsrcDefinitions: catsrcDefinitions,
		config:            config,
	}
}

// Ensure checks if the given CatalogSource source is one of the
// defaults and if it is, it ensures it is present or absent on the cluster
// based on the config.
func (d *defaults) Ensure(ctx context.Context, client wrapper.Client, sourceName string) error {
	catsrc, present := d.catsrcDefinitions[sourceName]
	if !present {
		return nil
	}
	return ensureCatsrc(ctx, client, d.config, catsrc)
}

// EnsureAll processes all the default Catalogsources and ensures they are present
// or absent on the cluster based on the config.
func (d *defaults) EnsureAll(ctx context.Context, client wrapper.Client) map[string]error {
	result := make(map[string]error)
	for name := range d.config {
		err := d.Ensure(ctx, client, name)
		if err != nil {
			result[name] = err
		}
	}
	return result
}

// GetGlobals returns the global CatalogSource definitions and the
// default config
func GetGlobals() (map[string]olmv1alpha1.CatalogSource, map[string]bool) {
	return globalCatsrcDefinitions, defaultConfig
}

// GetGlobalCatalogSourceDefinitions returns the global CatalogSource definitions
func GetGlobalCatalogSourceDefinitions() map[string]olmv1alpha1.CatalogSource {
	return globalCatsrcDefinitions
}

// GetDefaultConfig returns the global OperatorHub configuration
func GetDefaultConfig() map[string]bool {
	return defaultConfig
}

// IsDefaultSource returns true if the given name is one of the default
// CatalogSources
func IsDefaultSource(name string) bool {
	_, present := defaultConfig[name]
	return present

}

// PopulateGlobals populates the global definitions and default config. If Dir
// is blank, the global definitions and config will be initialized but empty.
func PopulateGlobals() error {
	var err error
	globalCatsrcDefinitions, defaultConfig, err = populateDefsConfig(Dir)
	return err
}

// populateDefsConfig returns populated CatalogSource definitions from files present
// in the @dir directory and an enabled config. It returns error on the first
// issue it runs into. The function also guarantees to return an empty map on error.
func populateDefsConfig(dir string) (map[string]olmv1alpha1.CatalogSource, map[string]bool, error) {
	catsrcDefinitions := make(map[string]olmv1alpha1.CatalogSource)
	config := make(map[string]bool)
	// Default directory has not been specified
	if dir == "" {
		return catsrcDefinitions, config, nil
	}

	_, err := os.Stat(Dir)
	if err != nil {
		return catsrcDefinitions, config, err
	}

	fileInfos, err := ioutil.ReadDir(Dir)
	if err != nil {
		return catsrcDefinitions, config, err
	}

	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		catsrc, err := getCatsrcDefinition(fileName)
		if err != nil {
			// Reinitialize the definitions as we hard error on even one failure
			catsrcDefinitions = make(map[string]olmv1alpha1.CatalogSource)
			config = make(map[string]bool)
			return catsrcDefinitions, config, err
		}
		catsrcDefinitions[catsrc.Name] = *catsrc
		config[catsrc.Name] = false
	}
	return catsrcDefinitions, config, nil
}
