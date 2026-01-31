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
		"description": "Real-time MTA transit tracking for NYC",
		"version":     "1.0.0",
		"endpoints": map[string]any{
			"core": map[string]string{
				"GET /":       "API information",
				"GET /health": "Health check",
			},
			"location": map[string]string{
				"GET /transit/location/info":                  "Service info",
				"GET /transit/location/boroughs":              "List all boroughs",
				"GET /transit/location/zipcodes/all":          "List all zip codes",
				"GET /transit/location/zip/{zipcode}":         "Find stops near zip code",
				"GET /transit/location/zip/{zipcode}/closest": "Get N closest stops",
			},
			"subway": map[string]string{
				"GET /transit/subway/j-train":            "J/Z arrivals at Woodhaven Blvd",
				"GET /transit/subway/station/{stopId}":  "Arrivals for any station",
			},
			"bus": map[string]string{
				"GET /transit/bus/arrivals": "Bus arrivals for Woodhaven area",
				"GET /transit/bus/stops":    "Woodhaven bus stops and routes",
			},
		},
	})
}

func (h *RootHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]any{
		"error":   "Route not found",
		"message": "Check the root endpoint (/) for available routes",
	})
}
