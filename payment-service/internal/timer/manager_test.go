package timer

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestManagerStartFiresExpiryCallback(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	done := make(chan struct{})
	var fired atomic.Int32

	mgr.Start("A1", 20*time.Millisecond, func() {
		fired.Add(1)
		close(done)
	})

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected timer callback to fire")
	}

	if fired.Load() != 1 {
		t.Fatalf("expected callback to fire once, got %d", fired.Load())
	}
}

func TestManagerStopPreventsExpiryCallback(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	var fired atomic.Int32

	mgr.Start("A2", 100*time.Millisecond, func() {
		fired.Add(1)
	})
	mgr.Stop("A2")

	time.Sleep(200 * time.Millisecond)

	if fired.Load() != 0 {
		t.Fatalf("expected callback not to fire after Stop, got %d", fired.Load())
	}
}

