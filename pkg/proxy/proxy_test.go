package proxy_test

import (
	"testing"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/proxy"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProxy  = "HTTP_PROXY"
	httpsProxy = "HTTPS_PROXY"
	noProxy    = "NO_PROXY"
)

func TestGetInstance(t *testing.T) {
	assert.NotNil(t, proxy.GetInstance())
}

func TestGetProxy(t *testing.T) {
	proxy := proxy.GetInstance()
	expected := []corev1.EnvVar{
		corev1.EnvVar{Name: noProxy, Value: ""},
		corev1.EnvVar{Name: httpProxy, Value: ""},
		corev1.EnvVar{Name: httpsProxy, Value: ""},
	}

	assert.Equal(t, expected, proxy.GetEnvVars())
}

func TestSetProxy(t *testing.T) {
	proxy := proxy.GetInstance()
	expected := []corev1.EnvVar{
		corev1.EnvVar{Name: httpProxy, Value: "HTTP_PROXY"},
		corev1.EnvVar{Name: httpsProxy, Value: "HTTPS_PROXY"},
		corev1.EnvVar{Name: noProxy, Value: "NO_PROXY"},
	}
	clusterProxy := &apiconfigv1.Proxy{}
	clusterProxy.Status.HTTPProxy = "HTTP_PROXY"
	clusterProxy.Status.HTTPSProxy = "HTTPS_PROXY"
	clusterProxy.Status.NoProxy = "NO_PROXY"

	proxy.SetProxy(clusterProxy)

	actual := proxy.GetEnvVars()
	for i := range expected {
		assert.True(t, contains(actual, expected[i]))
	}
}

func contains(envVars []corev1.EnvVar, envVar corev1.EnvVar) bool {
	for i := range envVars {
		if envVar == envVars[i] {
			return true
		}
	}
	return false
}
