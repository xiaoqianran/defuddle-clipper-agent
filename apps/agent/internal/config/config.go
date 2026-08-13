package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	DataDir      string
	Token        string
	MaxBodyBytes int64
	AI           AIConfig
}

type AIConfig struct {
	Enabled    bool
	BaseURL    string
	APIKey     string
	Model      string
	ChunkChars int
	Timeout    time.Duration
}

func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Addr:         env("DCA_ADDR", "127.0.0.1:27123"),
		DataDir:      env("DCA_DATA_DIR", filepath.Join(home, ".defuddle-clipper-agent")),
		Token:        os.Getenv("DCA_TOKEN"),
		MaxBodyBytes: envInt64("DCA_MAX_BODY_BYTES", 10*1024*1024),
		AI: AIConfig{
			Enabled:    envBool("DCA_AI_ENABLED", false),
			BaseURL:    env("DCA_OPENAI_BASE_URL", "https://api.openai.com/v1"),
			APIKey:     os.Getenv("DCA_OPENAI_API_KEY"),
			Model:      os.Getenv("DCA_OPENAI_MODEL"),
			ChunkChars: envInt("DCA_AI_CHUNK_CHARS", 12000),
			Timeout:    time.Duration(envInt("DCA_AI_TIMEOUT_SECONDS", 90)) * time.Second,
		},
	}

	if cfg.MaxBodyBytes < 1024 {
		return Config{}, fmt.Errorf("DCA_MAX_BODY_BYTES is too small")
	}
	if cfg.AI.ChunkChars < 2000 {
		return Config{}, fmt.Errorf("DCA_AI_CHUNK_CHARS must be >= 2000")
	}
	if cfg.AI.Enabled && strings.TrimSpace(cfg.AI.Model) == "" {
		return Config{}, fmt.Errorf("DCA_OPENAI_MODEL is required when DCA_AI_ENABLED=true")
	}

	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DCA_ADDR: %w", err)
	}
	ip := net.ParseIP(host)
	isLoopback := host == "localhost" || (ip != nil && ip.IsLoopback())
	if !isLoopback && cfg.Token == "" {
		return Config{}, fmt.Errorf("DCA_TOKEN is required when binding to a non-loopback address")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
