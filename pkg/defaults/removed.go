package defaults

import (
	"context"
	"fmt"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// removedCatalogSources is the set of CatalogSources that used to be shipped in the
// defaults directory and have since been dropped from it. Clusters that were
// upgraded from a release that shipped them still have them on cluster, so
// any CatalogSource matching one of these namespaced keys is deleted. The match
// is on the exact namespace and name only; labels, annotations and owner
// references are deliberately not considered.
var removedCatalogSources = map[types.NamespacedName]struct{}{
	// Previously defined in defaults/04_redhat_marketplace.yaml.
	{Namespace: "openshift-marketplace", Name: "redhat-marketplace"}: {},
}

// IsRemovedSource returns true if the given namespaced key identifies a
// CatalogSource that is no longer a default and must be removed from the
// cluster.
func IsRemovedSource(key types.NamespacedName) bool {
	_, present := removedCatalogSources[key]
	return present
}

// EnsureRemoved deletes the CatalogSource identified by key, which must be one
// of the removed default CatalogSources; callers are expected to check
// IsRemovedSource first. Any other key is rejected with an error before the
// client is used, so nothing outside the deletion scope is read or deleted. A
// CatalogSource that is not present on the cluster, or is already being
// deleted, is treated as successfully removed. Any other error is returned so
// that the caller can retry.
func EnsureRemoved(ctx context.Context, client wrapper.Client, key types.NamespacedName) error {
	if !IsRemovedSource(key) {
		return fmt.Errorf("refusing to delete CatalogSource %s: it is not a removed default CatalogSource", key)
	}

	cluster := &olmv1alpha1.CatalogSource{}
	if err := client.Get(ctx, key, cluster); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("[defaults] Error getting removed CatalogSource %s - %v", key, err)
		return err
	}

	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		logrus.Infof("[defaults] Removed CatalogSource %s has been marked for deletion", key)
		return nil
	}

	if err := client.Delete(ctx, cluster); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("[defaults] Error deleting removed CatalogSource %s - %v", key, err)
		return err
	}
	logrus.Infof("[defaults] Deleted CatalogSource %s as it is no longer a default", key)

	return nil
}
