#!/bin/bash
set -e

# aimemo installation script
# Usage: curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="MyAgentHubs/aimemo"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="aimemo"

# Helper functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

info() {
    echo -e "${BLUE}==>${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning: $1${NC}"
}

# Check if running as root
is_root() {
    [ "$(id -u)" -eq 0 ]
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        *)          echo "unknown" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64)     echo "amd64" ;;
        aarch64)    echo "arm64" ;;
        arm64)      echo "arm64" ;;
        *)          echo "unknown" ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -sSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v?([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        error "Failed to fetch latest version from GitHub"
    fi

    echo "$version"
}

# Download and install
install_aimemo() {
    local os arch version download_url tmp_dir

    info "Detecting system information..."
    os=$(detect_os)
    arch=$(detect_arch)

    if [ "$os" = "unknown" ]; then
        error "Unsupported operating system: $(uname -s)"
    fi

    if [ "$arch" = "unknown" ]; then
        error "Unsupported architecture: $(uname -m)"
    fi

    success "Detected: ${os}/${arch}"

    info "Fetching latest version..."
    version=$(get_latest_version)
    success "Latest version: v${version}"

    # Construct download URL
    local archive_name="aimemo_${version}_${os}_${arch}.tar.gz"
    download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"

    info "Downloading ${archive_name}..."
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    if ! curl -sSL "$download_url" -o "${tmp_dir}/${archive_name}"; then
        error "Failed to download from ${download_url}"
    fi

    success "Downloaded to ${tmp_dir}/${archive_name}"

    # Verify checksum
    info "Verifying checksum..."
    local checksum_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    if ! curl -sSL "$checksum_url" -o "${tmp_dir}/checksums.txt"; then
        warn "Failed to download checksums.txt, skipping verification"
    else
        # Change to tmp_dir for checksum verification
        (
            cd "$tmp_dir" || exit 1

            if command -v sha256sum >/dev/null 2>&1; then
                if grep "${archive_name}" checksums.txt | sha256sum -c --status; then
                    success "Checksum verification passed"
                else
                    error "Checksum verification failed - file may be corrupted or tampered"
                fi
            elif command -v shasum >/dev/null 2>&1; then
                if grep "${archive_name}" checksums.txt | shasum -a 256 -c --status; then
                    success "Checksum verification passed"
                else
                    error "Checksum verification failed - file may be corrupted or tampered"
                fi
            else
                warn "No checksum tool found (sha256sum or shasum), skipping verification"
            fi
        )
    fi

    info "Extracting archive..."
    if ! tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"; then
        error "Failed to extract archive"
    fi

    success "Extracted successfully"

    # Install binary
    info "Installing aimemo to ${INSTALL_DIR}..."

    if is_root; then
        # Running as root, install directly
        if ! mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"; then
            error "Failed to install binary to ${INSTALL_DIR}"
        fi
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        # Not root, try with sudo
        if command -v sudo >/dev/null 2>&1; then
            if ! sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"; then
                error "Failed to install binary to ${INSTALL_DIR} (sudo required)"
            fi
            sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
        else
            error "Installation to ${INSTALL_DIR} requires root privileges. Please run with sudo or as root."
        fi
    fi

    success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

    # Verify installation
    info "Verifying installation..."
    if ! command -v aimemo >/dev/null 2>&1; then
        warn "${INSTALL_DIR} may not be in your PATH"

        # Detect user's shell and suggest appropriate config file
        local shell_config
        if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
            shell_config="~/.zshrc"
        elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
            shell_config="~/.bashrc"
        else
            shell_config="~/.profile"
        fi

        warn "Add this line to your ${shell_config}:"
        echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
    fi

    local installed_version
    installed_version=$(aimemo --version 2>&1 | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")

    if [ "$installed_version" = "v${version}" ] || [ "$installed_version" = "${version}" ]; then
        success "Verification passed: aimemo ${installed_version}"
    else
        warn "Version mismatch: expected ${version}, got ${installed_version}"
    fi

    echo ""
    success "aimemo installed successfully!"
    echo ""
}

# Print next steps
print_next_steps() {
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}   Next Steps${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "1. Initialize memory for your project:"
    echo -e "   ${BLUE}cd your-project && aimemo init${NC}"
    echo ""
    echo "2. Register with your AI coding client:"
    echo ""
    echo "   ${BLUE}# Claude Code${NC}"
    echo "   claude mcp add-json aimemo-memory '{\"command\":\"aimemo\",\"args\":[\"serve\"]}'"
    echo ""
    echo "   ${BLUE}# OpenClaw${NC}"
    echo "   Add to ~/.openclaw/openclaw.json:"
    echo "   {"
    echo "     \"mcpServers\": {"
    echo "       \"aimemo-memory\": {"
    echo "         \"command\": \"${INSTALL_DIR}/aimemo\","
    echo "         \"args\": [\"serve\"]"
    echo "       }"
    echo "     }"
    echo "   }"
    echo ""
    echo "   ${BLUE}# Cursor / Windsurf / Cline / Continue / Zed${NC}"
    echo "   See: https://github.com/MyAgentHubs/aimemo#client-support"
    echo ""
    echo "3. Restart your AI coding client"
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Documentation:"
    echo "  • Main README: https://github.com/MyAgentHubs/aimemo"
    echo "  • OpenClaw Integration: https://github.com/MyAgentHubs/aimemo/blob/main/docs/openclaw-integration.md"
    echo ""
    echo "Need help? Open an issue: https://github.com/MyAgentHubs/aimemo/issues"
    echo ""
}

# Main
main() {
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}   aimemo installer${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""

    # Check dependencies
    if ! command -v curl >/dev/null 2>&1; then
        error "curl is required but not installed"
    fi

    if ! command -v tar >/dev/null 2>&1; then
        error "tar is required but not installed"
    fi

    install_aimemo
    print_next_steps
}

main "$@"
