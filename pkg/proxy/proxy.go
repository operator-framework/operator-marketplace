package proxy

import (
	"context"
	"sort"
	"sync"

	apiconfigv1 "github.com/openshift/api/config/v1"
	"golang.org/x/net/http/httpproxy"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// HTTPProxy is the name of the environment variable that sets the proxy for HTTP requests.
	HTTPProxy = "HTTP_PROXY"

	// HTTPSProxy is the name of the environment variable that sets the proxy for HTTPS requests.
	HTTPSProxy = "HTTPS_PROXY"

	// NoProxy is the name of the environment variable that has a list of domains for which the proxy should not be used.
	NoProxy = "NO_PROXY"

	// ClusterProxyName is the name of the global config proxy.
	ClusterProxyName = "cluster"
)

var (
	// ClusterProxyKey is the key for the cluster proxy.
	ClusterProxyKey = types.NamespacedName{Name: ClusterProxyName}
)

// Proxy is an interface that provides function to set, use, and consume a proxy.
type Proxy interface {
	// SetProxy updates the proxy field.
	SetProxy(proxy *apiconfigv1.Proxy)

	// GetEnvVars returns a list proxy EnvVars.
	GetEnvVars() []corev1.EnvVar

	// GetProxyConfig returns an httpproxy.Config used to send http requests through
	// the proxy defined in the structure.
	GetProxyConfig() *httpproxy.Config

	// CheckDeploymentEnvVars returns true if the deployment with the given name and
	// namespace needs to have its environment variables update to match those defined
	// in the proxy.
	CheckDeploymentEnvVars(client cli.Client, name, namespace string) (bool, error)
}

// proxy implements Proxy.
type proxy struct {
	proxy *apiconfigv1.Proxy
	lock  sync.Mutex
}

// instance is a singleton proxy used to store and update proxy settings.
var instance *proxy

// once is used to ensure that instance is only initialized once.
var once sync.Once

// GetInstance returns the global proxy.
func GetInstance() Proxy {
	once.Do(func() {
		instance = &proxy{
			proxy: &apiconfigv1.Proxy{},
			lock:  sync.Mutex{},
		}
	})
	return instance
}

func (p *proxy) SetProxy(proxy *apiconfigv1.Proxy) {
	instance.lock.Lock()
	defer instance.lock.Unlock()
	p.proxy = proxy
}

func (p *proxy) GetEnvVars() []corev1.EnvVar {
	instance.lock.Lock()
	defer instance.lock.Unlock()
	return []corev1.EnvVar{
		corev1.EnvVar{Name: NoProxy, Value: p.proxy.Status.NoProxy},
		corev1.EnvVar{Name: HTTPProxy, Value: p.proxy.Status.HTTPProxy},
		corev1.EnvVar{Name: HTTPSProxy, Value: p.proxy.Status.HTTPSProxy},
	}
}

func (p *proxy) GetProxyConfig() *httpproxy.Config {
	instance.lock.Lock()
	defer instance.lock.Unlock()
	return &httpproxy.Config{
		HTTPProxy:  p.proxy.Status.HTTPProxy,
		HTTPSProxy: p.proxy.Status.HTTPSProxy,
		NoProxy:    p.proxy.Status.NoProxy,
	}
}

func (p *proxy) CheckDeploymentEnvVars(client cli.Client, name, namespace string) (bool, error) {
	// Check if the Proxy API exists.
	if !IsAPIAvailable() {
		return false, nil
	}

	// Get the Deployment with the given namespacedname.
	deployment := &apps.Deployment{}
	key := types.NamespacedName{Namespace: namespace, Name: name}
	if err := client.Get(context.TODO(), key, deployment); err != nil {
		return false, err
	}

	// Get the array of EnvVar objects from the deployment and the proxy singleton.
	deploymentEnv := deployment.Spec.Template.Spec.Containers[0].Env
	operatorEnv := p.GetEnvVars()

	// Sort the two arrays.
	p.sortEnvVars(deploymentEnv)
	p.sortEnvVars(operatorEnv)

	// Check if the deployment environment variables are in sync with the proxy singleton.
	if p.equalEnvVars(deploymentEnv, operatorEnv) {
		return false, nil
	}

	return true, nil
}

// sortEnvVars will sort a list of EnvVar objects alphabetically by thier name.
func (p *proxy) sortEnvVars(envVars []corev1.EnvVar) {
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})
}

// equalEnvVars checks if two arrays of EnvVars contain the same elements in the same order.
func (p *proxy) equalEnvVars(a, b []corev1.EnvVar) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
