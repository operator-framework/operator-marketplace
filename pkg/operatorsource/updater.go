package operatorsource

import (
	"context"
	"errors"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Updater is an interface that can be used to check whether a remote registry
// has any update(s) and trigger a rebuild of in cluster OperatorSource cache.
type Updater interface {
	// Check contacts the remote registry associated with the specified
	// OperatorSource object, fetches release metadata and determines whether
	// the remote registry has new update(s).
	//
	// It returns true if the remote registry has update(s), otherwise it
	// returns false.
	// If the function encounters any error then it returns (false, err).
	Check(source *datastore.OperatorSourceKey) (bool, error)

	// Trigger triggers a rebuild of the cache associated with the
	// specified OperatorSource object.
	//
	// It fetches the latest copy of the specified OperatorSource object and
	// then sets the phase to 'Purging' so that the cache is invalidated and
	// reconciliation can start new.
	//
	// On return, deleted is set to true if the object has already been deleted.
	// updateErr is set to the error the function encounters while it tries
	// to update the OperatorSource object.
	Trigger(source *datastore.OperatorSourceKey) (deleted bool, updateErr error)
}

// updater implements the Updater interface.
type updater struct {
	factory      appregistry.ClientFactory
	datastore    datastore.Writer
	client       client.Client
	transitioner phase.Transitioner
}

func (u *updater) Check(source *datastore.OperatorSourceKey) (bool, error) {
	// Get the latest version of the operator source from underlying datastore.
	source, exists := u.datastore.GetOperatorSource(source.UID)
	if !exists {
		return false, errors.New("The given OperatorSource object does not exist in datastore")
	}

	registry, err := u.factory.New(source.Spec.Type, source.Spec.Endpoint)
	if err != nil {
		return false, err
	}

	metadata, err := registry.ListPackages(source.Spec.RegistryNamespace)
	if err != nil {
		return false, err
	}

	updated, err := u.datastore.OperatorSourceHasUpdate(source.UID, metadata)
	if err != nil {
		return false, err
	}

	return updated, nil
}

func (u *updater) Trigger(source *datastore.OperatorSourceKey) (deleted bool, updateErr error) {
	instance := &v1alpha1.OperatorSource{}

	// Get the current state of the given object before we make any decision.
	if err := u.client.Get(context.TODO(), source.Name, instance); err != nil {
		// Not found, the given OperatorSource object could have been deleted.
		// Treat it as no error and indicate that the object has been deleted.
		if k8s_errors.IsNotFound(err) {
			deleted = true
			return
		}

		// Otherwise, it is an error.
		updateErr = err
		return
	}

	// Needed because sdk does not get the gvk.
	instance.EnsureGVK()

	if instance.GetCurrentPhaseName() == phase.OperatorSourceDownloading {
		return
	}

	// We want to purge the OperatorSource object so that the cache can rebuild.
	nextPhase := &v1alpha1.Phase{
		Name:    phase.OperatorSourcePurging,
		Message: "Remote registry has been updated",
	}
	if !u.transitioner.TransitionInto(&instance.Status.CurrentPhase, nextPhase) {
		// No need to update since the object is already in purging phase.
		return
	}

	if err := u.client.Update(context.TODO(), instance); err != nil {
		updateErr = err
		return
	}

	return
}
