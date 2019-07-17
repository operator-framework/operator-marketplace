package watches

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type childResourceToCatalogSourceConfig struct {
	client client.Client
}

func (m *childResourceToCatalogSourceConfig) Map(obj handler.MapObject) []reconcile.Request {
	// We don't need to check if the key is nil as we are already doing that in the delete
	// predicate function.
	key := getCscOwnerKey(obj.Meta.GetLabels())
	csc := &v2.CatalogSourceConfig{}
	err := m.client.Get(context.TODO(), *key, csc)
	if err != nil {
		return nil
	}

	log.Infof("Child resource %s/%s owned by a CatalogSourceConfig was deleted", obj.Meta.GetNamespace(), obj.Meta.GetName())

	if !csc.GetDeletionTimestamp().IsZero() {
		log.Infof("CatalogSourceConfig %s/%s was marked for deletion. No action taken.", csc.GetNamespace(), csc.GetName())
		return nil
	}

	return []reconcile.Request{
		reconcile.Request{NamespacedName: *key},
	}
}

// ChildResourceToCatalogSourceConfig returns a mapper that maps the
// deleted child resource to the CatalogSourceConfig that it belonged to.
func ChildResourceToCatalogSourceConfig(client client.Client) handler.Mapper {
	return &childResourceToCatalogSourceConfig{client: client}
}
