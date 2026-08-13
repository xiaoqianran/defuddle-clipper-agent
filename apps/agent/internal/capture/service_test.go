package capture

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

type fakeAnalyzer struct {
	summary string
	delay   time.Duration
	err     error
	calls   atomic.Int32
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, _ protocol.ContentPacket) (ai.Analysis, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ai.Analysis{}, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return ai.Analysis{}, err
	}
	if f.err != nil {
		return ai.Analysis{}, f.err
	}
	summary := f.summary
	if summary == "" {
		summary = "ok"
	}
	return ai.Analysis{Summary: summary, Tags: []string{"tag"}}, nil
}

func testPacket(id string) protocol.ContentPacket {
	return protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       id,
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: protocol.Source{
			URL:   "https://example.com",
			Title: "Title",
		},
		Content: protocol.Content{Markdown: "Body"},
	}
}

func captureDir(root, id string) string {
	return filepath.Join(root, "captures", "2026", "08", "13", id)
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func TestProcessReturnsPendingThenWritesDerived(t *testing.T) {
	root := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	analyzer := &gatedAnalyzer{started: started, release: release}
	service := New(storage.Store{Root: root}, analyzer)
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		service.waitForIdle()
	})
	packet := testPacket("capture-1")

	result, err := service.Process(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusPending {
		t.Fatalf("expected pending, got %s", result.AIStatus)
	}

	dir := captureDir(root, "capture-1")
	if _, err := os.Stat(filepath.Join(dir, "packet.json")); err != nil {
		t.Fatalf("missing packet.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis-pending")); err != nil {
		t.Fatalf("missing pending marker: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("analysis.json must not exist before Analyze finishes")
	}
	note, err := os.ReadFile(filepath.Join(dir, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(note), "Analysis is pending") {
		t.Fatalf("pending note missing section:\n%s", note)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("Analyze did not start")
	}
	close(release)
	service.waitForIdle()
	waitFile(t, filepath.Join(dir, "analysis.json"))
	if _, err := os.Stat(filepath.Join(dir, "analysis-pending")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pending marker should be cleared")
	}
	if got := storage.NewPaths(dir).AIStatus(); got != storage.StatusOK {
		t.Fatalf("expected ok after analysis, got %s", got)
	}
}

func TestProcessNilAnalyzerWritesDisabledNote(t *testing.T) {
	root := t.TempDir()
	service := New(storage.Store{Root: root}, nil)
	result, err := service.Process(context.Background(), testPacket("capture-disabled"))
	if err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusDisabled {
		t.Fatalf("expected disabled, got %s", result.AIStatus)
	}

	dir := captureDir(root, "capture-disabled")
	if _, err := os.Stat(filepath.Join(dir, "note.md")); err != nil {
		t.Fatalf("missing note.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("analysis.json should not exist when AI is disabled")
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis-pending")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pending marker should not exist when AI is disabled")
	}
}

func TestProcessCancelledContextDoesNotCancelAnalyze(t *testing.T) {
	root := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	analyzer := &gatedAnalyzer{started: started, release: release}
	service := New(storage.Store{Root: root}, analyzer)
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		service.waitForIdle()
	})

	ctx, cancel := context.WithCancel(context.Background())
	result, err := service.Process(ctx, testPacket("capture-cancel"))
	if err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusPending {
		t.Fatalf("expected pending, got %s", result.AIStatus)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("Analyze did not start")
	}
	cancel()
	close(release)
	service.waitForIdle()

	if analyzer.canceled.Load() {
		t.Fatal("background Analyze saw a cancelled context")
	}
	dir := captureDir(root, "capture-cancel")
	waitFile(t, filepath.Join(dir, "analysis.json"))
	if got := storage.NewPaths(dir).AIStatus(); got != storage.StatusOK {
		t.Fatalf("expected ok, got %s", got)
	}
}

