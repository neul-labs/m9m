# Makefile for m9m

# Build variables
BINARY_NAME=m9m
MAIN_FILE=cmd/m9m/main.go

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	go build -o ${BINARY_NAME} ${MAIN_FILE}

# Install dependencies
.PHONY: deps
deps:
	go mod tidy

# Run tests
.PHONY: test
test:
	go test ./...

# Run tests with coverage
.PHONY: coverage
coverage:
	go test -cover ./...

# Run tests with coverage and open in browser
.PHONY: coverage-html
coverage-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Vet code
.PHONY: vet
vet:
	go vet ./...

# Check code with lint
.PHONY: lint
lint:
	golint ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f ${BINARY_NAME}
	rm -f coverage.out
	rm -rf web/dist
	rm -rf web/node_modules

# Frontend build targets
WEB_DIR=web

# Install frontend dependencies
.PHONY: web-deps
web-deps:
	cd ${WEB_DIR} && npm install

# Build frontend for production
.PHONY: web-build
web-build: web-deps
	cd ${WEB_DIR} && npm run build
	@echo "Frontend built to ${WEB_DIR}/dist"

# Run frontend dev server
.PHONY: web-dev
web-dev: web-deps
	cd ${WEB_DIR} && npm run dev

# Build with embedded frontend
.PHONY: build-with-frontend
build-with-frontend: web-build
	mkdir -p internal/web/dist
	cp -r ${WEB_DIR}/dist/* internal/web/dist/
	go build -o ${BINARY_NAME} ${MAIN_FILE}

# Run development mode (backend + frontend proxy)
.PHONY: dev
dev:
	@echo "Starting m9m in development mode..."
	@echo "Run 'make web-dev' in another terminal for frontend hot-reload"
	./${BINARY_NAME} --mode control

# Install golint if not present
.PHONY: install-lint
install-lint:
	go install golang.org/x/lint/golint@latest

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all                 - Build the application (default)"
	@echo "  build               - Build the application"
	@echo "  build-with-frontend - Build with embedded frontend"
	@echo "  deps                - Install Go dependencies"
	@echo "  test                - Run tests"
	@echo "  coverage            - Run tests with coverage"
	@echo "  coverage-html       - Run tests with coverage and open in browser"
	@echo "  fmt                 - Format code"
	@echo "  vet                 - Vet code"
	@echo "  lint                - Check code with lint"
	@echo "  clean               - Clean build artifacts"
	@echo "  install-lint        - Install golint"
	@echo ""
	@echo "Frontend targets:"
	@echo "  web-deps            - Install frontend dependencies"
	@echo "  web-build           - Build frontend for production"
	@echo "  web-dev             - Run frontend dev server (hot-reload)"
	@echo "  dev                 - Run backend in development mode"
	@echo ""
	@echo "  help                - Show this help"