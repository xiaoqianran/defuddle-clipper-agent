package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/browserstate"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/capture"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	server := Server{
		Token:        "secret",
		MaxBodyBytes: 1 << 20,
		Captures: capture.Service{
			Store: storage.Store{Root: t.TempDir()},
		},
	}
	return server.Handler()
}

func TestCaptureAuthAndSave(t *testing.T) {
	handler := testServer(t)
	packet := protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       "capture-http",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: protocol.Source{URL: "https://example.com", Title: "Title"},
		Content: protocol.Content{Markdown: "Body"},
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
