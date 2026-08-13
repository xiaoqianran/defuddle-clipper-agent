package capture

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/render"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

type Service struct {
	Store    storage.Store
	Analyzer ai.Analyzer
	Logger   *log.Logger
	jobs     *jobSet
}

type Result struct {
	CaptureID string `json:"captureId"`
	NotePath  string `json:"notePath,omitempty"`
	AIStatus  string `json:"aiStatus"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type jobSet struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
	wg    sync.WaitGroup
}

func New(store storage.Store, analyzer ai.Analyzer) Service {
	return Service{
		Store:    store,
		Analyzer: analyzer,
		jobs:     newJobSet(),
	}
}

func newJobSet() *jobSet {
	return &jobSet{locks: make(map[string]*sync.Mutex)}
}

func (s Service) Process(ctx context.Context, packet protocol.ContentPacket) (Result, error) {
	paths, existed, err := s.Store.SavePacket(packet)
	if err != nil {
		return Result{}, err
	}

	if existed && s.Store.DerivedComplete(paths) {
		return Result{
			CaptureID: packet.CaptureID,
			NotePath:  absNote(paths.Note),
			AIStatus:  paths.AIStatus(),
			Duplicate: true,
		}, nil
	}

	return s.startDerived(ctx, packet, paths)
}

func (s Service) Reprocess(ctx context.Context, captureID string) (Result, error) {
	packet, paths, err := s.Store.Load(captureID)
	if err != nil {
		return Result{}, err
	}
	return s.startDerived(ctx, packet, paths)
}

func (s Service) startDerived(ctx context.Context, packet protocol.ContentPacket, paths storage.Paths) (Result, error) {
	notePath := absNote(paths.Note)

	if s.Analyzer == nil {
		note := render.Markdown(packet, nil, nil)
		if err := s.Store.WriteDerived(paths, nil, nil, note); err != nil {
			return Result{}, err
		}
		return Result{
			CaptureID: packet.CaptureID,
			NotePath:  notePath,
			AIStatus:  storage.StatusDisabled,
		}, nil
	}

	if err := s.Store.MarkPending(paths, render.PendingMarkdown(packet)); err != nil {
		return Result{}, err
	}

	// HTTP 请求 context 会在客户端中止或 handler 返回时被取消。
	// 分析必须比它活得更久；出站调用仍由 AI 客户端超时约束。
	s.enqueue(context.WithoutCancel(ctx), packet, paths)
	return Result{
		CaptureID: packet.CaptureID,
		NotePath:  notePath,
		AIStatus:  storage.StatusPending,
	}, nil
}

func (s Service) enqueue(ctx context.Context, packet protocol.ContentPacket, paths storage.Paths) {
	jobs := s.jobs
	if jobs == nil {
		panic("capture.Service must be constructed with capture.New")
	}
	lock := jobs.lock(packet.CaptureID)
	jobs.wg.Add(1)
	go func() {
		defer jobs.wg.Done()
		lock.Lock()
		defer lock.Unlock()
		s.runAnalysis(ctx, packet, paths)
	}()
}

func (s Service) runAnalysis(ctx context.Context, packet protocol.ContentPacket, paths storage.Paths) {
	var analysis *ai.Analysis
	var aiErr error
	value, err := s.Analyzer.Analyze(ctx, packet)
	if err != nil {
		aiErr = err
	} else {
		analysis = &value
	}

	note := render.Markdown(packet, analysis, aiErr)
	if err := s.Store.WriteDerived(paths, analysis, aiErr, note); err != nil {
		s.logf("write derived for %s failed: %v", packet.CaptureID, err)
	}
}

func (s Service) waitForIdle() {
	if s.jobs != nil {
		s.jobs.wg.Wait()
	}
}

func (j *jobSet) lock(id string) *sync.Mutex {
	j.mu.Lock()
	defer j.mu.Unlock()
	if m, ok := j.locks[id]; ok {
		return m
	}
	m := &sync.Mutex{}
	j.locks[id] = m
	return m
}

func (s Service) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func absNote(note string) string {
	notePath, err := filepath.Abs(note)
	if err != nil {
		return note
	}
	return notePath
}
