.PHONY: test run build clean docker help

# Default target
help:
	@echo "Available targets:"
	@echo "  test        - Run Go tests"
	@echo "  test-v      - Run Go tests with verbose output"
	@echo "  run         - Run the plugin with go run"
	@echo "  build       - Build the plugin binary"
	@echo "  build-all   - Build for all platforms (linux amd64/arm/arm64, windows)"
	@echo "  docker      - Build Docker image for linux/amd64"
	@echo "  clean       - Remove build artifacts"

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Run the plugin
run:
	go run main.go

# Build the plugin binary for current platform
build:
	CGO_ENABLED=0 go build -o drone-github-app

# Build for all platforms using the existing script
build-all:
	./scripts/build.sh

# Build Docker image
docker:
	docker build -t rssnyder/drone-github-app -f docker/Dockerfile.linux.amd64 .

# Clean build artifacts
clean:
	rm -f drone-github-app
	rm -rf release/
	rm -f jwt.txt token.txt output.json
