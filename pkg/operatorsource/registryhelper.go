package operatorsource

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupRegistryOptions generates an Options object based on the OperatorSource spec. It passes along
// the opsrc endpoint and, if defined, retrieves the authorization token from the specified Secret
// object.
func SetupRegistryOptions(client k8sclient.Client, spec *v1alpha1.OperatorSourceSpec) (appregistry.Options, error) {
	options := appregistry.Options{
		Source: spec.Endpoint,
	}

	auth := spec.AuthorizationToken
	if auth.SecretName != "" && auth.SecretNamespace != "" {
		secret := corev1.Secret{}
		key := k8sclient.ObjectKey{
			Name:      auth.SecretName,
			Namespace: auth.SecretNamespace,
		}
		err := client.Get(context.TODO(), key, &secret)
		if err != nil {
			return options, err
		}

		options.AuthToken = string(secret.Data["token"])
	}

	return options, nil
}
