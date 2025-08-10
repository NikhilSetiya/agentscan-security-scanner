#!/bin/bash
set -e

# AgentScan CLI Installation Script
# Usage: curl -sSL https://install.agentscan.dev | sh

AGENTSCAN_VERSION="${AGENTSCAN_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.agentscan/bin}"
BINARY_NAME="agentscan-cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os
    local arch
    
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)          log_error "Unsupported operating system: $(uname -s)"; exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        armv7l)         arch="arm" ;;
        i386|i686)      arch="386" ;;
        *)              log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Get the latest version from GitHub releases
get_latest_version() {
    if [ "$AGENTSCAN_VERSION" = "latest" ]; then
        log_info "Fetching latest version..."
        local latest_version
        latest_version=$(curl -s https://api.github.com/repos/agentscan/agentscan/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        if [ -z "$latest_version" ]; then
            log_warning "Could not fetch latest version, using v1.0.0"
            echo "v1.0.0"
        else
            echo "$latest_version"
        fi
    else
        echo "$AGENTSCAN_VERSION"
    fi
}

# Download and install the binary
install_binary() {
    local platform="$1"
    local version="$2"
    local download_url="https://github.com/agentscan/agentscan/releases/download/${version}/agentscan-cli-${platform}.tar.gz"
    local temp_dir
    
    log_info "Installing AgentScan CLI ${version} for ${platform}..."
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Download binary
    log_info "Downloading from ${download_url}..."
    if command -v curl >/dev/null 2>&1; then
        curl -sL "$download_url" -o "$temp_dir/agentscan-cli.tar.gz"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$download_url" -O "$temp_dir/agentscan-cli.tar.gz"
    else
        log_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
    
    # Extract binary
    log_info "Extracting binary..."
    tar -xzf "$temp_dir/agentscan-cli.tar.gz" -C "$temp_dir"
    
    # Move binary to install directory
    mv "$temp_dir/$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    log_success "AgentScan CLI installed to $INSTALL_DIR/$BINARY_NAME"
}

# Add to PATH
setup_path() {
    local shell_profile
    
    # Detect shell and profile file
    case "$SHELL" in
        */bash)
            if [ -f "$HOME/.bashrc" ]; then
                shell_profile="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                shell_profile="$HOME/.bash_profile"
            else
                shell_profile="$HOME/.profile"
            fi
            ;;
        */zsh)
            shell_profile="$HOME/.zshrc"
            ;;
        */fish)
            shell_profile="$HOME/.config/fish/config.fish"
            ;;
        *)
            shell_profile="$HOME/.profile"
            ;;
    esac
    
    # Check if already in PATH
    if echo "$PATH" | grep -q "$INSTALL_DIR"; then
        log_info "AgentScan CLI is already in PATH"
        return
    fi
    
    # Add to PATH in profile
    if [ -f "$shell_profile" ]; then
        if ! grep -q "$INSTALL_DIR" "$shell_profile"; then
            echo "" >> "$shell_profile"
            echo "# AgentScan CLI" >> "$shell_profile"
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$shell_profile"
            log_success "Added $INSTALL_DIR to PATH in $shell_profile"
            log_info "Please run 'source $shell_profile' or restart your terminal"
        else
            log_info "PATH already configured in $shell_profile"
        fi
    else
        log_warning "Could not find shell profile file. Please manually add $INSTALL_DIR to your PATH"
    fi
}

# Verify installation
verify_installation() {
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        log_success "Installation verified!"
        
        # Test the binary
        if "$INSTALL_DIR/$BINARY_NAME" version >/dev/null 2>&1; then
            log_info "AgentScan CLI is working correctly"
            "$INSTALL_DIR/$BINARY_NAME" version
        else
            log_warning "Binary installed but may not be working correctly"
        fi
    else
        log_error "Installation failed - binary not found"
        exit 1
    fi
}

# Print usage information
print_usage() {
    echo ""
    echo "🔒 AgentScan CLI Installation Complete!"
    echo ""
    echo "Usage:"
    echo "  agentscan-cli scan                    # Run security scan"
    echo "  agentscan-cli scan --help             # Show scan options"
    echo "  agentscan-cli version                 # Show version"
    echo ""
    echo "Examples:"
    echo "  # Basic scan"
    echo "  agentscan-cli scan"
    echo ""
    echo "  # Scan with API integration"
    echo "  agentscan-cli scan --api-url=https://api.agentscan.dev --api-token=\$TOKEN"
    echo ""
    echo "  # Fail on medium or high severity findings"
    echo "  agentscan-cli scan --fail-on-severity=medium"
    echo ""
    echo "  # Exclude specific paths"
    echo "  agentscan-cli scan --exclude-path=node_modules --exclude-path=vendor"
    echo ""
    echo "Documentation: https://docs.agentscan.dev"
    echo "Support: https://github.com/agentscan/agentscan/issues"
}

# Main installation flow
main() {
    log_info "Starting AgentScan CLI installation..."
    
    # Check dependencies
    if ! command -v tar >/dev/null 2>&1; then
        log_error "tar is required but not installed"
        exit 1
    fi
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Get version
    local version
    version=$(get_latest_version)
    log_info "Installing version: $version"
    
    # Install binary
    install_binary "$platform" "$version"
    
    # Setup PATH
    setup_path
    
    # Verify installation
    verify_installation
    
    # Print usage
    print_usage
}

# Run main function
main "$@"