func TestProcessDuplicateCompletedShortCircuits(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{}
	service := New(storage.Store{Root: root}, analyzer)
	packet := testPacket("capture-dup")

	if _, err := service.Process(context.Background(), packet); err != nil {
		t.Fatal(err)
	}
	service.waitForIdle()
	if analyzer.calls.Load() != 1 {
		t.Fatalf("expected 1 analyze call, got %d", analyzer.calls.Load())
	}

	second, err := service.Process(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatal("expected idempotent duplicate")
	}
	if second.AIStatus != storage.StatusOK {
		t.Fatalf("duplicate status: %s", second.AIStatus)
	}
	service.waitForIdle()
	if analyzer.calls.Load() != 1 {
		t.Fatalf("duplicate should not re-analyze, calls=%d", analyzer.calls.Load())
	}
}

func TestReprocessRunsAnalyzeAgain(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{summary: "first"}
	service := New(storage.Store{Root: root}, analyzer)
	packet := testPacket("capture-reprocess")

	if _, err := service.Process(context.Background(), packet); err != nil {
		t.Fatal(err)
	}
	service.waitForIdle()

	analyzer.summary = "second"
	result, err := service.Reprocess(context.Background(), packet.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Duplicate {
		t.Fatal("reprocess must not be treated as a duplicate")
	}
	if result.AIStatus != storage.StatusPending {
		t.Fatalf("expected pending reprocess, got %s", result.AIStatus)
	}
	service.waitForIdle()
	if analyzer.calls.Load() != 2 {
		t.Fatalf("expected 2 analyze calls, got %d", analyzer.calls.Load())
	}

	raw, err := os.ReadFile(filepath.Join(captureDir(root, packet.CaptureID), "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	var analysis ai.Analysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		t.Fatal(err)
	}
	if analysis.Summary != "second" {
		t.Fatalf("expected reprocessed summary, got %q", analysis.Summary)
	}
}

func TestReprocessUnknownID(t *testing.T) {
	service := New(storage.Store{Root: t.TempDir()}, &fakeAnalyzer{})
	_, err := service.Reprocess(context.Background(), "missing-id")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist, got %v", err)
	}
}

func TestConcurrentProcessSerializesDerivedWrites(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{delay: 30 * time.Millisecond}
	service := New(storage.Store{Root: root}, analyzer)
	packet := testPacket("capture-race")

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Process(context.Background(), packet)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	service.waitForIdle()

	dir := captureDir(root, packet.CaptureID)
	raw, err := os.ReadFile(filepath.Join(dir, "analysis.json"))
	if err != nil {
		t.Fatal(err)
	}
	var analysis ai.Analysis
	if err := json.Unmarshal(raw, &analysis); err != nil {
		t.Fatalf("corrupt analysis.json: %v\n%s", err, raw)
	}
	note, err := os.ReadFile(filepath.Join(dir, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(note), "## Original Content") {
		t.Fatalf("corrupt note.md:\n%s", note)
	}
	if storage.NewPaths(dir).AIStatus() != storage.StatusOK {
		t.Fatalf("status=%s", storage.NewPaths(dir).AIStatus())
	}
}

func TestBackgroundFailureWritesErrorSidecar(t *testing.T) {
	root := t.TempDir()
	analyzer := &fakeAnalyzer{err: errors.New("model timeout")}
	service := New(storage.Store{Root: root}, analyzer)
	if _, err := service.Process(context.Background(), testPacket("capture-fail")); err != nil {
		t.Fatal(err)
	}
	service.waitForIdle()

	dir := captureDir(root, "capture-fail")
	waitFile(t, filepath.Join(dir, "analysis-error.txt"))
	if got := storage.NewPaths(dir).AIStatus(); got != storage.StatusFailed {
		t.Fatalf("expected failed, got %s", got)
	}
}

type gatedAnalyzer struct {
	started  chan struct{}
	release  chan struct{}
	canceled atomic.Bool
}

func (g *gatedAnalyzer) Analyze(ctx context.Context, _ protocol.ContentPacket) (ai.Analysis, error) {
	close(g.started)
	select {
	case <-g.release:
	case <-ctx.Done():
		g.canceled.Store(true)
		return ai.Analysis{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		g.canceled.Store(true)
		return ai.Analysis{}, err
	}
	return ai.Analysis{Summary: "survived"}, nil
}
