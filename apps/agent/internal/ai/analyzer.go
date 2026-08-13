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
	PageType    string    `json:"pageType"`
	Summary     string    `json:"summary"`
	KeyPoints   []string  `json:"keyPoints"`
	Concepts    []Concept `json:"concepts"`
	Conclusions []string  `json:"conclusions"`
	Actions     []string  `json:"actions,omitempty"`
	Questions   []string  `json:"questions,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

type Analyzer interface {
	Analyze(context.Context, protocol.ContentPacket) (Analysis, error)
}
