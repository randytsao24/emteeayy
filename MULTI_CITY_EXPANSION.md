# Multi-City Expansion Plan

Roadmap for expanding emteeayy beyond NYC to support transit tracking in other US metropolitan areas.

## Transit API Landscape

| City | Agency | API Key | Format | Difficulty |
|------|--------|---------|--------|------------|
| NYC | MTA | No (subway), Yes (bus) | GTFS-RT | ✅ Done |
| Philadelphia | SEPTA | **No** | GTFS-RT | Easy |
| SF Bay Area | BART | **No** | GTFS-RT | Easy |
| Boston | MBTA | Free (recommended) | GTFS-RT + JSON | Medium |
| Washington DC | WMATA | Free signup | GTFS-RT | Medium |
| Chicago | CTA | Yes | Custom API | Harder |
| Los Angeles | Metro | Yes (Swiftly form) | GTFS-RT | Harder |

### Feed URLs Reference

**SEPTA (Philadelphia)** - No auth required
- Bus Trip Updates: `https://www3.septa.org/gtfsrt/septa-pa-us/Trip/rtTripUpdates.pb`
- Bus Vehicle Positions: `https://www3.septa.org/gtfsrt/septa-pa-us/Vehicle/rtVehiclePosition.pb`
- Rail Trip Updates: `https://www3.septa.org/gtfsrt/septarail-pa-us/Trip/rtTripUpdates.pb`
- Rail Vehicle Positions: `https://www3.septa.org/gtfsrt/septarail-pa-us/Vehicle/rtVehiclePosition.pb`
- Static GTFS: `https://www3.septa.org/developer/gtfs_public.zip`

**BART (SF Bay Area)** - No auth required
- Trip Updates: `http://api.bart.gov/gtfsrt/tripupdate.aspx`
- Alerts: `http://api.bart.gov/gtfsrt/alerts.aspx`
- Static GTFS: `https://www.bart.gov/dev/schedules/google_transit.zip`

**MBTA (Boston)** - Free key recommended (1000 req/min)
- Trip Updates: `https://cdn.mbta.com/realtime/TripUpdates.pb`
- Vehicle Positions: `https://cdn.mbta.com/realtime/VehiclePositions.pb`
- Alerts: `https://cdn.mbta.com/realtime/Alerts.pb`
- Also has JSON API: `https://api-v3.mbta.com/`

**WMATA (DC)** - Free key required
- Developer portal: `https://developer.wmata.com/`
- Rail Trip Updates: `https://api.wmata.com/gtfs/rail-gtfsrt-tripupdates.pb`
- Rail Vehicle Positions: `https://api.wmata.com/gtfs/rail-gtfsrt-vehiclepositions.pb`

---

## Architecture Changes

### Current Architecture (NYC-only)

```
User enters zip code
    → Look up in nyc-zipcodes.json
    → Find nearby stops from stops.txt
    → Fetch from hardcoded MTA feed URLs
    → Return arrivals
```

### Target Architecture (Multi-city)

```
User enters zip code
    → Look up in unified zip code database
    → Detect which metro area
    → Load city-specific transit config
    → Find nearby stops from city's stops data
    → Fetch from city's feed URLs
    → Normalize response format
    → Return arrivals
```

### Key Abstractions Needed

#### 1. Transit Provider Interface

```go
// internal/transit/provider.go

type Provider interface {
    // Info
    City() string
    Agency() string
    
    // Data loading
    LoadStops() error
    
    // Queries
    GetNearbyStops(lat, lng float64, radiusMeters int) []models.Stop
    GetArrivals(stopID string) ([]models.Arrival, error)
    GetNearbyArrivals(lat, lng float64, radiusMeters, limit int) ([]models.StationArrivals, error)
}
```

#### 2. City Configuration

```go
// internal/config/cities.go

type CityConfig struct {
    ID          string       // "nyc", "philadelphia", "boston"
    Name        string       // "New York City"
    Agency      string       // "MTA"
    ZipPrefixes []string     // ["100", "101", "102", "103", "104", "110", "111", "112", "113", "114", "116"]
    Feeds       []FeedConfig
    StopsFile   string       // path to GTFS stops.txt
    RequiresKey bool
    KeyEnvVar   string       // "MTA_BUS_API_KEY"
}

type FeedConfig struct {
    Name   string   // "ACE", "BDFM", "Broad Street Line"
    URL    string
    Routes []string // which routes this feed covers
}
```

#### 3. Unified Zip Code Database

