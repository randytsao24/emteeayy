# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS requests to MTA APIs
RUN apk --no-cache add ca-certificates

# Download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage - minimal image
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Copy the binary
COPY --from=builder /app/server .

# Copy data files
COPY data/ ./data/

EXPOSE 8080

ENTRYPOINT ["./server"]
