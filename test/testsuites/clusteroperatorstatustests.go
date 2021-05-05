package testsuites

import (
	"context"
	"fmt"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ClusterOperatorStatusOnStartup is a test suite that ensures the ClusterOperator resource which
// defines the status of the marketplace operator has the correct status upon initialization. It
// also confirms that the ClusterOperator's RelatedObjects contains the expected list of objects.
func ClusterOperatorStatusOnStartup(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	// Get namespace
	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	// Check that the ClusterOperator resource has the correct status
	clusterOperatorName := "marketplace"
	expectedTypeStatus := map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
		configv1.OperatorUpgradeable: configv1.ConditionTrue,
		configv1.OperatorProgressing: configv1.ConditionFalse,
		configv1.OperatorAvailable:   configv1.ConditionTrue,
		configv1.OperatorDegraded:    configv1.ConditionFalse,
	}

	// Poll to ensure ClusterOperator is present and has the correct status
	// i.e. ConditionType has a ConditionStatus matching expectedTypeStatus
	namespacedName := types.NamespacedName{Name: clusterOperatorName, Namespace: namespace}
	result := &configv1.ClusterOperator{}
	RetryInterval := time.Second * 5
	Timeout := time.Minute * 5
	client := test.Global.Client

	err = wait.PollImmediate(RetryInterval, Timeout, func() (done bool, err error) {
		err = client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			return false, err
		}
		for _, condition := range result.Status.Conditions {
			if expectedTypeStatus[condition.Type] != condition.Status {
				return false, fmt.Errorf("expecting condition type %v of status %v but got %v", condition.Type, expectedTypeStatus[condition.Type], condition.Status)
			}
		}
		return true, nil
	})
	assert.NoError(t, err, "ClusterOperator never reached expected status")

	// Check if the expected default ObjectReferences are present in RelatedObjects
	expectedRelatedObjects := []configv1.ObjectReference{
		{
			Resource: "namespaces",
			Name:     namespace,
		},
		{
			Group:     olm.GroupName,
			Resource:  "catalogsources",
			Namespace: namespace,
		},
	}
	assert.ElementsMatch(t, result.Status.RelatedObjects, expectedRelatedObjects, "ClusterOperator did not list the expected RelatedObjects")
}
