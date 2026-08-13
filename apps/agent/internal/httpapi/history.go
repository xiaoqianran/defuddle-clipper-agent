package httpapi

import (
	"errors"
	"net/http"
	"os"
	"strconv"
)

func (s Server) listCaptures(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 500 {
			writeError(w, http.StatusBadRequest, "limit must be an integer from 1 to 500")
			return
		}
		limit = value
	}

	items, err := s.Captures.Store.List(limit)
	if err != nil {
		s.logf("list captures failed: %v", err)
		writeError(w, http.StatusInternalServerError, "capture history unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s Server) readCapture(w http.ResponseWriter, r *http.Request) {
	view, err := s.Captures.Store.Read(r.PathValue("captureID"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "capture not found")
		return
	}
	if err != nil {
		s.logf("read capture failed: %v", err)
		writeError(w, http.StatusInternalServerError, "capture unavailable")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s Server) reprocessCapture(w http.ResponseWriter, r *http.Request) {
	captureID := r.PathValue("captureID")
	result, err := s.Captures.Reprocess(r.Context(), captureID)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "capture not found")
		return
	}
	if err != nil {
		s.logf("reprocess %s failed: %v", captureID, err)
		writeError(w, http.StatusInternalServerError, "capture reprocess failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
