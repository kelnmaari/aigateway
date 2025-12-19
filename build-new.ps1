# Cross-compilation build script для Windows
# AIGateway Platform v3.x
# Supports Legacy and Svelte WebUI

param(
    [Parameter(Position=0)]
    [string]$Command = "all",
    
    [string]$Version = "",
    
    [ValidateSet("legacy", "svelte", "both")]
    [string]$WebUI = "both"
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
$SvelteDir = "web-svelte"
$SvelteBuildDir = "internal/web/svelte-build"
$LegacyDir = "web"
$LegacyEmbedDir = "internal/web/static"

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

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "=== $Message ===" -ForegroundColor Cyan
}

# Check Node.js installation
function Test-NodeInstallation {
    try {
        $nodeVersion = node --version 2>$null
        if (-not $nodeVersion) {
            return $false
        }
        return $true
    } catch {
        return $false
    }
}

# Check required tools installation
function Test-RequiredTools {
    $requiredTools = @("go", "node", "npm")
    foreach ($tool in $requiredTools) {
        if (!(Get-Command $tool -ErrorAction SilentlyContinue)) {
            Write-ErrorMsg "$tool is not installed or not in PATH"
            exit 1
        }
    }
}

# Build Svelte WebUI
function Build-SvelteUI {
    Write-Step "Building Svelte WebUI"
    
    if (-not (Test-Path $SvelteDir)) {
        Write-ErrorMsg "Svelte project not found at $SvelteDir"
        Write-Warn "Run 'npm create svelte@latest web-svelte' to create it"
        exit 1
    }
    
    if (-not (Test-NodeInstallation)) {
        Write-ErrorMsg "Node.js is not installed. Please install Node.js 18+ for Svelte build."
        exit 1
    }
    
    Push-Location $SvelteDir
    try {
        # Install dependencies if needed
        if (-not (Test-Path "node_modules")) {
            Write-Info "Installing npm dependencies..."
            npm ci
            if ($LASTEXITCODE -ne 0) {
                Write-Warn "npm ci failed, trying npm install..."
                npm install
                if ($LASTEXITCODE -ne 0) {
                    Write-ErrorMsg "Failed to install dependencies"
                    exit 1
                }
            }
        }
        
        # Build
        Write-Info "Running npm build..."
        npm run build
        if ($LASTEXITCODE -ne 0) {
            Write-ErrorMsg "Svelte build failed"
            exit 1
        }
        
        # Verify build output
        if (-not (Test-Path "build")) {
            Write-ErrorMsg "Build output not found at $SvelteDir/build"
            exit 1
        }
        
        Write-Info "Svelte build completed successfully"
    } finally {
        Pop-Location
    }
    
    # Copy to embed directory
    Write-Info "Copying Svelte build to $SvelteBuildDir..."
    
    # Remove old build
    if (Test-Path $SvelteBuildDir) {
        Remove-Item -Path $SvelteBuildDir -Recurse -Force
    }
    
    # Create directory and copy
    New-Item -ItemType Directory -Path $SvelteBuildDir -Force | Out-Null
    Copy-Item -Path "$SvelteDir/build/*" -Destination $SvelteBuildDir -Recurse
    
    $fileCount = (Get-ChildItem -Path $SvelteBuildDir -Recurse -File).Count
    $totalSize = (Get-ChildItem -Path $SvelteBuildDir -Recurse -File | Measure-Object -Property Length -Sum).Sum
    $sizeMB = [math]::Round($totalSize / 1MB, 2)
    
    Write-Info "Copied $fileCount files ($sizeMB MB) to $SvelteBuildDir"
}

# Copy Legacy WebUI to embed directory
function Copy-LegacyUI {
    Write-Step "Preparing Legacy WebUI"
    
    if (-not (Test-Path $LegacyDir)) {
        Write-ErrorMsg "Legacy web directory not found at $LegacyDir"
        exit 1
    }
    
    # Remove old copy
    if (Test-Path $LegacyEmbedDir) {
        Remove-Item -Path $LegacyEmbedDir -Recurse -Force
    }
    
    # Create directory and copy
    New-Item -ItemType Directory -Path $LegacyEmbedDir -Force | Out-Null
    Copy-Item -Path "$LegacyDir/*" -Destination $LegacyEmbedDir -Recurse
    
    $fileCount = (Get-ChildItem -Path $LegacyEmbedDir -Recurse -File).Count
    $totalSize = (Get-ChildItem -Path $LegacyEmbedDir -Recurse -File | Measure-Object -Property Length -Sum).Sum
    $sizeKB = [math]::Round($totalSize / 1KB, 2)
    
    Write-Info "Copied $fileCount files ($sizeKB KB) to $LegacyEmbedDir"
}

# Build WebUI based on selected option
function Build-WebUI {
    switch ($WebUI) {
        "svelte" {
            Build-SvelteUI
        }
        "legacy" {
            Copy-LegacyUI
        }
        "both" {
            Copy-LegacyUI
            Build-SvelteUI
        }
    }
}

