package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

type OpenAICompatible struct {
	baseURL    string
	apiKey     string
	model      string
	chunkChars int
	client     *http.Client
	now        func() time.Time
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type contentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *imageURLPart `json:"image_url,omitempty"`
}

type imageURLPart struct {
	URL string `json:"url"`
}

func NewOpenAICompatible(baseURL, apiKey, model string, chunkChars int, timeout time.Duration) *OpenAICompatible {
	return &OpenAICompatible{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		chunkChars: chunkChars,
		client:     &http.Client{Timeout: timeout},
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (c *OpenAICompatible) Analyze(ctx context.Context, packet protocol.ContentPacket) (Analysis, error) {
	chunks := SplitMarkdown(packet.Content.Markdown, c.chunkChars)
	if len(chunks) == 0 {
		return Analysis{}, errors.New("empty Markdown")
	}

	imageURL := usableCoverImageURL(packet)
	imageSent := false
	imageSkipped := ""
	var analysis Analysis
	var err error
	if len(chunks) == 1 {
		analysis, imageSent, err = c.analyzeText(ctx, packet.Source.Title, packet.Source.URL, chunks[0],
			"Analyze the complete captured page.", imageURL)
		if err == nil && imageURL != "" && !imageSent {
			imageSkipped = ImageSkippedProviderMediaFetch
		}
	} else {
		partials := make([]Analysis, 0, len(chunks))
		droppedImage := false
		for i, chunk := range chunks {
			partial, usedImage, chunkErr := c.analyzeText(
				ctx,
				packet.Source.Title,
				packet.Source.URL,
				chunk,
				fmt.Sprintf("This is chunk %d of %d. Summarize only information supported by this chunk.", i+1, len(chunks)),
				imageURL,
			)
			if chunkErr != nil {
				return Analysis{}, fmt.Errorf("analyze chunk %d/%d: %w", i+1, len(chunks), chunkErr)
			}
			if imageURL != "" && !usedImage {
				droppedImage = true
				imageURL = ""
			}
			if usedImage {
				imageSent = true
			}
			partials = append(partials, partial)
		}
		if droppedImage {
			imageSent = false
			imageSkipped = ImageSkippedProviderMediaFetch
		}

		raw, marshalErr := json.Marshal(partials)
		if marshalErr != nil {
			return Analysis{}, marshalErr
		}
		analysis, _, err = c.analyzeText(
			ctx,
			packet.Source.Title,
			packet.Source.URL,
			string(raw),
			"Synthesize the following JSON analyses of consecutive chunks into one non-redundant page-level analysis. Do not invent facts not present in the chunk analyses.",
			"",
		)
	}
	if err != nil {
		return Analysis{}, err
	}
	c.attachProvenance(&analysis, imageSent, imageSkipped)
	return analysis, nil
}

func (c *OpenAICompatible) analyzeText(ctx context.Context, title, sourceURL, content, instruction, imageURL string) (Analysis, bool, error) {
	system := `You are a precise knowledge extraction engine.
Return ONLY one valid JSON object with exactly this semantic shape:
{
  "pageType": "short page/content type",
  "summary": "concise but information-dense summary",
  "keyPoints": ["..."],
  "concepts": [{"name":"...", "explanation":"..."}],
  "conclusions": ["..."],
  "actions": ["..."],
  "questions": ["..."],
  "tags": ["..."]
}
Use the source language unless translation is explicitly required.
Distinguish claims from conclusions. Do not fabricate missing facts.
Keep tags concise. Arrays may be empty.`

	user := fmt.Sprintf(
		"%s\n\nTITLE:\n%s\n\nSOURCE:\n%s\n\nCONTENT:\n%s",
		instruction, title, sourceURL, content,
	)

	analysis, err := c.chatCompletions(ctx, system, user, imageURL)
	if err != nil {
		if imageURL != "" && isProviderMediaFetchError(err) {
			analysis, err = c.chatCompletions(ctx, system, user, "")
			if err != nil {
				return Analysis{}, false, err
			}
			return analysis, false, nil
		}
		return Analysis{}, false, err
	}
	return analysis, imageURL != "", nil
}

type providerHTTPError struct {
	StatusCode int
	Body       string
}

func (e *providerHTTPError) Error() string {
	return fmt.Sprintf("provider HTTP %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
}

func isProviderMediaFetchError(err error) bool {
	var pe *providerHTTPError
	if !errors.As(err, &pe) {
		return false
	}
	if pe.StatusCode < 400 || pe.StatusCode >= 500 || pe.StatusCode == http.StatusUnauthorized {
		return false
	}
	return mediaFetchBlocked(pe.Body)
}

func mediaFetchBlocked(body string) bool {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "failed media") || strings.Contains(lower, "fetch media") {
		return true
	}
	if strings.Contains(lower, "upstream returned http 403") || strings.Contains(lower, "upstream returned http 401") {
		return true
	}
	return false
}

func (c *OpenAICompatible) chatCompletions(ctx context.Context, system, user, imageURL string) (Analysis, error) {
	reqBody := map[string]any{
		"model":       c.model,
		"messages":    []chatMessage{systemMessage(system), userMessage(user, imageURL)},
		"temperature": 0.2,
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return Analysis{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return Analysis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return Analysis{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Analysis{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Analysis{}, &providerHTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Analysis{}, fmt.Errorf("decode provider response: %w", err)
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return Analysis{}, errors.New("provider returned no message content")
	}

	return decodeAnalysisJSON(envelope.Choices[0].Message.Content)
}

func systemMessage(text string) chatMessage {
	return chatMessage{Role: "system", Content: text}
}

func userMessage(text, imageURL string) chatMessage {
	if imageURL == "" {
		return chatMessage{Role: "user", Content: text}
	}
	return chatMessage{
		Role: "user",
		Content: []contentPart{
			{Type: "text", Text: text},
			{Type: "image_url", ImageURL: &imageURLPart{URL: imageURL}},
		},
	}
}

func usableCoverImageURL(packet protocol.ContentPacket) string {
	if packet.Metadata == nil {
		return ""
	}
	raw, ok := packet.Metadata["image"]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	return value
}

func (c *OpenAICompatible) attachProvenance(analysis *Analysis, imageSent bool, imageSkipped string) {
	analysis.Provenance = Provenance{
		Model:         c.model,
		ProviderHost:  providerHost(c.baseURL),
		PromptVersion: PromptVersion,
		ImageSent:     imageSent,
		ImageSkipped:  imageSkipped,
		AnalyzedAt:    c.now().Format(time.RFC3339),
	}
}

func providerHost(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Host
}

func decodeAnalysisJSON(text string) (Analysis, error) {
	text = extractFirstJSONObject(stripCodeFence(strings.TrimSpace(text)))
	var analysis Analysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return Analysis{}, fmt.Errorf("decode analysis JSON: %w", err)
	}
	return analysis, nil
}

func stripCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) >= 3 && strings.HasPrefix(lines[0], "```") && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
	}
	return text
}

func extractFirstJSONObject(text string) string {
	text = strings.TrimSpace(text)
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return text
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(text[start : i+1])
			}
		}
	}
	return text
}
