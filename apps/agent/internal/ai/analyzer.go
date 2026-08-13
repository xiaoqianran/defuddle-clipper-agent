package ai

import (
	"context"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

type Concept struct {
	Name        string `json:"name"`
	Explanation string `json:"explanation"`
}

type Analysis struct {
	PageType    string     `json:"pageType"`
	Summary     string     `json:"summary"`
	KeyPoints   []string   `json:"keyPoints"`
	Concepts    []Concept  `json:"concepts"`
	Conclusions []string   `json:"conclusions"`
	Actions     []string   `json:"actions,omitempty"`
	Questions   []string   `json:"questions,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Provenance  Provenance `json:"provenance,omitempty"`
}

// Provenance 记录一次分析是如何产生的。绝不能包含
// API key 或其他密钥。
type Provenance struct {
	Model         string `json:"model"`
	ProviderHost  string `json:"providerHost"`
	PromptVersion string `json:"promptVersion"`
	ImageSent     bool   `json:"imageSent"`
	ImageSkipped  string `json:"imageSkipped,omitempty"`
	AnalyzedAt    string `json:"analyzedAt"`
}

// ImageSkippedProviderMediaFetch 表示可用封面图因 provider 拉取失败
// 而被省略（例如 Wikimedia 403 / CDN 401）。
const (
	ImageSkippedProviderMediaFetch = "provider-media-fetch"
	// 分析用 system prompt 变更时递增 PromptVersion。
	PromptVersion = "dca-analysis-v1"
)

type Analyzer interface {
	Analyze(context.Context, protocol.ContentPacket) (Analysis, error)
}
