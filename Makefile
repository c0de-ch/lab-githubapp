.PHONY: dev build clean docker

# Development with auto-reload
dev:
	DEV_MODE=true go run .

# Build production binary
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/ghapp .

# Clean build artifacts
clean:
	rm -rf bin/ data/

# Run tests
test:
	go test ./... -v

# Run linting
vet:
	go vet ./...

# Docker image
docker:
	docker build -t ghapp .

# Docker run
docker-run:
	docker run --rm -p 8080:8080 \
		-v $(PWD)/private-key.pem:/app/private-key.pem:ro \
		-v $(PWD)/data:/app/data \
		--env-file .env \
		ghapp

# Format code
fmt:
	go fmt ./...
