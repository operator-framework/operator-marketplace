package catalogsourceconfig

import (
	"context"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/operator-framework/operator-marketplace/pkg/watches"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewSucceededReconciler returns a Reconciler that reconciles a
// CatalogSourceConfig object in the "Succeeded" phase.
func NewSucceededReconciler(log *logrus.Entry, client client.Client) Reconciler {
	return &succeededReconciler{
		log:    log,
		client: client,
	}
}

// succeededReconciler is an implementation of Reconciler interface that
// reconciles an CatalogSourceConfig object in the "Succeeded" phase.
type succeededReconciler struct {
	log    *logrus.Entry
	client client.Client
}

// Reconcile reconciles an CatalogSourceConfig object that is in "Succeeded"
// phase. Since this phase indicates that the object has been successfully
// reconciled, no further action is taken.
//
// Given that nil is returned here, it implies that no phase transition is
// expected.
func (r *succeededReconciler) Reconcile(ctx context.Context, in *v2.CatalogSourceConfig) (out *v2.CatalogSourceConfig, nextPhase *shared.Phase, err error) {
	if in.Status.CurrentPhase.Name != phase.Succeeded {
		err = phase.ErrWrongReconcilerInvoked
		return
	}

	// No change is being made, so return the CatalogSourceConfig object as is.
	out = in

	msg := "No action taken, the object has already been reconciled"

	secretIsPresent, err := r.isSecretPresent(in)
	if err != nil {
		return
	}

	if watches.CheckChildResources(r.client, in.Name, in.Namespace, in.Spec.TargetNamespace, secretIsPresent) {
		// A child resource has been deleted. Drop the existing Status field so that reconciliation can start anew.
		out.Status = v2.CatalogSourceConfigStatus{}
		nextPhase = phase.GetNext(phase.Configuring)
		msg = "Child resource(s) have been deleted, scheduling for configuring"
	}

	r.log.Info(msg)

	return
}

// isSecretPresent checks if the OperatorSource specified in the CatalogSourceConfig contains an authorization
// token and returns true if it does. An error is returned if there is an issue retreiving the OperatorSource.
func (r *succeededReconciler) isSecretPresent(csc *v2.CatalogSourceConfig) (secretIsPresent bool, err error) {
	// Get the OperatorSource so that we can check if an authorization token is present
	opsrc := &v1.OperatorSource{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: csc.Namespace, Name: csc.Spec.Source}, opsrc)
	if err != nil {
		return false, err
	}

	if opsrc.Spec.AuthorizationToken.SecretName != "" {
		return true, nil
	}
	return false, nil
}
