#!/bin/bash
# Package script - создание distribution пакетов
# Ollama-OpenAI Proxy v1.3.0

set -e

VERSION=${1:-1.3.0}
OUTPUT_DIR=${2:-dist}

echo "Creating distribution packages for version $VERSION..."

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Package each binary
for binary in "$OUTPUT_DIR"/ollama-proxy-*; do
    if [ ! -f "$binary" ]; then
        continue
    fi
    
    basename=$(basename "$binary")
    platform="${basename#ollama-proxy-}"
    platform="${platform%.*}"  # Remove .exe if exists
    
    echo "Packaging $platform..."
    
    pkg_name="ollama-proxy-$VERSION-$platform"
    pkg_dir="$TEMP_DIR/$pkg_name"
    
    # Create package directory
    mkdir -p "$pkg_dir"
    
    # Copy binary
    cp "$binary" "$pkg_dir/"
    chmod +x "$pkg_dir/$(basename "$binary")" 2>/dev/null || true
    
    # Copy configs
    cp -r configs "$pkg_dir/"
    cp -r web "$pkg_dir/"
    
    # Copy documentation
    cp README.md "$pkg_dir/" 2>/dev/null || echo "# Ollama-OpenAI Proxy $VERSION" > "$pkg_dir/README.md"
    cp LICENSE "$pkg_dir/" 2>/dev/null || true
    
    # Create start script
    if [[ "$platform" == *"windows"* ]]; then
        cat > "$pkg_dir/start.bat" << 'EOF'
@echo off
echo Starting Ollama-OpenAI Proxy...
ollama-proxy-windows-amd64.exe
EOF
        
        # Create zip archive
        (cd "$TEMP_DIR" && zip -r "$pkg_name.zip" "$pkg_name")
        mv "$TEMP_DIR/$pkg_name.zip" "$OUTPUT_DIR/"
        echo "✓ Created: $OUTPUT_DIR/$pkg_name.zip"
    else
        cat > "$pkg_dir/start.sh" << 'EOF'
#!/bin/bash
echo "Starting Ollama-OpenAI Proxy..."
./ollama-proxy-*
EOF
        chmod +x "$pkg_dir/start.sh"
        
        # Create tar.gz archive
        tar -czf "$OUTPUT_DIR/$pkg_name.tar.gz" -C "$TEMP_DIR" "$pkg_name"
        echo "✓ Created: $OUTPUT_DIR/$pkg_name.tar.gz"
    fi
done

echo "Packaging completed!"

