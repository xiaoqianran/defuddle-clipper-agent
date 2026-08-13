package events

import (
	"testing"
	"time"
)

func TestPublishReachesSubscriber(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe()
	defer cancel()

	hub.Publish(Event{Type: CaptureSaved, CaptureID: "cap-1"})

	select {
	case ev := <-ch:
		if ev.Type != CaptureSaved || ev.CaptureID != "cap-1" || ev.At == "" {
			t.Fatalf("event=%+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublishDoesNotBlockWhenBufferFull(t *testing.T) {
	hub := NewHub()
	_, cancel := hub.Subscribe()
	defer cancel()

	done := make(chan struct{})
	go func() {
		for i := 0; i < subscriberBuffer+8; i++ {
			hub.Publish(Event{Type: CaptureUpdated, CaptureID: "cap"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}

func TestCancelStopsDelivery(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe()
	cancel()
	hub.Publish(Event{Type: PolicyUpdated})
	if _, ok := <-ch; ok {
		t.Fatal("expected closed subscription")
	}
}
