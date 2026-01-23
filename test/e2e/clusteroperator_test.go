package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	cohelpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("clusteroperator", func() {
	var (
		co                  = &configv1.ClusterOperator{}
		ctx                 = context.Background()
		clusterOperatorName = "marketplace"
		expectedTypeStatus  = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
			configv1.OperatorUpgradeable: configv1.ConditionTrue,
			configv1.OperatorProgressing: configv1.ConditionFalse,
			configv1.OperatorAvailable:   configv1.ConditionTrue,
			configv1.OperatorDegraded:    configv1.ConditionFalse,
		}
	)

	It("Should contain the expected status conditions", func() {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterOperatorName}, co)
		Expect(err).ToNot(HaveOccurred())
		Expect(co).ToNot(BeNil())

		for _, cond := range co.Status.Conditions {
			Expect(cond.Status).To(Equal(expectedTypeStatus[cond.Type]))
		}
	})

	It("Should contain the correct related objects", func() {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterOperatorName}, co)
		Expect(err).ToNot(HaveOccurred())
		Expect(co).ToNot(BeNil())

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

	It("Should set Progressing=True when ClusterOperator version is updated", func() {
		By("getting the on-cluster ClusterOperator")

		// Client for handling reporting of operator status
		Expect(restConfig).ToNot(BeNil())
		configClient, err := configclient.NewForConfig(restConfig)
		Expect(err).ToNot(HaveOccurred())

		clusterCO, err := configClient.ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(clusterCO).ToNot(BeNil())

		By("updating the ClusterOperator with an older version")
		clusterCO.Status.Versions = []configv1.OperandVersion{
			{
				Name:    "operator",
				Version: "0.0.0-fake",
			},
		}
		_, err = configClient.ClusterOperators().UpdateStatus(context.TODO(), clusterCO, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// The ClusterOperator monitor runs on a 20 second loop; therefore min wait time is <1s, max is 20s but
		// extra time is built it for possibility of a slower response/cache update
		By("waiting for marketplace-operator to set Progressing condition status to True")
		Eventually(func() error {
			co, err := configClient.ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if condition := cohelpers.FindStatusCondition(co.Status.Conditions, configv1.OperatorProgressing); condition == nil {
				return fmt.Errorf("waiting for Progressing condition to appear")
			} else if condition.Status != configv1.ConditionTrue {
				return fmt.Errorf("waiting for Progressing condition status to become True")
			}
			return nil
		}, 60*time.Second, 3).Should(BeNil())

		// 20s after the Progressing condition has been set to True, the ClusterOperator monitor
		// will run again and update the condition
		By("waiting for marketplace-operator to set Progressing condition status to False")
		Eventually(func() error {
			co, err := configClient.ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if condition := cohelpers.FindStatusCondition(co.Status.Conditions, configv1.OperatorProgressing); condition == nil {
				return fmt.Errorf("waiting for Progressing condition to appear")
			} else if condition.Status != configv1.ConditionFalse {
				return fmt.Errorf("waiting for Progressing condition status to become False")
			}
			return nil
		}, 60*time.Second, 3).Should(BeNil())
	})
})
