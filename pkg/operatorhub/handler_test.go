package operatorhub

import (
	"context"
	"sort"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/operator-marketplace/pkg/defaults"
)

func TestHandleSetsDeterministicSourceStatusOrder(t *testing.T) {
	oldDir := defaults.Dir
	t.Cleanup(func() {
		defaults.Dir = oldDir
		require.NoError(t, defaults.PopulateGlobals(""))
	})
	defaults.Dir = "../../defaults"
	require.NoError(t, defaults.PopulateGlobals(""))

	scheme := runtime.NewScheme()
	require.NoError(t, configv1.AddToScheme(scheme))
	require.NoError(t, olmv1alpha1.AddToScheme(scheme))

	hub := &configv1.OperatorHub{
		ObjectMeta: metav1.ObjectMeta{Name: DefaultName},
		Spec: configv1.OperatorHubSpec{
			DisableAllDefaultSources: true,
			Sources: []configv1.HubSource{
				{Name: "not-a-default-source", Disabled: false},
			},
		},
	}
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(hub).
		WithStatusSubresource(hub).
		Build()
	handler := NewHandler(client)
	ctx := context.Background()

	var first []configv1.HubSourceStatus
	for i := 0; i < 20; i++ {
		in := &configv1.OperatorHub{}
		require.NoError(t, client.Get(ctx, types.NamespacedName{Name: DefaultName}, in))
		require.NoError(t, handler.Handle(ctx, in))

		out := &configv1.OperatorHub{}
		require.NoError(t, client.Get(ctx, types.NamespacedName{Name: DefaultName}, out))
		sources := out.Status.Sources
		require.Len(t, sources, len(defaults.GetDefaultConfig())+1)
		require.True(t, sort.SliceIsSorted(sources, func(a, b int) bool {
			return sources[a].Name < sources[b].Name
		}), "status sources are not sorted by name: %v", sources)

		if first == nil {
			first = sources
			continue
		}
		require.Equal(t, first, sources, "status sources changed between reconciles of the same spec")
	}
}
