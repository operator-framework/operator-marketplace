package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultTimeout = 30 * time.Second
	defaultPoll    = 1 * time.Second
)

var _ = Describe("operatorhub", func() {
	var (
		operatorhubName = "cluster"
		globalNamespace = "openshift-marketplace"
		ctx             = context.Background()
		nn              = types.NamespacedName{Name: operatorhubName}
	)

	// TODO: verify garbage collection of underlying catalogsource resources works as intended

	Context("The OperatorHub Controller", func() {
		AfterEach(func() {
			Eventually(func() error {
				og := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, og); err != nil {
					return err
				}
				og.Spec = configv1.OperatorHubSpec{}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, defaultPoll).Should(BeNil())
		})

		It("should ensure default catalogsources are deleted when spec.disableAllSources is set to true", func() {
			By("setting spec.disableAllSources to true")
			Eventually(func() error {
				og := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, og); err != nil {
					return err
				}
				og.Spec = configv1.OperatorHubSpec{
					DisableAllDefaultSources: true,
				}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, 3).Should(BeNil())

			By("ensuring all catalogsources have been deleted")
			Eventually(func() error {
				css := &olmv1alpha1.CatalogSourceList{}
				if err := k8sClient.List(ctx, css); err != nil {
					return err
				}
				if len(css.Items) != 0 {
					return fmt.Errorf("waiting for all default catalogsources to be deleted")
				}
				return nil
			}, defaultTimeout, 3).Should(BeNil())

			By("ensuring that the marketplace clusteroperator resource is still available")
			co := &configv1.ClusterOperator{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "marketplace"}, co)
			Expect(err).ToNot(HaveOccurred())

			expectedTypeStatus := map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
				configv1.OperatorUpgradeable: configv1.ConditionTrue,
				configv1.OperatorProgressing: configv1.ConditionFalse,
				configv1.OperatorAvailable:   configv1.ConditionTrue,
				configv1.OperatorDegraded:    configv1.ConditionFalse,
			}

			for _, cond := range co.Status.Conditions {
				Expect(cond.Status).To(Equal(expectedTypeStatus[cond.Type]))
			}
		})

		It("should ensure non-default catalogsources are not deleted when spec.disableAllSources is set to true", func() {
			nonDefaultName := "marketplace-non-default-cs-test"
			nonDefaultCS := &olmv1alpha1.CatalogSource{}
			nonDefaultCS.SetName(nonDefaultName)
			nonDefaultCS.SetNamespace(globalNamespace)

			err := k8sClient.Create(ctx, nonDefaultCS)
			Expect(err).ToNot(HaveOccurred())

			defer func() {
				err := k8sClient.Delete(ctx, nonDefaultCS)
				Expect(err).ToNot(HaveOccurred())
			}()

			Eventually(func() error {
				og := &configv1.OperatorHub{}
				err = k8sClient.Get(ctx, nn, og)
				Expect(err).NotTo(HaveOccurred())

				og.Spec = configv1.OperatorHubSpec{
					DisableAllDefaultSources: true,
				}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, defaultPoll).Should(BeNil())

			Eventually(func() error {
				css := &olmv1alpha1.CatalogSourceList{}
				err = k8sClient.List(ctx, css)
				Expect(err).ToNot(HaveOccurred())
				Expect(css).ToNot(BeNil())
				Expect(css.Items).To(HaveLen(1), "unexpected number of catalogsource resources returned")

				cs := css.Items[0]
				Expect(cs).ToNot(BeNil())
				Expect(cs.GetName()).To(Equal(nonDefaultCS.GetName()))
				return nil
			}, defaultTimeout, defaultPoll).Should(BeNil())
		})

		It("should ensure disabling a single catalogsource", func() {
			By("disabling the redhat-operators source in the operatorhub resource")
			disabledName := "redhat-operators"
			disabledNN := types.NamespacedName{
				Name:      disabledName,
				Namespace: globalNamespace,
			}

			Eventually(func() error {
				og := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, og); err != nil {
					return err
				}
				og.Spec = configv1.OperatorHubSpec{
					DisableAllDefaultSources: false,
					Sources: []configv1.HubSource{
						{
							Name:     disabledName,
							Disabled: true,
						},
					},
				}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, 3).Should(BeNil())

			By("checking the redhat-operators catalogsource does not exist")
			Eventually(func() bool {
				cs := &olmv1alpha1.CatalogSource{}
				if err := k8sClient.Get(ctx, disabledNN, cs); apierrors.IsNotFound(err) {
					return true
				}
				return false
			}, defaultTimeout, 3).Should(BeTrue())

			By("re-enabling the redhat-operators source in the operatorhub resource")
			Eventually(func() error {
				og := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, og); err != nil {
					return err
				}
				og.Spec = configv1.OperatorHubSpec{}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, 3).Should(BeNil())

			By("checking the redhat-operators catalogsource has been re-created")
			Eventually(func() error {
				cs := &olmv1alpha1.CatalogSource{}
				err := k8sClient.Get(ctx, disabledNN, cs)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(cs).NotTo(BeNil())
				Expect(cs.GetName()).To(Equal(disabledName))
				return nil
			}, defaultTimeout, 3).Should(BeNil())
		})

		It("should prefer spec.sources[*].disabled over spec.disableAllSources", func() {
			Eventually(func() error {
				og := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, og); err != nil {
					return err
				}
				og.Spec = configv1.OperatorHubSpec{
					DisableAllDefaultSources: true,
					Sources: []configv1.HubSource{
						{
							Name:     "community-operators",
							Disabled: false,
						},
					},
				}
				return k8sClient.Update(ctx, og)
			}, defaultTimeout, defaultPoll).Should(BeNil())

			Eventually(func() bool {
				css := &olmv1alpha1.CatalogSourceList{}
				if err := k8sClient.List(ctx, css); err != nil {
					return false
				}
				if len(css.Items) != 1 {
					return false
				}
				if css.Items[0].Name != "community-operators" {
					return false
				}
				return true
			}, defaultTimeout, 3).Should(BeTrue())
		})
	})
})
