package defaults

import (
	"context"
	"errors"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var removedMarketplaceKey = types.NamespacedName{Namespace: "openshift-marketplace", Name: "redhat-marketplace"}

func newTestCatsrc(key types.NamespacedName) *olmv1alpha1.CatalogSource {
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

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, olmv1alpha1.AddToScheme(scheme))
	return scheme
}

// countDeletes returns interceptor funcs that pass Delete calls through to the
// fake client and record how many were made in deletes.
func countDeletes(deletes *int) interceptor.Funcs {
	return interceptor.Funcs{
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			*deletes++
			return c.Delete(ctx, obj, opts...)
		},
	}
}

// countClientCalls returns interceptor funcs that pass every read and write
// call through to the fake client and record how many were made in calls.
func countClientCalls(calls *int) interceptor.Funcs {
	return interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			*calls++
			return c.Get(ctx, key, obj, opts...)
		},
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			*calls++
			return c.List(ctx, list, opts...)
		},
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			*calls++
			return c.Create(ctx, obj, opts...)
		},
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			*calls++
			return c.Delete(ctx, obj, opts...)
		},
		DeleteAllOf: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteAllOfOption) error {
			*calls++
			return c.DeleteAllOf(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			*calls++
			return c.Update(ctx, obj, opts...)
		},
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			*calls++
			return c.Patch(ctx, obj, patch, opts...)
		},
	}
}

func TestIsRemovedSource(t *testing.T) {
	tests := []struct {
		name string
		key  types.NamespacedName
		want bool
	}{
		{
			name: "redhat-marketplace in openshift-marketplace",
			key:  removedMarketplaceKey,
			want: true,
		},
		{
			name: "redhat-marketplace in another namespace",
			key:  types.NamespacedName{Namespace: "other", Name: "redhat-marketplace"},
			want: false,
		},
		{
			name: "redhat-marketplace without a namespace",
			key:  types.NamespacedName{Name: "redhat-marketplace"},
			want: false,
		},
		{
			name: "default CatalogSource in openshift-marketplace",
			key:  types.NamespacedName{Namespace: "openshift-marketplace", Name: "redhat-operators"},
			want: false,
		},
		{
			name: "name with a common prefix in openshift-marketplace",
			key:  types.NamespacedName{Namespace: "openshift-marketplace", Name: "redhat-marketplace-mirror"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRemovedSource(tt.key))
		})
	}
}

