package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/browserstate"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/capture"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/events"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

type fakeAnalyzer struct {
	calls   atomic.Int32
	summary atomic.Value
}

func (f *fakeAnalyzer) Analyze(context.Context, protocol.ContentPacket) (ai.Analysis, error) {
	f.calls.Add(1)
	summary, _ := f.summary.Load().(string)
	if summary == "" {
		summary = "http-ok"
	}
	return ai.Analysis{Summary: summary}, nil
}

func testServer(t *testing.T) http.Handler {
	t.Helper()
	hub := events.NewHub()
	captures := capture.New(storage.Store{Root: t.TempDir()}, nil)
	captures.OnChange = func(kind, captureID string) {
		hub.Publish(events.Event{Type: kind, CaptureID: captureID})
	}
	server := Server{
		Token:        "secret",
		MaxBodyBytes: 1 << 20,
		Captures:     captures,
		Events:       hub,
	}
	return server.Handler()
}

func testPacket(id string) protocol.ContentPacket {
	return protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       id,
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source:          protocol.Source{URL: "https://example.com", Title: "Title"},
		Content:         protocol.Content{Markdown: "Body"},
	}
}

func doJSON(t *testing.T, handler http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func waitCaptureStatus(t *testing.T, handler http.Handler, id, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		w := doJSON(t, handler, http.MethodGet, "/v1/captures/"+id, "secret", nil)
		if w.Code != http.StatusOK {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		var view storage.CaptureView
		if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
			t.Fatal(err)
		}
		last = view.AIStatus
		if last == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for aiStatus=%s (last %q)", want, last)
}

func TestCaptureAuthAndSave(t *testing.T) {
	handler := testServer(t)
	packet := protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       "capture-http",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source:          protocol.Source{URL: "https://example.com", Title: "Title"},
		Content:         protocol.Content{Markdown: "Body"},
	}
	body, _ := json.Marshal(packet)

	unauthorized := httptest.NewRequest(http.MethodPost, "/v1/captures", bytes.NewReader(body))
	unauthorized.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, unauthorized)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/captures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBrowserActiveStateRoundTrip(t *testing.T) {
	handler := testServer(t)
	body := []byte(`{"url":"https://example.com/docs","title":"Docs","observedAt":"2026-08-13T12:00:00Z","tabId":7,"windowId":2}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/browser/active", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/browser/state", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var state browserstate.State
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if !state.Active || state.Page.URL != "https://example.com/docs" || state.Page.Title != "Docs" {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestCaptureReturnsDisabledWithoutAnalyzer(t *testing.T) {
	handler := testServer(t)
	body, _ := json.Marshal(testPacket("capture-disabled"))
	w := doJSON(t, handler, http.MethodPost, "/v1/captures", "secret", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result capture.Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusDisabled {
		t.Fatalf("expected disabled, got %s", result.AIStatus)
	}

	w = doJSON(t, handler, http.MethodGet, "/v1/captures/capture-disabled", "secret", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var view storage.CaptureView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.AIStatus != storage.StatusDisabled {
		t.Fatalf("GET aiStatus=%s", view.AIStatus)
	}
}

func TestCapturePendingThenOKAndReprocess(t *testing.T) {
	analyzer := &fakeAnalyzer{}
	analyzer.summary.Store("first")
	server := Server{
		Token:        "secret",
		MaxBodyBytes: 1 << 20,
		AIEnabled:    true,
		Captures:     capture.New(storage.Store{Root: t.TempDir()}, analyzer),
	}
	handler := server.Handler()

	body, _ := json.Marshal(testPacket("capture-ai"))
	w := doJSON(t, handler, http.MethodPost, "/v1/captures", "secret", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result capture.Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusPending {
		t.Fatalf("expected pending, got %s", result.AIStatus)
	}

	waitCaptureStatus(t, handler, "capture-ai", storage.StatusOK)
	if analyzer.calls.Load() != 1 {
		t.Fatalf("expected 1 analyze call, got %d", analyzer.calls.Load())
	}

	list := doJSON(t, handler, http.MethodGet, "/v1/captures?limit=10", "secret", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	var history struct {
		Items []storage.CaptureSummary `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 1 || history.Items[0].AIStatus != storage.StatusOK {
		t.Fatalf("unexpected list: %+v", history.Items)
	}

	analyzer.summary.Store("second")
	w = doJSON(t, handler, http.MethodPost, "/v1/captures/capture-ai/reprocess", "secret", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.AIStatus != storage.StatusPending || result.Duplicate {
		t.Fatalf("unexpected reprocess result: %+v", result)
	}
	waitCaptureStatus(t, handler, "capture-ai", storage.StatusOK)
	if analyzer.calls.Load() != 2 {
		t.Fatalf("reprocess should analyze again, calls=%d", analyzer.calls.Load())
	}

	w = doJSON(t, handler, http.MethodGet, "/v1/captures/capture-ai", "secret", nil)
	var view storage.CaptureView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	analysis, _ := view.Analysis.(map[string]any)
	if analysis["summary"] != "second" {
		t.Fatalf("expected reprocessed analysis, got %#v", view.Analysis)
	}
}

func TestReprocessNotFoundAndAuth(t *testing.T) {
	analyzer := &fakeAnalyzer{}
	server := Server{
		Token:        "secret",
		MaxBodyBytes: 1 << 20,
		Captures:     capture.New(storage.Store{Root: t.TempDir()}, analyzer),
	}
	handler := server.Handler()

	unauthorized := doJSON(t, handler, http.MethodPost, "/v1/captures/missing/reprocess", "", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}

	missing := doJSON(t, handler, http.MethodPost, "/v1/captures/missing-id/reprocess", "secret", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", missing.Code, missing.Body.String())
	}
}

func TestPolicyAndStatusRoundTrip(t *testing.T) {
	handler := testServer(t)
	body := []byte(`{"autoCapture":false,"archiveAll":false,"captureDelayMs":900,"domainAllowlist":["Example.com"],"domainDenylist":[]}`)
	w := doJSON(t, handler, http.MethodPut, "/v1/policy", "secret", body)
	if w.Code != http.StatusOK {
		t.Fatalf("put policy: %d %s", w.Code, w.Body.String())
	}

	got := doJSON(t, handler, http.MethodGet, "/v1/policy", "secret", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get policy: %d %s", got.Code, got.Body.String())
	}
	var doc struct {
		AutoCapture    bool     `json:"autoCapture"`
		ArchiveAll     bool     `json:"archiveAll"`
		CaptureDelayMs int      `json:"captureDelayMs"`
		Allowlist      []string `json:"domainAllowlist"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.AutoCapture || doc.ArchiveAll || doc.CaptureDelayMs != 900 || len(doc.Allowlist) != 1 || doc.Allowlist[0] != "example.com" {
		t.Fatalf("policy=%+v", doc)
	}

	beat := doJSON(t, handler, http.MethodPost, "/v1/sensor/heartbeat", "secret", []byte(`{"queueLength":2,"lastError":"retry"}`))
	if beat.Code != http.StatusOK {
		t.Fatalf("heartbeat: %d %s", beat.Code, beat.Body.String())
	}

	status := doJSON(t, handler, http.MethodGet, "/v1/status?limit=10", "secret", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("status: %d %s", status.Code, status.Body.String())
	}
	var payload struct {
		ProtocolVersion string `json:"protocolVersion"`
		Policy          struct {
			AutoCapture bool `json:"autoCapture"`
		} `json:"policy"`
		Sensor struct {
			Connected   bool `json:"connected"`
			QueueLength int  `json:"queueLength"`
		} `json:"sensor"`
		Captures []any `json:"captures"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "1.0" || payload.Policy.AutoCapture || !payload.Sensor.Connected || payload.Sensor.QueueLength != 2 {
		t.Fatalf("status=%+v", payload)
	}
	if payload.Captures == nil {
		t.Fatal("expected captures array")
	}
}

func TestEventsStreamEmitsCaptureSaved(t *testing.T) {
	handler := testServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, req)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(w.Body.String(), ": connected") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	body, _ := json.Marshal(testPacket("capture-event"))
	posted := doJSON(t, handler, http.MethodPost, "/v1/captures", "secret", body)
	if posted.Code != http.StatusOK {
		t.Fatalf("capture: %d %s", posted.Code, posted.Body.String())
	}

	for time.Now().Before(deadline) {
		if strings.Contains(w.Body.String(), `"type":"capture.saved"`) {
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("sse handler did not return")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	t.Fatalf("missing capture.saved in stream: %s", w.Body.String())
}
