package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/ai"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/capture"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/config"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/httpapi"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/storage"
)

func main() {
	logger := log.New(os.Stderr, "dca: ", log.LstdFlags|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
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

	api := httpapi.Server{
		Token:        cfg.Token,
		MaxBodyBytes: cfg.MaxBodyBytes,
		AIEnabled:    cfg.AI.Enabled,
		Captures:     captures,
		Logger:       logger,
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Printf("listening on http://%s", cfg.Addr)
	logger.Printf("data dir: %s", cfg.DataDir)
	if cfg.Token == "" {
		logger.Printf("warning: DCA_TOKEN is empty; loopback-only mode is relying on local network isolation")
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}
