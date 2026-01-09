.PHONY: build run dev clean test fmt vet

# Binary name
BINARY_NAME=tmpemail-api

# Local development paths
DEV_DATA_DIR=./data
DEV_DB_PATH=$(DEV_DATA_DIR)/tmpemail.db
DEV_STORAGE_PATH=$(DEV_DATA_DIR)/mail

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Run the application (production mode)
run: build
	./$(BINARY_NAME)

# Run in development mode with local paths
dev: build
	@mkdir -p $(DEV_DATA_DIR) $(DEV_STORAGE_PATH)
	TMPEMAIL_DB_PATH=$(DEV_DB_PATH) \
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
