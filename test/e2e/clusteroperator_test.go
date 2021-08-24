package e2e

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("clusteroperator", func() {
	var (
		ctx                 = context.Background()
		clusterOperatorName = "marketplace"
		expectedTypeStatus  = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
			configv1.OperatorUpgradeable: configv1.ConditionTrue,
			configv1.OperatorProgressing: configv1.ConditionFalse,
			configv1.OperatorAvailable:   configv1.ConditionTrue,
			configv1.OperatorDegraded:    configv1.ConditionFalse,
		}
	)
	co := &configv1.ClusterOperator{}

	It("Should contain the expected status conditions", func() {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterOperatorName}, co)
		Expect(err).ToNot(HaveOccurred())
		Expect(co).ToNot(BeNil())

		for _, cond := range co.Status.Conditions {
			Expect(cond.Status).To(Equal(expectedTypeStatus[cond.Type]))
		}
	})

	It("Should contain the correct related objects", func() {
		expectedRelatedObjects := []configv1.ObjectReference{
			{
				Resource: "namespaces",
				Name:     "openshift-marketplace",
			},
			{
				Group:     olmv1alpha1.GroupName,
				Resource:  "catalogsources",
				Namespace: "openshift-marketplace",
			},
		}
		Expect(co.Status.RelatedObjects).To(ContainElements(expectedRelatedObjects))
	})
})
