package transit

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// MTA GTFS-RT feed URLs by line group
var feedURLs = map[string]string{
	"ace":    "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	"bdfm":   "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	"g":      "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",
	"jz":     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",
	"nqrw":   "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	"l":      "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",
	"1234567": "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	"si":     "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-si",
}

// routeToFeed maps route letters to their feed
var routeToFeed = map[string]string{
	"A": "ace", "C": "ace", "E": "ace",
	"B": "bdfm", "D": "bdfm", "F": "bdfm", "M": "bdfm",
	"G": "g",
	"J": "jz", "Z": "jz",
	"N": "nqrw", "Q": "nqrw", "R": "nqrw", "W": "nqrw",
	"L": "l",
	"1": "1234567", "2": "1234567", "3": "1234567", "4": "1234567",
	"5": "1234567", "6": "1234567", "7": "1234567",
	"SI": "si",
}

// Arrival represents an upcoming train arrival
type Arrival struct {
	Route       string    `json:"route"`
	StopID      string    `json:"stop_id"`
	Direction   string    `json:"direction"`
	ArrivalTime time.Time `json:"arrival_time"`
	MinutesAway int       `json:"minutes_away"`
}

// SubwayService fetches real-time subway arrivals
type SubwayService struct {
	client  *http.Client
	timeout time.Duration
}

// NewSubwayService creates a new subway service
func NewSubwayService(timeout time.Duration) *SubwayService {
	return &SubwayService{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// GetArrivals fetches arrivals for a specific stop
func (s *SubwayService) GetArrivals(stopID string, routes []string) ([]Arrival, error) {
	// Determine which feeds to fetch based on routes
	feeds := s.getFeedsForRoutes(routes)
	
	var allArrivals []Arrival
	for _, feedName := range feeds {
		arrivals, err := s.fetchFeed(feedName, stopID)
		if err != nil {
			continue // Skip failed feeds, try others
		}
		allArrivals = append(allArrivals, arrivals...)
	}

	// Sort by arrival time
	sort.Slice(allArrivals, func(i, j int) bool {
		return allArrivals[i].ArrivalTime.Before(allArrivals[j].ArrivalTime)
	})

	return allArrivals, nil
}

// GetArrivalsForStation fetches arrivals for a station (both directions)
func (s *SubwayService) GetArrivalsForStation(baseStopID string) (map[string][]Arrival, error) {
	// MTA stop IDs: base = parent, N = northbound, S = southbound
	northID := baseStopID + "N"
	southID := baseStopID + "S"

	// Fetch all feeds for comprehensive coverage
	var northArrivals, southArrivals []Arrival

	for feedName := range feedURLs {
		arrivals, err := s.fetchFeed(feedName, "")
		if err != nil {
			continue
		}

		for _, arr := range arrivals {
			if arr.StopID == northID {
				northArrivals = append(northArrivals, arr)
			} else if arr.StopID == southID {
				southArrivals = append(southArrivals, arr)
			}
		}
	}

	sortArrivals(northArrivals)
	sortArrivals(southArrivals)

	return map[string][]Arrival{
		"northbound": northArrivals,
		"southbound": southArrivals,
	}, nil
}

func (s *SubwayService) fetchFeed(feedName, filterStopID string) ([]Arrival, error) {
	url, ok := feedURLs[feedName]
	if !ok {
		return nil, fmt.Errorf("unknown feed: %s", feedName)
	}

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	feed := &gtfs.FeedMessage{}
	if err := proto.Unmarshal(body, feed); err != nil {
		return nil, fmt.Errorf("parsing protobuf: %w", err)
	}

	return s.parseArrivals(feed, filterStopID), nil
}

func (s *SubwayService) parseArrivals(feed *gtfs.FeedMessage, filterStopID string) []Arrival {
	var arrivals []Arrival
	now := time.Now()

	for _, entity := range feed.GetEntity() {
		tripUpdate := entity.GetTripUpdate()
		if tripUpdate == nil {
			continue
		}

		routeID := tripUpdate.GetTrip().GetRouteId()

		for _, stopTimeUpdate := range tripUpdate.GetStopTimeUpdate() {
			stopID := stopTimeUpdate.GetStopId()

			// Filter by stop if specified
			if filterStopID != "" && !strings.HasPrefix(stopID, filterStopID) {
				continue
			}

			arrivalTime := stopTimeUpdate.GetArrival().GetTime()
			if arrivalTime == 0 {
				arrivalTime = stopTimeUpdate.GetDeparture().GetTime()
			}
			if arrivalTime == 0 {
				continue
			}

			arrTime := time.Unix(arrivalTime, 0)
			if arrTime.Before(now) {
				continue // Skip past arrivals
			}

			direction := "unknown"
			if strings.HasSuffix(stopID, "N") {
				direction = "northbound"
			} else if strings.HasSuffix(stopID, "S") {
				direction = "southbound"
			}

			arrivals = append(arrivals, Arrival{
				Route:       routeID,
				StopID:      stopID,
				Direction:   direction,
				ArrivalTime: arrTime,
				MinutesAway: int(arrTime.Sub(now).Minutes()),
			})
		}
	}

	return arrivals
}

func (s *SubwayService) getFeedsForRoutes(routes []string) []string {
	if len(routes) == 0 {
		// Return all feeds
		feeds := make([]string, 0, len(feedURLs))
		for name := range feedURLs {
			feeds = append(feeds, name)
		}
		return feeds
	}

	seen := make(map[string]bool)
	var feeds []string
	for _, route := range routes {
		if feed, ok := routeToFeed[strings.ToUpper(route)]; ok && !seen[feed] {
			seen[feed] = true
			feeds = append(feeds, feed)
		}
	}
	return feeds
}

func sortArrivals(arrivals []Arrival) {
	sort.Slice(arrivals, func(i, j int) bool {
		return arrivals[i].ArrivalTime.Before(arrivals[j].ArrivalTime)
	})
}
