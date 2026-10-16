BINARY   := ikctl
PKG      := github.com/rustyrien/ikctl
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --verify -q HEAD 2>/dev/null)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

.PHONY: all build install test cover lint fmt tidy clean

all: lint test build

build: ## Build bin/ikctl
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ikctl

install: ## Install ikctl into GOBIN
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/ikctl

test: ## Run unit tests with the race detector
	go test -race -count=1 ./...

cover: ## Run tests with coverage
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint: ## Run linters
	go vet ./...
	$(GOLANGCI_LINT) run ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

clean:
	rm -rf bin coverage.out dist
