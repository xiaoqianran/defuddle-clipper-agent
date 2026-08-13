package host

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServesHealthProtocolVersion(t *testing.T) {
	t.Setenv("DCA_DATA_DIR", t.TempDir())
	t.Setenv("DCA_ADDR", "127.0.0.1:27123")
	t.Setenv("DCA_TOKEN", "")
	t.Setenv("DCA_AI_ENABLED", "false")

	srv, err := New(log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	if srv.Addr != "127.0.0.1:27123" {
		t.Fatalf("addr=%q", srv.Addr)
	}
	if srv.HTTP == nil || srv.HTTP.Handler == nil {
		t.Fatal("missing HTTP handler")
	}
	if srv.HTTP.ReadHeaderTimeout != ReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout=%s", srv.HTTP.ReadHeaderTimeout)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		Status          string `json:"status"`
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if ProtocolVersion != "1.0" {
		t.Fatalf("host.ProtocolVersion=%q, desktop probe expects 1.0", ProtocolVersion)
	}
	if payload.Status != "ok" || payload.ProtocolVersion != "1.0" {
		t.Fatalf("health=%+v", payload)
	}
}
