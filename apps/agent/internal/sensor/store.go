// Package sensor 记录扩展心跳。桌面用它判断传感器是否还活着。
package sensor

import (
	"sync"
	"time"
)

const DefaultFreshness = 15 * time.Second

type Heartbeat struct {
	QueueLength int    `json:"queueLength"`
	LastError   string `json:"lastError,omitempty"`
	Version     string `json:"version,omitempty"`
}

type View struct {
	Connected   bool   `json:"connected"`
	SeenAt      string `json:"seenAt,omitempty"`
	QueueLength int    `json:"queueLength"`
	LastError   string `json:"lastError,omitempty"`
	Version     string `json:"version,omitempty"`
}

type Store struct {
	mu        sync.Mutex
	heartbeat Heartbeat
	seenAt    time.Time
}

func (s *Store) Beat(hb Heartbeat) View {
	if s == nil {
		return View{}
	}
	if hb.QueueLength < 0 {
		hb.QueueLength = 0
	}
	s.mu.Lock()
	s.heartbeat = hb
	s.seenAt = time.Now().UTC()
	s.mu.Unlock()
	return s.View(DefaultFreshness)
}

func (s *Store) View(fresh time.Duration) View {
	if s == nil {
		return View{}
	}
	if fresh <= 0 {
		fresh = DefaultFreshness
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seenAt.IsZero() {
		return View{}
	}
	return View{
		Connected:   time.Since(s.seenAt) <= fresh,
		SeenAt:      s.seenAt.Format(time.RFC3339Nano),
		QueueLength: s.heartbeat.QueueLength,
		LastError:   s.heartbeat.LastError,
		Version:     s.heartbeat.Version,
	}
}
