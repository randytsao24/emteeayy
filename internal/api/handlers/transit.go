package handlers

import (
	"net/http"
	"strings"

	"github.com/randytsao24/emteeayy/internal/transit"
)

type TransitHandler struct {
	subway *transit.SubwayService
	bus    *transit.BusService
}

func NewTransitHandler(subway *transit.SubwayService, bus *transit.BusService) *TransitHandler {
	return &TransitHandler{
		subway: subway,
		bus:    bus,
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
	// J15 is Woodhaven Blvd station
	arrivals, err := h.subway.GetArrivals("J15", []string{"J", "Z"})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch J train arrivals",
			"message": err.Error(),
		})
		return
	}

	// Split by direction
	var manhattan, brooklyn []transit.Arrival
	for _, arr := range arrivals {
		if strings.HasSuffix(arr.StopID, "N") {
			manhattan = append(manhattan, arr)
		} else {
			brooklyn = append(brooklyn, arr)
		}
	}

	// Limit to next 5 each
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

// GetBusArrivals returns bus arrivals for Woodhaven area
func (h *TransitHandler) GetBusArrivals(w http.ResponseWriter, r *http.Request) {
	if !h.bus.HasAPIKey() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "Bus service unavailable",
			"message": "MTA_BUS_API_KEY not configured",
		})
		return
	}

	arrivals, err := h.bus.GetWoodhavenArrivals()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "Failed to fetch bus arrivals",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"arrivals": arrivals,
	})
}

// GetBusStops returns Woodhaven bus stop info
func (h *TransitHandler) GetBusStops(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"stops":   h.bus.GetWoodhavenStops(),
		"routes":  h.bus.GetWoodhavenRoutes(),
	})
}
