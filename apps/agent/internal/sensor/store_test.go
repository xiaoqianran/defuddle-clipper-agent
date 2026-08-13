package sensor

import (
	"testing"
	"time"
)

func TestBeatMarksConnected(t *testing.T) {
	var store Store
	view := store.Beat(Heartbeat{QueueLength: 3, LastError: "agent down", Version: "0.1.0"})
	if !view.Connected || view.QueueLength != 3 || view.LastError != "agent down" || view.SeenAt == "" {
		t.Fatalf("view=%+v", view)
	}
}

func TestStaleHeartbeatDisconnects(t *testing.T) {
	var store Store
	store.Beat(Heartbeat{})
	store.mu.Lock()
	store.seenAt = time.Now().UTC().Add(-time.Minute)
	store.mu.Unlock()
	view := store.View(DefaultFreshness)
	if view.Connected {
		t.Fatalf("expected disconnected, got %+v", view)
	}
}
