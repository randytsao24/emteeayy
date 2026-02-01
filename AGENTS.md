# Agent Instructions

Real-time transit tracking for NYC (expanding to other US metros). Privacy-focused: no logins, no data collection, no monetization.

## Project Structure

```
cmd/server/main.go       # Entry point - initializes services, starts server
internal/
  api/
    router.go            # Route definitions (Go 1.22+ patterns)
    middleware.go        # Recovery, Logging, CORS, Timeout chain
    handlers/            # HTTP handlers (one file per domain)
  transit/
    subway.go            # GTFS-RT feed fetching & protobuf parsing
    bus.go               # MTA Bus API (SIRI format)
  location/
    zipcode.go           # Zip code lookup service
    stops.go             # Subway stop search (spatial)
    distance.go          # Haversine distance calculation
  models/models.go       # Shared data types
  cache/cache.go         # Generic TTL cache (available but unused)
  config/config.go       # Environment config loading
web/
  index.html             # Embedded SPA frontend
  embed.go               # Go embed directive
data/
  nyc-zipcodes.json      # NYC zip codes with lat/lng
  stops.txt              # GTFS stops file
```

## Code Patterns

### Handler Structure

Handlers receive service dependencies via constructor injection:

```go
type TransitHandler struct {
    subway   *transit.SubwayService
    bus      *transit.BusService
    stops    *location.StopService
    zipCodes *location.ZipCodeService
}

func NewTransitHandler(...) *TransitHandler { ... }
```

### JSON Responses

Always use the `writeJSON` helper from `handlers/response.go`:

```go
// Success
writeJSON(w, http.StatusOK, map[string]any{
    "success":  true,
    "data":     result,
    "count":    len(result),
})

// Error
writeJSON(w, http.StatusBadRequest, map[string]any{
    "error":   "Short error message",
    "message": "Detailed explanation",
})
```

### Error Handling

- Use `fmt.Errorf("context: %w", err)` for wrapping
- Log with `slog.Error("message", "error", err, "key", value)`
- Return user-friendly error messages in responses
- Continue on partial failures (e.g., skip failed feeds, try others)

### Query Parameter Parsing

Use `parseIntQueryParam` for bounded integer params:

```go
radius := parseIntQueryParam(r, "radius", defaultVal, minVal, maxVal)
```

### Path Parameters

Use Go 1.22+ `r.PathValue()`:

```go
zipCode := r.PathValue("zipcode")
stopID := r.PathValue("stopId")
```

## Data Flow

1. **Zip code lookup**: User zip → find in `ZipCodeService` → get lat/lng
2. **Stop search**: lat/lng + radius → `StopService.FindNearby()` → sorted by distance
3. **Arrivals**: stop IDs → `SubwayService.GetArrivalsForStations()` → fetch GTFS-RT feeds → parse protobuf → filter & sort

## GTFS-RT Feed Parsing

MTA subway data comes as Protocol Buffers. Key types:

- `gtfs.FeedMessage` - top level
- `entity.GetTripUpdate()` - trip info
- `tripUpdate.GetStopTimeUpdate()` - arrival times per stop

Stop ID convention: `101N` = stop 101 northbound, `101S` = southbound.

## Adding New Features

### New Endpoint

1. Add handler method to appropriate handler struct
2. Register route in `router.go`
3. Follow existing response patterns

### New Service

1. Create in `internal/` with constructor `NewXxxService()`
2. Initialize in `main.go`
3. Inject into handlers that need it

### Multi-City Expansion

See `MULTI_CITY_EXPANSION.md` for the roadmap. Key abstraction needed:

```go
type Provider interface {
    City() string
    GetNearbyStops(lat, lng float64, radiusMeters int) []models.Stop
    GetArrivals(stopID string) ([]models.Arrival, error)
}
```

## Testing Locally

```bash
cp .env.example .env
# Add MTA_BUS_API_KEY if you want bus data
go run cmd/server/main.go
```

Server runs at http://localhost:3000. Frontend at root, API at `/transit/*`.

## Key Files to Read First

1. `cmd/server/main.go` - see how everything wires together
2. `internal/api/router.go` - all endpoints
3. `internal/transit/subway.go` - GTFS-RT parsing example
4. `internal/api/handlers/transit.go` - handler patterns

## Don'ts

- Don't add user accounts or authentication
- Don't persist user data server-side
- Don't add analytics or tracking
- Don't break the single-file frontend (keep it simple)
