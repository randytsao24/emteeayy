package transit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const (
	defaultBusRadius = 400 // meters
	maxBusStops      = 5
)

// BusStop represents a bus stop from the MTA API
type BusStop struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	Direction string   `json:"direction,omitempty"`
	Routes    []string `json:"routes,omitempty"`
}

// BusArrival represents an upcoming bus arrival
type BusArrival struct {
	Route           string    `json:"route"`
	Destination     string    `json:"destination"`
	StopID          string    `json:"stop_id"`
	StopName        string    `json:"stop_name,omitempty"`
	StopsAway       int       `json:"stops_away"`
	Feet            int       `json:"feet_away"`
	ExpectedArrival time.Time `json:"expected_arrival"`
	MinutesAway     int       `json:"minutes_away"`
}

// BusService fetches real-time bus arrivals from MTA SIRI API
type BusService struct {
	apiKey  string
	client  *http.Client
}

// NewBusService creates a new bus service
func NewBusService(apiKey string, timeout time.Duration) *BusService {
	return &BusService{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// HasAPIKey returns true if the service has an API key configured
func (s *BusService) HasAPIKey() bool {
	return s.apiKey != ""
}

// FindStopsNear finds bus stops near a location
func (s *BusService) FindStopsNear(lat, lng float64, radiusMeters int) ([]BusStop, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("MTA_BUS_API_KEY not configured")
	}

	if radiusMeters <= 0 {
		radiusMeters = defaultBusRadius
	}

	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("lat", fmt.Sprintf("%f", lat))
	params.Set("lon", fmt.Sprintf("%f", lng))
	params.Set("radius", fmt.Sprintf("%d", radiusMeters))

	apiURL := "https://bustime.mta.info/api/where/stops-for-location.json?" + params.Encode()
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching stops: %w", err)
	}
	defer resp.Body.Close()

	var result stopsForLocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var stops []BusStop
	for _, stop := range result.Data.Stops {
		stops = append(stops, BusStop{
			ID:        stop.ID,
			Name:      stop.Name,
			Lat:       stop.Lat,
			Lng:       stop.Lon,
			Direction: stop.Direction,
		})
	}

	return stops, nil
}

// GetArrivalsNear finds stops near a location and fetches arrivals for each
func (s *BusService) GetArrivalsNear(lat, lng float64, radiusMeters int) ([]BusArrival, error) {
	stops, err := s.FindStopsNear(lat, lng, radiusMeters)
	if err != nil {
		return nil, err
	}

	// Limit number of stops to query
	if len(stops) > maxBusStops {
		stops = stops[:maxBusStops]
	}

	var allArrivals []BusArrival
	for _, stop := range stops {
		arrivals, err := s.GetArrivalsForStop(stop.ID)
		if err != nil {
			continue
		}
		// Add stop name to each arrival
		for i := range arrivals {
			arrivals[i].StopName = stop.Name
		}
		allArrivals = append(allArrivals, arrivals...)
	}

	// Sort by arrival time
	sort.Slice(allArrivals, func(i, j int) bool {
		return allArrivals[i].ExpectedArrival.Before(allArrivals[j].ExpectedArrival)
	})

	return allArrivals, nil
}

// GetArrivalsForStop fetches arrivals for a specific stop
func (s *BusService) GetArrivalsForStop(stopID string) ([]BusArrival, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("MTA_BUS_API_KEY not configured")
	}

	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("MonitoringRef", stopID)
	params.Set("version", "2")

	apiURL := "https://bustime.mta.info/api/siri/stop-monitoring.json?" + params.Encode()
	resp, err := s.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching bus data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bus API returned status %d", resp.StatusCode)
	}

	var result siriResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return s.parseArrivals(result, stopID), nil
}

func (s *BusService) parseArrivals(resp siriResponse, stopID string) []BusArrival {
	var arrivals []BusArrival
	now := time.Now()

	delivery := resp.Siri.ServiceDelivery.StopMonitoringDelivery
	if len(delivery) == 0 {
		return arrivals
	}

	for _, visit := range delivery[0].MonitoredStopVisit {
		journey := visit.MonitoredVehicleJourney

		expectedTime := journey.MonitoredCall.ExpectedArrivalTime
		if expectedTime.IsZero() {
			expectedTime = journey.MonitoredCall.ExpectedDepartureTime
		}

		// Skip entries with no valid arrival time
		if expectedTime.IsZero() {
			continue
		}

		route := getFirstString(journey.PublishedLineName)
		destination := getFirstString(journey.DestinationName)

		stopsAway := 0
		feetAway := 0
		if journey.MonitoredCall.Extensions.Distances.StopsFromCall != nil {
			stopsAway = *journey.MonitoredCall.Extensions.Distances.StopsFromCall
		}
		if journey.MonitoredCall.Extensions.Distances.DistanceFromCall != nil {
			feetAway = *journey.MonitoredCall.Extensions.Distances.DistanceFromCall
		}

		arrivals = append(arrivals, BusArrival{
			Route:           route,
			Destination:     destination,
			StopID:          stopID,
			StopsAway:       stopsAway,
			Feet:            feetAway,
			ExpectedArrival: expectedTime,
			MinutesAway:     int(expectedTime.Sub(now).Minutes()),
		})
	}

	return arrivals
}

// getFirstString handles fields that can be string or []string
func getFirstString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		if len(val) > 0 {
			if s, ok := val[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// API response structures
type stopsForLocationResponse struct {
	Data struct {
		Stops []struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			Direction string  `json:"direction"`
		} `json:"stops"`
	} `json:"data"`
}

type siriResponse struct {
	Siri struct {
		ServiceDelivery struct {
			StopMonitoringDelivery []struct {
				MonitoredStopVisit []struct {
					MonitoredVehicleJourney monitoredVehicleJourney `json:"MonitoredVehicleJourney"`
				} `json:"MonitoredStopVisit"`
			} `json:"StopMonitoringDelivery"`
		} `json:"ServiceDelivery"`
	} `json:"Siri"`
}

type monitoredVehicleJourney struct {
	PublishedLineName any `json:"PublishedLineName"`
	DestinationName   any `json:"DestinationName"`
	MonitoredCall     struct {
		ExpectedArrivalTime   time.Time `json:"ExpectedArrivalTime"`
		ExpectedDepartureTime time.Time `json:"ExpectedDepartureTime"`
		Extensions            struct {
			Distances struct {
				StopsFromCall    *int `json:"StopsFromCall"`
				DistanceFromCall *int `json:"DistanceFromCall"`
			} `json:"Distances"`
		} `json:"Extensions"`
	} `json:"MonitoredCall"`
}
