package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/randytsao24/emteeayy/internal/api"
	"github.com/randytsao24/emteeayy/internal/api/handlers"
	"github.com/randytsao24/emteeayy/internal/config"
	"github.com/randytsao24/emteeayy/internal/location"
	"github.com/randytsao24/emteeayy/internal/transit"
)

// ---------------------------------------------------------------------------
// Mock providers
// ---------------------------------------------------------------------------

type mockSubwayProvider struct {
	arrivals []transit.Arrival
	err      error
}

func (m *mockSubwayProvider) GetArrivalsForStation(stopID string) (map[string][]transit.Arrival, error) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string][]transit.Arrival{
		"northbound": m.arrivals,
		"southbound": m.arrivals,
	}, nil
}

func (m *mockSubwayProvider) GetArrivalsForStations(stopIDs []string) ([]transit.StationArrivals, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]transit.StationArrivals, len(stopIDs))
	for i, id := range stopIDs {
		result[i] = transit.StationArrivals{
			StopID:     id,
			Northbound: m.arrivals,
			Southbound: m.arrivals,
		}
	}
	return result, nil
}

type mockBusProvider struct {
	hasKey   bool
	stops    []transit.BusStop
	arrivals []transit.BusArrival
	err      error
}

func (m *mockBusProvider) HasAPIKey() bool { return m.hasKey }

func (m *mockBusProvider) FindStopsNear(lat, lng float64, radiusMeters int) ([]transit.BusStop, error) {
	return m.stops, m.err
}

func (m *mockBusProvider) GetArrivalsNear(lat, lng float64, radiusMeters, limit int) ([]transit.BusArrival, error) {
	return m.arrivals, m.err
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func dataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "../../data")
}

func newTestServer(t *testing.T, subway handlers.SubwayProvider, bus handlers.BusProvider) *httptest.Server {
	t.Helper()

	dir := dataDir(t)

	zipSvc := location.NewZipCodeService()
	if err := zipSvc.Load(filepath.Join(dir, "nyc-zipcodes.json")); err != nil {
		t.Fatalf("load zip codes: %v", err)
	}

	stopSvc := location.NewStopService()
	if err := stopSvc.Load(filepath.Join(dir, "stops.txt")); err != nil {
		t.Fatalf("load stops: %v", err)
	}

	cfg := &config.Config{HTTPTimeout: 5 * time.Second}
	router := api.NewRouter(cfg, zipSvc, stopSvc, subway, bus, nil)
	return httptest.NewServer(router)
}

func defaultSubway() *mockSubwayProvider {
	return &mockSubwayProvider{
		arrivals: []transit.Arrival{
			{
				Route:       "A",
				StopID:      "127N",
				Direction:   "northbound",
				ArrivalTime: time.Now().Add(5 * time.Minute),
				MinutesAway: 5,
			},
		},
	}
}

func defaultBus() *mockBusProvider {
	return &mockBusProvider{
		hasKey: true,
		stops: []transit.BusStop{
			{ID: "MTA_305423", Name: "5 AV/W 34 ST", Lat: 40.748817, Lng: -73.985428},
		},
		arrivals: []transit.BusArrival{
			{
				Route:           "M34",
				Destination:     "34 St Ferry",
				StopID:          "MTA_305423",
				ExpectedArrival: time.Now().Add(3 * time.Minute),
				MinutesAway:     3,
			},
		},
	}
}

func get(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return m
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

func assertSuccess(t *testing.T, body map[string]any) {
	t.Helper()
	if body["success"] != true {
		t.Errorf("expected success=true, body: %v", body)
	}
}

func assertField(t *testing.T, body map[string]any, field string) {
	t.Helper()
	if _, ok := body[field]; !ok {
		t.Errorf("missing field %q in response: %v", field, body)
	}
}

// ---------------------------------------------------------------------------
// Health & root
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/health")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertField(t, body, "status")
	assertField(t, body, "uptime")

	if body["status"] != "OK" {
		t.Errorf("status = %v, want OK", body["status"])
	}
}

func TestAPIRoot(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/api")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertField(t, body, "endpoints")
}

// ---------------------------------------------------------------------------
// Location endpoints (use real data, no external calls)
// ---------------------------------------------------------------------------

func TestLocationBoroughs(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/boroughs")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "boroughs")

	boroughs, ok := body["boroughs"].([]any)
	if !ok || len(boroughs) == 0 {
		t.Error("expected non-empty boroughs list")
	}
}

func TestLocationInfo(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/info")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "coverage")

	coverage, ok := body["coverage"].(map[string]any)
	if !ok {
		t.Fatal("coverage should be an object")
	}
	if coverage["zipcodes"] == nil {
		t.Error("coverage.zipcodes should be present")
	}
	if count, _ := coverage["zipcodes"].(float64); count == 0 {
		t.Error("coverage.zipcodes should be > 0")
	}
}

func TestLocationStopsByZip(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"valid NYC zip", "/transit/location/zip/10001", http.StatusOK},
		{"non-NYC zip", "/transit/location/zip/99999", http.StatusNotFound},
		{"too short", "/transit/location/zip/100", http.StatusBadRequest},
		{"letters", "/transit/location/zip/abcde", http.StatusNotFound}, // 5 chars but not found
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(t, srv, tc.path)
			assertStatus(t, resp, tc.status)
		})
	}
}

