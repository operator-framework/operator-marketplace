package certificateauthority

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewHandler returns a new Handler.
func NewHandler(client client.Client) Handler {
	return &configmapHandler{
		client: client,
	}
}

// Handler is the interface that wraps the Handle method
//
// Handle handles a new event associated with OperatorSource type.
//
// ctx represents the parent context.
// event encapsulates the event fired by operator sdk.
type Handler interface {
	Handle(ctx context.Context, operatorSource *corev1.ConfigMap) error
}

// configmapHandler implements the Handler interface
type configmapHandler struct {
	client client.Client
}

// Handle handles events associated with the ConfigMap type.
func (h *configmapHandler) Handle(ctx context.Context, in *corev1.ConfigMap) error {
	log := logrus.WithFields(logrus.Fields{
		"type": in.TypeMeta.Kind,
		"name": in.GetName(),
	})

	// Retrieve the Certificate Authority bundle from the ConfigMap.
	caBundle := in.Data[CABundleKey]

	// Retrieve the Certificate Authority bundle from Disk.
	// If an error is returned and is not related to a nonexistant file, return an error.
	caOnDisk, err := getCaOnDisk()
	if err != nil && !os.IsNotExist(err) {
		log.Infof("[ca] Error reading from disk: %v", err)
		return err
	}

	// Compare the Certificate Authority bundle on disk with the one in the ConfigMap.
	if caBundle != string(caOnDisk) {
		log.Infof("[ca] Certificate Authorization ConfigMap %s/%s is not in sync with disk, restarting marketplace.", in.Namespace, in.Name)
		os.Exit(0)
	}

	log.Infof("[ca] Certificate Authorization ConfigMap %s/%s is in sync with disk.", in.Namespace, in.Name)
	return nil
}
