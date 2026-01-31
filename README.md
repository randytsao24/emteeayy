# emteeayy

Real-time MTA transit tracking for NYC.

## Quick Start

```bash
# Copy env file and add your MTA Bus API key
cp .env.example .env

# Run
go run cmd/server/main.go
```

Server starts at http://localhost:3000

## Endpoints

### Core
| Endpoint | Description |
|----------|-------------|
| `GET /` | API info |
| `GET /health` | Health check |

### Location
| Endpoint | Description |
|----------|-------------|
| `GET /transit/location/zip/{zipcode}` | Find stops near zip code |
| `GET /transit/location/zip/{zipcode}/closest?limit=N` | Get N closest stops |
| `GET /transit/location/zipcodes/all` | List all zip codes |
| `GET /transit/location/boroughs` | List boroughs |

### Subway (Real-time)
| Endpoint | Description |
|----------|-------------|
| `GET /transit/subway/j-train` | J/Z arrivals at Woodhaven Blvd |
| `GET /transit/subway/station/{stopId}` | Arrivals for any station |

### Bus (Real-time)
| Endpoint | Description |
|----------|-------------|
| `GET /transit/bus/arrivals` | Woodhaven area bus arrivals |
| `GET /transit/bus/stops` | Woodhaven stops and routes |

## Config

```bash
PORT=3000
ENV=development
MTA_BUS_API_KEY=xxx  # Get at https://register.developer.obanyc.com/
CACHE_TTL_SECONDS=120
HTTP_TIMEOUT_SECONDS=10
```

## Requirements

- Go 1.21+
- MTA Bus API key (for bus data)
