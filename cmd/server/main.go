// Package main is the entry point for the emteeayy server
package main

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/randytsao24/emteeayy/internal/api"
	"github.com/randytsao24/emteeayy/internal/config"
	"github.com/randytsao24/emteeayy/internal/location"
	"github.com/randytsao24/emteeayy/internal/transit"
	"github.com/randytsao24/emteeayy/web"
)

func main() {
	// Load .env file (ignore error if not found)
	_ = godotenv.Load()
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

	// Find data directory
	dataDir := findDataDir()

	// Initialize location services
	zipSvc := location.NewZipCodeService()
	if err := zipSvc.Load(filepath.Join(dataDir, "nyc-zipcodes.json")); err != nil {
		log.Fatal("Failed to load zip codes: ", err)
	}
	slog.Info("loaded zip codes", "count", zipSvc.Count())

	stopSvc := location.NewStopService()
	if err := stopSvc.Load(filepath.Join(dataDir, "stops.txt")); err != nil {
		log.Fatal("Failed to load stops: ", err)
	}
	slog.Info("loaded subway stops", "total", stopSvc.Count(), "stations", stopSvc.ParentStationCount())

	// Initialize transit services
	subwaySvc := transit.NewSubwayService(cfg.HTTPTimeout, cfg.CacheTTL)
	slog.Info("initialized subway service", "cache_ttl", cfg.CacheTTL)

	busSvc := transit.NewBusService(cfg.MTABusAPIKey, cfg.HTTPTimeout, cfg.CacheTTL)
	if busSvc.HasAPIKey() {
		slog.Info("initialized bus service")
	} else {
		slog.Warn("bus service disabled - MTA_BUS_API_KEY not set")
	}

	alertSvc := transit.NewAlertService(cfg.HTTPTimeout, cfg.CacheTTL)
	slog.Info("initialized alerts service")

	// In development, serve web files from disk so frontend changes are
	// picked up instantly without rebuilding the binary.
	var webFS fs.FS = web.FS
	if cfg.IsDevelopment() {
		webFS = os.DirFS("web")
		slog.Info("serving frontend from disk (dev mode)")
	}

	// Create router with all routes and middleware
	router := api.NewRouter(cfg, zipSvc, stopSvc, subwaySvc, busSvc, alertSvc, webFS)

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

func findDataDir() string {
	if _, err := os.Stat("data"); err == nil {
		return "data"
	}

	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "data")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	return "data"
}
