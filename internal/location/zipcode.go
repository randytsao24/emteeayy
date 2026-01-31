// Package location handles zip code and stop lookups
package location

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/randytsao24/emteeayy/internal/models"
)

// ZipCodeService manages zip code data
type ZipCodeService struct {
	zipCodes map[string]models.ZipCode
	mu       sync.RWMutex
	loaded   bool
}

// NewZipCodeService creates a new zip code service
func NewZipCodeService() *ZipCodeService {
	return &ZipCodeService{
		zipCodes: make(map[string]models.ZipCode),
	}
}

// Load reads zip code data from a JSON file
func (s *ZipCodeService) Load(filepath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("reading zip code file: %w", err)
	}

	// The JSON is a map of zip code string -> location data
	var raw map[string]struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		City    string  `json:"city"`
		Borough string  `json:"borough"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing zip code JSON: %w", err)
	}

	// Convert to our model
	for code, loc := range raw {
		s.zipCodes[code] = models.ZipCode{
			Code:    code,
			Lat:     loc.Lat,
			Lng:     loc.Lng,
			City:    loc.City,
			Borough: loc.Borough,
		}
	}

	s.loaded = true
	return nil
}

// Get returns a zip code by its code
func (s *ZipCodeService) Get(code string) (models.ZipCode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	zip, exists := s.zipCodes[code]
	return zip, exists
}

// GetAll returns all zip codes
func (s *ZipCodeService) GetAll() []models.ZipCode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.ZipCode, 0, len(s.zipCodes))
	for _, zip := range s.zipCodes {
		result = append(result, zip)
	}
	return result
}

// GetByBorough returns all zip codes in a borough
func (s *ZipCodeService) GetByBorough(borough string) []models.ZipCode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.ZipCode
	for _, zip := range s.zipCodes {
		if zip.Borough == borough {
			result = append(result, zip)
		}
	}
	return result
}

// Boroughs returns a list of all unique boroughs
func (s *ZipCodeService) Boroughs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var boroughs []string
	for _, zip := range s.zipCodes {
		if !seen[zip.Borough] {
			seen[zip.Borough] = true
			boroughs = append(boroughs, zip.Borough)
		}
	}
	return boroughs
}

// Count returns the number of loaded zip codes
func (s *ZipCodeService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.zipCodes)
}

// IsLoaded returns true if data has been loaded
func (s *ZipCodeService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}
