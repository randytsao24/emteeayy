package handlers

import "github.com/randytsao24/emteeayy/internal/transit"

// SubwayProvider abstracts the subway data source for testability.
type SubwayProvider interface {
	GetArrivalsForStation(stopID string) (map[string][]transit.Arrival, error)
	GetArrivalsForStations(stopIDs []string) ([]transit.StationArrivals, error)
}

// BusProvider abstracts the bus data source for testability.
type BusProvider interface {
	HasAPIKey() bool
	FindStopsNear(lat, lng float64, radiusMeters int) ([]transit.BusStop, error)
	GetArrivalsNear(lat, lng float64, radiusMeters, limit int) ([]transit.BusArrival, error)
}

// AlertProvider abstracts the service alerts data source.
type AlertProvider interface {
	GetAlerts(routes []string) ([]transit.ServiceAlert, error)
}
