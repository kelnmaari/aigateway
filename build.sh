#!/bin/bash
# Cross-compilation build script для Linux и Windows
# Ollama-OpenAI Proxy v1.4.3

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Build configuration
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%d_%H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="dist"

# Platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Clean previous builds
clean() {
    log_info "Cleaning previous builds..."
    rm -rf "$OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"
}

# Build for specific platform
build_platform() {
    local os=$1
    local arch=$2
    local output_name=$3
    
    log_info "Building $os/$arch..."
    
    # Set environment
    export GOOS=$os
    export GOARCH=$arch
    export CGO_ENABLED=0  # Disable CGO for cross-compilation
    
    # Build flags with version information
    local ldflags="-w -s"
    ldflags="$ldflags -X 'ollama-openai-proxy/internal/version.Version=$VERSION'"
    ldflags="$ldflags -X 'ollama-openai-proxy/internal/version.GitCommit=$GIT_COMMIT'"
    ldflags="$ldflags -X 'ollama-openai-proxy/internal/version.BuildDate=$BUILD_DATE'"
    
    # Build tags for SQLite
    local tags="sqlite_fts5 sqlite_json1"
    
    # Output file
    local output="$OUTPUT_DIR/$output_name"
    
    # Build
    go build \
        -ldflags="$ldflags" \
        -tags "$tags" \
        -trimpath \
        -o "$output" \
        ./cmd/server
    
    if [ $? -eq 0 ]; then
        log_info "✓ Built: $output ($(du -h "$output" | cut -f1))"
    else
        log_error "✗ Failed to build $os/$arch"
        exit 1
    fi
}

# Build all platforms
build_all() {
    log_info "Starting build process..."
    log_info "Version: $VERSION"
    log_info "Build Time: $BUILD_TIME"
    log_info "Git Commit: $GIT_COMMIT"
    echo ""
    
    # Linux AMD64
    build_platform "linux" "amd64" "ollama-proxy-linux-amd64"
    
    # Linux ARM64
    build_platform "linux" "arm64" "ollama-proxy-linux-arm64"
    
    # Windows AMD64
    build_platform "windows" "amd64" "ollama-proxy-windows-amd64.exe"
    
    echo ""
    log_info "Build completed successfully!"
    log_info "Binaries are in: $OUTPUT_DIR/"
    echo ""
    ls -lh "$OUTPUT_DIR/"
}

# Build with Docker (alternative method with CGO support)
build_docker() {
    log_info "Building with Docker (with CGO support)..."
    
    # Linux AMD64
    log_info "Building Linux AMD64 (with Docker)..."
    docker run --rm \
        -v "$PWD":/src \
        -w /src \
        golang:1.25-alpine \
        sh -c "apk add --no-cache git gcc musl-dev sqlite-dev && \
               go build -ldflags='-w -s -X main.Version=$VERSION' \
               -tags 'sqlite_fts5 sqlite_json1' \
               -o dist/ollama-proxy-linux-amd64-cgo ./cmd/server"
    
    log_info "✓ Built with Docker: dist/ollama-proxy-linux-amd64-cgo"
}

# Package binaries with configs
package() {
    log_info "Creating distribution packages..."
    
    local version_dir="$OUTPUT_DIR/ollama-proxy-$VERSION"
    
    for binary in "$OUTPUT_DIR"/ollama-proxy-*; do
        if [ -f "$binary" ]; then
            local basename=$(basename "$binary")
            local platform="${basename#ollama-proxy-}"
            platform="${platform%.*}"  # Remove .exe if exists
            
            local pkg_dir="$version_dir-$platform"
            mkdir -p "$pkg_dir"
            
            # Copy binary
            cp "$binary" "$pkg_dir/"
            
            # Copy configs
            cp -r configs "$pkg_dir/"
            cp -r web "$pkg_dir/"
            
            # Copy docs
            cp README.md "$pkg_dir/" 2>/dev/null || true
            cp LICENSE "$pkg_dir/" 2>/dev/null || true
            
            # Create archive
            if [[ "$platform" == *"windows"* ]]; then
                (cd "$OUTPUT_DIR" && zip -r "ollama-proxy-$VERSION-$platform.zip" "$(basename "$pkg_dir")")
                log_info "✓ Created: $OUTPUT_DIR/ollama-proxy-$VERSION-$platform.zip"
            else
                tar -czf "$OUTPUT_DIR/ollama-proxy-$VERSION-$platform.tar.gz" -C "$OUTPUT_DIR" "$(basename "$pkg_dir")"
                log_info "✓ Created: $OUTPUT_DIR/ollama-proxy-$VERSION-$platform.tar.gz"
            fi
            
            # Cleanup temp dir
            rm -rf "$pkg_dir"
        fi
    done
    
    log_info "Packaging completed!"
}

# Display usage
usage() {
    cat << EOF
Usage: $0 [COMMAND]

Commands:
    all         Build for all platforms (default)
    linux       Build for Linux only (amd64 + arm64)
    windows     Build for Windows only (amd64)
    docker      Build with Docker (CGO support)
    package     Create distribution packages
    clean       Clean build artifacts
    help        Display this help message

Environment Variables:
    VERSION     Version tag (default: 1.3.0)

Examples:
    # Build all platforms
    ./build.sh all

    # Build Linux only
    ./build.sh linux

    # Build with custom version
    VERSION=1.3.1 ./build.sh all

    # Build and package
    ./build.sh all && ./build.sh package

EOF
}

# Main
main() {
    local command="${1:-all}"
    
    case $command in
        all)
            clean
            build_all
            ;;
        linux)
            clean
            build_platform "linux" "amd64" "ollama-proxy-linux-amd64"
            build_platform "linux" "arm64" "ollama-proxy-linux-arm64"
            log_info "Linux builds completed!"
            ;;
        windows)
            clean
            build_platform "windows" "amd64" "ollama-proxy-windows-amd64.exe"
            log_info "Windows build completed!"
            ;;
        docker)
            clean
            build_docker
            ;;
        package)
            package
            ;;
        clean)
            clean
            log_info "Cleaned!"
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Check Go installation
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.25 or later."
    exit 1
fi

# Run main
main "$@"

