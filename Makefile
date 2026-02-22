.PHONY: run build

run:
	go run cmd/server/main.go

build:
	go build -o emteeayy cmd/server/main.go
