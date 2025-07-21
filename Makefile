.PHONY: build test test-unit test-integration run clean docker-build docker-run docker-test

# Build the application
build:
	go build -o bin/email-catch ./cmd/server
	go build -o bin/certmanager ./cmd/certmanager

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test ./tests/unit/... -v

# Run integration tests
test-integration:
	go test ./tests/integration/... -v

# Run the application
run:
	go run ./cmd/server -config config/config.yaml

# Run with test config
run-test:
	go run ./cmd/server -config config/test-config.yaml

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf emails/
	rm -rf test-emails/
	rm -rf logs/

# Build Docker image
docker-build:
	docker build -t email-catch .

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Run tests with Docker Compose
docker-test:
	docker-compose -f docker-compose.test.yaml up --build --abort-on-container-exit

# Stop Docker services
docker-stop:
	docker-compose down
	docker-compose -f docker-compose.test.yaml down

# Test email sending (requires running server)
test-send-simple:
	go run ./tests/scripts/test_email_sender.go -type simple -count 1

test-send-attachments:
	go run ./tests/scripts/test_email_sender.go -type attachments -count 1

test-send-all-ports:
	go run ./tests/scripts/test_email_sender.go -ports "2525,2526,2527,2528" -type all -count 1

test-send-load:
	go run ./tests/scripts/test_email_sender.go -count 10 -delay 50ms

# Development helpers
dev-setup:
	mkdir -p emails logs certs
	mkdir -p test-emails

dev-minio:
	docker run -d \
		--name dev-minio \
		-p 9000:9000 \
		-p 9001:9001 \
		-e MINIO_ROOT_USER=minioadmin \
		-e MINIO_ROOT_PASSWORD=minioadmin \
		minio/minio:latest server /data --console-address ":9001"

dev-stop-minio:
	docker stop dev-minio || true
	docker rm dev-minio || true

# Linting and formatting
fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

# Dependencies
deps:
	go mod tidy
	go mod download

# Generate test certificates
gen-certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
		-subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Generate self-signed certificates using certmanager
gen-self-signed:
	go run ./cmd/certmanager -action generate-self-signed -domains "localhost,127.0.0.1"

# Get Let's Encrypt certificate information
cert-info:
	go run ./cmd/certmanager -action info

# Renew Let's Encrypt certificates
cert-renew:
	go run ./cmd/certmanager -action renew

# Run with Let's Encrypt configuration
run-letsencrypt:
	go run ./cmd/server -config config/config-letsencrypt.yaml

# All-in-one test suite
test-all: clean dev-setup build test docker-build docker-test

# Quick development test
test-quick: build run-test &
	sleep 2
	$(MAKE) test-send-simple
	pkill -f "email-catch"