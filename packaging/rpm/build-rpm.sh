#!/bin/bash
set -e

# Build RPM for ollama-openai-proxy
# Usage: ./build-rpm.sh <version> <binary_path>

VERSION=${1:-$(cat VERSION 2>/dev/null || echo "0.0.0")}
BINARY_PATH=${2:-"bin/server"}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Building RPM for version: $VERSION"
echo "Binary path: $BINARY_PATH"

# Create RPM build directories
RPM_BUILD_DIR="$PROJECT_ROOT/rpmbuild"
mkdir -p "$RPM_BUILD_DIR"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Copy spec file
cp "$SCRIPT_DIR/ollama-openai-proxy.spec" "$RPM_BUILD_DIR/SPECS/"

# Copy sources
cp "$PROJECT_ROOT/$BINARY_PATH" "$RPM_BUILD_DIR/SOURCES/server"
cp "$SCRIPT_DIR/oop.service" "$RPM_BUILD_DIR/SOURCES/"

# Build RPM
export VERSION
rpmbuild \
    --define "_topdir $RPM_BUILD_DIR" \
    --define "version $VERSION" \
    -bb "$RPM_BUILD_DIR/SPECS/ollama-openai-proxy.spec"

# Find and report built RPM
RPM_FILE=$(find "$RPM_BUILD_DIR/RPMS" -name "*.rpm" -type f | head -1)
if [ -n "$RPM_FILE" ]; then
    echo ""
    echo "========================================"
    echo "RPM built successfully!"
    echo "File: $RPM_FILE"
    echo "Size: $(du -h "$RPM_FILE" | cut -f1)"
    echo "========================================"
    
    # Copy to project root for easy access
    cp "$RPM_FILE" "$PROJECT_ROOT/"
    echo "Copied to: $PROJECT_ROOT/$(basename "$RPM_FILE")"
else
    echo "ERROR: RPM build failed!"
    exit 1
fi

