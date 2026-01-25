#!/bin/bash
# m9m installer - downloads and installs the latest release
# Usage: curl -sSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash

set -e

REPO="neul-labs/m9m"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="m9m"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
        *) error "Unsupported architecture: $arch" ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        error "Failed to fetch latest version"
    fi
    echo "$version"
}

# Download and verify binary
download_binary() {
    local os="$1"
    local arch="$2"
    local version="$3"
    local artifact_name
    local download_url
    local checksum_url

    if [ "$os" = "windows" ]; then
        artifact_name="${BINARY_NAME}-${os}-${arch}.exe"
    else
        artifact_name="${BINARY_NAME}-${os}-${arch}"
    fi

    download_url="https://github.com/${REPO}/releases/download/${version}/${artifact_name}"
    checksum_url="${download_url}.sha256"

    info "Downloading ${artifact_name}..."

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download binary
    if ! curl -sSL -o "${tmp_dir}/${artifact_name}" "$download_url"; then
        error "Failed to download binary from ${download_url}"
    fi

    # Download and verify checksum
    info "Verifying checksum..."
    if curl -sSL -o "${tmp_dir}/${artifact_name}.sha256" "$checksum_url" 2>/dev/null; then
        cd "$tmp_dir"
        if command -v sha256sum &> /dev/null; then
            if ! sha256sum -c "${artifact_name}.sha256" &> /dev/null; then
                error "Checksum verification failed"
            fi
        elif command -v shasum &> /dev/null; then
            if ! shasum -a 256 -c "${artifact_name}.sha256" &> /dev/null; then
                error "Checksum verification failed"
            fi
        else
            warn "No checksum tool available, skipping verification"
        fi
        cd - > /dev/null
    else
        warn "Checksum file not available, skipping verification"
    fi

    # Install binary
    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmp_dir}/${artifact_name}" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "${tmp_dir}/${artifact_name}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    info "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Main installation
main() {
    echo ""
    echo "m9m Installer"
    echo "============="
    echo ""

    local os arch version

    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected OS: ${os}"
    info "Detected Architecture: ${arch}"

    version=$(get_latest_version)
    info "Latest version: ${version}"

    download_binary "$os" "$arch" "$version"

    echo ""
    info "Installation complete!"
    echo ""
    echo "Run 'm9m --help' to get started."
    echo ""
}

main "$@"
