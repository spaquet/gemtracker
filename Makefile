.PHONY: build build-dev build-release test clean help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

help:
	@echo "gemtracker - Makefile targets"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build the gemtracker binary (development)"
	@echo "  build-release   Build release binaries for macOS (amd64 + arm64)"
	@echo "  test            Run tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Environment Variables:"
	@echo "  VERSION         Version string (default: git tag or 'dev')"
	@echo "  COMMIT          Commit hash (default: git rev)"
	@echo "  DATE            Build date (default: current time)"

build: build-dev

build-dev:
	go build $(LDFLAGS) -o gemtracker ./cmd/gemtracker

build-release:
	@echo "Building release binaries..."
	mkdir -p dist
	# macOS Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/gemtracker-darwin-amd64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker-darwin-amd64.tar.gz gemtracker-darwin-amd64
	# macOS Apple Silicon (arm64)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/gemtracker-darwin-arm64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker-darwin-arm64.tar.gz gemtracker-darwin-arm64
	@echo "Release binaries created:"
	@ls -lh dist/gemtracker-darwin-*.tar.gz

test:
	go test -v ./...

clean:
	rm -f gemtracker
	rm -rf dist/
	go clean
