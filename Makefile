.PHONY: build build-dev build-release test lint clean help

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
	@echo "  lint            Run linter (golangci-lint)"
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
	@echo "Building release binaries for all platforms..."
	mkdir -p dist
	# macOS Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/gemtracker-darwin-amd64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker_darwin_amd64.tar.gz gemtracker-darwin-amd64
	rm dist/gemtracker-darwin-amd64
	# macOS Apple Silicon (arm64)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/gemtracker-darwin-arm64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker_darwin_arm64.tar.gz gemtracker-darwin-arm64
	rm dist/gemtracker-darwin-arm64
	# Linux x86-64 (amd64)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/gemtracker-linux-amd64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker_linux_amd64.tar.gz gemtracker-linux-amd64
	rm dist/gemtracker-linux-amd64
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/gemtracker-linux-arm64 ./cmd/gemtracker
	tar -C dist -czf dist/gemtracker_linux_arm64.tar.gz gemtracker-linux-arm64
	rm dist/gemtracker-linux-arm64
	# Windows x86-64 (amd64)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/gemtracker-windows-amd64.exe ./cmd/gemtracker
	cd dist && zip -q gemtracker_windows_amd64.zip gemtracker-windows-amd64.exe && rm gemtracker-windows-amd64.exe
	# Windows ARM64
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/gemtracker-windows-arm64.exe ./cmd/gemtracker
	cd dist && zip -q gemtracker_windows_arm64.zip gemtracker-windows-arm64.exe && rm gemtracker-windows-arm64.exe
	@echo "Release binaries created:"
	@ls -lh dist/

test:
	go test -v ./...

lint:
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "gofmt found formatting issues:"; \
		gofmt -s -d .; \
		exit 1; \
	fi
	go vet ./...

clean:
	rm -f gemtracker
	rm -rf dist/
	go clean
