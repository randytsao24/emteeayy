# emteeayy

Real-time MTA transit tracking for NYC.

Current URL (will change): https://emteeayy.fly.dev

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

| Endpoint      | Description  |
| ------------- | ------------ |
| `GET /`       | API info     |
| `GET /health` | Health check |

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
