package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const (
	defaultTimeout = 30 * time.Second
	defaultPoll    = 1 * time.Second
)

var _ = Describe("operatorhub", func() {
	var (
		operatorhubName           = "cluster"
		globalNamespace           = "openshift-marketplace"
		ctx                       = context.Background()
		nn                        = types.NamespacedName{Name: operatorhubName}
		defaultCatalogSourceNames = []string{"redhat-operators", "certified-operators", "community-operators", "redhat-marketplace"}
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
		It("should create the default catalogsources in restricted mode", func() {
			// keysToListFunc takes a map and returns the keys as a string array
			keysToListFunc := func(m map[string]struct{}) []string {
				keys := []string{}
				for key := range m {
					keys = append(keys, key)
				}
				return keys
			}

			Eventually(func() error {
				csl := &olmv1alpha1.CatalogSourceList{}
				if err := k8sClient.List(ctx, csl); err != nil {
					return err
				}

				defaultSources := map[string]struct{}{}
				for _, name := range defaultCatalogSourceNames {
					defaultSources[name] = struct{}{}
				}

				for _, cs := range csl.Items {
					if cs.Spec.GrpcPodConfig.SecurityContextConfig == olmv1alpha1.Restricted {
						delete(defaultSources, cs.GetName())
					}
				}

				defaultSourcesNotFoundInRestrictedMode := keysToListFunc(defaultSources)
				if len(defaultSourcesNotFoundInRestrictedMode) != 0 {
					return fmt.Errorf("The following default catalogsources were not found in restricted mode: %v", defaultSourcesNotFoundInRestrictedMode)
				}

				return nil
			}, defaultTimeout, 3).Should(BeNil())
		})
		Context("default CatalogSource value enforcement", func() {
			var (
				cs                 olmv1alpha1.CatalogSource
				originalCatSrcSpec olmv1alpha1.CatalogSourceSpec
				catSrcSpec         olmv1alpha1.CatalogSourceSpec
				catSrcNN           types.NamespacedName = types.NamespacedName{Name: "certified-operators", Namespace: globalNamespace}
			)
			flipCase := func(s string) string {
				flip := func(r rune) rune {
					if unicode.IsUpper(r) {
						return unicode.ToLower(r)
					} else if unicode.IsLower(r) {
						return unicode.ToUpper(r)
					}
					return r
				}
				return strings.Map(flip, s)
			}
			BeforeEach(func() {
				cs = olmv1alpha1.CatalogSource{}
				err := k8sClient.Get(ctx, catSrcNN, &cs)
				Expect(err).ToNot(HaveOccurred())
				originalCatSrcSpec = cs.Spec
			})
			AfterEach(func() {
				// Return the spec back to default when finished testing, in case we fail somewhere before the end
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					cs := olmv1alpha1.CatalogSource{}
					err := k8sClient.Get(ctx, catSrcNN, &cs)
					if err != nil {
						return err
					}
					cs.Spec = originalCatSrcSpec
					return k8sClient.Update(ctx, &cs)
				})
				Expect(err).ToNot(HaveOccurred())
			})
			It("should maintain the values of default catalogsources, ignoring character case in strings", func() {
				By("swapping the letter case of spec.SourceType, ConfigMap, Address, DisplayName, and Publisher")
				Eventually(func() error {
					// Flip the letter case of all case-insensitive fields except Image
					cs.Spec.SourceType = olmv1alpha1.SourceType(flipCase(string(cs.Spec.SourceType)))
					cs.Spec.ConfigMap = flipCase(cs.Spec.ConfigMap)
					cs.Spec.Address = flipCase(cs.Spec.Address)
					cs.Spec.DisplayName = flipCase(cs.Spec.DisplayName)
					cs.Spec.Publisher = flipCase(cs.Spec.Publisher)
					// Update our Spec to track changes so far
					catSrcSpec = cs.Spec
					return k8sClient.Update(ctx, &cs)
				}, defaultTimeout, 3).Should(BeNil())
				Eventually(func() (olmv1alpha1.CatalogSourceSpec, error) {
					newCs := &olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, newCs); err != nil {
						return olmv1alpha1.CatalogSourceSpec{}, err
					}
					return newCs.Spec, nil
					// Spec should not have been reverted (no differences detected)
				}, defaultTimeout, 3).Should(Equal(catSrcSpec))

				By("swapping the letter case of spec.Image")
				Eventually(func() error {
					cs = olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, &cs); err != nil {
						return err
					}
					// Flip the letter case of spec.Image
					cs.Spec.Image = flipCase(cs.Spec.Image)
					// Update our Spec to track changes so far
					catSrcSpec.Image = cs.Spec.Image
					return k8sClient.Update(ctx, &cs)
				}, defaultTimeout, 3).Should(BeNil())
				Eventually(func() (olmv1alpha1.CatalogSourceSpec, error) {
					newCs := &olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, newCs); err != nil {
						return olmv1alpha1.CatalogSourceSpec{}, err
					}
					return newCs.Spec, nil
					// Spec should not have been reverted (no differences detected)
				}, defaultTimeout, 3).Should(Equal(catSrcSpec))

				By("setting the value of spec.ConfigMap to a new value")
				Eventually(func() error {
					cs = olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, &cs); err != nil {
						return err
					}
					// Add a new suffix to the existing ConfigMap value
					cs.Spec.ConfigMap = cs.Spec.ConfigMap + "-foo"
					return k8sClient.Update(ctx, &cs)
				}, defaultTimeout, 3).Should(BeNil())
				Eventually(func() (olmv1alpha1.CatalogSourceSpec, error) {
					newCs := &olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, newCs); err != nil {
						return olmv1alpha1.CatalogSourceSpec{}, err
					}
					return newCs.Spec, nil
					// Spec should eventually be reverted back to default values due to a detected difference
				}, defaultTimeout, 3).Should(Equal(originalCatSrcSpec))

				By("changing the content of spec.grpcPodConfig")
				Eventually(func() error {
					cs = olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, &cs); err != nil {
						return err
					}
					// Set the value of Spec.GrpcPodConfig.PriorityClassName
					newClassName := "foo"
					cs.Spec.GrpcPodConfig.PriorityClassName = &newClassName
					return k8sClient.Update(ctx, &cs)
				}, defaultTimeout, 3).Should(BeNil())
				Eventually(func() (olmv1alpha1.CatalogSourceSpec, error) {
					newCs := &olmv1alpha1.CatalogSource{}
					if err := k8sClient.Get(ctx, catSrcNN, newCs); err != nil {
						return olmv1alpha1.CatalogSourceSpec{}, err
					}
					return newCs.Spec, nil
					// Spec should eventually be reverted back to default values due to a detected difference
				}, defaultTimeout, 3).Should(Equal(originalCatSrcSpec))
			})
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

			By("ensuring the operatorHub status reflects that the default catalogsources have been disabled successfully")
			Eventually(func() error {
				oh := &configv1.OperatorHub{}
				if err := k8sClient.Get(ctx, nn, oh); err != nil {
					return err
				}
				defaultSources := map[string]struct{}{}
				for _, name := range defaultCatalogSourceNames {
					defaultSources[name] = struct{}{}
				}

				for _, source := range oh.Status.Sources {
					if source.Disabled == true && source.Status == "Success" && source.Message == "" {
						delete(defaultSources, source.Name)
					}
				}
				if len(defaultSources) != 0 {
					return fmt.Errorf("operatorHub.Status.Sources not in expected state: %v", oh.Status.Sources)
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
				if err := k8sClient.List(ctx, css); err != nil {
					return err
				}
				if len(css.Items) != 1 {
					return fmt.Errorf("unexpected number of catalogsources returned from list call")
				}
				if css.Items[0].ObjectMeta.Name != nonDefaultName {
					return fmt.Errorf("unexpected catalogsource returned from list call")
				}
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
