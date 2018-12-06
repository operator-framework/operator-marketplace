package operatorsource

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The prefix to a name we use to create CatalogSourceConfig object.
	catalogSourceConfigPrefix = "opsrc"
)

// NewConfiguringReconciler returns a Reconciler that reconciles
// an OperatorSource object in "Configuring" phase.
func NewConfiguringReconciler(logger *log.Entry, datastore datastore.Writer, client client.Client) Reconciler {
	return &configuringReconciler{
		logger:    logger,
		datastore: datastore,
		client:    client,
		builder:   &CatalogSourceConfigBuilder{},
	}
}

// configuringReconciler is an implementation of Reconciler interface that
// reconciles an OperatorSource object in "Configuring" phase.
type configuringReconciler struct {
	logger    *log.Entry
	datastore datastore.Writer
	client    client.Client
	builder   *CatalogSourceConfigBuilder
}

// Reconcile reconciles an OperatorSource object that is in "Configuring" phase.
// It ensures that a corresponding CatalogSourceConfig object exists.
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
// Upon success, it returns "Succeeded" as the next and final desired phase.
// On error, the function returns "Failed" as the next desied phase
// and Message is set to appropriate error message.
//
// If the corresponding CatalogSourceConfig object already exists
// then no further action is taken.
func (r *configuringReconciler) Reconcile(ctx context.Context, in *v1alpha1.OperatorSource) (out *v1alpha1.OperatorSource, nextPhase *v1alpha1.Phase, err error) {
	if in.GetCurrentPhaseName() != phase.Configuring {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	out = in

	cscName := getCatalogSourceConfigName(in.Name)
	cscNamespacedName := types.NamespacedName{Name: cscName, Namespace: in.Namespace}
	cscRetrievedInto := r.builder.WithMeta(in.Namespace, cscName).CatalogSourceConfig()

	err = r.client.Get(ctx, cscNamespacedName, cscRetrievedInto)

	if err == nil {
		r.logger.Infof("No action taken, CatalogSourceConfig [name=%s] already exists", cscName)
		nextPhase = phase.GetNext(phase.Succeeded)
		return
	}

	if !k8s_errors.IsNotFound(err) {
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	manifests := r.datastore.GetPackageIDs()

	csc := r.builder.WithMeta(in.Namespace, cscName).
		WithSpec(in.Namespace, manifests).
		WithOwner(in).
		CatalogSourceConfig()

	err = r.client.Create(ctx, csc)
	if err != nil {
		r.logger.Infof("Unexpected error: %s", err.Error())
		nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
		return
	}

	nextPhase = phase.GetNext(phase.Succeeded)
	r.logger.Info("The object has been successfully reconciled")

	return
}

// Given a name of OperatorSource object, this function returns the name
// of the corresponding CatalogSourceConfig type object.
func getCatalogSourceConfigName(operatorsourceName string) string {
	return fmt.Sprintf("%s-%s", catalogSourceConfigPrefix, operatorsourceName)
}
