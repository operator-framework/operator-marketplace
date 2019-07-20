package watches

import (
	"context"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/proxy"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type proxyToOperatorSources struct {
	client client.Client
}

// Map will update the proxy environment variables and create a reconcile request for
// all OperatorSources.
func (m *proxyToOperatorSources) Map(obj handler.MapObject) []reconcile.Request {
	clusterProxy := &apiconfigv1.Proxy{}
	err := m.client.Get(context.TODO(), proxy.ClusterProxyKey, clusterProxy)
	if err != nil {
		return nil
	}

	// Ensure that proxy is up to date.
	proxy.GetInstance().SetProxy(clusterProxy)

	// Get the list of OperatorSources.
	opsrcs := &v1.OperatorSourceList{}
	if err := m.client.List(context.TODO(), &client.ListOptions{}, opsrcs); err != nil {
		return nil
	}

	// Add each OperatorSource to the request
	requests := []reconcile.Request{}
	for _, opsrc := range opsrcs.Items {
		requests = append(requests, reconcile.Request{types.NamespacedName{Name: opsrc.GetName(), Namespace: opsrc.GetNamespace()}})
	}
	return requests
}

// ProxyToOperatorSources returns a mapper that maps the proxy to the OperatorSources.
func ProxyToOperatorSources(client client.Client) handler.Mapper {
	return &proxyToOperatorSources{client: client}
}
