.PHONY: build test clean help

help:
	@echo "gemtracker - Makefile targets"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the gemtracker binary"
	@echo "  test        Run tests"
	@echo "  clean       Remove build artifacts"
	@echo "  help        Show this help message"

build:
	go build -o gemtracker ./cmd/gemtracker

test:
	go test -v ./...

clean:
	rm -f gemtracker
	rm -rf dist/
	go clean
