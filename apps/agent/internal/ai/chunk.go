package ai

import "strings"

// SplitMarkdown keeps headings with the following content when possible and
// falls back to paragraph boundaries. maxChars is an approximate provider
// budget, not a token limit.
func SplitMarkdown(markdown string, maxChars int) []string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return nil
	}
	if len(markdown) <= maxChars {
		return []string{markdown}
	}

	paragraphs := strings.Split(markdown, "\n\n")
	var chunks []string
	var current strings.Builder

	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
	}

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		if len(paragraph) > maxChars {
			flush()
			for len(paragraph) > maxChars {
				cut := maxChars
				if idx := strings.LastIndex(paragraph[:maxChars], "\n"); idx > maxChars/2 {
					cut = idx
				}
				chunks = append(chunks, strings.TrimSpace(paragraph[:cut]))
				paragraph = strings.TrimSpace(paragraph[cut:])
			}
			if paragraph != "" {
				current.WriteString(paragraph)
			}
			continue
		}

		extra := len(paragraph)
		if current.Len() > 0 {
			extra += 2
		}
		if current.Len()+extra > maxChars {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
	}
	flush()
	return chunks
}
