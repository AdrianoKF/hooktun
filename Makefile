.PHONY: build-relay build-client build-all run-relay run-client test clean docker-build docker-run release release-snapshot

# Build for current platform
build-relay:
	@echo "Building hooktun server..."
	@go build -o bin/hooktun ./cmd/relay

build-client:
	@echo "Building hooktun client..."
	@go build -o bin/hooktun-client ./cmd/client

build-all: build-relay build-client

# Run targets
run-relay:
	@go run ./cmd/relay --port=8080 --log-level=info

run-client:
	@go run ./cmd/client \
		--relay-url=http://localhost:8080 \
		--channel-id=test123 \
		--target-url=http://localhost:3000 \
		--log-level=info

# Test
test:
	@go test -v ./...

# Clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Docker targets
docker-build:
	@docker build -f deployments/Dockerfile -t hooktun:latest .

docker-run:
	@docker run -p 8080:8080 hooktun:latest

# Install dependencies
deps:
	@go mod download
	@go mod tidy

# GoReleaser targets
release:
	@echo "Creating release with GoReleaser..."
	@goreleaser release --clean

release-snapshot:
	@echo "Creating snapshot release (no git tags required)..."
	@goreleaser release --snapshot --clean

release-check:
	@echo "Checking GoReleaser configuration..."
	@goreleaser check
