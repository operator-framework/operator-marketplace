package defaults

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	// Dir is the directory where the default OperatorSource definitions are
	// placed on disk. It will be empty if defaults are not required.
	Dir string

	// globalDefinitions is used to keep an in-memory record of default
	// OperatorSources as found on disk. It is a map of OperatorSource name
	// to the OperatorSource definition in the defaults directory. It is
	// populated just once during runtime to prevent new defaults from being
	// injected into the operator image.
	globalDefinitions = make(map[string]v1.OperatorSource)

	// defaultConfig is the default configuration for the cluster in the absence
	// of a an OperatorHub config object or if there is one with an empty spec.
	// The default is for all the OperatorSources in the globalDefinitions to be
	// enabled.
	defaultConfig = make(map[string]bool)
)

// Defaults is the interface that can be used to ensure default OperatorSources
// are always present on the cluster.
type Defaults interface {
	EnsureAll(client wrapper.Client) map[string]error
	Ensure(client wrapper.Client, opsrcName string) error
	RestoreSpecIfDefault(in *v1.OperatorSource)
}

type defaults struct {
	definitions map[string]v1.OperatorSource
	config      map[string]bool
}

// New returns an instance of defaults
func New(definitions map[string]v1.OperatorSource, config map[string]bool) Defaults {
	// Doing this to remove the need for checking at calls sites. This can be
	// made to return an error if error checking at calls sites is preferable.
	if definitions == nil || config == nil {
		panic("Defaults cannot be initialized with nil definitions or config")
	}
	return &defaults{
		definitions: definitions,
		config:      config,
	}
}

// RestoreSpecIfDefault takes an operator source and, if it is one of the defaults,
// sets the spec back to the expected spec in order to prevent any changes.
func (d *defaults) RestoreSpecIfDefault(in *v1.OperatorSource) {
	defOpsrc, present := d.definitions[in.Name]
	if !present {
		return
	}

	in.Spec = defOpsrc.Spec

	return
}

// Ensure checks if the given OperatorSource source is one of the
// defaults and if it is, it ensures it is present or absent on the cluster
// based on the config.
func (d *defaults) Ensure(client wrapper.Client, opsrcName string) error {
	opsrc, present := d.definitions[opsrcName]
	if !present {
		return nil
	}

	disable, present := d.config[opsrcName]
	if !present {
		disable = false
	}

	err := processOpSrc(client, opsrc, disable)
	if err != nil {
		return err
	}

	return nil
}

// EnsureAll processes all the default OperatorSources and ensures they are
// present or absent on the cluster based on the config.
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

// GetGlobals returns the global OperatorSource definitions and the
// default config
func GetGlobals() (map[string]v1.OperatorSource, map[string]bool) {
	return globalDefinitions, defaultConfig
}

// GetGlobalDefinitions returns the global OperatorSource definitions
func GetGlobalDefinitions() map[string]v1.OperatorSource {
	return globalDefinitions
}

// GetDefaultConfig returns the global OperatorSource definitions
func GetDefaultConfig() map[string]bool {
	return defaultConfig
}

// IsDefaultSource returns true if the given name is one of the default
// OperatorSources
func IsDefaultSource(name string) bool {
	_, present := defaultConfig[name]
	return present

}

// PopulateGlobals populates the global definitions and default config. If Dir
// is blank, the global definitions and config will be initialized but empty.
func PopulateGlobals() error {
	var err error
	globalDefinitions, defaultConfig, err = populateDefsConfig(Dir)
	return err
}

// populateDefsConfig returns populated OperatorSource definitions from files
// present in dir and an enabled config. It returns error on the first issues it
// runs into. The function also guarantees to return an empty map on error.
func populateDefsConfig(dir string) (map[string]v1.OperatorSource, map[string]bool, error) {
	definitions := make(map[string]v1.OperatorSource)
	config := make(map[string]bool)
	// Default directory has not been specified
	if dir == "" {
		return definitions, config, nil
	}

	_, err := os.Stat(Dir)
	if err != nil {
		return definitions, config, err
	}

	fileInfos, err := ioutil.ReadDir(Dir)
	if err != nil {
		return definitions, config, err
	}

	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		opsrc, err := getOpSrcDefinition(fileName)
		if err != nil {
			// Reinitialize the definitions as we hard error on even one failure
			definitions = make(map[string]v1.OperatorSource)
			config = make(map[string]bool)
			return definitions, config, err
		}
		definitions[opsrc.Name] = *opsrc
		config[opsrc.Name] = false
	}
	return definitions, config, nil
}

// getOpSrcDefinition returns an OperatorSource definition from the given file
// in the defaults directory. It only supports decoding OperatorSources. Any
// other resource type will result in an error.
func getOpSrcDefinition(fileName string) (*v1.OperatorSource, error) {
	file, err := os.Open(filepath.Join(Dir, fileName))
	if err != nil {
		return nil, err
	}

	opsrc := &v1.OperatorSource{}
	decoder := yaml.NewYAMLOrJSONDecoder(file, 1024)
	err = decoder.Decode(opsrc)
	if err != nil {
		return nil, err
	}
	return opsrc, nil
}

// processOpSrc will ensure that the given OperatorSource is present or not on
// the cluster based on the disable flag.
func processOpSrc(client wrapper.Client, def v1.OperatorSource, disable bool) error {
	// Get OperatorSource on the cluster
	cluster := &v1.OperatorSource{}
	err := client.Get(context.TODO(), wrapper.ObjectKey{
		Name:      def.Name,
		Namespace: def.Namespace},
		cluster)
	if err != nil && !errors.IsNotFound(err) {
		logrus.Errorf("[defaults] Error getting OperatorSource %s - %v", def.Name, err)
		return err
	}

	if disable {
		err = ensureAbsent(client, def, cluster)
	} else {
		err = ensurePresent(client, def, cluster)
	}

	if err != nil {
		logrus.Errorf("[defaults] Error processing OperatorSource %s - %v", def.Name, err)
	}

	return err
}

// ensureAbsent ensure that that the default OperatorSource is not present on the cluster
func ensureAbsent(client wrapper.Client, def v1.OperatorSource, cluster *v1.OperatorSource) error {
	// OperatorSource is not present on the cluster or has been marked for deletion
	if cluster.Name == "" || !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		logrus.Infof("[defaults] OperatorSource %s not present or has been marked for deletion", def.Name)
		return nil
	}

	err := client.Delete(context.TODO(), cluster)
	if err != nil {
		return err
	}

	logrus.Infof("[defaults] Deleting OperatorSource %s", def.Name)

	return err
}

// ensurePresent ensure that that the default OperatorSource is present on the cluster
func ensurePresent(client wrapper.Client, def v1.OperatorSource, cluster *v1.OperatorSource) error {
	// Create if not present or is deleted
	if cluster.Name == "" || (!cluster.ObjectMeta.DeletionTimestamp.IsZero() && len(cluster.Finalizers) == 0) {
		err := client.Create(context.TODO(), &def)
		if err != nil {
			return err
		}
		logrus.Infof("[defaults] Creating OperatorSource %s", def.Name)
		return nil
	}

	if def.Spec.IsEqual(&cluster.Spec) {
		logrus.Infof("[defaults] OperatorSource %s default and on cluster specs are same", def.Name)
		return nil
	}

	// Update if the spec has changed
	cluster.Spec = def.Spec
	err := client.Update(context.TODO(), cluster)
	if err != nil {
		return err
	}

	logrus.Infof("[defaults] Restoring OperatorSource %s", def.Name)

	return nil
}
