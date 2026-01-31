package api

import (
	"net/http"
	"time"

	"github.com/randytsao24/emteeayy/internal/api/handlers"
	"github.com/randytsao24/emteeayy/internal/config"
)

// NewRouter creates and configures the HTTP router with all routes and middleware
func NewRouter(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()
	rootHandler := handlers.NewRootHandler()

	// Register routes
	mux.HandleFunc("GET /", rootHandler.Index)
	mux.HandleFunc("GET /health", healthHandler.Health)

	// Apply middleware stack
	handler := Chain(mux,
		Recovery,
		Logging,
		CORS,
		Timeout(15*time.Second),
	)

	return handler
}
