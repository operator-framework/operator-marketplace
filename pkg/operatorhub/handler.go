package operatorhub

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewHandler returns a new Handler
func NewHandler(client client.Client) Handler {
	return &confighandler{
		client: client,
	}
}

// Handler is the interface that wraps the Handle method
type Handler interface {
	Handle(ctx context.Context, operatorSource *configv1.OperatorHub) error
}

type confighandler struct {
	client client.Client
}

// Handle handles events associated with the OperatorHub type.
func (h *confighandler) Handle(ctx context.Context, in *configv1.OperatorHub) error {
	log := logrus.WithFields(logrus.Fields{
		"type": in.TypeMeta.Kind,
		"name": in.GetName(),
	})

	// Set the in memory configuration. This will be used by the OperatorSource reconcilers
	current := GetSingleton()
	current.Set(in.Spec)
	currentConfig := current.Get()

	// Apply the configuration to the default OperatorSources
	opsrcDefinitions, catsrcDefinitions := defaults.GetGlobalDefinitions()
	result := defaults.New(opsrcDefinitions, catsrcDefinitions, currentConfig).EnsureAll(h.client)

	err := h.updateStatus(ctx, log, in, currentConfig, result)
	if err != nil {
		log.Errorf("Error updating cluster OperatorHub - %v", err)
	}
	return err
}

// updateStatus reflects the current state of applying the configuration into
// status subresource of the object.
func (h *confighandler) updateStatus(ctx context.Context, log *logrus.Entry, in *configv1.OperatorHub, currentConfig map[string]bool, result map[string]error) error {
	var statuses []configv1.HubSourceStatus
	for name, disabled := range currentConfig {
		status := configv1.HubSourceStatus{}
		status.Name = name
		status.Disabled = disabled

		// Check if there were any errors in the processing of actual default OperatorSources
		if defaults.IsDefaultSource(name) {
			err, present := result[name]
			if !present {
				status.Status = "Success"
				status.Message = ""
			} else {
				status.Status = "Error"
				status.Message = err.Error()
			}
		} else {
			// A non-default or non-existent OperatorSource was present in the spec
			status.Status = "Error"
			status.Message = "Not present in the default definitions"
		}
		statuses = append(statuses, status)
	}
	in.Status = configv1.OperatorHubStatus{Sources: statuses}

	// The first status update will result in another event as there as been a
	// change to the object. The second update will be a no-op.
	return h.client.Status().Update(ctx, in)
}
