package protocol

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const Version = "1.0"

var captureIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type ContentPacket struct {
	ProtocolVersion string           `json:"protocolVersion"`
	CaptureID       string           `json:"captureId"`
	CapturedAt      string           `json:"capturedAt"`
	Source          Source           `json:"source"`
	Content         Content          `json:"content"`
	Selection       *Selection       `json:"selection,omitempty"`
	Highlights      []map[string]any `json:"highlights,omitempty"`
	Metadata        map[string]any   `json:"metadata,omitempty"`
	Media           map[string]any   `json:"media,omitempty"`
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

type Selection struct {
	Markdown string `json:"markdown,omitempty"`
	HTML     string `json:"html,omitempty"`
}

func (p ContentPacket) Validate() error {
	if p.ProtocolVersion != Version {
		return fmt.Errorf("unsupported protocolVersion %q", p.ProtocolVersion)
	}
	if !captureIDPattern.MatchString(p.CaptureID) {
		return errors.New("captureId must match ^[A-Za-z0-9._-]{1,128}$")
	}
	if _, err := time.Parse(time.RFC3339Nano, p.CapturedAt); err != nil {
		return fmt.Errorf("capturedAt must be RFC3339: %w", err)
	}
	u, err := url.ParseRequestURI(p.Source.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("source.url must be an absolute URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("source.url must use http or https")
	}
	if strings.TrimSpace(p.Source.Title) == "" {
		return errors.New("source.title is required")
	}
	if strings.TrimSpace(p.Content.Markdown) == "" {
		return errors.New("content.markdown is required")
	}
	return nil
}
