package operatorsource

import (
	"context"
	"errors"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// NewDownloadingReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Downloading" phase.
func NewDownloadingReconciler(logger *log.Entry, factory appregistry.ClientFactory, datastore datastore.Writer) Reconciler {
	return &downloadingReconciler{
		logger:    logger,
		factory:   factory,
		datastore: datastore,
	}
}

// downloadingReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Downloading" phase.
type downloadingReconciler struct {
	logger    *log.Entry
	factory   appregistry.ClientFactory
	datastore datastore.Writer
}

// Reconcile reconciles an OperatorSource object that is in "Downloading" phase.
// It connects to the corresponding operator manifest registry, downloads all
// manifest metadata available and saves the metadata to the underlying datastore.
//
// in represents the original OperatorSource object received from the sdk
// and before reconciliation has started.
//
// out represents the OperatorSource object after reconciliation has completed
// and could be different from the original. The OperatorSource object received
// (in) should be deep copied into (out) before changes are made.
//
// nextPhase represents the next desired phase for the given OperatorSource
// object. If nil is returned, it implies that no phase transition is expected.
//
// Upon success, it returns "Configuring" as the next desired phase for the
// given OperatorSource object.
// On error, the function returns "Failed" as the next desied phase
// and Message is set to appropriate error message.
func (r *downloadingReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *v1alpha1.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.OperatorSourceDownloading {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	r.logger.Infof("Downloading from [%s]", in.Spec.Endpoint)

	registry, err := r.factory.New(in.Spec.Type, in.Spec.Endpoint)
	if err != nil {
		nextPhase = phase.GetNextWithMessage(phase.OperatorSourceDownloading, err.Error())
		return
	}

	manifests, err := registry.ListPackages(in.Spec.RegistryNamespace)
	if err != nil {
		nextPhase = phase.GetNextWithMessage(phase.OperatorSourceDownloading, err.Error())
		return
	}

	if len(manifests) == 0 {
		err = errors.New("The operator source endpoint returned an empty manifest list")

		// Moving it to 'Failed' phase since human intervention is required to
		// resolve this situation. As soon as the user pushes new operator
		// manifest(s) registry sync will detect a new release and will trigger
		// a new reconciliation.
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	r.logger.Infof("Downloaded %d manifest(s) from the operator source endpoint", len(manifests))

	count, err := r.datastore.Write(in, manifests)
	if err != nil {
		if count == 0 {
			// No operator manifest was written, move to Failed phase.
			nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
			return
		}

		r.logger.Infof("There were some faulty operator manifest(s), errors - %v", err)
		err = nil
	}

	packages := r.datastore.GetPackageIDsByOperatorSource(out.GetUID())
	out.Status.Packages = packages

	r.logger.Infof("Successfully stored %d operator manifest(s)", count)
	r.logger.Info("Download complete, scheduling for configuration")

	nextPhase = phase.GetNext(phase.Configuring)
	return
}
