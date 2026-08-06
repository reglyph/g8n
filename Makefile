.PHONY: build test lint

build:
	go build -o bin/g8n ./cmd/g8n

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...