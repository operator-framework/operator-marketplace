package defaults

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func ensureCatsrc(
	ctx context.Context,
	client wrapper.Client,
	config map[string]bool,
	catsrc olmv1alpha1.CatalogSource,
) error {
	disable, present := config[catsrc.Name]
	if !present {
		disable = false
	}

	err := processCatsrc(ctx, client, catsrc, disable)
	if err != nil {
		return err
	}

	return nil
}

// getCatsrcDefinition returns a CatalogSource definition from the given file
// in the defaults directory. It only supports decoding CatalogSources. Any
// other resource type will result in an error.
func getCatsrcDefinition(fileName string) (*olmv1alpha1.CatalogSource, error) {
	file, err := os.Open(filepath.Join(Dir, fileName))
	if err != nil {
		return nil, err
	}

	catsrc := &olmv1alpha1.CatalogSource{}
	decoder := yaml.NewYAMLOrJSONDecoder(file, 1024)
	err = decoder.Decode(catsrc)
	if err != nil {
		return nil, err
	}
	if strings.Compare(catsrc.Kind, "CatalogSource") != 0 {
		return nil, errors.New("Not an CatalogSource")
	}
	return catsrc, nil
}

// processCatsrc will ensure that the given CatalogSource is present or not on
// the cluster based on the disable flag.
func processCatsrc(ctx context.Context, client wrapper.Client, def olmv1alpha1.CatalogSource, disable bool) error {
	// Get CatalogSource on the cluster
	cluster := &olmv1alpha1.CatalogSource{}
	if err := client.Get(ctx, wrapper.ObjectKey{
		Name:      def.Name,
		Namespace: def.Namespace,
	}, cluster); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Errorf("[defaults] Error getting CatalogSource %s - %v", def.Name, err)
		return err
	}

	var err error
	if disable {
		if cluster.Annotations[defaultCatsrcAnnotationKey] == defaultCatsrcAnnotationValue {
			err = ensureCatsrcAbsent(ctx, client, def, cluster)
		}
	} else {
		err = ensureCatsrcPresent(ctx, client, def, cluster)
	}

	if err != nil {
		logrus.Errorf("[defaults] Error processing CatalogSource %s - %v", def.Name, err)
	}

	return err
}

// ensureCatsrcAbsent ensure that that the default CatalogSource is not present on the cluster
func ensureCatsrcAbsent(
	ctx context.Context,
	client wrapper.Client,
	def olmv1alpha1.CatalogSource,
	cluster *olmv1alpha1.CatalogSource,
) error {
	// CatalogSource is not present on the cluster or has been marked for deletion
	if cluster.Name == "" || !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		logrus.Infof("[defaults] CatalogSource %s not present or has been marked for deletion", def.Name)
		return nil
	}

	if err := client.Delete(ctx, cluster); err != nil {
		return err
	}
	logrus.Infof("[defaults] Deleting CatalogSource %s", def.Name)

	return nil
}

