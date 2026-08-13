package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

func Markdown(packet protocol.ContentPacket, analysis *ai.Analysis, aiErr error) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("title: " + strconv.Quote(packet.Source.Title) + "\n")
	b.WriteString("source: " + strconv.Quote(packet.Source.URL) + "\n")
	b.WriteString("captured: " + strconv.Quote(packet.CapturedAt) + "\n")
	if packet.Source.Site != "" {
		b.WriteString("site: " + strconv.Quote(packet.Source.Site) + "\n")
	}
	if packet.Source.Author != "" {
		b.WriteString("author: " + strconv.Quote(packet.Source.Author) + "\n")
	}
	if analysis != nil && len(analysis.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, tag := range analysis.Tags {
			b.WriteString("  - " + strconv.Quote(tag) + "\n")
		}
	}
	b.WriteString("---\n\n")

	b.WriteString("# " + packet.Source.Title + "\n\n")
	b.WriteString(fmt.Sprintf("> Source: %s\n>\n> Captured: %s\n\n", packet.Source.URL, packet.CapturedAt))

	if analysis != nil {
		section(&b, "AI Summary", analysis.Summary)
		listSection(&b, "Key Points", analysis.KeyPoints)

		if len(analysis.Concepts) > 0 {
			b.WriteString("## Concepts\n\n")
			for _, concept := range analysis.Concepts {
				b.WriteString("### " + concept.Name + "\n\n" + concept.Explanation + "\n\n")
			}
		}
		listSection(&b, "Conclusions", analysis.Conclusions)
		listSection(&b, "Actions", analysis.Actions)
		listSection(&b, "Questions", analysis.Questions)
	} else if aiErr != nil {
		b.WriteString("## AI Analysis\n\n")
		b.WriteString("> Analysis failed. The original capture is preserved and can be reprocessed later.\n\n")
	}

	if packet.Media != nil {
		if transcript, ok := packet.Media["transcript"].(string); ok && strings.TrimSpace(transcript) != "" {
			section(&b, "Transcript", transcript)
		}
	}

	b.WriteString("## Original Content\n\n")
	b.WriteString(strings.TrimSpace(packet.Content.Markdown))
	b.WriteString("\n")

	return b.String()
}

func section(b *strings.Builder, title, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	b.WriteString("## " + title + "\n\n")
	b.WriteString(strings.TrimSpace(body) + "\n\n")
}

func listSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("## " + title + "\n\n")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			b.WriteString("- " + item + "\n")
		}
	}
	b.WriteString("\n")
}
