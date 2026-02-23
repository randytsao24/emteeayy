package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/randytsao24/emteeayy/internal/location"
	"github.com/randytsao24/emteeayy/internal/transit"
)

const (
	defaultSubwayRadius  = 800  // ~0.5 mile in meters
	maxSubwayRadius      = 3200 // ~2 miles
	minSubwayRadius      = 100
	defaultStationsLimit = 3
	maxStationsLimit     = 5
)

type TransitHandler struct {
	subway   SubwayProvider
	bus      BusProvider
	alerts   AlertProvider
	stops    *location.StopService
	zipCodes *location.ZipCodeService
}

func NewTransitHandler(subway SubwayProvider, bus BusProvider, alerts AlertProvider, stops *location.StopService, zips *location.ZipCodeService) *TransitHandler {
	return &TransitHandler{
		subway:   subway,
		bus:      bus,
		alerts:   alerts,
		stops:    stops,
		zipCodes: zips,
	}
}

// GetSubwayArrivals returns arrivals for a station
func (h *TransitHandler) GetSubwayArrivals(w http.ResponseWriter, r *http.Request) {
	stopID := r.PathValue("stopId")
	if stopID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Stop ID is required",
		})
		return
	}

	arrivals, err := h.subway.GetArrivalsForStation(stopID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch arrivals",
			"message": err.Error(),
		})
		return
	}

	h.resolveDestinations(arrivals["northbound"])
	h.resolveDestinations(arrivals["southbound"])

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"stop_id":  stopID,
		"arrivals": arrivals,
	})
}

// GetSubwayArrivalsNearZip returns subway arrivals near a zip code
func (h *TransitHandler) GetSubwayArrivalsNearZip(w http.ResponseWriter, r *http.Request) {
	zipCode := r.PathValue("zipcode")
	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid zip code format",
		})
		return
	}

	zip, found := h.zipCodes.Get(zipCode)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "Zip code not found",
			"message": "Zip code " + zipCode + " is not in our NYC database",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", defaultSubwayRadius, minSubwayRadius, maxSubwayRadius)
	limit := parseIntQueryParam(r, "limit", defaultStationsLimit, 1, maxStationsLimit)

	// Find nearby subway stations
	nearbyStops := h.stops.FindNearby(zip.Lat, zip.Lng, float64(radius))
	if len(nearbyStops) > limit {
		nearbyStops = nearbyStops[:limit]
	}

	if len(nearbyStops) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"success":       true,
			"zip_code":      zipCode,
			"location":      zip,
			"radius_meters": radius,
			"stations":      []any{},
			"count":         0,
			"message":       "No subway stations found within radius",
		})
		return
	}

	// Extract stop IDs for arrival lookup
	stopIDs := make([]string, len(nearbyStops))
	for i, stop := range nearbyStops {
		stopIDs[i] = stop.ID
	}

	// Fetch arrivals for all nearby stations
	stationArrivals, err := h.subway.GetArrivalsForStations(stopIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch subway arrivals",
			"message": err.Error(),
		})
		return
	}

	// Enrich station arrivals with stop info
	for i := range stationArrivals {
		if i < len(nearbyStops) {
			stationArrivals[i].StopName = nearbyStops[i].Name
			stationArrivals[i].Lat = nearbyStops[i].Lat
			stationArrivals[i].Lng = nearbyStops[i].Lng
			stationArrivals[i].DistanceMeters = nearbyStops[i].DistanceMeters
			stationArrivals[i].DistanceMiles = nearbyStops[i].DistanceMiles
		}
	}
	h.resolveStationDestinations(stationArrivals)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"zip_code":      zipCode,
		"location":      zip,
		"radius_meters": radius,
		"stations":      stationArrivals,
		"count":         len(stationArrivals),
	})
}

