package defaults

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	// Dir is the directory where the default OperatorSource definitions are
	// placed on disk. It will be empty if defaults are not required.
	Dir string

	// defaultsTracker is used to keep an in-memory record of default
	// OperatorSources. It is a map of OperatorSource name to the OperatorSource
	// definition in the defaults directory.
	defaultsTracker = make(map[string]v1.OperatorSource)
)

// Defaults is the interface that can be used to ensure default OperatorSources
// are always present on the cluster.
type Defaults interface {
	EnsureAll(client wrapper.Client) error
	Ensure(client wrapper.Client, opsrcName string) error
	RestoreSpecIfDefault(in *v1.OperatorSource)
}

type defaults struct {
}

// New returns a the singleton defaults
func New() Defaults {
	return &defaults{}
}

// RestoreSpecIfDefault takes an operator source and, if it is one of the defaults,
// sets the spec back to the expected spec in order to prevent any changes.
func (d *defaults) RestoreSpecIfDefault(in *v1.OperatorSource) {
	defOpsrc, present := defaultsTracker[in.Name]
	if !present {
		return
	}

	in.Spec = defOpsrc.Spec

	return
}

// Ensure checks if the given OperatorSource source is one of the
// defaults and if it is, it ensures it is present on the cluster.
func (d *defaults) Ensure(client wrapper.Client, opsrcName string) error {
	opsrc, present := defaultsTracker[opsrcName]
	if !present {
		return nil
	}

	err := processOpSrc(client, opsrc)
	if err != nil {
		return err
	}

	return nil
}

// EnsureAll processes all the default OperatorSources and ensures they are
// present on the cluster.
func (d *defaults) EnsureAll(client wrapper.Client) error {
	allErrors := []error{}
	for name := range defaultsTracker {
		err := d.Ensure(client, name)
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("Error handling %s - %v", name, err))
		}
	}
	return utilerrors.NewAggregate(allErrors)
}

// PopulateTracker populates the defaultsTracker on initialization
func PopulateTracker() error {
	// Default directory has not been specified
	if Dir == "" {
		return nil
	}

	_, err := os.Stat(Dir)
	if err != nil {
		return err
	}

	fileInfos, err := ioutil.ReadDir(Dir)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		opsrc, err := getOpSrcDefinition(fileName)
		if err != nil {
			// Reinitialize the tracker as we hard error on even one failure
			defaultsTracker = make(map[string]v1.OperatorSource)
			return err
		}
		defaultsTracker[opsrc.Name] = *opsrc
	}
	return nil
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

// processOpSrc will ensure that the given OperatorSource exists on the cluster.
func processOpSrc(client wrapper.Client, defaultOpsrc v1.OperatorSource) error {
	// Get OperatorSource on the cluster
	opsrcCluster := &v1.OperatorSource{}
	err := client.Get(context.TODO(), wrapper.ObjectKey{
		Name:      defaultOpsrc.Name,
		Namespace: defaultOpsrc.Namespace},
		opsrcCluster)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Create if not present or is deleted
	if errors.IsNotFound(err) || !opsrcCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		err = client.Create(context.TODO(), &defaultOpsrc)
		if err != nil {
			return err
		}
		logrus.Infof("[defaults] Creating OperatorSource %s", defaultOpsrc.Name)
		return nil
	}

	if defaultOpsrc.Spec.IsEqual(&opsrcCluster.Spec) {
		logrus.Infof("[defaults] OperatorSource %s default and on cluster specs are same", defaultOpsrc.Name)
		return nil
	}

	// Update if the spec has changed
	opsrcCluster.Spec = defaultOpsrc.Spec
	err = client.Update(context.TODO(), opsrcCluster)
	if err != nil {
		return err
	}
	logrus.Infof("[defaults] Restoring OperatorSource %s", defaultOpsrc.Name)

	return nil
}
