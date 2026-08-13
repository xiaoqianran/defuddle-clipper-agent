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

// Provenance records how an analysis was produced. It must never include
// API keys or other secrets.
type Provenance struct {
	Model         string `json:"model"`
	ProviderHost  string `json:"providerHost"`
	PromptVersion string `json:"promptVersion"`
	ImageSent     bool   `json:"imageSent"`
	AnalyzedAt    string `json:"analyzedAt"`
}

// PromptVersion is bumped when the analysis system prompt changes.
const PromptVersion = "dca-analysis-v1"

type Analyzer interface {
	Analyze(context.Context, protocol.ContentPacket) (Analysis, error)
}
