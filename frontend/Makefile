.PHONY: dev build build-prod preview clean install lint

# Production URLs (override these for your environment)
PROD_API_URL ?= https://api.tmpemail.xyz
PROD_WS_URL ?= wss://api.tmpemail.xyz

# Install dependencies
install:
	npm install

# Run development server (uses localhost:8080 defaults)
dev:
	npm run dev

# Build for production with production URLs
build:
	VITE_API_URL=$(PROD_API_URL) \
	VITE_WS_URL=$(PROD_WS_URL) \
	npm run build

# Build with local URLs (for testing prod build locally)
build-local:
	npm run build

# Preview production build locally
preview: build-local
	npm run preview

# Run linter
lint:
	npm run lint

# Clean build artifacts
clean:
	rm -rf dist node_modules/.vite
