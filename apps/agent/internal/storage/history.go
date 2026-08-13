package storage

import "github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"

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

type CaptureView struct {
	Packet         protocol.ContentPacket `json:"packet"`
	SourceMarkdown string                 `json:"sourceMarkdown"`
	Analysis       any                    `json:"analysis,omitempty"`
	Note           string                 `json:"note,omitempty"`
}
