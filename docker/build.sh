#!/bin/bash
# Build script для multi-platform Docker images
# Поддержка Linux (amd64, arm64) и Windows (amd64)

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
VERSION="${VERSION:-1.3.0}"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
REGISTRY="${REGISTRY:-ghcr.io/yourusername}"
IMAGE_NAME="ollama-openai-proxy"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker buildx version &> /dev/null; then
        log_error "Docker Buildx is not available"
        exit 1
    fi
    
    log_info "Prerequisites OK"
}

# Create buildx builder if not exists
setup_builder() {
    log_info "Setting up Docker Buildx..."
    
    if ! docker buildx inspect multiarch &> /dev/null; then
        log_info "Creating new buildx builder 'multiarch'..."
        docker buildx create --name multiarch --use
    else
        log_info "Using existing builder 'multiarch'"
        docker buildx use multiarch
    fi
    
    docker buildx inspect --bootstrap
}

# Build for specific platform
build_single_platform() {
    local platform=$1
    local tag_suffix=$2
    
    log_info "Building for platform: $platform"
    
    docker buildx build \
        --platform $platform \
        --target production \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME=$BUILD_TIME \
        --build-arg GIT_COMMIT=$GIT_COMMIT \
        --tag ${REGISTRY}/${IMAGE_NAME}:${VERSION}${tag_suffix} \
        --tag ${REGISTRY}/${IMAGE_NAME}:latest${tag_suffix} \
        --file docker/Dockerfile \
        --load \
        .
    
    log_info "Build completed for $platform"
}

# Build multi-platform and push
build_multi_platform() {
    log_info "Building multi-platform images..."
    
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --target production \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME=$BUILD_TIME \
        --build-arg GIT_COMMIT=$GIT_COMMIT \
        --tag ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
        --tag ${REGISTRY}/${IMAGE_NAME}:latest \
        --file docker/Dockerfile \
        --push \
        .
    
    log_info "Multi-platform build and push completed"
}

# Build all platforms separately (for local testing)
build_all_local() {
    log_info "Building all platforms locally..."
    
    # Linux AMD64
    build_single_platform "linux/amd64" "-linux-amd64"
    
    # Linux ARM64
    build_single_platform "linux/arm64" "-linux-arm64"
    
    log_warn "Note: Windows containers require Windows Docker host"
    log_warn "To build Windows image: docker build --platform windows/amd64 ..."
    
    log_info "All local builds completed"
}

# Build and save images to tar files
build_and_export() {
    log_info "Building and exporting images to tar files..."
    
    mkdir -p ./dist
    
    # Linux AMD64
    log_info "Exporting Linux AMD64..."
    docker buildx build \
        --platform linux/amd64 \
        --target production \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME=$BUILD_TIME \
        --build-arg GIT_COMMIT=$GIT_COMMIT \
        --tag ${IMAGE_NAME}:${VERSION}-linux-amd64 \
        --file docker/Dockerfile \
        --output type=docker,dest=./dist/${IMAGE_NAME}-${VERSION}-linux-amd64.tar \
        .
    
    # Linux ARM64
    log_info "Exporting Linux ARM64..."
    docker buildx build \
        --platform linux/arm64 \
        --target production \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME=$BUILD_TIME \
        --build-arg GIT_COMMIT=$GIT_COMMIT \
        --tag ${IMAGE_NAME}:${VERSION}-linux-arm64 \
        --file docker/Dockerfile \
        --output type=docker,dest=./dist/${IMAGE_NAME}-${VERSION}-linux-arm64.tar \
        .
    
    log_info "Images exported to ./dist/"
    ls -lh ./dist/
}

# Display usage
usage() {
    cat << EOF
Usage: $0 [COMMAND]

Commands:
    local       Build for current platform only (default)
    all-local   Build all platforms locally (linux/amd64, linux/arm64)
    multi       Build multi-platform and push to registry
    export      Build and export images to tar files
    help        Display this help message

Environment Variables:
    VERSION     Version tag (default: 1.3.0)
    REGISTRY    Container registry (default: ghcr.io/yourusername)
    BUILD_TIME  Build timestamp (default: current UTC time)
    GIT_COMMIT  Git commit hash (default: auto-detected)

Examples:
    # Build for current platform
    ./docker/build.sh local

    # Build all platforms locally
    ./docker/build.sh all-local

    # Build and push multi-platform
    VERSION=1.3.1 REGISTRY=docker.io/myuser ./docker/build.sh multi

    # Export images to tar files
    ./docker/build.sh export

EOF
}

# Main
main() {
    local command="${1:-local}"
    
    case $command in
        local)
            check_prerequisites
            setup_builder
            build_single_platform "$(uname -m | sed 's/x86_64/linux\/amd64/;s/aarch64/linux\/arm64/')" ""
            ;;
        all-local)
            check_prerequisites
            setup_builder
            build_all_local
            ;;
        multi)
            check_prerequisites
            setup_builder
            build_multi_platform
            ;;
        export)
            check_prerequisites
            setup_builder
            build_and_export
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
    
    log_info "Build process completed successfully!"
}

# Run main
main "$@"

