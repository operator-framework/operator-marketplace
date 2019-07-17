package operatorsource

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	"github.com/sirupsen/logrus"
)

// NewOtherNamespaceReconciler returns a Reconciler that reconciles a
// OperatorSource object created in a namespace other than the
// operator namespace.
func NewOtherNamespaceReconciler(log *logrus.Entry) Reconciler {
	return &otherNamespaceReconciler{
		log: log,
	}
}

// initialReconciler is an implementation of Reconciler interface that
// reconciles a OperatorSource object created in a namespace other
// than the operator namespace.
type otherNamespaceReconciler struct {
	log *logrus.Entry
}

// Reconcile reconciles a OperatorSource object created in a namespace other
// than the operator namespace. It returns "Failed" as the next desired phase
// unless the objects is already in the "Failed" phase.
func (r *otherNamespaceReconciler) Reconcile(ctx context.Context, in *v1.OperatorSource) (out *v1.OperatorSource, nextPhase *shared.Phase, err error) {
	// Do nothing as this object has already been placed in the failed phase.
	if in.Status.CurrentPhase.Name == phase.Failed {
		return
	}

	err = fmt.Errorf("Will only reconcile resources in the operator's namespace")
	r.log.Error(err)
	out = in
	nextPhase = phase.GetNextWithMessage(phase.Failed, err.Error())
	return
}
