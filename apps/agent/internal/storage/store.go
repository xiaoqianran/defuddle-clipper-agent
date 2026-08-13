package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

const (
	StatusPending  = "pending"
	StatusOK       = "ok"
	StatusFailed   = "failed"
	StatusDisabled = "disabled"
	StatusUnknown  = "unknown"
)

type Store struct {
	Root string
}

type Paths struct {
	Dir      string
	Packet   string
	Source   string
	Analysis string
	Error    string
	Note     string
	Pending  string
}

func NewPaths(dir string) Paths {
	return Paths{
		Dir:      dir,
		Packet:   filepath.Join(dir, "packet.json"),
		Source:   filepath.Join(dir, "source.md"),
		Analysis: filepath.Join(dir, "analysis.json"),
		Error:    filepath.Join(dir, "analysis-error.txt"),
		Note:     filepath.Join(dir, "note.md"),
		Pending:  filepath.Join(dir, "analysis-pending"),
	}
}

func (s Store) SavePacket(packet protocol.ContentPacket) (Paths, bool, error) {
	capturedAt, err := time.Parse(time.RFC3339Nano, packet.CapturedAt)
	if err != nil {
		return Paths{}, false, err
	}

	dir := filepath.Join(
		s.Root,
		"captures",
		capturedAt.Format("2006"),
		capturedAt.Format("01"),
		capturedAt.Format("02"),
		packet.CaptureID,
	)
	paths := NewPaths(dir)

	if _, err := os.Stat(paths.Packet); err == nil {
		return paths, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Paths{}, false, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Paths{}, false, err
	}

	raw, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return Paths{}, false, err
	}
	raw = append(raw, '\n')

	if err := atomicWrite(paths.Packet, raw, 0o600); err != nil {
		return Paths{}, false, fmt.Errorf("write packet: %w", err)
	}
	if err := atomicWrite(paths.Source, []byte(packet.Content.Markdown+"\n"), 0o600); err != nil {
		return Paths{}, false, fmt.Errorf("write source: %w", err)
	}
	return paths, false, nil
}

func (s Store) NoteExists(paths Paths) bool {
	return exists(paths.Note)
}

// DerivedComplete reports whether derived artifacts finished (success, failure, or
// AI disabled). A pending marker means analysis is still in flight, so retries
// and reprocess must be allowed to proceed.
func (s Store) DerivedComplete(paths Paths) bool {
	return paths.Complete()
}

func (p Paths) Complete() bool {
	return exists(p.Note) && !exists(p.Pending)
}

func (p Paths) AIStatus() string {
	switch {
	case exists(p.Pending):
		return StatusPending
	case exists(p.Analysis):
		return StatusOK
	case exists(p.Error):
		return StatusFailed
	case exists(p.Note):
		return StatusDisabled
	default:
		return StatusUnknown
	}
}

func (s Store) MarkPending(paths Paths, note string) error {
	if err := atomicWrite(paths.Pending, []byte("pending\n"), 0o600); err != nil {
		return fmt.Errorf("write pending marker: %w", err)
	}
	if note != "" {
		if err := atomicWrite(paths.Note, []byte(note), 0o600); err != nil {
			return fmt.Errorf("write pending note: %w", err)
		}
	}
	return nil
}

func (s Store) WriteDerived(paths Paths, analysis *ai.Analysis, aiErr error, note string) error {
	if analysis != nil {
		raw, err := json.MarshalIndent(analysis, "", "  ")
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		if err := atomicWrite(paths.Analysis, raw, 0o600); err != nil {
			return err
		}
	} else {
		_ = os.Remove(paths.Analysis)
	}

	if aiErr != nil {
		if err := atomicWrite(paths.Error, []byte(aiErr.Error()+"\n"), 0o600); err != nil {
			return err
		}
	} else {
		_ = os.Remove(paths.Error)
	}

	if err := atomicWrite(paths.Note, []byte(note), 0o600); err != nil {
		return err
	}
	_ = os.Remove(paths.Pending)
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
