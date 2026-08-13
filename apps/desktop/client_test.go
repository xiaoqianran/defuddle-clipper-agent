// 运行：go test -tags dcatest ./...
package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSnapshotDecodesAIStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r, "secret")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/browser/state":
			writeJSON(w, map[string]any{
				"active":    true,
				"page":      map[string]any{"url": "https://example.com", "title": "Example", "observedAt": "2026-08-13T15:00:00Z"},
				"updatedAt": "2026-08-13T15:00:00Z",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/captures":
			writeJSON(w, map[string]any{
				"items": []map[string]any{{
					"captureId":   "cap-1",
					"capturedAt":  "2026-08-13T15:00:00Z",
					"title":       "Example",
					"url":         "https://example.com",
					"hasAnalysis": false,
					"hasNote":     true,
					"aiStatus":    "pending",
				}},
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snap, err := testClient(server, "secret").Snapshot(50)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Connected || len(snap.Captures) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	if snap.Captures[0].AIStatus != "pending" || snap.Captures[0].CaptureID != "cap-1" {
		t.Fatalf("aiStatus not decoded: %+v", snap.Captures[0])
	}
}

func TestReadCaptureDecodesAIStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r, "secret")
		if r.Method != http.MethodGet || r.URL.Path != "/v1/captures/cap-1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{
			"packet":         map[string]any{"captureId": "cap-1", "capturedAt": "2026-08-13T15:00:00Z", "source": map[string]any{"url": "https://example.com", "title": "Example"}, "content": map[string]any{"markdown": "# Hi"}},
			"sourceMarkdown": "# Hi",
			"note":           "Analysis is pending.",
			"aiStatus":       "ok",
			"analysis":       map[string]any{"summary": "done"},
		})
	}))
	defer server.Close()

	view, err := testClient(server, "secret").ReadCapture("cap-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.AIStatus != "ok" || view.Note != "Analysis is pending." || view.Packet.CaptureID != "cap-1" {
		t.Fatalf("view not decoded: %+v", view)
	}
}

func TestReprocessCapturePostsPathAndAuth(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %q", body)
		}
		writeJSON(w, map[string]any{
			"captureId": "cap-1",
			"aiStatus":  "pending",
			"notePath":  "/tmp/note.md",
		})
	}))
	defer server.Close()

	result, err := testClient(server, "secret").ReprocessCapture("cap-1")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%s", gotMethod)
	}
	if gotPath != "/v1/captures/cap-1/reprocess" {
		t.Fatalf("path=%s", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if result.CaptureID != "cap-1" || result.AIStatus != "pending" || result.Duplicate {
		t.Fatalf("result=%+v", result)
	}
}

func TestReprocessCaptureHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"capture not found"}`))
	}))
	defer server.Close()

	_, err := testClient(server, "secret").ReprocessCapture("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "HTTP 404") || !strings.Contains(msg, "capture not found") {
		t.Fatalf("error=%q", msg)
	}
}

func TestReadCaptureHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := testClient(server, "").ReadCapture("cap-1")
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("error=%v", err)
	}
}

func TestCaptureIDRequired(t *testing.T) {
	client := &AgentClient{baseURL: "http://127.0.0.1:1", http: &http.Client{}}
	if _, err := client.ReadCapture(""); err == nil {
		t.Fatal("expected read error")
	}
	if _, err := client.ReprocessCapture(""); err == nil {
		t.Fatal("expected reprocess error")
	}
}

func testClient(server *httptest.Server, token string) *AgentClient {
	return &AgentClient{
		baseURL: server.URL,
		token:   token,
		http:    server.Client(),
	}
}

func assertBearer(t *testing.T, r *http.Request, token string) {
	t.Helper()
	want := "Bearer " + token
	if got := r.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization=%q, want %q", got, want)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
