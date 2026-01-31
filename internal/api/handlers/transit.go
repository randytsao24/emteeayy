package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/randytsao24/emteeayy/internal/location"
	"github.com/randytsao24/emteeayy/internal/transit"
)

type TransitHandler struct {
	subway   *transit.SubwayService
	bus      *transit.BusService
	zipCodes *location.ZipCodeService
}

func NewTransitHandler(subway *transit.SubwayService, bus *transit.BusService, zips *location.ZipCodeService) *TransitHandler {
	return &TransitHandler{
		subway:   subway,
		bus:      bus,
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

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"stop_id":  stopID,
		"arrivals": arrivals,
	})
}

// GetJTrainArrivals returns J/Z train arrivals for Woodhaven Blvd
func (h *TransitHandler) GetJTrainArrivals(w http.ResponseWriter, r *http.Request) {
	arrivals, err := h.subway.GetArrivals("J15", []string{"J", "Z"})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch J train arrivals",
			"message": err.Error(),
		})
		return
	}

	var manhattan, brooklyn []transit.Arrival
	for _, arr := range arrivals {
		if strings.HasSuffix(arr.StopID, "N") {
			manhattan = append(manhattan, arr)
		} else {
			brooklyn = append(brooklyn, arr)
		}
	}

	if len(manhattan) > 5 {
		manhattan = manhattan[:5]
	}
	if len(brooklyn) > 5 {
		brooklyn = brooklyn[:5]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"station": "Woodhaven Blvd",
		"stop_id": "J15",
		"arrivals": map[string]any{
			"manhattan_bound": manhattan,
			"brooklyn_bound":  brooklyn,
		},
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

	radius := parseIntQueryParam(r, "radius", 400, 100, 1000)
	arrivals, err := h.bus.GetArrivalsNear(zip.Lat, zip.Lng, radius)
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

	radius := parseIntQueryParam(r, "radius", 400, 100, 1000)
	arrivals, err := h.bus.GetArrivalsNear(lat, lng, radius)
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

	radius := parseIntQueryParam(r, "radius", 400, 100, 1000)
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
