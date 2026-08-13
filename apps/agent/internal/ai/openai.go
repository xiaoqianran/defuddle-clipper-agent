package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
}

func NewOpenAICompatible(baseURL, apiKey, model string, chunkChars int, timeout time.Duration) *OpenAICompatible {
	return &OpenAICompatible{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		chunkChars: chunkChars,
		client:     &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatible) Analyze(ctx context.Context, packet protocol.ContentPacket) (Analysis, error) {
	chunks := SplitMarkdown(packet.Content.Markdown, c.chunkChars)
	if len(chunks) == 0 {
		return Analysis{}, errors.New("empty Markdown")
	}
	if len(chunks) == 1 {
		return c.analyzeText(ctx, packet.Source.Title, packet.Source.URL, chunks[0],
			"Analyze the complete captured page.")
	}

	partials := make([]Analysis, 0, len(chunks))
	for i, chunk := range chunks {
		partial, err := c.analyzeText(
			ctx,
			packet.Source.Title,
			packet.Source.URL,
			chunk,
			fmt.Sprintf("This is chunk %d of %d. Summarize only information supported by this chunk.", i+1, len(chunks)),
		)
		if err != nil {
			return Analysis{}, fmt.Errorf("analyze chunk %d/%d: %w", i+1, len(chunks), err)
		}
		partials = append(partials, partial)
	}

	raw, err := json.Marshal(partials)
	if err != nil {
		return Analysis{}, err
	}
	return c.analyzeText(
		ctx,
		packet.Source.Title,
		packet.Source.URL,
		string(raw),
		"Synthesize the following JSON analyses of consecutive chunks into one non-redundant page-level analysis. Do not invent facts not present in the chunk analyses.",
	)
}

func (c *OpenAICompatible) analyzeText(ctx context.Context, title, sourceURL, content, instruction string) (Analysis, error) {
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

	reqBody := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
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
		return Analysis{}, fmt.Errorf("provider HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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

	text := stripCodeFence(envelope.Choices[0].Message.Content)
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
