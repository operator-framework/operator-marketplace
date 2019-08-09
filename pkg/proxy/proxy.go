package proxy

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	// HTTPProxy is the name of the environment variable that sets the proxy for HTTP requests.
	HTTPProxy = "HTTP_PROXY"

	// HTTPSProxy is the name of the environment variable that sets the proxy for HTTPS requests.
	HTTPSProxy = "HTTPS_PROXY"

	// NoProxy is the name of the environment variable that has a list of domains for which the proxy should not be used.
	NoProxy = "NO_PROXY"
)

// GetProxyEnvVars returns an array of proxy EnvVars.
func GetProxyEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		envVar(HTTPProxy), envVar(HTTPSProxy), envVar(NoProxy),
	}
}

// envVar takes a string and returns an EnvVar object with a matching name
// and the environment variable value associated with that name.
func envVar(varName string) corev1.EnvVar {
	return corev1.EnvVar{Name: varName, Value: os.Getenv(varName)}
}