// ensureCatsrcPresent ensure that that the default CatalogSource is present on the cluster
func ensureCatsrcPresent(
	ctx context.Context,
	client wrapper.Client,
	def olmv1alpha1.CatalogSource,
	cluster *olmv1alpha1.CatalogSource,
) error {
	def = *def.DeepCopy()
	if def.Annotations == nil {
		def.Annotations = make(map[string]string)
	}
	def.Annotations[defaultCatsrcAnnotationKey] = defaultCatsrcAnnotationValue

	// Create if not present or is deleted
	if cluster.Name == "" || (!cluster.ObjectMeta.DeletionTimestamp.IsZero() && len(cluster.Finalizers) == 0) {
		err := client.Create(ctx, &def)
		if err != nil {
			return err
		}
		logrus.Infof("[defaults] Creating CatalogSource %s", def.Name)
		return nil
	}

	shallowSpecComparison, deepSpecComparison := AreCatsrcSpecsEqual(&def.Spec, &cluster.Spec)

	if shallowSpecComparison && shallowSpecComparison != deepSpecComparison {
		// If the spec has not changed according to the old shallow comparison method but a change was
		// detected using the new deep comparison method, then set Upgradeable status to False but
		// do not reset the spec.
		cluster.Status.Conditions = append(cluster.Status.Conditions, v1.Condition{
			Type:               "Upgradeable",
			Status:             v1.ConditionFalse,
			Message:            "CatalogSource not Upgradeable",
			Reason:             "CatalogSource has been modified from default settings and is no longer Upgradeable",
			LastTransitionTime: v1.Now(),
		})
		logrus.Infof("[defaults] A change to the default CatalogSource %s was detected, setting 'Upgradeable' condition to 'False'", def.Name)
		// If the spec needs to be reset or the 'Upgradeable'='False' condition was added then update the CatalogSource
		err := client.Status().Update(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if cluster.Annotations[defaultCatsrcAnnotationKey] == defaultCatsrcAnnotationValue && shallowSpecComparison && deepSpecComparison {
		// If both the shallow and deep spec comparisons detect no change then we can leave the cluster CatalogSource as-is
		logrus.Infof("[defaults] CatalogSource %s is annotated and its spec is the same as the default spec", def.Name)
		return nil
	}

	// If the spec has changed according to the old shallow comparison method then reset the spec.
	cluster.Spec = def.Spec
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[defaultCatsrcAnnotationKey] = defaultCatsrcAnnotationValue
	logrus.Infof("[defaults] Restoring CatalogSource %s", def.Name)

	// If the spec needs to be reset or the 'Upgradeable'='False' condition was added then update the CatalogSource
	err := client.Update(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

// AreCatsrcSpecsEqual performs two comparisons and returns two bools:
// The first bool is a 'shallow' comparison which maintains past cluster behavior
// by allowing users to modify fields such as the RegistryPoll settings.
// The second bool is the result of a deep comparison which will be sensitive to
// any and all changes made to the default CatalogSource spec.
//
// Both comparisons perform case insensitive comparisons of corresponding attributes.
//
// If either of the Specs received is nil, then the function returns false for both bools.
func AreCatsrcSpecsEqual(spec1 *olmv1alpha1.CatalogSourceSpec, spec2 *olmv1alpha1.CatalogSourceSpec) (bool, bool) {
	if spec1 == nil || spec2 == nil {
		return false, false
	}
	spec1Copy := spec1.DeepCopy()
	spec2Copy := spec2.DeepCopy()

	spec1Copy.SourceType = olmv1alpha1.SourceType(strings.ToLower(string(spec1Copy.SourceType)))
	spec2Copy.SourceType = olmv1alpha1.SourceType(strings.ToLower(string(spec2Copy.SourceType)))

	spec1Copy.ConfigMap = strings.ToLower(spec1Copy.ConfigMap)
	spec2Copy.ConfigMap = strings.ToLower(spec2Copy.ConfigMap)

	spec1Copy.Address = strings.ToLower(spec1Copy.Address)
	spec2Copy.Address = strings.ToLower(spec2Copy.Address)

	spec1Copy.DisplayName = strings.ToLower(spec1Copy.DisplayName)
	spec2Copy.DisplayName = strings.ToLower(spec2Copy.DisplayName)

	spec1Copy.Publisher = strings.ToLower(spec1Copy.Publisher)
	spec2Copy.Publisher = strings.ToLower(spec2Copy.Publisher)

	spec1Copy.Image = strings.ToLower(spec1Copy.Image)
	spec2Copy.Image = strings.ToLower(spec2Copy.Image)

	deepComparison := reflect.DeepEqual(spec1Copy, spec2Copy)

	if !strings.EqualFold(string(spec1.SourceType), string(spec2.SourceType)) ||
		!strings.EqualFold(spec1.ConfigMap, spec2.ConfigMap) ||
		!strings.EqualFold(spec1.Address, spec2.Address) ||
		!strings.EqualFold(spec1.DisplayName, spec2.DisplayName) ||
		!strings.EqualFold(spec1.Publisher, spec2.Publisher) ||
		!strings.EqualFold(spec1.Image, spec2.Image) {
		return false, deepComparison
	}
	if spec1.UpdateStrategy != nil && spec2.UpdateStrategy == nil {
		return false, deepComparison
	}
	if spec1.UpdateStrategy == nil && spec2.UpdateStrategy != nil {
		return false, deepComparison
	}
	return true, deepComparison
}
