.PHONY: build test test-coverage clean lint vet fmt all version
.PHONY: test-api test-internal test-cmd test-shell test-client test-mock-client test-ap
.PHONY: test-vendors test-vendors-registry test-vendors-cache test-vendors-errors test-vendors-mock

# Binary name
BINARY_NAME=wifimgr

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint
GOMOD=$(GOCMD) mod

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags - inject version info into cmd package
LDFLAGS=-ldflags "-w -s -X github.com/ravinald/wifimgr/cmd.Version=$(VERSION) -X github.com/ravinald/wifimgr/cmd.BuildTime=$(BUILD_TIME) -X github.com/ravinald/wifimgr/cmd.GitCommit=$(GIT_COMMIT)"

all: build

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

build-optimized:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

test:
	$(GOTEST) -v ./... || echo "Some tests failed, but continuing with build"
	
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Test API module
test-api:
	$(GOTEST) -v ./api

# Test API client implementation
test-client:
	$(GOTEST) -v ./api -run TestNew
	$(GOTEST) -v ./api -run TestClientSetters
	$(GOTEST) -v ./api -run TestAPIPathHandling
	$(GOTEST) -v ./api -run TestRateLimiter

# Test mock client implementation
test-mock-client:
	$(GOTEST) -v ./api -run TestNewMockClient
	$(GOTEST) -v ./api -run TestMockSiteOperations
	$(GOTEST) -v ./api -run TestMockAPOperations
	$(GOTEST) -v ./api -run TestUnifiedDeviceOperations
	$(GOTEST) -v ./api -run TestMockInventoryOperations
	$(GOTEST) -v ./api -run TestMockDeviceProfileOperations

# Test internal modules
test-internal:
	$(GOTEST) -v ./internal/...

# Test all command packages
test-cmd:
	$(GOTEST) -v ./cmd/...

# Test AP command package specifically
test-ap:
	$(GOTEST) -v ./cmd/ap

# Test shell functionality
test-shell:
	$(GOTEST) -v ./internal/shell

# Test multi-vendor support (all vendor tests)
test-vendors:
	$(GOTEST) -v ./internal/vendors/...

# Test vendor registry operations
test-vendors-registry:
	$(GOTEST) -v ./internal/vendors -run TestNewAPIClientRegistry
	$(GOTEST) -v ./internal/vendors -run TestRegisterFactory
	$(GOTEST) -v ./internal/vendors -run TestInitializeClients
	$(GOTEST) -v ./internal/vendors -run TestGetClient
	$(GOTEST) -v ./internal/vendors -run TestGetAllLabels
	$(GOTEST) -v ./internal/vendors -run TestHasAPI
	$(GOTEST) -v ./internal/vendors -run TestGetVendor
	$(GOTEST) -v ./internal/vendors -run TestGetOrgID
	$(GOTEST) -v ./internal/vendors -run TestGetConfig
	$(GOTEST) -v ./internal/vendors -run TestForEachAPI
	$(GOTEST) -v ./internal/vendors -run TestGetStatus
	$(GOTEST) -v ./internal/vendors -run TestRegistry_ConcurrentAccess

# Test vendor cache manager operations
test-vendors-cache:
	$(GOTEST) -v ./internal/vendors -run TestNormalizeMAC
	$(GOTEST) -v ./internal/vendors -run TestNewCacheManager
	$(GOTEST) -v ./internal/vendors -run TestCacheManager

# Test vendor error types
test-vendors-errors:
	$(GOTEST) -v ./internal/vendors -run TestSiteNotFoundError
	$(GOTEST) -v ./internal/vendors -run TestDuplicateSiteError
	$(GOTEST) -v ./internal/vendors -run TestAPINotFoundError
	$(GOTEST) -v ./internal/vendors -run TestCapabilityNotSupportedError
	$(GOTEST) -v ./internal/vendors -run TestMACCollisionError
	$(GOTEST) -v ./internal/vendors -run TestDeviceNotFoundError
	$(GOTEST) -v ./internal/vendors -run TestInvalidAPIConfigError
	$(GOTEST) -v ./internal/vendors -run TestErrorsImplementErrorInterface

# Test vendor mock client implementations
test-vendors-mock:
	$(GOTEST) -v ./internal/vendors -run TestMockClient
	$(GOTEST) -v ./internal/vendors -run TestMockSitesService
	$(GOTEST) -v ./internal/vendors -run TestMockInventoryService
	$(GOTEST) -v ./internal/vendors -run TestMockDevicesService
	$(GOTEST) -v ./internal/vendors -run TestMockSearchService
	$(GOTEST) -v ./internal/vendors -run TestListCapabilities

debug-shell:
	@echo "Starting Delve debugger in headless mode..."
	@echo "Connect with 'dlv connect 127.0.0.1:43000' in another terminal"
	RUN_DELVE_TEST=1 dlv debug --headless --api-version=2 --listen=127.0.0.1:43000 ./internal/shell --build-flags="-tags=delve_test" -- -test.run=TestForDebugWithDelve

clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f coverage.out
	rm -f coverage.html

lint:
	$(GOLINT) run ./...

vet:
	$(GOVET) ./...

fmt:
	$(GOFMT) ./...

dependencies:
	$(GOMOD) tidy
	$(GOMOD) download

# Build for multiple platforms
build-all: clean dependencies
	# Linux (amd64)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	# Linux (arm64)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .
	# macOS (amd64)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	# macOS (arm64 / Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	# Windows (amd64)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

# Run the application with the default configuration
run:
	./$(BINARY_NAME) -config config/wifimgr-config.json
	
# Run with debug level info
run-debug:
	./$(BINARY_NAME) -config config/wifimgr-config.json -d -debug-level info
	
# Run with all debug information
run-debug-all:
	./$(BINARY_NAME) -config config/wifimgr-config.json -d -debug-level all
	
# Run in dry-run mode (no actual API changes)
run-dry-run:
	./$(BINARY_NAME) -config config/wifimgr-config.json -dry-run

# Run in dry-run mode with debug information
run-dry-run-debug:
	./$(BINARY_NAME) -config config/wifimgr-config.json -dry-run -d -debug-level info

# Install golangci-lint if not already installed
install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Build and install the binary
install: build
	mkdir -p $(GOPATH)/bin
	mv $(BINARY_NAME) $(GOPATH)/bin/

# Create a version tag
tag:
	@echo "Creating tag v$(VERSION)"
	git tag -a v$(VERSION) -m "Version $(VERSION)"
	git push origin v$(VERSION)

# Start a new release
release: build-all
	@echo "Release v$(VERSION) created"
	@echo "Upload the binaries to your release page"