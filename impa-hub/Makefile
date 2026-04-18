.PHONY: build run dev docker-build docker-run test lint clean

# Build
build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/impa-hub ./cmd/impa-hub

build-local:
	go build -o bin/impa-hub ./cmd/impa-hub

# Run
run: build-local
	./bin/impa-hub

dev:
	go run ./cmd/impa-hub

# Docker
docker-build:
	docker compose build

docker-build-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t impa365/impa-hub:latest --push .

docker-build-multiarch-manager:
	cd ../impa-hub-manager && docker buildx build --platform linux/amd64,linux/arm64 -t impa365/impa-hub-manager:latest --push .

docker-build-all:
	docker buildx build --platform linux/amd64,linux/arm64 -t impa365/impa-hub:latest --push .
	cd ../impa-hub-manager && docker buildx build --platform linux/amd64,linux/arm64 -t impa365/impa-hub-manager:latest --push .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f impa-hub

# Test
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	go fmt ./...
	go vet ./...

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Dependencies
deps:
	go mod tidy
	go mod download

# Full check
check: deps lint test
