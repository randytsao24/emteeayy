package api

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/randytsao24/emteeayy/internal/api/handlers"
	"github.com/randytsao24/emteeayy/internal/config"
	"github.com/randytsao24/emteeayy/internal/location"
	"github.com/randytsao24/emteeayy/internal/transit"
)

// NewRouter creates and configures the HTTP router with all routes and middleware
func NewRouter(
	cfg *config.Config,
	zipSvc *location.ZipCodeService,
	stopSvc *location.StopService,
	subwaySvc *transit.SubwayService,
	busSvc *transit.BusService,
	webFS fs.FS,
) http.Handler {
	mux := http.NewServeMux()

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()
	rootHandler := handlers.NewRootHandler()
	locationHandler := handlers.NewLocationHandler(zipSvc, stopSvc)
	transitHandler := handlers.NewTransitHandler(subwaySvc, busSvc, stopSvc, zipSvc)

	// Serve frontend (if provided)
	if webFS != nil {
		mux.Handle("GET /", http.FileServer(http.FS(webFS)))
	} else {
		mux.HandleFunc("GET /", rootHandler.Index)
	}

	// Core routes
	mux.HandleFunc("GET /api", rootHandler.Index)
	mux.HandleFunc("GET /health", healthHandler.Health)

	// Location routes (subway stops)
	mux.HandleFunc("GET /transit/location/info", locationHandler.GetLocationInfo)
	mux.HandleFunc("GET /transit/location/boroughs", locationHandler.GetBoroughs)
	mux.HandleFunc("GET /transit/location/zipcodes/all", locationHandler.GetAllZipCodes)
	mux.HandleFunc("GET /transit/location/zip/{zipcode}/closest", locationHandler.GetClosestStops)
	mux.HandleFunc("GET /transit/location/zip/{zipcode}", locationHandler.GetStopsByZip)

	// Subway routes - station-specific
	mux.HandleFunc("GET /transit/subway/station/{stopId}", transitHandler.GetSubwayArrivals)

	// Subway routes - dynamic location-based
	mux.HandleFunc("GET /transit/subway/near/{zipcode}", transitHandler.GetSubwayArrivalsNearZip)
	mux.HandleFunc("GET /transit/subway/near", transitHandler.GetSubwayArrivalsNearCoords)
	mux.HandleFunc("GET /transit/subway/stops/{zipcode}", transitHandler.GetSubwayStopsNear)

	// Bus routes - dynamic location-based
	mux.HandleFunc("GET /transit/bus/near/{zipcode}", transitHandler.GetBusArrivalsNearZip)
	mux.HandleFunc("GET /transit/bus/near", transitHandler.GetBusArrivalsNearCoords)
	mux.HandleFunc("GET /transit/bus/stops/{zipcode}", transitHandler.GetBusStopsNear)

	// Apply middleware stack
	handler := Chain(mux,
		Recovery,
		Logging,
		CORS,
		Timeout(15*time.Second),
	)

	return handler
}
