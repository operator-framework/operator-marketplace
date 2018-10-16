package phase

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

// NewTransitionerWithClock returns a new Transitioner with the given clock.
// This function can be used for unit testing Transitioner.
func NewTransitionerWithClock(clock clock.Clock) Transitioner {
	return &transitioner{
		clock: clock,
	}
}

// NewTransitioner returns a new PhaseTransitioner with the default RealClock.
func NewTransitioner() Transitioner {
	clock := &clock.RealClock{}
	return NewTransitionerWithClock(clock)
}

// Transitioner is an interface that wraps the TransitionInto method
//
// TransitionInto transitions a given OperatorSource object into the specified next phase.
// If the given OperatorSource object is nil, the function returns false to indicate no transition took place.
// If the given OperatorSource object has the same phase and message specified in next phase,
// then the function returns false to indicate no transition took place.
// If a new phase is being set then LastTransitionTime is set appropriately, otherwise it is left untouched.
type Transitioner interface {
	TransitionInto(opsrc *v1alpha1.OperatorSource, nextPhase *NextPhase) (changed bool)
}

// transitioner implements Transitioner interface.s
type transitioner struct {
	clock clock.Clock
}

func (t *transitioner) TransitionInto(opsrc *v1alpha1.OperatorSource, nextPhase *NextPhase) (changed bool) {
	if opsrc == nil || nextPhase == nil {
		return false
	}

	status := &opsrc.Status
	if !hasPhaseChanged(status, nextPhase) {
		return false
	}

	now := metav1.NewTime(t.clock.Now())
	status.LastUpdateTime = now
	status.Message = nextPhase.Message

	if status.Phase != nextPhase.Phase {
		status.LastTransitionTime = now
		status.Phase = nextPhase.Phase
	}

	return true
}

// hasPhaseChanged returns true if the current phase specified in NextPhase
// has changed from that of the given OperatorSourceStatus object.
//
// If both Phase and Message are equal, the function will return false
// indicating no change. Otherwise, the function will return true.
func hasPhaseChanged(status *v1alpha1.OperatorSourceStatus, NextPhase *NextPhase) bool {
	if status.Phase == NextPhase.Phase && status.Message == NextPhase.Message {
		return false
	}

	return true
}
