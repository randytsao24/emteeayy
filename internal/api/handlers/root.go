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
				"GET /transit/location/zip/{zipcode}":         "Find subway stops near zip",
				"GET /transit/location/zip/{zipcode}/closest": "Get N closest subway stops",
			},
			"subway": map[string]string{
				"GET /transit/subway/station/{stopId}":  "Arrivals for any station",
				"GET /transit/subway/near/{zipcode}":    "Subway arrivals near zip code",
				"GET /transit/subway/near?lat=X&lng=Y":  "Subway arrivals near coordinates",
				"GET /transit/subway/stops/{zipcode}":   "Subway stops near zip code",
			},
			"bus": map[string]string{
				"GET /transit/bus/near/{zipcode}":      "Bus arrivals near zip code",
				"GET /transit/bus/near?lat=X&lng=Y":    "Bus arrivals near coordinates",
				"GET /transit/bus/stops/{zipcode}":     "Bus stops near zip code",
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
