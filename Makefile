# VMTerminal Makefile

BINARY=vmterminal
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/javanstorm/vmterminal/internal/version.Version=$(VERSION) -X github.com/javanstorm/vmterminal/internal/version.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test vet fmt check install run status help

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/vmterminal

# Build for release (with optimizations)
release:
	CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BINARY) ./cmd/vmterminal

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f $(BINARY)-*
	go clean

# Run tests
test:
	go test -v ./...

# Run go vet
vet:
	go vet ./...

# Check formatting
fmt:
	gofmt -l .
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt-fix' to fix formatting" && exit 1)

# Fix formatting
fmt-fix:
	gofmt -w .

# Run all checks
check: fmt vet build
	@echo "All checks passed!"

# Install to /usr/local/bin
install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

# Uninstall from /usr/local/bin
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled from /usr/local/bin/$(BINARY)"

# Run the VM
run: build
	./$(BINARY) run

# Check VM status
status: build
	./$(BINARY) status

# Show version
version: build
	./$(BINARY) version

# Show help
help: build
	./$(BINARY) --help

# Cross-compile for different platforms
cross:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/vmterminal
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 ./cmd/vmterminal
	@echo "Note: macOS builds require CGO and must be built on macOS"

# Download dependencies
deps:
	go mod download
	go mod tidy

# Show dependencies
deps-list:
	go list -m all

# Generate completion scripts
completion: build
	./$(BINARY) completion bash > vmterminal.bash
	./$(BINARY) completion zsh > _vmterminal
	./$(BINARY) completion fish > vmterminal.fish
	@echo "Generated completion scripts: vmterminal.bash, _vmterminal, vmterminal.fish"

# Test login shell detection
test-login-shell: build
	@echo "Creating symlink to test login shell detection..."
	ln -sf ./$(BINARY) ./-$(BINARY)
	@echo "Run './-$(BINARY)' to test login shell mode"
	@echo "Clean up with: rm ./-$(BINARY)"
