// Package policy 是捕获控制面。桌面写入，扩展拉取；文件可重建。
package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	fileName          = "settings.json"
	minCaptureDelayMs = 250
	maxCaptureDelayMs = 30_000
	defaultDelayMs    = 1200
)

type Document struct {
	Revision        int      `json:"revision"`
	AutoCapture     bool     `json:"autoCapture"`
	ArchiveAll      bool     `json:"archiveAll"`
	CaptureDelayMs  int      `json:"captureDelayMs"`
	DomainAllowlist []string `json:"domainAllowlist"`
	DomainDenylist  []string `json:"domainDenylist"`
}

type Store struct {
	mu      sync.Mutex
	path    string
	current Document
}

func Default() Document {
	return Document{
		Revision:        1,
		AutoCapture:     true,
		ArchiveAll:      true,
		CaptureDelayMs:  defaultDelayMs,
		DomainAllowlist: []string{},
		DomainDenylist:  []string{},
	}
}

func Memory() *Store {
	return &Store{current: Default()}
}

func Load(dir string) (*Store, error) {
	store := &Store{
		path:    filepath.Join(dir, fileName),
		current: Default(),
	}
	raw, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	store.current = normalize(doc)
	if store.current.Revision < 1 {
		store.current.Revision = 1
	}
	return store, nil
}

func (s *Store) Get() Document {
	if s == nil {
		return Default()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(normalize(s.current))
}

func (s *Store) Put(next Document) (Document, error) {
	if s == nil {
		return Default(), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := normalize(next)
	doc.Revision = s.current.Revision + 1
	if s.path != "" {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return Document{}, err
		}
		raw, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return Document{}, err
		}
		raw = append(raw, '\n')
		if err := atomicWrite(s.path, raw, 0o600); err != nil {
			return Document{}, err
		}
	}
	s.current = doc
	return clone(doc), nil
}

func normalize(doc Document) Document {
	if doc.CaptureDelayMs < minCaptureDelayMs || doc.CaptureDelayMs > maxCaptureDelayMs {
		doc.CaptureDelayMs = defaultDelayMs
	}
	if doc.DomainAllowlist == nil {
		doc.DomainAllowlist = []string{}
	}
	if doc.DomainDenylist == nil {
		doc.DomainDenylist = []string{}
	}
	doc.DomainAllowlist = compactDomains(doc.DomainAllowlist)
	doc.DomainDenylist = compactDomains(doc.DomainDenylist)
	return doc
}

func compactDomains(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		domain := normalizeDomain(value)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}
	return out
}

func normalizeDomain(domain string) string {
	start, end := 0, len(domain)
	for start < end && (domain[start] == ' ' || domain[start] == '.') {
		start++
	}
	for end > start && (domain[end-1] == ' ' || domain[end-1] == '.') {
		end--
	}
	if start >= end {
		return ""
	}
	buf := make([]byte, end-start)
	for i := start; i < end; i++ {
		c := domain[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[i-start] = c
	}
	return string(buf)
}

func clone(doc Document) Document {
	doc.DomainAllowlist = append([]string(nil), doc.DomainAllowlist...)
	doc.DomainDenylist = append([]string(nil), doc.DomainDenylist...)
	return doc
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
