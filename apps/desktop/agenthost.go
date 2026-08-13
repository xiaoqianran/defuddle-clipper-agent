package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultAgentAddr    = "127.0.0.1:27123"
	healthProtocolVer   = "1.0"
	probeHTTPTimeout    = 800 * time.Millisecond
	ownedShutdownWait   = 5 * time.Second
	desktopLogFileName  = "desktop.log"
	defaultAgentDataDir = ".defuddle-clipper-agent"
)

// StartFunc 在 addr 上启动捕获 HTTP 服务。成功时返回 Shutdown 与实际绑定地址。
type StartFunc func(ctx context.Context, addr string) (shutdown func(context.Context) error, boundAddr string, err error)

// ProbeFunc 判断 baseURL 是否已经在提供本 agent 的 GET /health（protocolVersion 1.0）。
type ProbeFunc func(ctx context.Context, baseURL string) (bool, error)

// AgentHost 决定是复用已在监听的 agent，还是在本进程内启动。
type AgentHost struct {
	Addr      string
	ClientURL string
	Logger    *log.Logger
	Probe     ProbeFunc
	Start     StartFunc

	mu       sync.Mutex
	owned    bool
	shutdown func(context.Context) error
}

func listenAddrFromEnv() string {
	if addr := strings.TrimSpace(os.Getenv("DCA_ADDR")); addr != "" {
		return addr
	}
	return defaultAgentAddr
}

func explicitAgentURLFromEnv() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("DCA_AGENT_URL")), "/")
}

func clientURLFromEnv() string {
	if base := explicitAgentURLFromEnv(); base != "" {
		return base
	}
	return "http://" + defaultAgentAddr
}

func httpBaseURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
			return strings.TrimRight(addr, "/")
		}
		return "http://" + addr
	}
	return "http://" + net.JoinHostPort(host, port)
}

// ProbeHealth 在 baseURL 上 GET /health，仅当 protocolVersion 为 1.0 时返回 true。
// 连接失败（端口空闲或拒绝）视为“不是本 agent”，返回 false, nil。
func ProbeHealth(ctx context.Context, baseURL string) (bool, error) {
	endpoint, err := url.JoinPath(strings.TrimRight(baseURL, "/"), "health")
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	client := &http.Client{Timeout: probeHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	var payload struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return false, nil
	}
	return payload.ProtocolVersion == healthProtocolVer, nil
}

// Ensure 若本机已有本 agent 则复用；否则在进程内启动。
func (h *AgentHost) Ensure(ctx context.Context) (clientBaseURL string, err error) {
	if h.Probe == nil {
		h.Probe = ProbeHealth
	}
	if h.Start == nil {
		return "", errors.New("agent host Start is not configured")
	}
	addr := h.Addr
	if addr == "" {
		addr = defaultAgentAddr
	}
	probeURL := httpBaseURL(addr)

	ok, err := h.Probe(ctx, probeURL)
	if err != nil {
		return "", err
	}
	if ok {
		h.mu.Lock()
		h.owned = false
		h.shutdown = nil
		h.mu.Unlock()
		h.logf("attached to existing agent at %s", probeURL)
		return h.clientURLForAttach(probeURL), nil
	}

	shutdown, bound, err := h.Start(ctx, addr)
	if err != nil {
		if isAddrInUse(err) {
			ok, probeErr := h.Probe(ctx, probeURL)
			if probeErr == nil && ok {
				h.mu.Lock()
				h.owned = false
				h.shutdown = nil
				h.mu.Unlock()
				h.logf("attached to existing agent at %s after bind conflict", probeURL)
				return h.clientURLForAttach(probeURL), nil
			}
		}
		return "", err
	}

	h.mu.Lock()
	h.owned = true
	h.shutdown = shutdown
	h.mu.Unlock()

	if bound == "" {
		bound = addr
	}
	startedURL := httpBaseURL(bound)
	h.logf("started in-process agent at %s", startedURL)
	return startedURL, nil
}

func (h *AgentHost) clientURLForAttach(probeURL string) string {
	if strings.TrimSpace(h.ClientURL) != "" {
		return strings.TrimRight(h.ClientURL, "/")
	}
	return probeURL
}

// Owned 报告本进程是否启动了 HTTP 服务器（关闭时才应 Stop）。
func (h *AgentHost) Owned() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.owned
}

// Stop 仅在本进程启动了服务器时关闭它；复用外部 agent 时是空操作。
func (h *AgentHost) Stop(ctx context.Context) error {
	h.mu.Lock()
	owned := h.owned
	shutdown := h.shutdown
	h.owned = false
	h.shutdown = nil
	h.mu.Unlock()
	if !owned || shutdown == nil {
		return nil
	}
	if err := shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	h.logf("stopped in-process agent")
	return nil
}

func (h *AgentHost) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "address already in use") {
		return true
	}
	// Windows: "bind: Only one usage of each socket address is normally permitted"
	if strings.Contains(msg, "only one usage of each socket address") {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) && op != nil && op.Err != nil && op.Err != err {
		return isAddrInUse(op.Err)
	}
	return false
}

func desktopDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DCA_DATA_DIR")); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, defaultAgentDataDir)
	}
	if cfg, err := os.UserConfigDir(); err == nil && cfg != "" {
		return filepath.Join(cfg, "defuddle-clipper-agent")
	}
	return os.TempDir()
}

func newDesktopLogger() (*log.Logger, io.Closer) {
	dir := desktopDataDir()
	writers := []io.Writer{os.Stderr}
	var closer io.Closer
	if err := os.MkdirAll(dir, 0o755); err == nil {
		f, err := os.OpenFile(filepath.Join(dir, desktopLogFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			writers = append(writers, f)
			closer = f
		}
	}
	return log.New(io.MultiWriter(writers...), "dca: ", log.LstdFlags|log.LUTC), closer
}

func waitHealthy(ctx context.Context, baseURL string, probe ProbeFunc) error {
	if probe == nil {
		probe = ProbeHealth
	}
	deadline := time.Now().Add(2 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ok, err := probe(ctx, baseURL)
		if err != nil {
			last = err
		} else if ok {
			return nil
		} else {
			last = fmt.Errorf("agent health not ready at %s", baseURL)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
	if last == nil {
		last = fmt.Errorf("agent health not ready at %s", baseURL)
	}
	return last
}
