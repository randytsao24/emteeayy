package transit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Woodhaven area bus stops
var woodhavenBusStops = map[string]string{
	"jamaicaWoodhaven":  "502494", // Jamaica Ave & Woodhaven Blvd
	"myrtleWoodhaven":   "502528", // Myrtle Ave & Woodhaven Blvd
	"91stWoodhaven":     "504080", // 91st Ave & Woodhaven Blvd
	"101stWoodhaven":    "504067", // 101st Ave & Woodhaven Blvd
}

var woodhavenRoutes = []string{"Q11", "Q52", "Q53", "Q55", "Q56"}

// BusArrival represents an upcoming bus arrival
type BusArrival struct {
	Route           string    `json:"route"`
	Destination     string    `json:"destination"`
	StopID          string    `json:"stop_id"`
	StopsAway       int       `json:"stops_away"`
	ExpectedArrival time.Time `json:"expected_arrival"`
	MinutesAway     int       `json:"minutes_away"`
}

// BusService fetches real-time bus arrivals from MTA SIRI API
type BusService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewBusService creates a new bus service
func NewBusService(apiKey string, timeout time.Duration) *BusService {
	return &BusService{
		apiKey:  apiKey,
		baseURL: "https://bustime.mta.info/api/siri/stop-monitoring.json",
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// HasAPIKey returns true if the service has an API key configured
func (s *BusService) HasAPIKey() bool {
	return s.apiKey != ""
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

	resp, err := s.client.Get(s.baseURL + "?" + params.Encode())
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

// GetWoodhavenArrivals fetches arrivals for all Woodhaven area stops
func (s *BusService) GetWoodhavenArrivals() (map[string][]BusArrival, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("MTA_BUS_API_KEY not configured")
	}

	results := make(map[string][]BusArrival)

	for name, stopID := range woodhavenBusStops {
		arrivals, err := s.GetArrivalsForStop(stopID)
		if err != nil {
			continue // Skip failed stops
		}
		results[name] = arrivals
	}

	return results, nil
}

// GetWoodhavenStops returns the configured Woodhaven stops
func (s *BusService) GetWoodhavenStops() map[string]string {
	return woodhavenBusStops
}

// GetWoodhavenRoutes returns the configured Woodhaven routes
func (s *BusService) GetWoodhavenRoutes() []string {
	return woodhavenRoutes
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

		stopsAway := 0
		if journey.MonitoredCall.Extensions.Distances.StopsFromCall != nil {
			stopsAway = *journey.MonitoredCall.Extensions.Distances.StopsFromCall
		}

		arrivals = append(arrivals, BusArrival{
			Route:           journey.PublishedLineName,
			Destination:     journey.DestinationName,
			StopID:          stopID,
			StopsAway:       stopsAway,
			ExpectedArrival: expectedTime,
			MinutesAway:     int(expectedTime.Sub(now).Minutes()),
		})
	}

	return arrivals
}

// SIRI API response structures
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
	PublishedLineName string `json:"PublishedLineName"`
	DestinationName   string `json:"DestinationName"`
	MonitoredCall     struct {
		ExpectedArrivalTime   time.Time `json:"ExpectedArrivalTime"`
		ExpectedDepartureTime time.Time `json:"ExpectedDepartureTime"`
		Extensions            struct {
			Distances struct {
				StopsFromCall *int `json:"StopsFromCall"`
			} `json:"Distances"`
		} `json:"Extensions"`
	} `json:"MonitoredCall"`
}