func TestEnsureRemoved(t *testing.T) {
	ctx := context.Background()
	otherNamespaceKey := types.NamespacedName{Namespace: "other", Name: "redhat-marketplace"}
	otherNameKey := types.NamespacedName{Namespace: "openshift-marketplace", Name: "redhat-operators"}
	prefixNameKey := types.NamespacedName{Namespace: "openshift-marketplace", Name: "redhat-marketplace-mirror"}

	t.Run("deletes the removed CatalogSource regardless of its metadata", func(t *testing.T) {
		catsrc := newTestCatsrc(removedMarketplaceKey)
		catsrc.Labels = map[string]string{"app": "custom"}
		catsrc.Annotations = map[string]string{"example.com/owner": "someone-else"}
		catsrc.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       "owner",
			UID:        "1234",
		}}
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(catsrc).Build()

		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))

		err := c.Get(ctx, removedMarketplaceKey, &olmv1alpha1.CatalogSource{})
		assert.True(t, k8sErrors.IsNotFound(err), "expected NotFound, got %v", err)
	})

	t.Run("is idempotent once the CatalogSource is gone", func(t *testing.T) {
		deletes := 0
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(newTestCatsrc(removedMarketplaceKey)).
			WithInterceptorFuncs(countDeletes(&deletes)).Build()

		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))
		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))
		assert.Equal(t, 1, deletes)
	})

	t.Run("succeeds when the CatalogSource is not present", func(t *testing.T) {
		deletes := 0
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithInterceptorFuncs(countDeletes(&deletes)).Build()

		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))
		assert.Zero(t, deletes)
	})

	t.Run("does not delete again while the CatalogSource is terminating", func(t *testing.T) {
		catsrc := newTestCatsrc(removedMarketplaceKey)
		now := metav1.Now()
		catsrc.DeletionTimestamp = &now
		catsrc.Finalizers = []string{"example.com/finalizer"}
		deletes := 0
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(catsrc).WithInterceptorFuncs(countDeletes(&deletes)).Build()

		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))
		assert.Zero(t, deletes)
	})

	t.Run("rejects keys outside the deletion scope without using the client", func(t *testing.T) {
		seeded := []types.NamespacedName{removedMarketplaceKey, otherNamespaceKey, otherNameKey, prefixNameKey}
		for _, tt := range []struct {
			name string
			key  types.NamespacedName
		}{
			{name: "same name in another namespace", key: otherNamespaceKey},
			{name: "same name without a namespace", key: types.NamespacedName{Name: "redhat-marketplace"}},
			{name: "default CatalogSource in openshift-marketplace", key: otherNameKey},
			{name: "name with a common prefix in openshift-marketplace", key: prefixNameKey},
		} {
			t.Run(tt.name, func(t *testing.T) {
				objs := make([]client.Object, 0, len(seeded))
				for _, key := range seeded {
					objs = append(objs, newTestCatsrc(key))
				}
				calls := 0
				c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(objs...).
					WithInterceptorFuncs(countClientCalls(&calls)).Build()

				before := map[types.NamespacedName]*olmv1alpha1.CatalogSource{}
				for _, key := range seeded {
					obj := &olmv1alpha1.CatalogSource{}
					require.NoError(t, c.Get(ctx, key, obj))
					before[key] = obj
				}
				calls = 0

				err := EnsureRemoved(ctx, c, tt.key)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.key.String())
				assert.Contains(t, err.Error(), "not a removed default CatalogSource")
				assert.Zero(t, calls, "expected no client calls for out-of-scope key %s", tt.key)

				for _, key := range seeded {
					after := &olmv1alpha1.CatalogSource{}
					require.NoError(t, c.Get(ctx, key, after))
					assert.Equal(t, before[key], after, "CatalogSource %s was modified", key)
				}
			})
		}
	})

	t.Run("treats NotFound on delete as success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(newTestCatsrc(removedMarketplaceKey)).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return k8sErrors.NewNotFound(schema.GroupResource{Group: "operators.coreos.com", Resource: "catalogsources"}, obj.GetName())
			},
		}).Build()

		require.NoError(t, EnsureRemoved(ctx, c, removedMarketplaceKey))
	})

	t.Run("returns get errors", func(t *testing.T) {
		getErr := errors.New("get failed")
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(newTestCatsrc(removedMarketplaceKey)).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return getErr
			},
		}).Build()

		assert.ErrorIs(t, EnsureRemoved(ctx, c, removedMarketplaceKey), getErr)
	})

	t.Run("returns delete errors", func(t *testing.T) {
		deleteErr := k8sErrors.NewForbidden(schema.GroupResource{Group: "operators.coreos.com", Resource: "catalogsources"}, removedMarketplaceKey.Name, errors.New("denied"))
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(newTestCatsrc(removedMarketplaceKey)).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				return deleteErr
			},
		}).Build()

		err := EnsureRemoved(ctx, c, removedMarketplaceKey)
		require.Error(t, err)
		assert.True(t, k8sErrors.IsForbidden(err), "expected Forbidden, got %v", err)
		assert.NoError(t, c.Get(ctx, removedMarketplaceKey, &olmv1alpha1.CatalogSource{}))
	})
}

// TestShippedDefaultsExcludeRemovedSources guards against a removed
// CatalogSource being added back to the defaults directory, which would make
// the operator create and delete the same CatalogSource in a loop.
func TestShippedDefaultsExcludeRemovedSources(t *testing.T) {
	originalDir := Dir
	t.Cleanup(func() { Dir = originalDir })
	Dir = "../../defaults"

	definitions, _, err := populateDefsConfig(Dir, "")
	require.NoError(t, err)
	require.NotEmpty(t, definitions)

	for _, def := range definitions {
		key := types.NamespacedName{Namespace: def.Namespace, Name: def.Name}
		assert.False(t, IsRemovedSource(key), "default CatalogSource %s is also listed as removed", key)
	}
}
