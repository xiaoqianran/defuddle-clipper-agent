package browserstate

import (
	"sync"
	"time"
)

type Page struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	ObservedAt string `json:"observedAt"`
	TabID      *int   `json:"tabId,omitempty"`
	WindowID   *int   `json:"windowId,omitempty"`
}

type State struct {
	Active    bool   `json:"active"`
	Page      Page   `json:"page"`
	UpdatedAt string `json:"updatedAt"`
}

type Store struct {
	mu    sync.RWMutex
	state State
}

func (s *Store) Set(page Page) State {
	s.mu.Lock()
	defer s.mu.Unlock()

	if page.ObservedAt == "" {
		page.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	s.state = State{
		Active:    true,
		Page:      page,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return s.state
}

func (s *Store) Get() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}
