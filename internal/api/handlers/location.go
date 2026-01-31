package handlers

import (
	"net/http"
	"strconv"

	"github.com/randytsao24/emteeayy/internal/location"
)

const (
	defaultRadius = 1600  // ~1 mile in meters
	maxRadius     = 8000  // ~5 miles
	minRadius     = 50
	defaultLimit  = 5
	maxLimit      = 20
)

type LocationHandler struct {
	zipCodes *location.ZipCodeService
	stops    *location.StopService
}

func NewLocationHandler(zips *location.ZipCodeService, stops *location.StopService) *LocationHandler {
	return &LocationHandler{
		zipCodes: zips,
		stops:    stops,
	}
}

// GetStopsByZip finds stops near a zip code
func (h *LocationHandler) GetStopsByZip(w http.ResponseWriter, r *http.Request) {
	zipCode := r.PathValue("zipcode")

	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "Invalid zip code format",
			"message": "Zip code must be exactly 5 digits",
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

	radius := parseIntParam(r, "radius", defaultRadius, minRadius, maxRadius)
	stops := h.stops.FindNearby(zip.Lat, zip.Lng, float64(radius))

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"zip_code":      zipCode,
		"location":      zip,
		"radius_meters": radius,
		"stops":         stops,
		"metadata": map[string]any{
			"stops_found": len(stops),
		},
	})
}

// GetClosestStops returns the N closest stops to a zip code
func (h *LocationHandler) GetClosestStops(w http.ResponseWriter, r *http.Request) {
	zipCode := r.PathValue("zipcode")

	if len(zipCode) != 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "Invalid zip code format",
			"message": "Zip code must be exactly 5 digits",
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

	limit := parseIntParam(r, "limit", defaultLimit, 1, maxLimit)
	stops := h.stops.FindClosest(zip.Lat, zip.Lng, limit)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"zip_code": zipCode,
		"location": zip,
		"stops":    stops,
		"metadata": map[string]any{
			"stops_found": len(stops),
		},
	})
}

// GetAllZipCodes returns all zip codes, optionally filtered by borough
func (h *LocationHandler) GetAllZipCodes(w http.ResponseWriter, r *http.Request) {
	borough := r.URL.Query().Get("borough")

	var zips []any
	if borough != "" {
		for _, z := range h.zipCodes.GetByBorough(borough) {
			zips = append(zips, z)
		}
	} else {
		for _, z := range h.zipCodes.GetAll() {
			zips = append(zips, z)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"count":    len(zips),
		"zipcodes": zips,
	})
}

// GetBoroughs returns all boroughs
func (h *LocationHandler) GetBoroughs(w http.ResponseWriter, r *http.Request) {
	boroughs := h.zipCodes.Boroughs()

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"count":    len(boroughs),
		"boroughs": boroughs,
	})
}

// GetLocationInfo returns service info
func (h *LocationHandler) GetLocationInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"service":     "NYC Zip Code Transit Lookup",
		"description": "Find nearby subway stops by entering a NYC zip code",
		"coverage": map[string]any{
			"zipcodes":       h.zipCodes.Count(),
			"subway_stations": h.stops.ParentStationCount(),
		},
		"defaults": map[string]any{
			"radius_meters": defaultRadius,
			"limit":         defaultLimit,
		},
	})
}

func parseIntParam(r *http.Request, name string, defaultVal, min, max int) int {
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
