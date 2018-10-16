package phase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/operator-marketplace/pkg/apis/marketplace/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource/phase"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

// Use Case: Phase and Message specified in both objects are identical.
// Expected Result: The function is expected to return false to indicate that no
// transition has taken place.
func TestTransitionInto_IdenticalPhase_FalseExpected(t *testing.T) {
	clock := clock.NewFakeClock(time.Now())
	transitioner := phase.NewTransitionerWithClock(clock)

	phaseWant, messageWant := "Validating", "Scheduled for validation"

	opsrcIn := &v1alpha1.OperatorSource{
		Status: v1alpha1.OperatorSourceStatus{
			Phase:   phaseWant,
			Message: messageWant,
		},
	}

	nextPhase := &phase.NextPhase{
		Phase:   phaseWant,
		Message: messageWant,
	}

	changedGot := transitioner.TransitionInto(opsrcIn, nextPhase)

	assert.False(t, changedGot)
	assert.Equal(t, phaseWant, opsrcIn.Status.Phase)
	assert.Equal(t, messageWant, opsrcIn.Status.Message)
}

// Use Case: Both Phase and Message specified in both objects are different.
// Expected Result: The function is expected to return true to indicate that a
// transition has taken place.
func TestTransitionInto_BothPhaseAndMessageAreDifferent_TrueExpected(t *testing.T) {
	now := time.Now()

	clock := clock.NewFakeClock(now)
	transitioner := phase.NewTransitionerWithClock(clock)

	phaseWant, messageWant := "Validating", "Scheduled for validation"

	opsrcIn := &v1alpha1.OperatorSource{
		Status: v1alpha1.OperatorSourceStatus{
			Phase:   "Initial",
			Message: "Not validated",
		},
	}

	nextPhase := &phase.NextPhase{
		Phase:   phaseWant,
		Message: messageWant,
	}

	changedGot := transitioner.TransitionInto(opsrcIn, nextPhase)

	assert.True(t, changedGot)
	assert.Equal(t, phaseWant, opsrcIn.Status.Phase)
	assert.Equal(t, messageWant, opsrcIn.Status.Message)
	assert.Equal(t, metav1.NewTime(now), opsrcIn.Status.LastTransitionTime)
	assert.Equal(t, metav1.NewTime(now), opsrcIn.Status.LastUpdateTime)
}

// Use Case: Phase specified in both objects are same but Message is different.
// Expected Result: The function is expected to return true to indicate that an
// update has taken place. LastTransitionTime is expected not to be changed.
func TestTransitionInto_MessageIsDifferent_TrueExpected(t *testing.T) {
	now := time.Now()
	clock := clock.NewFakeClock(now)
	transitioner := phase.NewTransitionerWithClock(clock)

	phaseWant, messageWant := "Failed", "Second try- reason 2"

	opsrcIn := &v1alpha1.OperatorSource{
		Status: v1alpha1.OperatorSourceStatus{
			Phase:   phaseWant,
			Message: "First try- reason 1",
		},
	}

	nextPhase := &phase.NextPhase{
		Phase:   phaseWant,
		Message: messageWant,
	}

	changedGot := transitioner.TransitionInto(opsrcIn, nextPhase)

	assert.True(t, changedGot)
	assert.Equal(t, phaseWant, opsrcIn.Status.Phase)
	assert.Equal(t, messageWant, opsrcIn.Status.Message)
	assert.Empty(t, opsrcIn.Status.LastTransitionTime)
	assert.Equal(t, metav1.NewTime(now), opsrcIn.Status.LastUpdateTime)
}
