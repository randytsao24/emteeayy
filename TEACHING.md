# Go Concepts & Learnings

A living document of Go concepts learned while building emteeayy.

---

## Modules & Packages

**Module** = A collection of packages with a `go.mod` file. Like a Node.js project with `package.json`.

**Package** = A directory of `.go` files. Every file starts with `package <name>`.

```go
// go.mod defines the module
module github.com/randytsao24/emteeayy

go 1.25.6
```

- Module path (`github.com/...`) is used for imports, doesn't need to be a real URL
- `go mod tidy` cleans up and downloads dependencies
- `go.sum` locks dependency versions (like `package-lock.json`)

---

## Project Structure Convention

```
cmd/           # Entry points (main packages)
internal/      # Private code - can't be imported by other projects
pkg/           # Public code - can be imported (we're not using this)
```

The `internal/` directory is special in Go - the compiler enforces that code inside can only be imported by code in the same module.

---

## Structs

Structs group related data together (like objects/classes in other languages).

```go
type Config struct {
    Port    string        // Exported (public) - uppercase
    timeout time.Duration // unexported (private) - lowercase
}
```

**Visibility rule:** Uppercase = exported (public), lowercase = unexported (private to package)

---

## Methods vs Functions

```go
// Function - standalone
func Load() *Config { ... }

// Method - attached to a type (has a "receiver")
func (c *Config) IsDevelopment() bool {
    return c.Env == "development"
}
```

Methods let you call `config.IsDevelopment()` instead of `IsDevelopment(config)`.

---

## Pointers

```go
func Load() *Config {    // Returns a POINTER to Config
    return &Config{...}  // & = "address of" (creates pointer)
}
```

- `*Config` = pointer to a Config
- `&Config{}` = create Config and return its address
- Why pointers? Avoids copying large structs, allows mutation

---

## HTTP Handlers

```go
func handleHealth(w http.ResponseWriter, r *http.Request) {
    // w = where you write the response
    // r = the incoming request
}
```

Go 1.22+ supports method+path patterns:

```go
mux.HandleFunc("GET /health", handleHealth)  // Only matches GET
mux.HandleFunc("POST /cache/clear", handleClear)
```

---

## JSON Struct Tags

```go
type Response struct {
    Status    string `json:"status"`     // JSON field = "status"
    Timestamp string `json:"timestamp"`
}
```

Tags control JSON serialization. Without tags, Go uses the field name as-is.

---

## Anonymous Structs

For one-off responses, define the struct inline:

```go
response := struct {
    Status string `json:"status"`
}{
    Status: "OK",
}
```

---

## Error Handling

Go doesn't have exceptions. Functions return errors explicitly:

```go
if err := cfg.Validate(); err != nil {
    log.Fatal("Configuration error: ", err)
}
```

Pattern: `if err != nil { handle it }` - you'll see this constantly.

---

## Zero Values

Uninitialized variables get "zero values":

- `string` → `""`
- `int` → `0`
- `bool` → `false`
- `pointer` → `nil`
- `struct` → all fields zero

---

## Middleware Pattern

Middleware wraps HTTP handlers to add cross-cutting functionality (logging, auth, CORS, etc).

```go
func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)  // Call the next handler
        slog.Info("request", "duration", time.Since(start))
    })
}
```

Middleware signature: `func(http.Handler) http.Handler` - takes a handler, returns a handler.

**Chaining middleware:**

```go
handler := Chain(mux, Recovery, Logging, CORS)
// Executes: Recovery -> Logging -> CORS -> actual handler
```

---

## Structured Logging (slog)

Go 1.21+ includes `log/slog` for structured logging:

```go
slog.Info("request",
    "method", r.Method,
    "path", r.URL.Path,
    "status", 200,
)
// Output: level=INFO msg=request method=GET path=/health status=200
```

Key-value pairs make logs searchable and parseable.

---

## Embedding and Wrapping Types

To capture response status for logging, we wrap `http.ResponseWriter`:

```go
type responseWriter struct {
    http.ResponseWriter  // Embedded - gets all methods automatically
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code  // Capture the status
    rw.ResponseWriter.WriteHeader(code)  // Call original
}
```

**Embedding** = Go's version of inheritance. The embedded type's methods are "promoted".

---

## Panic & Recover

Go has `panic` (like throwing an exception) and `recover` (like catching):

```go
defer func() {
    if err := recover(); err != nil {
        // Handle the panic
    }
}()
```

`defer` = runs when function exits. Recovery middleware uses this to catch panics and return 500 instead of crashing.

---

## Handler Structs

Instead of plain functions, we use structs for handlers when they need state:

```go
type HealthHandler struct {
    startTime time.Time
}

func NewHealthHandler() *HealthHandler {
    return &HealthHandler{startTime: time.Now()}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    // Can access h.startTime for uptime calculation
}
```

This pattern (constructor + methods) is common in Go.

---

## map[string]any

For dynamic JSON responses:

```go
writeJSON(w, http.StatusOK, map[string]any{
    "status": "OK",
    "count":  42,
})
```

`any` is an alias for `interface{}` - can hold any type.

---

## Formatting & Linting

Go has a built-in, official formatter that everyone uses - no debates about style like in other languages.
gofmt / go fmt

# Format a filegofmt -w main.go# Format entire projectgo fmt ./...

This is built into Go itself. It enforces one true style - tabs for indentation, specific spacing rules, etc.

---

## Generics (Type Parameters)

Go 1.18+ supports generics. Our cache uses them:

```go
// T is a type parameter - can be any type
type Cache[T any] struct {
    items map[string]item[T]
}

// Create caches for different types
zipCache := cache.New[models.ZipCode](time.Minute)
stopCache := cache.New[[]models.Stop](time.Minute)
```

`[T any]` means "T can be any type".

---

## sync.RWMutex (Read-Write Mutex)

For thread-safe access to shared data:

```go
type Service struct {
    data map[string]string
    mu   sync.RWMutex
}

func (s *Service) Get(key string) string {
    s.mu.RLock()         // Multiple readers allowed
    defer s.mu.RUnlock()
    return s.data[key]
}

func (s *Service) Set(key, value string) {
    s.mu.Lock()          // Exclusive access for writing
    defer s.mu.Unlock()
    s.data[key] = value
}
```

---

## Error Wrapping

Use `fmt.Errorf` with `%w` to wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("reading file: %w", err)
}
```

---

## Slices

Slices are dynamic arrays:

```go
var stops []models.Stop           // nil slice
stops = append(stops, newStop)    // append grows it
stops = stops[:5]                 // first 5 elements
```

---

## Sorting

```go
sort.Slice(results, func(i, j int) bool {
    return results[i].Distance < results[j].Distance
})
```

---

## Channels for Signaling

```go
stop := make(chan struct{})

select {
case <-ticker.C:
    doWork()
case <-stop:
    return  // Exit when stop is closed
}
```

---

_This document is updated as new concepts are introduced during development._
