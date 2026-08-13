package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnsureStartsWhenNothingListens(t *testing.T) {
	addr := freeLoopbackAddr(t)
	var started atomic.Bool
	host := &AgentHost{
		Addr:   addr,
		Logger: silentLogger(),
		Probe:  ProbeHealth,
		Start: func(ctx context.Context, gotAddr string) (func(context.Context) error, string, error) {
			if gotAddr != addr {
				t.Fatalf("start addr=%q want %q", gotAddr, addr)
			}
			started.Store(true)
			return startTestAgent(t, gotAddr)
		},
	}

	clientURL, err := host.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stopHost(t, host)

	if !started.Load() {
		t.Fatal("expected in-process start")
	}
	if !host.Owned() {
		t.Fatal("expected owned server")
	}
	if clientURL != "http://"+addr {
		t.Fatalf("clientURL=%q", clientURL)
	}
	if ok, err := ProbeHealth(context.Background(), clientURL); err != nil || !ok {
		t.Fatalf("started server not healthy: ok=%v err=%v", ok, err)
	}
}

func TestEnsureAttachesWhenHealthProtocolMatches(t *testing.T) {
	existing := serveExistingAgent(t, "1.0")
	defer existing.Close()

	var started atomic.Bool
	host := &AgentHost{
		Addr:      hostPort(t, existing.URL),
		ClientURL: existing.URL,
		Logger:    silentLogger(),
		Probe:     ProbeHealth,
		Start: func(context.Context, string) (func(context.Context) error, string, error) {
			started.Store(true)
			return nil, "", errors.New("should not start in-process")
		},
	}

	clientURL, err := host.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started.Load() {
		t.Fatal("started in-process despite existing agent")
	}
	if host.Owned() {
		t.Fatal("attached host must not own the server")
	}
	if clientURL != existing.URL {
		t.Fatalf("clientURL=%q want %q", clientURL, existing.URL)
	}

	if err := host.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeHealth(context.Background(), existing.URL); err != nil {
		t.Fatal(err)
	}
	ok, err := ProbeHealth(context.Background(), existing.URL)
	if err != nil || !ok {
		t.Fatalf("attached server should still be healthy after Stop: ok=%v err=%v", ok, err)
	}
}

func TestEnsureAttachesWhenStartHitsAddrInUse(t *testing.T) {
	existing := serveExistingAgent(t, "1.0")
	defer existing.Close()
	addr := hostPort(t, existing.URL)

	var probes atomic.Int32
	host := &AgentHost{
		Addr:   addr,
		Logger: silentLogger(),
		Probe: func(ctx context.Context, baseURL string) (bool, error) {
			n := probes.Add(1)
			if n == 1 {
				return false, nil
			}
			return ProbeHealth(ctx, baseURL)
		},
		Start: func(ctx context.Context, gotAddr string) (func(context.Context) error, string, error) {
			ln, err := net.Listen("tcp", gotAddr)
			if err != nil {
				return nil, "", err
			}
			_ = ln.Close()
			return nil, "", errors.New("listen unexpectedly succeeded")
		},
	}

	clientURL, err := host.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if host.Owned() {
		t.Fatal("expected attach after bind conflict")
	}
	if clientURL != "http://"+addr {
		t.Fatalf("clientURL=%q", clientURL)
	}
	if probes.Load() < 2 {
		t.Fatalf("probes=%d, want re-probe after addr-in-use", probes.Load())
	}
}

func TestEnsureErrorsWhenPortBusyWithOtherServer(t *testing.T) {
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"other"}`))
	}))
	defer foreign.Close()
	addr := hostPort(t, foreign.URL)
	host := &AgentHost{
		Addr:   addr,
		Logger: silentLogger(),
		Probe:  ProbeHealth,
		Start: func(ctx context.Context, gotAddr string) (func(context.Context) error, string, error) {
			other, err := net.Listen("tcp", gotAddr)
			if err != nil {
				return nil, "", err
			}
			_ = other.Close()
			return nil, "", errors.New("listen unexpectedly succeeded")
		},
	}

	if _, err := host.Ensure(context.Background()); err == nil {
		t.Fatal("expected error when a different server owns the port")
	}
	if host.Owned() {
		t.Fatal("must not claim ownership of a foreign server")
	}
}

func TestEnsureStopShutsDownOwnedServer(t *testing.T) {
	addr := freeLoopbackAddr(t)
	host := &AgentHost{
		Addr:   addr,
		Logger: silentLogger(),
		Probe:  ProbeHealth,
		Start: func(ctx context.Context, gotAddr string) (func(context.Context) error, string, error) {
			return startTestAgent(t, gotAddr)
		},
	}
	clientURL, err := host.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !host.Owned() {
		t.Fatal("expected owned server")
	}
	stopHost(t, host)
	if host.Owned() {
		t.Fatal("owned flag should clear after Stop")
	}
	ok, err := ProbeHealth(context.Background(), clientURL)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("owned server should be gone after Stop")
	}
}

func TestProbeHealthRejectsWrongProtocol(t *testing.T) {
	server := serveExistingAgent(t, "2.0")
	defer server.Close()
	ok, err := ProbeHealth(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("protocolVersion 2.0 must not match")
	}
}

func TestHttpBaseURLBracketsIPv6(t *testing.T) {
	if got := httpBaseURL("[::1]:27123"); got != "http://[::1]:27123" {
		t.Fatalf("got %q", got)
	}
	if got := httpBaseURL("127.0.0.1:27123"); got != "http://127.0.0.1:27123" {
		t.Fatalf("got %q", got)
	}
}

func TestNewDesktopLoggerWritesDataDirFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DCA_DATA_DIR", dir)
	logger, closer := newDesktopLogger()
	if closer != nil {
		defer closer.Close()
	}
	logger.Printf("hello-desktop-log")
	if closer != nil {
		_ = closer.Close()
	}
	raw, err := os.ReadFile(filepath.Join(dir, desktopLogFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "hello-desktop-log") {
		t.Fatalf("log file contents=%q", raw)
	}
}

func TestNewAgentClientFromBoundAddr(t *testing.T) {
	client, err := NewAgentClient("http://127.0.0.1:27123", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if client.baseURL != "http://127.0.0.1:27123" || client.token != "tok" {
		t.Fatalf("client=%+v", client)
	}
}

func silentLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func serveExistingAgent(t *testing.T, protocolVersion string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":          "ok",
			"protocolVersion": protocolVersion,
		})
	}))
}

func startTestAgent(t *testing.T, addr string) (func(context.Context) error, string, error) {
	t.Helper()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":          "ok",
			"protocolVersion": healthProtocolVer,
		})
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		_ = srv.Serve(ln)
	}()
	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := waitHealthy(waitCtx, httpBaseURL(ln.Addr().String()), ProbeHealth); err != nil {
		_ = srv.Close()
		return nil, "", err
	}
	return srv.Shutdown, ln.Addr().String(), nil
}

func hostPort(t *testing.T, rawURL string) string {
	t.Helper()
	u := strings.TrimPrefix(rawURL, "http://")
	u = strings.TrimPrefix(u, "https://")
	return u
}

func stopHost(t *testing.T, host *AgentHost) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := host.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
