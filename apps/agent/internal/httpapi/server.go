package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/browserstate"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/capture"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/events"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/policy"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/sensor"
)

type Server struct {
	Token        string
	MaxBodyBytes int64
	AIEnabled    bool
	Captures     capture.Service
	Browser      *browserstate.Store
	Policy       *policy.Store
	Events       *events.Hub
	Sensor       *sensor.Store
	Logger       *log.Logger
}

func (s Server) Handler() http.Handler {
	if s.Browser == nil {
		s.Browser = &browserstate.Store{}
	}
	if s.Policy == nil {
		s.Policy = policy.Memory()
	}
	if s.Events == nil {
		s.Events = events.NewHub()
	}
	if s.Sensor == nil {
		s.Sensor = &sensor.Store{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /v1/captures", s.auth(s.capture))
	mux.HandleFunc("GET /v1/captures", s.auth(s.listCaptures))
	mux.HandleFunc("GET /v1/captures/{captureID}", s.auth(s.readCapture))
	mux.HandleFunc("POST /v1/captures/{captureID}/reprocess", s.auth(s.reprocessCapture))
	mux.HandleFunc("POST /v1/browser/active", s.auth(s.activePage))
	mux.HandleFunc("GET /v1/browser/state", s.auth(s.browserState))
	mux.HandleFunc("GET /v1/policy", s.auth(s.getPolicy))
	mux.HandleFunc("PUT /v1/policy", s.auth(s.putPolicy))
	mux.HandleFunc("GET /v1/status", s.auth(s.status))
	mux.HandleFunc("POST /v1/sensor/heartbeat", s.auth(s.sensorHeartbeat))
	mux.HandleFunc("GET /v1/events", s.auth(s.streamEvents))
	return s.logging(mux)
}

func (s Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"protocolVersion": protocol.Version,
		"aiEnabled":       s.AIEnabled,
		"time":            time.Now().UTC().Format(time.RFC3339),
	})
}

func (s Server) capture(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.MaxBodyBytes)
	defer r.Body.Close()

	var packet protocol.ContentPacket
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&packet); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "capture payload is too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}
	if err := packet.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.Captures.Process(r.Context(), packet)
	if err != nil {
		s.logf("capture %s failed: %v", packet.CaptureID, err)
		writeError(w, http.StatusInternalServerError, "capture processing failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s Server) activePage(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()

	var page browserstate.Page
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&page); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}

	parsed, err := url.Parse(page.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "active page URL must be http or https")
		return
	}
	if page.ObservedAt != "" {
		if _, err := time.Parse(time.RFC3339, page.ObservedAt); err != nil {
			writeError(w, http.StatusBadRequest, "observedAt must be RFC3339")
			return
		}
	}

	state := s.Browser.Set(page)
	s.Events.Publish(events.Event{Type: events.BrowserActive})
	writeJSON(w, http.StatusOK, state)
}

func (s Server) browserState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Browser.Get())
}

func (s Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Token == "" {
			next(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(got) != len(s.Token) || subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/events" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func (s Server) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSONStatus(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	writeJSONStatus(w, status, value)
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
