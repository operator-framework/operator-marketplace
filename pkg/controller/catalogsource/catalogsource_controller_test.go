package catalogsource

import (
	"context"
	"errors"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const marketplaceNamespace = "openshift-marketplace"

var (
	removedKey        = types.NamespacedName{Namespace: marketplaceNamespace, Name: "redhat-marketplace"}
	otherNamespaceKey = types.NamespacedName{Namespace: "other", Name: "redhat-marketplace"}
	defaultKey        = types.NamespacedName{Namespace: marketplaceNamespace, Name: "redhat-operators"}
	customKey         = types.NamespacedName{Namespace: marketplaceNamespace, Name: "custom-catalog"}
)

func newCatsrc(key types.NamespacedName) *olmv1alpha1.CatalogSource {
	return &olmv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: olmv1alpha1.CatalogSourceSpec{
			SourceType: olmv1alpha1.SourceTypeGrpc,
			Image:      "registry.example.com/catalog:latest",
		},
	}
}

func newFakeClientBuilder(t *testing.T) *fake.ClientBuilder {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, olmv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme)
}

func TestPredicate(t *testing.T) {
	pred := newPredicate(map[string]olmv1alpha1.CatalogSource{
		defaultKey.Name: *newCatsrc(defaultKey),
	})

	tests := []struct {
		name        string
		key         types.NamespacedName
		wantCreate  bool
		wantUpdate  bool
		wantDelete  bool
		wantGeneric bool
	}{
		{
			name:        "removed CatalogSource",
			key:         removedKey,
			wantCreate:  true,
			wantUpdate:  true,
			wantDelete:  false,
			wantGeneric: true,
		},
		{
			name:        "removed CatalogSource name in another namespace",
			key:         otherNamespaceKey,
			wantCreate:  false,
			wantUpdate:  false,
			wantDelete:  false,
			wantGeneric: false,
		},
		{
			name:        "default CatalogSource",
			key:         defaultKey,
			wantCreate:  false,
			wantUpdate:  true,
			wantDelete:  true,
			wantGeneric: true,
		},
		{
			name:        "non-default CatalogSource",
			key:         customKey,
			wantCreate:  false,
			wantUpdate:  false,
			wantDelete:  false,
			wantGeneric: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := newCatsrc(tt.key)
			assert.Equal(t, tt.wantCreate, pred.Create(event.CreateEvent{Object: obj}), "create")
			assert.Equal(t, tt.wantCreate, pred.Create(event.CreateEvent{Object: obj, IsInInitialList: true}), "create in initial list")
			assert.Equal(t, tt.wantUpdate, pred.Update(event.UpdateEvent{ObjectOld: obj, ObjectNew: obj.DeepCopy()}), "update")
			assert.Equal(t, tt.wantDelete, pred.Delete(event.DeleteEvent{Object: obj}), "delete")
			assert.False(t, pred.Delete(event.DeleteEvent{Object: obj, DeleteStateUnknown: true}), "delete with unknown state")
			assert.Equal(t, tt.wantGeneric, pred.Generic(event.GenericEvent{Object: obj}), "generic")
		})
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes the removed CatalogSource and leaves others untouched", func(t *testing.T) {
		c := newFakeClientBuilder(t).WithObjects(
			newCatsrc(removedKey),
			newCatsrc(otherNamespaceKey),
			newCatsrc(customKey),
		).Build()
		r := &ReconcileCatalogSource{client: c}

		for _, key := range []types.NamespacedName{removedKey, otherNamespaceKey, customKey} {
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			require.NoError(t, err, "reconciling %s", key)
			assert.Equal(t, reconcile.Result{}, res)
		}

		err := c.Get(ctx, removedKey, &olmv1alpha1.CatalogSource{})
		assert.True(t, k8sErrors.IsNotFound(err), "expected NotFound, got %v", err)
		assert.NoError(t, c.Get(ctx, otherNamespaceKey, &olmv1alpha1.CatalogSource{}))
		assert.NoError(t, c.Get(ctx, customKey, &olmv1alpha1.CatalogSource{}))
	})

	t.Run("succeeds when the CatalogSource is already gone", func(t *testing.T) {
		r := &ReconcileCatalogSource{client: newFakeClientBuilder(t).Build()}

		for i := 0; i < 2; i++ {
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: removedKey})
			require.NoError(t, err)
			assert.Equal(t, reconcile.Result{}, res)
		}
	})

	t.Run("still ensures default CatalogSources", func(t *testing.T) {
		originalDir := defaults.Dir
		t.Cleanup(func() {
			defaults.Dir = originalDir
			assert.NoError(t, defaults.PopulateGlobals(""))
			operatorhub.GetSingleton().Set(configv1.OperatorHubSpec{})
		})
		defaults.Dir = "../../../defaults"
		require.NoError(t, defaults.PopulateGlobals(""))
		operatorhub.GetSingleton().Set(configv1.OperatorHubSpec{})

		c := newFakeClientBuilder(t).Build()
		r := &ReconcileCatalogSource{client: c}

		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: defaultKey})
		require.NoError(t, err)

		catsrc := &olmv1alpha1.CatalogSource{}
		require.NoError(t, c.Get(ctx, defaultKey, catsrc))
		assert.Equal(t, "marketplace-operator", catsrc.Annotations["operatorframework.io/managed-by"])
	})

	t.Run("returns errors so the request is retried", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		c := newFakeClientBuilder(t).WithObjects(newCatsrc(removedKey)).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return deleteErr
			},
		}).Build()
		r := &ReconcileCatalogSource{client: c}

		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: removedKey})
		assert.ErrorIs(t, err, deleteErr)
		assert.NoError(t, c.Get(ctx, removedKey, &olmv1alpha1.CatalogSource{}))
	})
}
