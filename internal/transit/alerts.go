package transit

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"github.com/randytsao24/emteeayy/internal/cache"
	"google.golang.org/protobuf/proto"
)

const alertsFeedURL = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/camsys%2Fall-alerts"

// ServiceAlert represents an active MTA service alert
type ServiceAlert struct {
	ID          string   `json:"id"`
	Routes      []string `json:"routes"`
	Header      string   `json:"header"`
	Description string   `json:"description"`
}

// AlertService fetches and caches MTA service alerts
type AlertService struct {
	client *http.Client
	cache  *cache.Cache[[]ServiceAlert]
}

// NewAlertService creates a new alert service
func NewAlertService(timeout time.Duration, cacheTTL time.Duration) *AlertService {
	return &AlertService{
		client: &http.Client{Timeout: timeout},
		cache:  cache.New[[]ServiceAlert](cacheTTL),
	}
}

// GetAlerts returns active service alerts, optionally filtered by route
func (s *AlertService) GetAlerts(routes []string) ([]ServiceAlert, error) {
	allAlerts, err := s.fetchAlerts()
	if err != nil {
		return nil, err
	}

	if len(routes) == 0 {
		return allAlerts, nil
	}

	routeSet := make(map[string]bool, len(routes))
	for _, r := range routes {
		routeSet[r] = true
	}

	var filtered []ServiceAlert
	for _, alert := range allAlerts {
		for _, r := range alert.Routes {
			if routeSet[r] {
				filtered = append(filtered, alert)
				break
			}
		}
	}
	return filtered, nil
}

func (s *AlertService) fetchAlerts() ([]ServiceAlert, error) {
	if cached, ok := s.cache.Get("all"); ok {
		return cached, nil
	}

	resp, err := s.client.Get(alertsFeedURL)
	if err != nil {
		return nil, fmt.Errorf("fetching alerts feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alerts feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading alerts response: %w", err)
	}

	feed := &gtfs.FeedMessage{}
	if err := proto.Unmarshal(body, feed); err != nil {
		return nil, fmt.Errorf("parsing alerts protobuf: %w", err)
	}

	alerts := s.parseAlerts(feed)
	s.cache.Set("all", alerts)
	return alerts, nil
}

func (s *AlertService) parseAlerts(feed *gtfs.FeedMessage) []ServiceAlert {
	var alerts []ServiceAlert
	now := time.Now().Unix()

	for _, entity := range feed.GetEntity() {
		alert := entity.GetAlert()
		if alert == nil {
			continue
		}

		active := len(alert.GetActivePeriod()) == 0
		for _, period := range alert.GetActivePeriod() {
			start := int64(period.GetStart())
			end := int64(period.GetEnd())
			if now >= start && (end == 0 || now < end) {
				active = true
				break
			}
		}
		if !active {
			continue
		}

		var routes []string
		seen := make(map[string]bool)
		for _, ie := range alert.GetInformedEntity() {
			if routeID := ie.GetRouteId(); routeID != "" && !seen[routeID] {
				seen[routeID] = true
				routes = append(routes, routeID)
			}
		}

		header := translatedText(alert.GetHeaderText())
		if header == "" {
			continue
		}

		alerts = append(alerts, ServiceAlert{
			ID:          entity.GetId(),
			Routes:      routes,
			Header:      header,
			Description: translatedText(alert.GetDescriptionText()),
		})
	}

	return alerts
}

func translatedText(ts *gtfs.TranslatedString) string {
	if ts == nil {
		return ""
	}
	for _, t := range ts.GetTranslation() {
		if t.GetLanguage() == "en" || t.GetLanguage() == "" {
			return t.GetText()
		}
	}
	if len(ts.GetTranslation()) > 0 {
		return ts.GetTranslation()[0].GetText()
	}
	return ""
}
