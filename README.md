# emteeayy

MTA transit tracking for NYC without the fluff.

## Quick Start

```bash
# Run directly
go run cmd/server/main.go

# Or build and run
go build -o emteeayy cmd/server/main.go
./emteeayy
```

Server starts at http://localhost:3000

## Config

Set via environment variables:

```bash
PORT=3000                  # Server port
ENV=development            # Environment
MTA_BUS_API_KEY=xxx        # MTA Bus Time API key
CACHE_TTL_SECONDS=120      # Cache duration
```

## Requirements

- Go 1.21+
