package operatorstatus

import (
	"sync"
)

// eventTracker is used to track the number of sync and phase transition failures.
type eventTracker struct {
	// syncs represents the number of recorded sync events.
	syncs uint

	// failedTransitions represents the number of recorded phase transition failures.
	failedTransitions uint

	// syncLimit is the value that syncs must reach before the sync
	// and failedTransitions fields are divided by the truncateValue.
	syncLimit uint

	// truncateValue is the value that the syncs and failedTransitions will be truncated by.
	truncateValue uint

	// lock is used to prevent multiple routines from modifying the number of syncs and failedTransitions at the same time.
	lock sync.Mutex
}

// incrementSyncs will increment the syncs field by one.
func (e *eventTracker) incrementSyncs() {
	e.lock.Lock()
	defer func() {
		e.lock.Unlock()
	}()
	e.syncs++
	e.preventOverflow()
}

// incrementFailedTransitions will increment the failedTranisitions field by one.
func (e *eventTracker) incrementFailedTransitions() {
	e.lock.Lock()
	defer func() {
		e.lock.Unlock()
	}()
	e.failedTransitions++
}

// getEvents returns the number of syncs and failed transitions.
func (e *eventTracker) getEvents() (uint, uint) {
	e.lock.Lock()
	defer func() {
		e.lock.Unlock()
	}()
	return e.syncs, e.failedTransitions
}

// preventOverflow prevents the sync and failedTransition fields from overflowing
// in long-running marketplace instances.
func (e *eventTracker) preventOverflow() {
	if e.syncs >= e.syncLimit {
		e.syncs = e.syncs / e.truncateValue
		e.failedTransitions = e.failedTransitions / e.truncateValue
	}
}