// GetSubwayArrivalsNearCoords returns subway arrivals near lat/lng coordinates
func (h *TransitHandler) GetSubwayArrivalsNearCoords(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")

	if latStr == "" || lngStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "lat and lng query parameters are required",
		})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid lat parameter",
		})
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid lng parameter",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", defaultSubwayRadius, minSubwayRadius, maxSubwayRadius)
	limit := parseIntQueryParam(r, "limit", defaultStationsLimit, 1, maxStationsLimit)

	// Find nearby subway stations
	nearbyStops := h.stops.FindNearby(lat, lng, float64(radius))
	if len(nearbyStops) > limit {
		nearbyStops = nearbyStops[:limit]
	}

	if len(nearbyStops) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"success":       true,
			"lat":           lat,
			"lng":           lng,
			"radius_meters": radius,
			"stations":      []any{},
			"count":         0,
			"message":       "No subway stations found within radius",
		})
		return
	}

	// Extract stop IDs for arrival lookup
	stopIDs := make([]string, len(nearbyStops))
	for i, stop := range nearbyStops {
		stopIDs[i] = stop.ID
	}

	// Fetch arrivals for all nearby stations
	stationArrivals, err := h.subway.GetArrivalsForStations(stopIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch subway arrivals",
			"message": err.Error(),
		})
		return
	}

	// Enrich station arrivals with stop info
	for i := range stationArrivals {
		if i < len(nearbyStops) {
			stationArrivals[i].StopName = nearbyStops[i].Name
			stationArrivals[i].Lat = nearbyStops[i].Lat
			stationArrivals[i].Lng = nearbyStops[i].Lng
			stationArrivals[i].DistanceMeters = nearbyStops[i].DistanceMeters
			stationArrivals[i].DistanceMiles = nearbyStops[i].DistanceMiles
		}
	}
	h.resolveStationDestinations(stationArrivals)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"lat":           lat,
		"lng":           lng,
		"radius_meters": radius,
		"stations":      stationArrivals,
		"count":         len(stationArrivals),
	})
}

// GetSubwayStopsNear returns subway stops near a zip code
func (h *TransitHandler) GetSubwayStopsNear(w http.ResponseWriter, r *http.Request) {
	zipCode := r.PathValue("zipcode")
	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid zip code format",
		})
		return
	}

	zip, found := h.zipCodes.Get(zipCode)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "Zip code not found",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", defaultSubwayRadius, minSubwayRadius, maxSubwayRadius)
	stops := h.stops.FindNearby(zip.Lat, zip.Lng, float64(radius))

	// Convert to simpler response format
	var stopsResponse []transit.SubwayStop
	for _, stop := range stops {
		stopsResponse = append(stopsResponse, transit.SubwayStop{
			ID:             stop.ID,
			Name:           stop.Name,
			Lat:            stop.Lat,
			Lng:            stop.Lng,
			DistanceMeters: stop.DistanceMeters,
			DistanceMiles:  stop.DistanceMiles,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"zip_code":      zipCode,
		"location":      zip,
		"radius_meters": radius,
		"stops":         stopsResponse,
		"count":         len(stopsResponse),
	})
}

// GetBusArrivalsNearZip returns bus arrivals near a zip code
func (h *TransitHandler) GetBusArrivalsNearZip(w http.ResponseWriter, r *http.Request) {
	if !h.bus.HasAPIKey() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "Bus service unavailable",
			"message": "MTA_BUS_API_KEY not configured",
		})
		return
	}

	zipCode := r.PathValue("zipcode")
	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid zip code format",
		})
		return
	}

	zip, found := h.zipCodes.Get(zipCode)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "Zip code not found",
			"message": "Zip code " + zipCode + " is not in our NYC database",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", 400, 100, maxSubwayRadius)
	limit := parseIntQueryParam(r, "limit", transit.DefaultBusLimit, 1, transit.MaxBusStops)
	arrivals, err := h.bus.GetArrivalsNear(zip.Lat, zip.Lng, radius, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch bus arrivals",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"zip_code":      zipCode,
		"location":      zip,
		"radius_meters": radius,
		"arrivals":      arrivals,
		"count":         len(arrivals),
	})
}

