package stub

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

// Handle function for handling CatalogSourceConfig and OperatorSource CR events
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.CatalogSourceConfig:
		// Ignore the delete event as the garbage collector will clean up the created resources as per the OwnerReference
		if event.Deleted {
			logrus.Infof("Deleted %s CatalogSourceConfig in %s namespace", o.Name, o.Spec.TargetNamespace)
			return nil
		}
		return createCatalogSource(o)

	case *v1alpha1.OperatorSource:
		err := sdk.Create(newbusyBoxPod2(o))
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("failed to create busybox pod : %v", err)
			return err
		}
	}
	return nil
}

func newbusyBoxPod2(cr *v1alpha1.OperatorSource) *corev1.Pod {
	labels := map[string]string{
		"app": "busy-box",
	}
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busy-box-operatorsource",
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, cr.GroupVersionKind()),
			},
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
