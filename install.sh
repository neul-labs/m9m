#!/bin/bash
# m9m installer - downloads and installs the latest release
# Falls back to local compilation if remote binaries are unavailable
#
# Usage: curl -sSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash
#
# Options (via environment variables):
#   INSTALL_DIR   - Installation directory (default: /usr/local/bin)
#   BUILD_LOCAL   - Force local build (set to "1" to skip download attempt)

set -e

REPO="neul-labs/m9m"
REPO_URL="https://github.com/${REPO}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="m9m"
MIN_GO_VERSION="1.21"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $os" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l) echo "arm" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Check if Go is installed and meets minimum version
check_go() {
    if ! command -v go &> /dev/null; then
        return 1
    fi

    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')

    # Compare versions
    local major minor min_major min_minor
    major=$(echo "$go_version" | cut -d. -f1)
    minor=$(echo "$go_version" | cut -d. -f2)
    min_major=$(echo "$MIN_GO_VERSION" | cut -d. -f1)
    min_minor=$(echo "$MIN_GO_VERSION" | cut -d. -f2)

    if [ "$major" -gt "$min_major" ]; then
        return 0
    elif [ "$major" -eq "$min_major" ] && [ "$minor" -ge "$min_minor" ]; then
        return 0
    else
        return 1
    fi
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    echo "$version"
}

# Try to download pre-built binary
try_download_binary() {
    local os="$1"
    local arch="$2"
    local version="$3"
    local artifact_name
    local download_url

    if [ -z "$version" ]; then
        warn "No release version found"
        return 1
    fi

    if [ "$os" = "windows" ]; then
        artifact_name="${BINARY_NAME}-${os}-${arch}.exe"
    else
        artifact_name="${BINARY_NAME}-${os}-${arch}"
    fi

    download_url="https://github.com/${REPO}/releases/download/${version}/${artifact_name}"

    step "Attempting to download ${artifact_name} (${version})..."

    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Try to download binary
    if curl -sSL --fail -o "${tmp_dir}/${artifact_name}" "$download_url" 2>/dev/null; then
        # Download succeeded, verify checksum if available
        local checksum_url="${download_url}.sha256"
        if curl -sSL -o "${tmp_dir}/${artifact_name}.sha256" "$checksum_url" 2>/dev/null; then
            info "Verifying checksum..."
            cd "$tmp_dir"
            if command -v sha256sum &> /dev/null; then
                if ! sha256sum -c "${artifact_name}.sha256" &> /dev/null; then
                    warn "Checksum verification failed"
                    cd - > /dev/null
                    rm -rf "$tmp_dir"
                    return 1
                fi
            elif command -v shasum &> /dev/null; then
                if ! shasum -a 256 -c "${artifact_name}.sha256" &> /dev/null; then
                    warn "Checksum verification failed"
                    cd - > /dev/null
                    rm -rf "$tmp_dir"
                    return 1
                fi
            fi
            cd - > /dev/null
        fi

        # Install binary
        install_binary "${tmp_dir}/${artifact_name}"
        rm -rf "$tmp_dir"
        return 0
    else
        rm -rf "$tmp_dir"
        return 1
    fi
}