func TestLocationStopsByZipResponse(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/zip/10001")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "stops")
	assertField(t, body, "zip_code")
	assertField(t, body, "metadata")

	stops, ok := body["stops"].([]any)
	if !ok || len(stops) == 0 {
		t.Error("expected non-empty stops for 10001")
	}
}

func TestLocationClosestStops(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/zip/10001/closest?limit=3")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)

	stops, ok := body["stops"].([]any)
	if !ok {
		t.Fatal("expected stops array")
	}
	if len(stops) > 3 {
		t.Errorf("limit=3 but got %d stops", len(stops))
	}
}

func TestLocationAllZipCodes(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/zipcodes/all")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)

	zips, ok := body["zipcodes"].([]any)
	if !ok || len(zips) == 0 {
		t.Error("expected non-empty zipcodes list")
	}
}

func TestLocationAllZipCodesBoroughFilter(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/location/zipcodes/all?borough=Manhattan")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "zipcodes")
}

// ---------------------------------------------------------------------------
// Subway endpoints
// ---------------------------------------------------------------------------

func TestSubwayStation(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/subway/station/127")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "arrivals")
	assertField(t, body, "stop_id")
}

func TestSubwayStationServiceError(t *testing.T) {
	failSubway := &mockSubwayProvider{err: errors.New("feed unavailable")}
	srv := newTestServer(t, failSubway, defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/subway/station/127")
	assertStatus(t, resp, http.StatusInternalServerError)

	body := decodeBody(t, resp)
	assertField(t, body, "error")
}

func TestSubwayNearZip(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"valid zip", "/transit/subway/near/10001", http.StatusOK},
		{"non-NYC zip", "/transit/subway/near/99999", http.StatusNotFound},
		{"too short", "/transit/subway/near/100", http.StatusBadRequest},
		{"with radius", "/transit/subway/near/10001?radius=1600", http.StatusOK},
		{"with limit", "/transit/subway/near/10001?limit=2", http.StatusOK},
	}

	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(t, srv, tc.path)
			assertStatus(t, resp, tc.status)
		})
	}
}

func TestSubwayNearZipResponse(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/subway/near/10001")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "stations")
	assertField(t, body, "count")
	assertField(t, body, "zip_code")
	assertField(t, body, "radius_meters")
}

func TestSubwayNearCoords(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"valid coords", "/transit/subway/near?lat=40.7484&lng=-73.9967", http.StatusOK},
		{"missing lat", "/transit/subway/near?lng=-73.9967", http.StatusBadRequest},
		{"missing lng", "/transit/subway/near?lat=40.7484", http.StatusBadRequest},
		{"invalid lat", "/transit/subway/near?lat=abc&lng=-73.9967", http.StatusBadRequest},
		{"invalid lng", "/transit/subway/near?lat=40.7484&lng=xyz", http.StatusBadRequest},
	}

	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(t, srv, tc.path)
			assertStatus(t, resp, tc.status)
		})
	}
}

func TestSubwayStopsNearZip(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/subway/stops/10001")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "stops")
	assertField(t, body, "count")
}

// ---------------------------------------------------------------------------
// Bus endpoints
// ---------------------------------------------------------------------------

func TestBusNearZipNoAPIKey(t *testing.T) {
	noKeyBus := &mockBusProvider{hasKey: false}
	srv := newTestServer(t, defaultSubway(), noKeyBus)
	defer srv.Close()

	resp := get(t, srv, "/transit/bus/near/10001")
	assertStatus(t, resp, http.StatusServiceUnavailable)

	body := decodeBody(t, resp)
	assertField(t, body, "error")
}

func TestBusNearZip(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"valid zip", "/transit/bus/near/10001", http.StatusOK},
		{"non-NYC zip", "/transit/bus/near/99999", http.StatusNotFound},
		{"too short", "/transit/bus/near/100", http.StatusBadRequest},
	}

	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(t, srv, tc.path)
			assertStatus(t, resp, tc.status)
		})
	}
}

func TestBusNearZipResponse(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/bus/near/10001")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "arrivals")
	assertField(t, body, "count")
	assertField(t, body, "zip_code")
}

func TestBusNearCoords(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"valid coords", "/transit/bus/near?lat=40.7484&lng=-73.9967", http.StatusOK},
		{"missing lat", "/transit/bus/near?lng=-73.9967", http.StatusBadRequest},
		{"missing lng", "/transit/bus/near?lat=40.7484", http.StatusBadRequest},
		{"invalid lat", "/transit/bus/near?lat=abc&lng=-73.9967", http.StatusBadRequest},
	}

	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(t, srv, tc.path)
			assertStatus(t, resp, tc.status)
		})
	}
}

func TestBusStopsNearZip(t *testing.T) {
	srv := newTestServer(t, defaultSubway(), defaultBus())
	defer srv.Close()

	resp := get(t, srv, "/transit/bus/stops/10001")
	assertStatus(t, resp, http.StatusOK)

	body := decodeBody(t, resp)
	assertSuccess(t, body)
	assertField(t, body, "stops")
	assertField(t, body, "count")
}

func TestBusServiceError(t *testing.T) {
	failBus := &mockBusProvider{hasKey: true, err: errors.New("upstream error")}
	srv := newTestServer(t, defaultSubway(), failBus)
	defer srv.Close()

	resp := get(t, srv, "/transit/bus/near/10001")
	assertStatus(t, resp, http.StatusInternalServerError)

	body := decodeBody(t, resp)
	assertField(t, body, "error")
}
