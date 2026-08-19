.PHONY: build test lint release hooks

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags "-X main.ver=$(VERSION)" -o bin/g8n ./cmd/g8n

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

release:
	goreleaser release --clean

hooks:
	@printf '#!/bin/sh\nmake lint && make test\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed"