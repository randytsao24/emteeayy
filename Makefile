.PHONY: run dev build

run:
	go run cmd/server/main.go

dev:
	$(shell go env GOPATH)/bin/air

build:
	go build -o emteeayy cmd/server/main.go
