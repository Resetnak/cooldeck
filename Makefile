BINARY := bin/cooldeck
# Strip the tag's leading v so local builds report the same shape as release
# builds (goreleaser injects {{.Version}}, which has no v).
VERSION ?= $(shell (git describe --tags --always --dirty 2>/dev/null || echo dev) | sed 's/^v//')
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/resetnak/cooldeck/internal/version.Version=$(VERSION) \
	-X github.com/resetnak/cooldeck/internal/version.Commit=$(COMMIT) \
	-X github.com/resetnak/cooldeck/internal/version.Date=$(DATE)

.PHONY: build run test test-race test-update-golden lint fmt fmt-check vet vuln check clean snapshot bench

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

# The same linters CI runs, pinned to the same versions, via go run so a fresh
# clone needs nothing preinstalled. v0.7.0 is staticcheck 2026.1.
lint:
	go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6 run --timeout=5m

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Pre-commit / CI local equivalent.
check: fmt-check vet lint test build

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:" && gofmt -l . && exit 1)

bench:
	go test ./internal/tui/views/ -bench=. -benchmem -count=1

clean:
	rm -rf bin dist

snapshot:
	goreleaser release --snapshot --clean
