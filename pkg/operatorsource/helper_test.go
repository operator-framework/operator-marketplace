package operatorsource

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newOperatorSourceType(namespace, name string) *v1alpha1.OperatorSource {
	opsrc := &v1alpha1.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
			Kind:       v1alpha1.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},

		Spec: v1alpha1.OperatorSourceSpec{
			Type:     "appregistry",
			Endpoint: "http://localhost:5000/cnr",
		},
	}

	return opsrc
}
