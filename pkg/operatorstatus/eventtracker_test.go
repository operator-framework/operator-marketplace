package operatorstatus

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventTrackerInit(t *testing.T) {
	e := eventTracker{0, 0, 100, 10, sync.Mutex{}}

	syncs, failures := e.getEvents()
	assert.Equal(t, uint(0), syncs)
	assert.Equal(t, uint(0), failures)
}

func TestEventTrackerSyncEvent(t *testing.T) {
	e := eventTracker{0, 0, 100, 10, sync.Mutex{}}

	for i := 0; i < 10; i++ {
		e.incrementSyncs()
	}

	for i := 0; i < 3; i++ {
		e.incrementFailedTransitions()
	}
	syncs, failures := e.getEvents()
	assert.Equal(t, uint(10), syncs)
	assert.Equal(t, uint(3), failures)
}

func TestEventTrackerTruncate(t *testing.T) {
	e := eventTracker{0, 0, 100, 10, sync.Mutex{}}

	for i := 0; i < 21; i++ {
		e.incrementFailedTransitions()
	}

	for i := 0; i < 100; i++ {
		e.incrementSyncs()
	}

	syncs, failures := e.getEvents()
	assert.Equal(t, uint(10), syncs)
	assert.Equal(t, uint(2), failures)
}
