// Package host 把 clipper-agent 的 HTTP 运行时装配成可复用的 *http.Server。
// 独立 cmd 与桌面进程都可以调用，而不必复制 capture/AI 逻辑。
package host

import (
	"log"
	"net/http"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/capture"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/config"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/events"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/httpapi"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/policy"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/sensor"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

const (
	// ReadHeaderTimeout 与独立 clipper-agent 入口一致。
	ReadHeaderTimeout = 5 * time.Second
	// ProtocolVersion 是 GET /health 用来识别本 agent 的契约版本。
	ProtocolVersion = protocol.Version
	DefaultAddr     = "127.0.0.1:27123"
)

// Server 是已装配的本地捕获 HTTP 服务，尚未开始监听。
type Server struct {
	HTTP    *http.Server
	Addr    string
	DataDir string
	Token   string
}

// New 按独立 agent 相同的环境变量装配服务（DCA_ADDR、DCA_DATA_DIR、DCA_TOKEN、DCA_AI_*、DCA_OPENAI_*）。
func New(logger *log.Logger) (*Server, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var analyzer ai.Analyzer
	if cfg.AI.Enabled {
		analyzer = ai.NewOpenAICompatible(
			cfg.AI.BaseURL,
			cfg.AI.APIKey,
			cfg.AI.Model,
			cfg.AI.ChunkChars,
			cfg.AI.Timeout,
		)
	}

	captures := capture.New(storage.Store{Root: cfg.DataDir}, analyzer)
	captures.Logger = logger

	hub := events.NewHub()
	captures.OnChange = func(kind, captureID string) {
		hub.Publish(events.Event{Type: kind, CaptureID: captureID})
	}

	pol, err := policy.Load(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	api := httpapi.Server{
		Token:        cfg.Token,
		MaxBodyBytes: cfg.MaxBodyBytes,
		AIEnabled:    cfg.AI.Enabled,
		Captures:     captures,
		Policy:       pol,
		Events:       hub,
		Sensor:       &sensor.Store{},
		Logger:       logger,
	}

	return &Server{
		HTTP: &http.Server{
			Addr:              cfg.Addr,
			Handler:           api.Handler(),
			ReadHeaderTimeout: ReadHeaderTimeout,
		},
		Addr:    cfg.Addr,
		DataDir: cfg.DataDir,
		Token:   cfg.Token,
	}, nil
}
