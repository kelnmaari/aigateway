# Build script для Windows
# PowerShell version для сборки Docker images

param(
    [Parameter(Position=0)]
    [string]$Command = "local",
    
    [string]$Version = "1.3.0",
    [string]$Registry = "ghcr.io/yourusername",
    [string]$ImageName = "ollama-openai-proxy"
)

# Colors for console output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Get build metadata
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
try {
    $GitCommit = (git rev-parse --short HEAD 2>$null)
    if (-not $GitCommit) { $GitCommit = "unknown" }
} catch {
    $GitCommit = "unknown"
}

# Check prerequisites
function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check Docker
    try {
        $dockerVersion = docker version 2>$null
        if (-not $dockerVersion) {
            Write-Error "Docker is not installed or not running"
            exit 1
        }
    } catch {
        Write-Error "Docker is not available"
        exit 1
    }
    
    # Check Docker Buildx
    try {
        $buildxVersion = docker buildx version 2>$null
        if (-not $buildxVersion) {
            Write-Error "Docker Buildx is not available"
            exit 1
        }
    } catch {
        Write-Error "Docker Buildx is not available"
        exit 1
    }
    
    Write-Info "Prerequisites OK"
}

# Setup Docker Buildx builder
function Initialize-Builder {
    Write-Info "Setting up Docker Buildx..."
    
    $builderExists = docker buildx inspect multiarch 2>$null
    
    if (-not $builderExists) {
        Write-Info "Creating new buildx builder 'multiarch'..."
        docker buildx create --name multiarch --use
    } else {
        Write-Info "Using existing builder 'multiarch'"
        docker buildx use multiarch
    }
    
    docker buildx inspect --bootstrap
}

# Build for single platform
function Build-SinglePlatform {
    param(
        [string]$Platform,
        [string]$TagSuffix = ""
    )
    
    Write-Info "Building for platform: $Platform"
    
    docker buildx build `
        --platform $Platform `
        --target production `
        --build-arg VERSION=$Version `
        --build-arg BUILD_TIME=$BuildTime `
        --build-arg GIT_COMMIT=$GitCommit `
        --tag "${Registry}/${ImageName}:${Version}${TagSuffix}" `
        --tag "${Registry}/${ImageName}:latest${TagSuffix}" `
        --file docker/Dockerfile `
        --load `
        .
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed for $Platform"
        exit 1
    }
    
    Write-Info "Build completed for $Platform"
}

# Build multi-platform and push
function Build-MultiPlatform {
    Write-Info "Building multi-platform images..."
    
    docker buildx build `
        --platform linux/amd64,linux/arm64 `
        --target production `
        --build-arg VERSION=$Version `
        --build-arg BUILD_TIME=$BuildTime `
        --build-arg GIT_COMMIT=$GitCommit `
        --tag "${Registry}/${ImageName}:${Version}" `
        --tag "${Registry}/${ImageName}:latest" `
        --file docker/Dockerfile `
        --push `
        .
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Multi-platform build failed"
        exit 1
    }
    
    Write-Info "Multi-platform build and push completed"
}

# Build all platforms locally
function Build-AllLocal {
    Write-Info "Building all platforms locally..."
    
    # Linux AMD64
    Build-SinglePlatform -Platform "linux/amd64" -TagSuffix "-linux-amd64"
    
    # Linux ARM64
    Build-SinglePlatform -Platform "linux/arm64" -TagSuffix "-linux-arm64"
    
    Write-Warn "Note: Windows containers require Windows Docker host"
    Write-Warn "To build Windows image: docker build --platform windows/amd64 ..."
    
    Write-Info "All local builds completed"
}

# Build and export to tar files
function Build-AndExport {
    Write-Info "Building and exporting images to tar files..."
    
    # Create dist directory
    if (-not (Test-Path "./dist")) {
        New-Item -ItemType Directory -Path "./dist" | Out-Null
    }
    
    # Linux AMD64
    Write-Info "Exporting Linux AMD64..."
    docker buildx build `
        --platform linux/amd64 `
        --target production `
        --build-arg VERSION=$Version `
        --build-arg BUILD_TIME=$BuildTime `
        --build-arg GIT_COMMIT=$GitCommit `
        --tag "${ImageName}:${Version}-linux-amd64" `
        --file docker/Dockerfile `
        --output type=docker,dest=./dist/${ImageName}-${Version}-linux-amd64.tar `
        .
    
    # Linux ARM64
    Write-Info "Exporting Linux ARM64..."
    docker buildx build `
        --platform linux/arm64 `
        --target production `
        --build-arg VERSION=$Version `
        --build-arg BUILD_TIME=$BuildTime `
        --build-arg GIT_COMMIT=$GitCommit `
        --tag "${ImageName}:${Version}-linux-arm64" `
        --file docker/Dockerfile `
        --output type=docker,dest=./dist/${ImageName}-${Version}-linux-arm64.tar `
        .
    
    Write-Info "Images exported to ./dist/"
    Get-ChildItem ./dist -Filter *.tar | Format-Table Name, Length -AutoSize
}

# Display usage
function Show-Usage {
    @"
Usage: .\docker\build.ps1 [COMMAND] [OPTIONS]

Commands:
    local       Build for current platform only (default)
    all-local   Build all platforms locally (linux/amd64, linux/arm64)
    multi       Build multi-platform and push to registry
    export      Build and export images to tar files
    help        Display this help message

Options:
    -Version <string>   Version tag (default: 1.3.0)
    -Registry <string>  Container registry (default: ghcr.io/yourusername)
    -ImageName <string> Image name (default: ollama-openai-proxy)

Examples:
    # Build for current platform
    .\docker\build.ps1 local

    # Build all platforms locally
    .\docker\build.ps1 all-local

    # Build and push multi-platform
    .\docker\build.ps1 multi -Version 1.3.1 -Registry docker.io/myuser

    # Export images to tar files
    .\docker\build.ps1 export

"@
}

# Main execution
function Main {
    switch ($Command.ToLower()) {
        "local" {
            Test-Prerequisites
            Initialize-Builder
            Build-SinglePlatform -Platform "linux/amd64"
        }
        "all-local" {
            Test-Prerequisites
            Initialize-Builder
            Build-AllLocal
        }
        "multi" {
            Test-Prerequisites
            Initialize-Builder
            Build-MultiPlatform
        }
        "export" {
            Test-Prerequisites
            Initialize-Builder
            Build-AndExport
        }
        "help" {
            Show-Usage
        }
        default {
            Write-Error "Unknown command: $Command"
            Show-Usage
            exit 1
        }
    }
    
    Write-Info "Build process completed successfully!"
}

# Run main
try {
    Main
} catch {
    Write-Error "An error occurred: $_"
    exit 1
}

