// Package main is the entry point for the emteeayy server.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/randytsao24/emteeayy/internal/config"
)

func main() {
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration error: ", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /", handleRoot)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("🚇 emteeayy server starting on port %s\n", cfg.Port)
	fmt.Printf("📍 Environment: %s\n", cfg.Env)
	fmt.Printf("🔗 http://localhost:%s\n", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
		Version   string `json:"version"`
	}{
		Status:    "OK",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "2.0.0-go",
	}

	writeJSON(w, http.StatusOK, response)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Version     string            `json:"version"`
		Endpoints   map[string]string `json:"endpoints"`
	}{
		Name:        "emteeayy",
		Description: "Real-time MTA transit tracking for NYC - Go edition!",
		Version:     "2.0.0-go",
		Endpoints: map[string]string{
			"GET /":       "API information",
			"GET /health": "Health check",
		},
	}

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
