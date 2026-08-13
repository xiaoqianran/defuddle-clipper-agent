package main

import (
	"bufio"
	"bytes"
	"context"
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

type Policy struct {
	Revision        int      `json:"revision"`
	AutoCapture     bool     `json:"autoCapture"`
	ArchiveAll      bool     `json:"archiveAll"`
	CaptureDelayMs  int      `json:"captureDelayMs"`
	DomainAllowlist []string `json:"domainAllowlist"`
	DomainDenylist  []string `json:"domainDenylist"`
}

type SensorView struct {
	Connected   bool   `json:"connected"`
	SeenAt      string `json:"seenAt,omitempty"`
	QueueLength int    `json:"queueLength"`
	LastError   string `json:"lastError,omitempty"`
	Version     string `json:"version,omitempty"`
}

type ChangeEvent struct {
	Type      string `json:"type"`
	CaptureID string `json:"captureId,omitempty"`
	At        string `json:"at"`
}

type Snapshot struct {
	Connected bool             `json:"connected"`
	Browser   BrowserState     `json:"browser"`
	Captures  []CaptureSummary `json:"captures"`
	Policy    Policy           `json:"policy"`
	Sensor    SensorView       `json:"sensor"`
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
	return c.doJSON(http.MethodGet, path, nil, out)
}

func (c *AgentClient) post(path string, out any) error {
	return c.doJSON(http.MethodPost, path, nil, out)
}

func (c *AgentClient) put(path string, body any, out any) error {
	return c.doJSON(http.MethodPut, path, body, out)
}

func (c *AgentClient) doJSON(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agentHTTPError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
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
	var status struct {
		Browser  BrowserState     `json:"browser"`
		Captures []CaptureSummary `json:"captures"`
		Policy   Policy           `json:"policy"`
		Sensor   SensorView       `json:"sensor"`
	}
	if err := c.get("/v1/status?limit="+strconv.Itoa(limit), &status); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Connected: true,
		Browser:   status.Browser,
		Captures:  status.Captures,
		Policy:    status.Policy,
		Sensor:    status.Sensor,
	}, nil
}

func (c *AgentClient) GetPolicy() (Policy, error) {
	var doc Policy
	if err := c.get("/v1/policy", &doc); err != nil {
		return Policy{}, err
	}
	return doc, nil
}

func (c *AgentClient) PutPolicy(doc Policy) (Policy, error) {
	var saved Policy
	if err := c.put("/v1/policy", doc, &saved); err != nil {
		return Policy{}, err
	}
	return saved, nil
}

func (c *AgentClient) WatchEvents(ctx context.Context, handle func(ChangeEvent)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/events", nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return agentHTTPError(resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var eventType, data string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if data != "" {
				var ev ChangeEvent
				if json.Unmarshal([]byte(data), &ev) == nil {
					if ev.Type == "" {
						ev.Type = eventType
					}
					handle(ev)
				}
			}
			eventType, data = "", ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return err
	}
	return ctx.Err()
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
