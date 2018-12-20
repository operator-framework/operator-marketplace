package operatorsource_test

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func helperGetContextLogger() *log.Entry {
	return log.NewEntry(log.New())
}

func helperNewOperatorSourceWithEndpoint(namespace, name, endpointType string) *v1alpha1.OperatorSource {
	return &v1alpha1.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind: v1alpha1.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},

		Spec: v1alpha1.OperatorSourceSpec{
			Type:     endpointType,
			Endpoint: "http://localhost:5000/cnr",
		},
	}
}

func helperNewOperatorSourceWithPhase(namespace, name, phase string) *v1alpha1.OperatorSource {
	return &v1alpha1.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind: v1alpha1.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},

		Spec: v1alpha1.OperatorSourceSpec{
			Type:     "appregistry",
			Endpoint: "http://localhost:5000/cnr",
		},

		Status: v1alpha1.OperatorSourceStatus{
			CurrentPhase: v1alpha1.ObjectPhase{
				Phase: v1alpha1.Phase{
					Name: phase,
				},
			},
		},
	}
}

func helperNewCatalogSourceConfig(namespace, name string) *v1alpha1.CatalogSourceConfig {
	return &v1alpha1.CatalogSourceConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind: v1alpha1.CatalogSourceConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func helperNewCatalogSourceConfigWithLabels(namespace, name string, opsrcLabels map[string]string) *v1alpha1.CatalogSourceConfig {
	csc := helperNewCatalogSourceConfig(namespace, name)

	// This is the default label that should get added to CatalogSourceConfig.
	labels := map[string]string{
		"opsrc-datastore": "true",
	}

	for key, value := range opsrcLabels {
		labels[key] = value
	}

	csc.SetLabels(labels)

	return csc
}