Replace `nyc-zipcodes.json` with a multi-city dataset. Options:

- **US Census ZCTA data** - Free, official, all US zip codes
- **SimpleMaps** - Free tier has 41k+ zip codes with lat/lng
- **Filter to supported metros** - Don't need all 41k, just cities we support

Structure:
```json
{
  "10001": {
    "lat": 40.7484,
    "lng": -73.9967,
    "city": "New York",
    "state": "NY",
    "metro": "nyc"
  },
  "19101": {
    "lat": 39.9526,
    "lng": -75.1652,
    "city": "Philadelphia", 
    "state": "PA",
    "metro": "philadelphia"
  }
}
```

---

## Data Requirements Per City

Each city needs:

1. **Zip codes** - Zip codes in that metro area with lat/lng
2. **Stops data** - GTFS `stops.txt` file from the transit agency
3. **Feed URLs** - GTFS-RT endpoints (or API endpoints)
4. **Route mappings** - Which routes are in which feeds (if split like MTA)

### Data Sources

| Data | Source |
|------|--------|
| Zip codes | US Census ZCTA or SimpleMaps |
| Stop locations | Each agency's GTFS static data |
| Feed discovery | [Transitland](https://transit.land/feeds) - 2,500+ feeds indexed |

---

## Implementation Phases

### Phase 1: Refactor for Extensibility

**Goal:** Abstract NYC-specific code into a provider pattern without breaking existing functionality.

Tasks:
- [ ] Create `Provider` interface in `internal/transit/provider.go`
- [ ] Refactor `subway.go` to implement the interface as `NYCSubwayProvider`
- [ ] Create city config structure
- [ ] Update handlers to use provider interface
- [ ] Add metro detection from zip code

### Phase 2: Add SEPTA (Philadelphia)

**Goal:** Validate the multi-city architecture with the easiest addition.

Why SEPTA first:
- Same GTFS-RT protobuf format as MTA (reuse parsing code)
- No API key required
- Different enough to validate the abstraction

Tasks:
- [ ] Download SEPTA GTFS static data
- [ ] Add Philadelphia zip codes to database
- [ ] Implement `SEPTAProvider`
- [ ] Test end-to-end

### Phase 3: Add BART (SF Bay Area)

**Goal:** Second no-auth city, different geography.

Tasks:
- [ ] Download BART GTFS static data
- [ ] Add SF Bay Area zip codes
- [ ] Implement `BARTProvider`
- [ ] Handle BART-specific quirks (Antioch trip IDs, Oakland Airport)

### Phase 4: Add MBTA (Boston)

**Goal:** First city with API key requirement.

Tasks:
- [ ] Register for free MBTA API key
- [ ] Add Boston metro zip codes
- [ ] Implement `MBTAProvider`
- [ ] Add key management (env var, graceful degradation if missing)

### Phase 5: Remaining Cities

Based on user demand:
- WMATA (DC) - Free key, good docs
- CTA (Chicago) - Different API format, more work
- LA Metro - Key through Swiftly, extra signup step

---

## Frontend Changes

The current frontend is NYC-focused. Updates needed:

1. **City selector** - Dropdown or auto-detect from zip
2. **City-specific styling** - Different agencies have different line colors
3. **Fallback messaging** - "Transit data not available for this area"

### Auto-detection Flow

```
User enters zip code
    → API detects metro area
    → Returns city info in response
    → Frontend updates UI accordingly
```

### Response Format Update

```json
{
  "metro": "philadelphia",
  "city": "Philadelphia",
  "agency": "SEPTA",
  "stations": [...]
}
```

---

## Considerations

### Stays True to Core Values

- **No logins** - City detection is automatic from zip code
- **No data collection** - All processing is stateless
- **No monetization** - Public transit data remains public

### Error Handling

- If a city's API is down, show error for that city only
- If zip code not in any supported metro, show friendly message
- Graceful degradation when optional API keys are missing

### Performance

- Cache GTFS static data (stops don't change often)
- Consider caching real-time data briefly (30-60s) to reduce API calls
- Load city data lazily or at startup based on config

---

## Resources

- **[Transitland](https://transit.land/feeds)** - GTFS feed discovery and metadata
- **[GTFS Realtime Spec](https://gtfs.org/realtime/)** - Protocol documentation
- **[US Census ZCTA](https://www.census.gov/programs-surveys/geography/guidance/geo-areas/zctas.html)** - Official zip code data
- **[MobilityData](https://gtfs.mobilitydata.org/)** - GTFS resources and tools
