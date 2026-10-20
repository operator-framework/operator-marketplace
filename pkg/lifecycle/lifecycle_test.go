package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeManager returns startErr from Start, or blocks until ctx is done when
// startErr is nil, like a healthy controller-runtime manager.
type fakeManager struct{ startErr error }

func (f fakeManager) Start(ctx context.Context) error {
	if f.startErr != nil {
		return f.startErr
	}
	<-ctx.Done()
	return nil
}

// fakeReporter closes its done channel only when ctx is done, exactly like
// status.reporter, whose stopCh is the leader-election ctx.Done().
type fakeReporter struct{ ctx context.Context }

func (f fakeReporter) StartReporting() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		<-f.ctx.Done()
		close(done)
	}()
	return done
}

// runWithTimeout returns the RunManager result, or fails the test if RunManager
// does not return within one second. A hang here is the OCPBUGS-130575 zombie.
func runWithTimeout(t *testing.T, ctx context.Context, cancel context.CancelFunc, mgr Starter, state *State) error {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- RunManager(ctx, cancel, mgr, fakeReporter{ctx: ctx}, state) }()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("RunManager did not return within 1s after the manager stopped: zombie")
		return nil
	}
}

func TestRunManagerReturnsWhenManagerFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startErr := errors.New("failed to wait for caches to sync")
	var state State

	err := runWithTimeout(t, ctx, cancel, fakeManager{startErr: startErr}, &state)

	if !errors.Is(err, startErr) {
		t.Fatalf("expected the manager error %q, got %v", startErr, err)
	}
	if ctx.Err() == nil {
		t.Fatal("expected the leader-election context to be cancelled so the lease is released")
	}
	if !state.Failed.Load() || !state.Stopped.Load() {
		t.Fatalf("expected Failed and Stopped after a manager error, got failed=%v stopped=%v", state.Failed.Load(), state.Stopped.Load())
	}
}

func TestRunManagerReturnsNilOnNormalStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Simulate lost leadership: OnStoppedLeading cancels the context.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	var state State

	err := runWithTimeout(t, ctx, cancel, fakeManager{}, &state)

	if err != nil {
		t.Fatalf("expected nil on a normal stop, got %v", err)
	}
	if state.Failed.Load() {
		t.Fatal("expected Failed to stay false on a normal stop")
	}
}

// blockedReporter never finishes, like a reporter stuck in its first status
// update. RunManager must still publish the stopped state.
type blockedReporter struct{}

func (blockedReporter) StartReporting() <-chan struct{} { return make(chan struct{}) }

func TestRunManagerPublishesStateBeforeReporterFinishes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var state State

	go RunManager(ctx, cancel, fakeManager{startErr: errors.New("boom")}, blockedReporter{}, &state)

	// The cancel is what leader election and main observe. Both flags must be
	// visible by then, while RunManager still waits for the reporter.
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context not cancelled within 1s while the reporter blocks")
	}
	if !state.Stopped.Load() {
		t.Fatal("Stopped not set at cancel time: healthz would stay 200")
	}
	if !state.Failed.Load() {
		t.Fatal("Failed not set at cancel time: main would exit 0")
	}
}

func TestRunManagerMarksStoppedOnNormalStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var state State

	_ = runWithTimeout(t, ctx, cancel, fakeManager{}, &state)

	if !state.Stopped.Load() {
		t.Fatal("expected the stopped flag after a normal manager stop, so healthz fails while the process winds down")
	}
}

func TestHealthzFollowsManagerState(t *testing.T) {
	var state State
	handler := Healthz(&state)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 while the manager runs, got %d", rec.Code)
	}

	state.Stopped.Store(true)
	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after the manager stopped, got %d", rec.Code)
	}
}
