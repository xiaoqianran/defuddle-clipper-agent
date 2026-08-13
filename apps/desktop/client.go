package main

import (
	"encoding/json"
	"fmt"
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

func NewAgentClientFromEnv() (*AgentClient, error) {
	base := strings.TrimRight(os.Getenv("DCA_AGENT_URL"), "/")
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
		token:   os.Getenv("DCA_TOKEN"),
		http:    &http.Client{Timeout: 8 * time.Second},
	}, nil
}

func (c *AgentClient) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
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
