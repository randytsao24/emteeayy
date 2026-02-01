# emteeayy Development Guide

## Learning Approach

This project is being built as a Go learning exercise. The approach:

- **Code comments** should be concise and professional - they explain *why*, not *how*
- **Chat conversations** are where teaching happens - Go concepts, design decisions, and explanations are discussed interactively
- **Ask questions** anytime something is unclear - that's the point!

### For AI Agents

When teaching new Go concepts during development:
1. Explain the concept in the chat conversation
2. **Update `TEACHING.md`** with the concept, a brief explanation, and a code example
3. Keep `TEACHING.md` organized by topic

## Project Structure

```
emteeayy/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/                  # Private application code
│   ├── config/               # Configuration loading
│   ├── api/                  # HTTP handlers and routing
│   ├── transit/              # MTA transit services
│   ├── location/             # Zip code and stop lookups
│   ├── cache/                # Caching utilities
│   └── models/               # Shared data types
├── data/                      # Static data files (GTFS, zip codes)
├── go.mod                     # Module definition
└── go.sum                     # Dependency checksums
```

## Running the Server

```bash
# Development
go run cmd/server/main.go

# Build and run
go build -o emteeayy cmd/server/main.go
./emteeayy
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3000 | Server port |
| `ENV` | development | Environment (development/production) |
| `MTA_BUS_API_KEY` | - | MTA Bus Time API key |
| `CACHE_TTL_SECONDS` | 120 | Cache time-to-live |
| `HTTP_TIMEOUT_SECONDS` | 10 | External HTTP request timeout |

## Go Concepts Reference

As we build, key Go concepts will be covered:

- [x] Modules and packages (`go.mod`, imports)
- [x] Structs and methods
- [x] HTTP server basics
- [x] Middleware pattern
- [x] Structured logging (slog)
- [x] Type embedding
- [x] Panic and recover
- [ ] Error handling patterns
- [ ] Interfaces
- [ ] Goroutines and concurrency
- [ ] JSON encoding with struct tags
- [ ] Testing
