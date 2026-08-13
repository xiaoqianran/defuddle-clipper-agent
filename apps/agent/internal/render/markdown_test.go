package render

import (
	"strings"
	"testing"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

func TestMarkdown(t *testing.T) {
	packet := protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       "id",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: protocol.Source{
			URL:   "https://example.com",
			Title: "Title",
		},
		Content: protocol.Content{Markdown: "Original"},
	}
	analysis := &ai.Analysis{Summary: "Summary", KeyPoints: []string{"One"}, Tags: []string{"test"}}
	got := Markdown(packet, analysis, nil)

	for _, want := range []string{"# Title", "## AI Summary", "Summary", "## Key Points", "Original"} {
		if !strings.Contains(got, want) {
			t.Fatalf("render missing %q:\n%s", want, got)
		}
	}

	pending := PendingMarkdown(packet)
	if !strings.Contains(pending, "Analysis is pending") {
		t.Fatalf("pending render missing status:\n%s", pending)
	}
}
