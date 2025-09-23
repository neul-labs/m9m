# Makefile for n8n-go

# Build variables
BINARY_NAME=n8n-go
MAIN_FILE=cmd/n8n-go/main.go

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

# Install golint if not present
.PHONY: install-lint
install-lint:
	go install golang.org/x/lint/golint@latest

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all             - Build the application (default)"
	@echo "  build           - Build the application"
	@echo "  deps            - Install dependencies"
	@echo "  test            - Run tests"
	@echo "  coverage        - Run tests with coverage"
	@echo "  coverage-html   - Run tests with coverage and open in browser"
	@echo "  fmt             - Format code"
	@echo "  vet             - Vet code"
	@echo "  lint            - Check code with lint"
	@echo "  clean           - Clean build artifacts"
	@echo "  install-lint    - Install golint"
	@echo "  help            - Show this help"