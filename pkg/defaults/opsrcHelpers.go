package defaults

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func ensureOpsrc(client wrapper.Client, config map[string]bool, opsrc v1.OperatorSource) error {

	disable, present := config[opsrc.Name]
	if !present {
		disable = false
	}

	err := processOpSrc(client, opsrc, disable)
	if err != nil {
		return err
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
	if strings.Compare(opsrc.Kind, "OperatorSource") != 0 {
		return nil, errors.New("Not an OperatorSource")
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
	if err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Errorf("[defaults] Error getting OperatorSource %s - %v", def.Name, err)
		return err
	}

	if disable {
		err = ensureOpsrcAbsent(client, def, cluster)
	} else {
		err = ensureOpsrcPresent(client, def, cluster)
	}

	if err != nil {
		logrus.Errorf("[defaults] Error processing OperatorSource %s - %v", def.Name, err)
	}

	return err
}

// ensureOpsrcAbsent ensure that that the default OperatorSource is not present on the cluster
func ensureOpsrcAbsent(client wrapper.Client, def v1.OperatorSource, cluster *v1.OperatorSource) error {
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

// ensureOpsrcPresent ensure that that the default OperatorSource is present on the cluster
func ensureOpsrcPresent(client wrapper.Client, def v1.OperatorSource, cluster *v1.OperatorSource) error {
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
