package e2e

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	restConfig *rest.Config
	k8sClient  client.Client
)

func TestMarketplace(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Integration Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("boostrapping test environment")
	var err error
	restConfig, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	Expect(err).NotTo(HaveOccurred())
	Expect(restConfig).NotTo(BeNil())

	err = configv1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	err = olmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})
