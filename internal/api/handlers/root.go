package handlers

import (
	"net/http"
)

type RootHandler struct{}

func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

func (h *RootHandler) Index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        "emteeayy",
		"description": "Real-time MTA transit tracking for NYC - Go edition!",
		"version":     "2.0.0-go",
		"endpoints": map[string]string{
			"GET /":       "API information",
			"GET /health": "Health check",
		},
	})
}

func (h *RootHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]any{
		"error":   "Route not found",
		"message": "Check the root endpoint (/) for available routes",
	})
}
