.PHONY: build run dev clean test fmt vet gen-certs

# Binary name
BINARY_NAME=tmpemail-email-service

# Local development paths (shared with API service)
DEV_STORAGE_PATH=../api/data/mail

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Run the application (production mode)
run: build
	./$(BINARY_NAME)

# Run in development mode with local paths
dev: build
	@mkdir -p $(DEV_STORAGE_PATH)
	TMPEMAIL_STORAGE_PATH=$(DEV_STORAGE_PATH) \
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .

# Generate self-signed TLS certificate for development
gen-certs:
	@mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
		-keyout certs/smtp.key \
		-out certs/smtp.crt \
		-subj "/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,DNS:mail.tmpemail.xyz,DNS:tmpemail.xyz,IP:127.0.0.1"
	@echo "Self-signed certificate generated in certs/"
	@echo "To enable TLS, set TMPEMAIL_TLS_ENABLED=true"
