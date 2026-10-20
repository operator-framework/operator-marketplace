// Package lifecycle ties the controller-runtime manager to leader election
// and ClusterOperator status reporting.
package lifecycle

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/operator-framework/operator-marketplace/pkg/status"
	"github.com/sirupsen/logrus"
)

// Starter is the part of manager.Manager that RunManager needs.
type Starter interface {
	Start(context.Context) error
}

// State is what the rest of the process needs to know about the manager.
// Both flags are set before RunManager cancels the leader-election context,
// so a reader that runs after leader election returned sees them.
type State struct {
	// Stopped is set as soon as the manager returned. Healthz fails from
	// then on, even while the status reporter is still shutting down.
	Stopped atomic.Bool
	// Failed is set when the manager returned an error. main exits with
	// status 1 when it is set after leader election returned.
	Failed atomic.Bool
}

// RunManager starts the status reporter, runs the manager until it stops, and
// waits for the reporter to finish. It returns the manager error, if any.
//
// ctx is the leader-election context and cancel cancels it. When the manager
// fails, RunManager cancels that context. The reporter stops (its stopCh is
// ctx.Done()), leader election releases the lease (ReleaseOnCancel), and the
// caller can exit so the kubelet restarts the operator. Without the cancel the
// process stays alive, holds the lease and reports healthy while no controller
// runs (OCPBUGS-130575).
//
// state is updated before RunManager waits for the reporter. The reporter
// calls the API server with context.TODO, so that wait has no deadline of
// its own.
func RunManager(ctx context.Context, cancel context.CancelFunc, mgr Starter, reporter status.Reporter, state *State) error {
	done := reporter.StartReporting()
	err := mgr.Start(ctx)
	state.Stopped.Store(true)
	if err != nil {
		// Log here, before the cancel: main may exit as soon as leader
		// election returns, before the caller gets to log the error.
		logrus.WithError(err).Error("unable to run manager")
		state.Failed.Store(true)
		cancel()
	}
	<-done
	return err
}

// Healthz returns a liveness handler that answers 200 until the manager
// stopped, then 503, so the kubelet restarts the container when the manager
// has stopped but the process is still alive.
func Healthz(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state.Stopped.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
