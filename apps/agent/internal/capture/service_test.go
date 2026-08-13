package capture

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

type fakeAnalyzer struct{}

func (fakeAnalyzer) Analyze(context.Context, protocol.ContentPacket) (ai.Analysis, error) {
	return ai.Analysis{Summary: "ok", Tags: []string{"tag"}}, nil
}

func TestProcessPersistsRawAndDerived(t *testing.T) {
	root := t.TempDir()
	service := Service{Store: storage.Store{Root: root}, Analyzer: fakeAnalyzer{}}
	packet := protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       "capture-1",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: protocol.Source{
			URL:   "https://example.com",
			Title: "Title",
		},
		Content: protocol.Content{Markdown: "Body"},
	}

	result, err := service.Process(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != "ok" {
		t.Fatalf("unexpected AI status: %s", result.AIStatus)
	}

	dir := filepath.Join(root, "captures", "2026", "08", "13", "capture-1")
	for _, name := range []string{"packet.json", "source.md", "analysis.json", "note.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}

	second, err := service.Process(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatal("expected idempotent duplicate")
	}
}
