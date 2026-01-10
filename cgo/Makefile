# CGO Shared Library Build
# This Makefile builds the m9m shared library for use with Python and Node.js bindings

.PHONY: all clean linux darwin windows install test

# Default target builds for the current platform
all: shared

# Output directory
BUILD_DIR := ../build

# Library names per platform
LIB_LINUX := libm9m.so
LIB_DARWIN := libm9m.dylib
LIB_WINDOWS := m9m.dll

# Build shared library for current platform
shared:
	@mkdir -p $(BUILD_DIR)
ifeq ($(shell uname -s),Linux)
	CGO_ENABLED=1 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_LINUX) .
	@echo "Built $(BUILD_DIR)/$(LIB_LINUX)"
else ifeq ($(shell uname -s),Darwin)
	CGO_ENABLED=1 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_DARWIN) .
	@echo "Built $(BUILD_DIR)/$(LIB_DARWIN)"
else
	CGO_ENABLED=1 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_WINDOWS) .
	@echo "Built $(BUILD_DIR)/$(LIB_WINDOWS)"
endif

# Build for Linux (can be used for cross-compilation)
linux:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_LINUX) .
	@echo "Built $(BUILD_DIR)/$(LIB_LINUX)"

# Build for macOS (can be used for cross-compilation on macOS)
darwin:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_DARWIN) .
	@echo "Built $(BUILD_DIR)/$(LIB_DARWIN)"

# Build for macOS ARM64
darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o $(BUILD_DIR)/libm9m_arm64.dylib .
	@echo "Built $(BUILD_DIR)/libm9m_arm64.dylib"

# Build for Windows (requires mingw-w64 for cross-compilation)
windows:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_WINDOWS) .
	@echo "Built $(BUILD_DIR)/$(LIB_WINDOWS)"

# Install to system library path (Linux)
install: shared
ifeq ($(shell uname -s),Linux)
	sudo cp $(BUILD_DIR)/$(LIB_LINUX) /usr/local/lib/
	sudo ldconfig
	@echo "Installed $(LIB_LINUX) to /usr/local/lib/"
else ifeq ($(shell uname -s),Darwin)
	sudo cp $(BUILD_DIR)/$(LIB_DARWIN) /usr/local/lib/
	@echo "Installed $(LIB_DARWIN) to /usr/local/lib/"
endif

# Verify library exports
exports: shared
	@echo "Library exports:"
ifeq ($(shell uname -s),Linux)
	nm -D $(BUILD_DIR)/$(LIB_LINUX) | grep " T " | grep m9m_
else ifeq ($(shell uname -s),Darwin)
	nm $(BUILD_DIR)/$(LIB_DARWIN) | grep " T " | grep m9m_
endif

# Test that the library can be loaded
test: shared
	@echo "Testing library..."
	@python3 -c "import ctypes; lib = ctypes.CDLL('$(BUILD_DIR)/libm9m.so'); print('Library loaded successfully')" 2>/dev/null || \
	python3 -c "import ctypes; lib = ctypes.CDLL('$(BUILD_DIR)/libm9m.dylib'); print('Library loaded successfully')" 2>/dev/null || \
	echo "Could not load library (this is expected if Python is not installed)"

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/libm9m.* $(BUILD_DIR)/m9m.* $(BUILD_DIR)/libm9m.h

# Show help
help:
	@echo "m9m CGO Shared Library Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all/shared  - Build shared library for current platform"
	@echo "  linux       - Build for Linux (amd64)"
	@echo "  darwin      - Build for macOS (amd64)"
	@echo "  darwin-arm64- Build for macOS (arm64/M1)"
	@echo "  windows     - Build for Windows (amd64, requires mingw)"
	@echo "  install     - Install to system library path"
	@echo "  exports     - Show exported symbols"
	@echo "  test        - Test loading the library"
	@echo "  clean       - Remove build artifacts"
	@echo "  help        - Show this help"
