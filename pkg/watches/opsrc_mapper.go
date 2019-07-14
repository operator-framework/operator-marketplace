package watches

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type childResourceToOperatorSource struct {
	client client.Client
}

func (m *childResourceToOperatorSource) Map(obj handler.MapObject) []reconcile.Request {
	// We don't need to check if the key is nil as we are already doing that in the delete
	// predicate function.
	key := getOpSrcOwnerKey(obj.Meta.GetLabels())
	opsrc := &v1.OperatorSource{}
	err := m.client.Get(context.TODO(), *key, opsrc)
	if err != nil {
		return nil
	}

	log.Infof("Child resource %s/%s owned by a OperatorSource was deleted", obj.Meta.GetNamespace(), obj.Meta.GetName())

	if !opsrc.GetDeletionTimestamp().IsZero() {
		log.Infof("OperatorSource %s/%s was marked for deletion. No action taken.", opsrc.GetNamespace(), opsrc.GetName())
		return nil
	}

	return []reconcile.Request{
		reconcile.Request{NamespacedName: *key},
	}
}

// ChildResourceToOperatorSource returns a mapper that returns a request for the
// OperatorSource whose child resource has been deleted.
func ChildResourceToOperatorSource(client client.Client) handler.Mapper {
	return &childResourceToOperatorSource{client: client}
}
