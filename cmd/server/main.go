// Package main is the entry point for the emteeayy server
package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/randytsao24/emteeayy/internal/api"
	"github.com/randytsao24/emteeayy/internal/config"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load and validate configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration error: ", err)
	}

	// Create router with all routes and middleware
	router := api.NewRouter(cfg)

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
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