// GetBusArrivalsNearCoords returns bus arrivals near lat/lng coordinates
func (h *TransitHandler) GetBusArrivalsNearCoords(w http.ResponseWriter, r *http.Request) {
	if !h.bus.HasAPIKey() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "Bus service unavailable",
			"message": "MTA_BUS_API_KEY not configured",
		})
		return
	}

	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")

	if latStr == "" || lngStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "lat and lng query parameters are required",
		})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid lat parameter",
		})
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid lng parameter",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", 400, 100, maxSubwayRadius)
	limit := parseIntQueryParam(r, "limit", transit.DefaultBusLimit, 1, transit.MaxBusStops)
	arrivals, err := h.bus.GetArrivalsNear(lat, lng, radius, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch bus arrivals",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"lat":           lat,
		"lng":           lng,
		"radius_meters": radius,
		"arrivals":      arrivals,
		"count":         len(arrivals),
	})
}

// GetBusStopsNear returns bus stops near a location
func (h *TransitHandler) GetBusStopsNear(w http.ResponseWriter, r *http.Request) {
	if !h.bus.HasAPIKey() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "Bus service unavailable",
		})
		return
	}

	zipCode := r.PathValue("zipcode")
	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid zip code format",
		})
		return
	}

	zip, found := h.zipCodes.Get(zipCode)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "Zip code not found",
		})
		return
	}

	radius := parseIntQueryParam(r, "radius", 400, 100, maxSubwayRadius)
	stops, err := h.bus.FindStopsNear(zip.Lat, zip.Lng, radius)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to find bus stops",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"zip_code":      zipCode,
		"location":      zip,
		"radius_meters": radius,
		"stops":         stops,
		"count":         len(stops),
	})
}

// GetServiceAlerts returns active service alerts, optionally filtered by route
func (h *TransitHandler) GetServiceAlerts(w http.ResponseWriter, r *http.Request) {
	routesParam := r.URL.Query().Get("routes")
	var routes []string
	if routesParam != "" {
		routes = strings.Split(routesParam, ",")
	}

	alerts, err := h.alerts.GetAlerts(routes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch service alerts",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"alerts":  alerts,
		"count":   len(alerts),
	})
}

// GetSubwayArrivalsForStops returns arrivals for specific station IDs (used by favorites)
func (h *TransitHandler) GetSubwayArrivalsForStops(w http.ResponseWriter, r *http.Request) {
	stopsParam := r.URL.Query().Get("stops")
	if stopsParam == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "stops query parameter is required (comma-separated stop IDs)",
		})
		return
	}

	stopIDs := strings.Split(stopsParam, ",")
	if len(stopIDs) > maxStationsLimit {
		stopIDs = stopIDs[:maxStationsLimit]
	}

	stationArrivals, err := h.subway.GetArrivalsForStations(stopIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch arrivals",
			"message": err.Error(),
		})
		return
	}

	for i := range stationArrivals {
		if stop, ok := h.stops.GetByID(stationArrivals[i].StopID); ok {
			stationArrivals[i].StopName = stop.Name
			stationArrivals[i].Lat = stop.Lat
			stationArrivals[i].Lng = stop.Lng
		}
	}
	h.resolveStationDestinations(stationArrivals)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"stations": stationArrivals,
		"count":    len(stationArrivals),
	})
}

func (h *TransitHandler) resolveDestinations(arrivals []transit.Arrival) {
	for i := range arrivals {
		if arrivals[i].Destination == "" {
			continue
		}
		stop, ok := h.stops.GetByID(arrivals[i].Destination)
		if !ok {
			continue
		}
		name := stop.Name
		if zip, found := h.zipCodes.FindNearest(stop.Lat, stop.Lng); found && zip.Borough != "" {
			name += " (" + zip.Borough + ")"
		}
		arrivals[i].Destination = name
	}
}

func (h *TransitHandler) resolveStationDestinations(stations []transit.StationArrivals) {
	for i := range stations {
		h.resolveDestinations(stations[i].Northbound)
		h.resolveDestinations(stations[i].Southbound)
	}
}

func parseIntQueryParam(r *http.Request, name string, defaultVal, min, max int) int {
	str := r.URL.Query().Get(name)
	if str == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal
	}

	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
