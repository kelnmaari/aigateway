# Cross-compilation build script для Windows
# AIGateway Platform v2.1.0

param(
    [Parameter(Position=0)]
    [string]$Command = "all",
    
    [string]$Version = ""
)

# Build configuration
if ([string]::IsNullOrEmpty($Version)) {
    if (Test-Path "VERSION") {
        $Version = Get-Content "VERSION" -Raw
        $Version = $Version.Trim()
    } else {
        $Version = "dev"
    }
}

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")
try {
    $ErrorActionPreference = "SilentlyContinue"
    $GitCommit = (git rev-parse --short HEAD 2>&1 | Out-String).Trim()
    $ErrorActionPreference = "Continue"
    if ([string]::IsNullOrWhiteSpace($GitCommit) -or $LASTEXITCODE -ne 0) { 
        $GitCommit = "unknown" 
    }
} catch {
    $GitCommit = "unknown"
}
$OutputDir = "dist"

# Colors
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-ErrorMsg {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Clean previous builds
function Invoke-Clean {
    Write-Info "Cleaning previous builds..."
    if (Test-Path $OutputDir) {
        Remove-Item -Path $OutputDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Build for specific platform
function Build-Platform {
    param(
        [string]$OS,
        [string]$Arch,
        [string]$OutputName
    )
    
    Write-Info "Building $OS/$Arch..."
    
    # Set environment
    $env:GOOS = $OS
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = "0"  # Disable CGO for cross-compilation
    $env:GOEXPERIMENT = "greenteagc"
    # Build flags with version information
    $ldflags = "-w -s"
    $ldflags += " -X 'aigateway/internal/version.Version=$Version'"
    $ldflags += " -X 'aigateway/internal/version.GitCommit=$GitCommit'"
    $ldflags += " -X 'aigateway/internal/version.BuildDate=$BuildDate'"
    
    # Build tags for SQLite
    $tags = "sqlite_fts5,sqlite_json1"
    
    # Output file
    $output = Join-Path $OutputDir $OutputName
    
    # Build
    go build `
        -buildvcs=false `
        -ldflags="$ldflags" `
        -tags $tags `
        -trimpath `
        -o $output `
        ./cmd/server
    
    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $output).Length
        $sizeKB = [math]::Round($size / 1KB, 2)
        $sizeMB = [math]::Round($size / 1MB, 2)
        if ($sizeMB -gt 1) {
            Write-Info "Built successfully: $output ($sizeMB MB)"
        } else {
            Write-Info "Built successfully: $output ($sizeKB KB)"
        }
    } else {
        Write-ErrorMsg "Failed to build $OS/$Arch"
        exit 1
    }
}

# Build all platforms
function Build-All {
    Write-Info "Starting build process..."
    Write-Info "Version: $Version"
    Write-Info "Build Date: $BuildDate"
    Write-Info "Git Commit: $GitCommit"
    Write-Host ""
    
    # Linux AMD64
    Build-Platform -OS "linux" -Arch "amd64" -OutputName "aigateway-linux-amd64"
    
    # Linux ARM64
    Build-Platform -OS "linux" -Arch "arm64" -OutputName "aigateway-linux-arm64"
    
    # Windows AMD64
    Build-Platform -OS "windows" -Arch "amd64" -OutputName "aigateway-windows-amd64.exe"
    
    Write-Host ""
    Write-Info "Build completed successfully!"
    Write-Info "Binaries are in: $OutputDir/"
    Write-Host ""
    Get-ChildItem $OutputDir | Format-Table Name, @{Name="Size (MB)";Expression={[math]::Round($_.Length / 1MB, 2)}} -AutoSize
}

# Build Linux only
function Build-Linux {
    Write-Info "Building Linux binaries..."
    Build-Platform -OS "linux" -Arch "amd64" -OutputName "aigateway-linux-amd64"
    Build-Platform -OS "linux" -Arch "arm64" -OutputName "aigateway-linux-arm64"
    Write-Info "Linux builds completed!"
}

# Build Windows only
function Build-Windows {
    Write-Info "Building Windows binary..."
    Build-Platform -OS "windows" -Arch "amd64" -OutputName "aigateway-windows-amd64.exe"
    Write-Info "Windows build completed!"
}

# Package binaries with configs
function Invoke-Package {
    Write-Info "Creating distribution packages..."
    
    $versionDir = "$OutputDir\aigateway-$Version"
    
    Get-ChildItem $OutputDir -Filter "aigateway-*" | ForEach-Object {
        $binary = $_.FullName
        $basename = $_.Name
        $platform = $basename -replace "aigateway-", "" -replace "\.exe$", ""
        
        $pkgDir = "$versionDir-$platform"
        New-Item -ItemType Directory -Path $pkgDir -Force | Out-Null
        
        # Copy binary
        Copy-Item $binary $pkgDir\
        
        # Copy configs
        Copy-Item -Path "configs" -Destination $pkgDir\ -Recurse
        Copy-Item -Path "web" -Destination $pkgDir\ -Recurse
        
        # Copy docs
        if (Test-Path "README.md") { Copy-Item "README.md" $pkgDir\ }
        if (Test-Path "LICENSE") { Copy-Item "LICENSE" $pkgDir\ }
        
        # Create archive
        $archiveName = "aigateway-$Version-$platform.zip"
        $archivePath = Join-Path $OutputDir $archiveName
        
        Compress-Archive -Path $pkgDir -DestinationPath $archivePath -Force
        Write-Info "Archive created: $archivePath"
        
        # Cleanup temp dir
        Remove-Item -Path $pkgDir -Recurse -Force
    }
    
    Write-Info "Packaging completed!"
    Write-Host ""
    Get-ChildItem $OutputDir -Filter "*.zip" | Format-Table Name, @{Name="Size (MB)";Expression={[math]::Round($_.Length / 1MB, 2)}} -AutoSize
}

# Display usage
function Show-Usage {
    @"
Usage: .\build.ps1 [COMMAND] [OPTIONS]

Commands:
    all         Build for all platforms (default)
    linux       Build for Linux only (amd64 + arm64)
    windows     Build for Windows only (amd64)
    package     Create distribution packages
    clean       Clean build artifacts
    help        Display this help message

Options:
    -Version <string>   Version tag (default: 1.3.0)

Examples:
    # Build all platforms
    .\build.ps1 all

    # Build Linux only
    .\build.ps1 linux

    # Build with custom version
    .\build.ps1 all -Version 1.3.1

    # Build and package
    .\build.ps1 all
    .\build.ps1 package

"@
}

# Check Go installation
function Test-GoInstallation {
    try {
        $goVersion = go version 2>$null
        if (-not $goVersion) {
            Write-ErrorMsg "Go is not installed. Please install Go 1.25 or later."
            exit 1
        }
    } catch {
        Write-ErrorMsg "Go is not installed. Please install Go 1.25 or later."
        exit 1
    }
}

# Main execution
function Main {
    Test-GoInstallation
    
    switch ($Command.ToLower()) {
        "all" {
            Invoke-Clean
            Build-All
        }
        "linux" {
            Invoke-Clean
            Build-Linux
        }
        "windows" {
            Invoke-Clean
            Build-Windows
        }
        "package" {
            Invoke-Package
        }
        "clean" {
            Invoke-Clean
            Write-Info "Cleaned!"
        }
        "help" {
            Show-Usage
        }
        default {
            Write-ErrorMsg "Unknown command: $Command"
            Show-Usage
            exit 1
        }
    }
}

# Run main
try {
    Main
} catch {
    Write-ErrorMsg "An error occurred: $_"
    exit 1
}