# Clean previous builds
function Invoke-Clean {
    Write-Info "Cleaning previous builds..."
    if (Test-Path $OutputDir) {
        Remove-Item -Path $OutputDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Clean WebUI builds
function Invoke-CleanWebUI {
    Write-Info "Cleaning WebUI builds..."
    if (Test-Path $SvelteBuildDir) {
        Remove-Item -Path $SvelteBuildDir -Recurse -Force
        Write-Info "Removed $SvelteBuildDir"
    }
    if (Test-Path $LegacyEmbedDir) {
        Remove-Item -Path $LegacyEmbedDir -Recurse -Force
        Write-Info "Removed $LegacyEmbedDir"
    }
    if (Test-Path "$SvelteDir/build") {
        Remove-Item -Path "$SvelteDir/build" -Recurse -Force
        Write-Info "Removed $SvelteDir/build"
    }
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
    Write-Step "Starting build process"
    Write-Info "Version: $Version"
    Write-Info "Build Date: $BuildDate"
    Write-Info "Git Commit: $GitCommit"
    Write-Info "WebUI: $WebUI"
    Write-Host ""
    
    # Build WebUI first
    Build-WebUI
    
    Write-Step "Building Go binaries"
    
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
    Build-WebUI
    Build-Platform -OS "linux" -Arch "amd64" -OutputName "aigateway-linux-amd64"
    Build-Platform -OS "linux" -Arch "arm64" -OutputName "aigateway-linux-arm64"
    Write-Info "Linux builds completed!"
}

# Build Windows only
function Build-Windows {
    Write-Info "Building Windows binary..."
    Build-WebUI
    Build-Platform -OS "windows" -Arch "amd64" -OutputName "aigateway-windows-amd64.exe"
    Write-Info "Windows build completed!"
}

# Build only Svelte UI (no Go build)
function Build-FrontendOnly {
    Write-Step "Building Svelte frontend only"
    Build-SvelteUI
    Write-Info "Frontend build completed!"
    Write-Info "Output: $SvelteBuildDir"
}

# Package binaries with configs
function Invoke-Package {
    Write-Info "Creating distribution packages..."
    
    $versionDir = "$OutputDir\aigateway-$Version"
    
    Get-ChildItem $OutputDir -Filter "aigateway-*" | Where-Object { -not $_.Name.EndsWith(".zip") } | ForEach-Object {
        $binary = $_.FullName
        $basename = $_.Name
        $platform = $basename -replace "aigateway-", "" -replace "\.exe$", ""
        
        $pkgDir = "$versionDir-$platform"
        New-Item -ItemType Directory -Path $pkgDir -Force | Out-Null
        
        # Copy binary
        Copy-Item $binary $pkgDir\
        
        # Copy configs
        Copy-Item -Path "configs" -Destination $pkgDir\ -Recurse
        
        # Copy legacy web (for external serving if needed)
        if (Test-Path "web") {
            Copy-Item -Path "web" -Destination $pkgDir\ -Recurse
        }
        
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
Usage: .\build-new.ps1 [COMMAND] [OPTIONS]

Commands:
    all           Build for all platforms (default)
    linux         Build for Linux only (amd64 + arm64)
    windows       Build for Windows only (amd64)
    frontend      Build Svelte frontend only (no Go build)
    package       Create distribution packages
    clean         Clean build artifacts
    clean-webui   Clean WebUI build artifacts
    help          Display this help message

Options:
    -Version <string>   Version tag (default: from VERSION file)
    -WebUI <string>     WebUI to include: legacy, svelte, or both (default: both)

Examples:
    # Build all platforms with both UIs
    .\build-new.ps1 all

    # Build with only Svelte UI
    .\build-new.ps1 all -WebUI svelte

    # Build with only Legacy UI
    .\build-new.ps1 all -WebUI legacy

    # Build only the Svelte frontend (no Go compilation)
    .\build-new.ps1 frontend

    # Build Linux only
    .\build-new.ps1 linux

    # Build with custom version
    .\build-new.ps1 all -Version 3.1.0

    # Build and package
    .\build-new.ps1 all
    .\build-new.ps1 package

    # Clean everything
    .\build-new.ps1 clean
    .\build-new.ps1 clean-webui

WebUI Modes:
    legacy    - Only HTML/JS frontend (smaller binary)
    svelte    - Only Svelte frontend (modern UI)
    both      - Both frontends embedded (switchable via --webui-version flag)

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
    switch ($Command.ToLower()) {
        "all" {
            Test-RequiredTools
            Test-GoInstallation
            Invoke-Clean
            Build-All
        }
        "linux" {
            Test-RequiredTools
            Test-GoInstallation
            Invoke-Clean
            Build-Linux
        }
        "windows" {
            Test-RequiredTools
            Test-GoInstallation
            Invoke-Clean
            Build-Windows
        }
        "frontend" {
            Test-RequiredTools
            Build-FrontendOnly
        }
        "package" {
            Invoke-Package
        }
        "clean" {
            Invoke-Clean
            Write-Info "Cleaned dist/"
        }
        "clean-webui" {
            Invoke-CleanWebUI
            Write-Info "Cleaned WebUI builds!"
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