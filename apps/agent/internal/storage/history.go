package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

type CaptureSummary struct {
	CaptureID   string `json:"captureId"`
	CapturedAt  string `json:"capturedAt"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Site        string `json:"site,omitempty"`
	Language    string `json:"language,omitempty"`
	HasAnalysis bool   `json:"hasAnalysis"`
	HasNote     bool   `json:"hasNote"`
}

type CaptureView struct {
	Packet         protocol.ContentPacket `json:"packet"`
	SourceMarkdown string                 `json:"sourceMarkdown"`
	Analysis       any                    `json:"analysis,omitempty"`
	Note           string                 `json:"note,omitempty"`
}

func (s Store) List(limit int) ([]CaptureSummary, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	root := filepath.Join(s.Root, "captures")
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return []CaptureSummary{}, nil
	} else if err != nil {
		return nil, err
	}

	items := make([]CaptureSummary, 0, limit)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "packet.json" {
			return nil
		}

		packet, err := loadPacket(path)
		if err != nil {
			return nil
		}
		dir := filepath.Dir(path)
		_, analysisErr := os.Stat(filepath.Join(dir, "analysis.json"))
		_, noteErr := os.Stat(filepath.Join(dir, "note.md"))
		items = append(items, CaptureSummary{
			CaptureID:   packet.CaptureID,
			CapturedAt:  packet.CapturedAt,
			Title:       packet.Source.Title,
			URL:         packet.Source.URL,
			Site:        packet.Source.Site,
			Language:    packet.Source.Language,
			HasAnalysis: analysisErr == nil,
			HasNote:     noteErr == nil,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool { return items[i].CapturedAt > items[j].CapturedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s Store) Read(captureID string) (CaptureView, error) {
	if !validCaptureID(captureID) {
		return CaptureView{}, os.ErrNotExist
	}

	pattern := filepath.Join(s.Root, "captures", "*", "*", "*", captureID, "packet.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return CaptureView{}, err
	}
	if len(matches) == 0 {
		return CaptureView{}, os.ErrNotExist
	}

	packetPath := matches[0]
	packet, err := loadPacket(packetPath)
	if err != nil {
		return CaptureView{}, err
	}
	dir := filepath.Dir(packetPath)
	view := CaptureView{Packet: packet, SourceMarkdown: packet.Content.Markdown}

	if raw, err := os.ReadFile(filepath.Join(dir, "source.md")); err == nil {
		view.SourceMarkdown = string(raw)
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "analysis.json")); err == nil {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			view.Analysis = value
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "note.md")); err == nil {
		view.Note = string(raw)
	}
	return view, nil
}

func loadPacket(path string) (protocol.ContentPacket, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return protocol.ContentPacket{}, err
	}
	var packet protocol.ContentPacket
	if err := json.Unmarshal(raw, &packet); err != nil {
		return protocol.ContentPacket{}, err
	}
	return packet, nil
}

func validCaptureID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-", r) {
			continue
		}
		return false
	}
	return true
}
