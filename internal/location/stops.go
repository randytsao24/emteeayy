package location

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/randytsao24/emteeayy/internal/models"
)

// StopService manages subway stop data
type StopService struct {
	stops  []models.Stop
	mu     sync.RWMutex
	loaded bool
}

// NewStopService creates a new stop service
func NewStopService() *StopService {
	return &StopService{}
}

// Load reads stop data from a GTFS stops.txt file
func (s *StopService) Load(filepath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("opening stops file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("stops file has no data rows")
	}

	// Skip header row
	for _, record := range records[1:] {
		if len(record) < 5 {
			continue
		}

		lat, _ := strconv.ParseFloat(record[2], 64)
		lng, _ := strconv.ParseFloat(record[3], 64)
		locationType, _ := strconv.Atoi(record[4])

		parentStation := ""
		if len(record) > 5 {
			parentStation = record[5]
		}

		s.stops = append(s.stops, models.Stop{
			ID:            record[0],
			Name:          record[1],
			Lat:           lat,
			Lng:           lng,
			LocationType:  locationType,
			ParentStation: parentStation,
		})
	}

	s.loaded = true
	return nil
}

// FindNearby returns stops within a radius (meters) of a point
func (s *StopService) FindNearby(lat, lng, radiusMeters float64) []models.StopWithDistance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []models.StopWithDistance

	for _, stop := range s.stops {
		// Only include parent stations (location_type = 1)
		if stop.LocationType != 1 {
			continue
		}

		dist := Haversine(lat, lng, stop.Lat, stop.Lng)
		if dist <= radiusMeters {
			results = append(results, models.StopWithDistance{
				Stop:           stop,
				DistanceMeters: dist,
				DistanceMiles:  MetersToMiles(dist),
			})
		}
	}

	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceMeters < results[j].DistanceMeters
	})

	return results
}

// FindClosest returns the N closest stops to a point
func (s *StopService) FindClosest(lat, lng float64, limit int) []models.StopWithDistance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []models.StopWithDistance

	for _, stop := range s.stops {
		// Only include parent stations
		if stop.LocationType != 1 {
			continue
		}

		dist := Haversine(lat, lng, stop.Lat, stop.Lng)
		results = append(results, models.StopWithDistance{
			Stop:           stop,
			DistanceMeters: dist,
			DistanceMiles:  MetersToMiles(dist),
		})
	}

	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceMeters < results[j].DistanceMeters
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results
}

// Count returns the number of loaded stops
func (s *StopService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.stops)
}

// ParentStationCount returns the count of parent stations only
func (s *StopService) ParentStationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, stop := range s.stops {
		if stop.LocationType == 1 {
			count++
		}
	}
	return count
}

// GetByID returns a stop by its ID
func (s *StopService) GetByID(id string) (models.Stop, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, stop := range s.stops {
		if stop.ID == id {
			return stop, true
		}
	}
	return models.Stop{}, false
}

// IsLoaded returns true if data has been loaded
func (s *StopService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}
