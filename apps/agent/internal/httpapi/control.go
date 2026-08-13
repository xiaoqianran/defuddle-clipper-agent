package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/events"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/policy"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/sensor"
)

type Status struct {
	ProtocolVersion string          `json:"protocolVersion"`
	AIEnabled       bool            `json:"aiEnabled"`
	Policy          policy.Document `json:"policy"`
	Browser         any             `json:"browser"`
	Sensor          sensor.View     `json:"sensor"`
	Captures        any             `json:"captures,omitempty"`
	Time            string          `json:"time"`
}

func (s Server) getPolicy(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Policy.Get())
}

func (s Server) putPolicy(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()

	var doc policy.Document
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&doc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}

	saved, err := s.Policy.Put(doc)
	if err != nil {
		s.logf("save policy failed: %v", err)
		writeError(w, http.StatusInternalServerError, "policy could not be saved")
		return
	}
	s.Events.Publish(events.Event{Type: events.PolicyUpdated})
	writeJSON(w, http.StatusOK, saved)
}

func (s Server) status(w http.ResponseWriter, r *http.Request) {
	payload := Status{
		ProtocolVersion: protocol.Version,
		AIEnabled:       s.AIEnabled,
		Policy:          s.Policy.Get(),
		Browser:         s.Browser.Get(),
		Sensor:          s.Sensor.View(sensor.DefaultFreshness),
		Time:            time.Now().UTC().Format(time.RFC3339),
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 500 {
			writeError(w, http.StatusBadRequest, "limit must be an integer from 1 to 500")
			return
		}
		items, err := s.Captures.Store.List(limit)
		if err != nil {
			s.logf("status list failed: %v", err)
			writeError(w, http.StatusInternalServerError, "capture history unavailable")
			return
		}
		payload.Captures = items
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s Server) sensorHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	defer r.Body.Close()

	var hb sensor.Heartbeat
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&hb); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}
	writeJSON(w, http.StatusOK, s.Sensor.Beat(hb))
}

func (s Server) streamEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ch, cancel := s.Events.Subscribe()
	defer cancel()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			raw, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, raw); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
