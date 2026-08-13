package capture

import (
	"context"
	"path/filepath"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/render"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

type Service struct {
	Store    storage.Store
	Analyzer ai.Analyzer
}

type Result struct {
	CaptureID string `json:"captureId"`
	NotePath  string `json:"notePath,omitempty"`
	AIStatus  string `json:"aiStatus"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

func (s Service) Process(ctx context.Context, packet protocol.ContentPacket) (Result, error) {
	paths, existed, err := s.Store.SavePacket(packet)
	if err != nil {
		return Result{}, err
	}

	if existed && s.Store.NoteExists(paths) {
		return Result{
			CaptureID: packet.CaptureID,
			NotePath:  paths.Note,
			AIStatus:  "unknown",
			Duplicate: true,
		}, nil
	}

	var analysis *ai.Analysis
	var aiErr error
	aiStatus := "disabled"

	if s.Analyzer != nil {
		value, err := s.Analyzer.Analyze(ctx, packet)
		if err != nil {
			aiErr = err
			aiStatus = "failed"
		} else {
			analysis = &value
			aiStatus = "ok"
		}
	}

	note := render.Markdown(packet, analysis, aiErr)
	if err := s.Store.WriteDerived(paths, analysis, aiErr, note); err != nil {
		return Result{}, err
	}

	notePath, _ := filepath.Abs(paths.Note)
	return Result{
		CaptureID: packet.CaptureID,
		NotePath:  notePath,
		AIStatus:  aiStatus,
	}, nil
}
