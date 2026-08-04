BINARY := bin/cooldeck
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/resetnak/cooldeck/internal/version.Version=$(VERSION) \
	-X github.com/resetnak/cooldeck/internal/version.Commit=$(COMMIT) \
	-X github.com/resetnak/cooldeck/internal/version.Date=$(DATE)

.PHONY: build run test test-race test-update-golden lint fmt vet vuln check clean snapshot

build:
	mkdir -p bin
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/cooldeck

run:
	go run ./cmd/cooldeck --demo

test:
	go test ./...

test-race:
	go test -race ./...

test-update-golden:
	UPDATE_GOLDEN=1 go test ./...

lint:
	staticcheck ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

vuln:
	govulncheck ./...

check: vet test build

clean:
	rm -rf bin dist

snapshot:
	goreleaser release --snapshot --clean
