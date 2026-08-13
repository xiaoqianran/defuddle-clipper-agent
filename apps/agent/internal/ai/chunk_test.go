package ai

import (
	"strings"
	"testing"
)

func TestSplitMarkdown(t *testing.T) {
	input := strings.Repeat("paragraph one\n\n", 30)
	chunks := SplitMarkdown(input, 120)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for _, chunk := range chunks {
		if len(chunk) > 140 {
			t.Fatalf("unexpected oversized chunk: %d", len(chunk))
		}
	}
}
