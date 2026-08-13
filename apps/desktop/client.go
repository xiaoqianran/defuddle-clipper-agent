package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
	AIStatus    string `json:"aiStatus"`
}

type Source struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Site        string `json:"site,omitempty"`
	Author      string `json:"author,omitempty"`
	Published   string `json:"published,omitempty"`
	Language    string `json:"language,omitempty"`
	Description string `json:"description,omitempty"`
}

type Content struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html,omitempty"`
}

type Packet struct {
	CaptureID  string         `json:"captureId"`
	CapturedAt string         `json:"capturedAt"`
	Source     Source         `json:"source"`
	Content    Content        `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Media      map[string]any `json:"media,omitempty"`
}

type CaptureView struct {
	Packet         Packet `json:"packet"`
	SourceMarkdown string `json:"sourceMarkdown"`
	Analysis       any    `json:"analysis,omitempty"`
	Note           string `json:"note,omitempty"`
	AIStatus       string `json:"aiStatus"`
}

type ReprocessResult struct {
	CaptureID string `json:"captureId"`
	NotePath  string `json:"notePath,omitempty"`
	AIStatus  string `json:"aiStatus"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type BrowserPage struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	ObservedAt string `json:"observedAt"`
}

type BrowserState struct {
	Active    bool        `json:"active"`
	Page      BrowserPage `json:"page"`
	UpdatedAt string      `json:"updatedAt"`
}

type Snapshot struct {
	Connected bool             `json:"connected"`
	Browser   BrowserState     `json:"browser"`
	Captures  []CaptureSummary `json:"captures"`
}

type AgentClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func agentTokenFromEnv() string {
	return os.Getenv("DCA_TOKEN")
}

func NewAgentClientFromEnv() (*AgentClient, error) {
	return NewAgentClient(clientURLFromEnv(), agentTokenFromEnv())
}

func NewAgentClient(base, token string) (*AgentClient, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "http://127.0.0.1:27123"
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid DCA_AGENT_URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return nil, fmt.Errorf("DCA_AGENT_URL must point to loopback")
	}
	return &AgentClient{
		baseURL: base,
		token:   token,
		http:    &http.Client{Timeout: 8 * time.Second},
	}, nil
}

func (c *AgentClient) get(path string, out any) error {
	return c.doJSON(http.MethodGet, path, out)
}

func (c *AgentClient) post(path string, out any) error {
	return c.doJSON(http.MethodPost, path, out)
}

func (c *AgentClient) doJSON(method, path string, out any) error {
	req, err := http.NewRequest(method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agentHTTPError(resp.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func agentHTTPError(status int, body []byte) error {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return fmt.Errorf("agent returned HTTP %d: %s", status, payload.Error)
	}
	return fmt.Errorf("agent returned HTTP %d", status)
}

func (c *AgentClient) Snapshot(limit int) (Snapshot, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var browser BrowserState
	if err := c.get("/v1/browser/state", &browser); err != nil {
		return Snapshot{}, err
	}
	var history struct {
		Items []CaptureSummary `json:"items"`
	}
	if err := c.get("/v1/captures?limit="+strconv.Itoa(limit), &history); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Connected: true, Browser: browser, Captures: history.Items}, nil
}

func (c *AgentClient) ReadCapture(captureID string) (CaptureView, error) {
	if captureID == "" {
		return CaptureView{}, fmt.Errorf("capture id is required")
	}
	var view CaptureView
	if err := c.get("/v1/captures/"+url.PathEscape(captureID), &view); err != nil {
		return CaptureView{}, err
	}
	return view, nil
}

func (c *AgentClient) ReprocessCapture(captureID string) (ReprocessResult, error) {
	if captureID == "" {
		return ReprocessResult{}, fmt.Errorf("capture id is required")
	}
	var result ReprocessResult
	if err := c.post("/v1/captures/"+url.PathEscape(captureID)+"/reprocess", &result); err != nil {
		return ReprocessResult{}, err
	}
	return result, nil
}
