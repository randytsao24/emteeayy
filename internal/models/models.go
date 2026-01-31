// Package models defines shared data types
package models

// ZipCode represents a NYC zip code with its location
type ZipCode struct {
	Code    string  `json:"code"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	City    string  `json:"city"`
	Borough string  `json:"borough"`
}

// Stop represents a subway station
type Stop struct {
	ID            string  `json:"stop_id"`
	Name          string  `json:"stop_name"`
	Lat           float64 `json:"stop_lat"`
	Lng           float64 `json:"stop_lon"`
	LocationType  int     `json:"location_type"`
	ParentStation string  `json:"parent_station"`
}

// StopWithDistance is a Stop with distance from a reference point
type StopWithDistance struct {
	Stop
	DistanceMeters float64 `json:"distance_meters"`
	DistanceMiles  float64 `json:"distance_miles"`
}

// Arrival represents a subway arrival
type Arrival struct {
	Route       string `json:"route"`
	Direction   string `json:"direction"`
	ArrivalTime string `json:"arrival_time"`
	MinutesAway int    `json:"minutes_away"`
}