# Build from source
build_from_source() {
    step "Building from source..."

    if ! check_go; then
        error "Go ${MIN_GO_VERSION}+ is required for local build. Please install Go from https://go.dev/dl/"
    fi

    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | sed 's/go//')
    info "Found Go version: ${go_version}"

    local build_dir
    local cleanup_build_dir=false

    # Check if we're already in the m9m repo
    if [ -f "go.mod" ] && grep -q "module github.com/neul-labs/m9m" go.mod 2>/dev/null; then
        info "Building from current directory..."
        build_dir="$(pwd)"
    elif [ -f "../go.mod" ] && grep -q "module github.com/neul-labs/m9m" ../go.mod 2>/dev/null; then
        info "Building from parent directory..."
        build_dir="$(cd .. && pwd)"
    else
        # Need to clone or download
        local tmp_dir
        tmp_dir=$(mktemp -d)
        build_dir="${tmp_dir}/m9m"
        cleanup_build_dir=true

        # Clone or download source
        if command -v git &> /dev/null; then
            info "Cloning repository..."
            if ! git clone --depth 1 "${REPO_URL}.git" "${build_dir}" 2>/dev/null; then
                # Try alternate URL
                if ! git clone --depth 1 "https://github.com/neul-labs/m9m.git" "${build_dir}" 2>/dev/null; then
                    rm -rf "$tmp_dir"
                    error "Failed to clone repository"
                fi
            fi
        else
            info "Downloading source archive..."
            local archive_url="${REPO_URL}/archive/refs/heads/main.tar.gz"
            if ! curl -sSL -o "${tmp_dir}/m9m.tar.gz" "$archive_url" 2>/dev/null; then
                # Try alternate URL
                archive_url="https://github.com/neul-labs/m9m/archive/refs/heads/main.tar.gz"
                if ! curl -sSL -o "${tmp_dir}/m9m.tar.gz" "$archive_url"; then
                    rm -rf "$tmp_dir"
                    error "Failed to download source"
                fi
            fi
            mkdir -p "${build_dir}"
            tar -xzf "${tmp_dir}/m9m.tar.gz" -C "${build_dir}" --strip-components=1
        fi
    fi

    local original_dir
    original_dir="$(pwd)"
    cd "${build_dir}"

    # Build
    info "Compiling m9m..."

    # Get version info
    local version commit build_date
    version=$(git describe --tags 2>/dev/null || echo "dev")
    commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    build_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Build with version info
    local ldflags="-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.BuildDate=${build_date}"

    if ! go build -ldflags "$ldflags" -o "${BINARY_NAME}" ./cmd/m9m; then
        cd "$original_dir"
        [ "$cleanup_build_dir" = true ] && rm -rf "$(dirname "$build_dir")"
        error "Build failed"
    fi

    info "Build successful!"

    # Install
    install_binary "${build_dir}/${BINARY_NAME}"

    # Cleanup
    rm -f "${build_dir}/${BINARY_NAME}"
    cd "$original_dir"
    if [ "$cleanup_build_dir" = true ]; then
        rm -rf "$(dirname "$build_dir")"
    fi
}

# Install binary to target directory
install_binary() {
    local binary_path="$1"

    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    info "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    if command -v "${BINARY_NAME}" &> /dev/null; then
        local installed_version
        installed_version=$("${BINARY_NAME}" version 2>/dev/null || echo "unknown")
        info "Verified: ${installed_version}"
        return 0
    elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        local installed_version
        installed_version=$("${INSTALL_DIR}/${BINARY_NAME}" version 2>/dev/null || echo "unknown")
        info "Verified: ${installed_version}"
        return 0
    else
        warn "Installation verification failed"
        return 1
    fi
}

# Main installation
main() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║         m9m Installer                 ║"
    echo "║   High-Performance Workflow Engine    ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    local os arch version
    local download_success=false

    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected: ${os}/${arch}"

    # Check if forced local build
    if [ "${BUILD_LOCAL}" = "1" ]; then
        info "BUILD_LOCAL=1, skipping download attempt"
    else
        # Try to get latest version and download
        version=$(get_latest_version)

        if [ -n "$version" ]; then
            info "Latest release: ${version}"

            if try_download_binary "$os" "$arch" "$version"; then
                download_success=true
            else
                warn "Pre-built binary not available for ${os}/${arch}"
            fi
        else
            warn "Could not fetch release information from GitHub"
        fi
    fi

    # Fall back to local build
    if [ "$download_success" = false ]; then
        echo ""
        step "Falling back to local compilation..."
        echo ""

        if check_go; then
            build_from_source
        else
            echo ""
            error "Cannot install m9m:
  - Pre-built binary not available
  - Go ${MIN_GO_VERSION}+ not found for local compilation

Please either:
  1. Install Go from https://go.dev/dl/ and re-run this script
  2. Build manually: git clone ${REPO_URL} && cd m9m && make build
  3. Download a release manually from ${REPO_URL}/releases"
        fi
    fi

    # Verify installation
    echo ""
    verify_installation

    echo ""
    echo "════════════════════════════════════════"
    info "Installation complete!"
    echo ""
    echo "  Get started:"
    echo "    m9m init              # Initialize workspace"
    echo "    m9m node list         # List available nodes"
    echo "    m9m --help            # Show all commands"
    echo ""
    echo "  Documentation: ${REPO_URL}"
    echo "════════════════════════════════════════"
    echo ""
}

main "$@"
