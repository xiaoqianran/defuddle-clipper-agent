// Package events 是进程内扇出总线。HTTP SSE 与桌面订阅共用，不落盘。
package events

import (
	"sync"
	"time"
)

const (
	CaptureSaved     = "capture.saved"
	CaptureUpdated   = "capture.updated"
	BrowserActive    = "browser.active"
	PolicyUpdated    = "policy.updated"
	SensorHeartbeat  = "sensor.heartbeat"
	subscriberBuffer = 16
)

type Event struct {
	Type      string `json:"type"`
	CaptureID string `json:"captureId,omitempty"`
	At        string `json:"at"`
}

type Hub struct {
	mu   sync.Mutex
	next int
	subs map[int]chan Event
}

func NewHub() *Hub {
	return &Hub{subs: make(map[int]chan Event)}
}

func (h *Hub) Publish(ev Event) {
	if h == nil {
		return
	}
	if ev.At == "" {
		ev.At = time.Now().UTC().Format(time.RFC3339Nano)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *Hub) Subscribe() (<-chan Event, func()) {
	if h == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan Event, subscriberBuffer)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if existing, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(existing)
		}
		h.mu.Unlock()
	}
}
