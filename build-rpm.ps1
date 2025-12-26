<#
.SYNOPSIS
    Build RPM package for Rocky Linux

.DESCRIPTION
    Builds the Go binary and packages it as RPM using Docker

.PARAMETER SkipBuild
    Skip Go build step (use existing binary)

.EXAMPLE
    .\build-rpm.ps1
    
.EXAMPLE
    .\build-rpm.ps1 -SkipBuild
#>

param(
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$ProjectRoot = $PSScriptRoot
$Version = Get-Content "$ProjectRoot\VERSION" -ErrorAction SilentlyContinue
if (-not $Version) { $Version = "0.0.0" }
$Version = $Version.Trim()

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Building RPM for AIGateway v$Version" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Step 1: Build binary for Linux
if (-not $SkipBuild) {
    Write-Host "`n[1/3] Building Linux binary..." -ForegroundColor Yellow
    
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    
    $BuildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
    $GitCommit = git rev-parse --short HEAD 2>$null
    if (-not $GitCommit) { $GitCommit = "unknown" }
    
    $LDFlags = "-X main.Version=$Version -X main.BuildTime=$BuildTime -X main.GitCommit=$GitCommit -s -w"
    
    go build -ldflags $LDFlags -o "$ProjectRoot\bin\server-linux" cmd/server/main.go
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build binary"
        exit 1
    }
    
    # Reset environment
    $env:GOOS = ""
    $env:GOARCH = ""
    
    Write-Host "  Binary built: bin/server-linux" -ForegroundColor Green
} else {
    Write-Host "`n[1/3] Skipping build (using existing binary)" -ForegroundColor Yellow
    
    if (-not (Test-Path "$ProjectRoot\bin\server-linux")) {
        Write-Error "Binary not found: bin/server-linux. Run without -SkipBuild first."
        exit 1
    }
}

# Step 2: Build RPM in Docker
Write-Host "`n[2/3] Building RPM in Rocky Linux container..." -ForegroundColor Yellow

# Create temp directory for Docker context
$TempDir = "$ProjectRoot\rpmbuild-temp"
if (Test-Path $TempDir) { Remove-Item -Recurse -Force $TempDir }
New-Item -ItemType Directory -Path $TempDir | Out-Null

# Copy required files
Copy-Item "$ProjectRoot\bin\server-linux" "$TempDir\server"
Copy-Item "$ProjectRoot\packaging\rpm\oop.service" "$TempDir\"
Copy-Item "$ProjectRoot\packaging\rpm\ollama-openai-proxy.spec" "$TempDir\"
Copy-Item "$ProjectRoot\VERSION" "$TempDir\"

# Create Dockerfile for RPM build
$Dockerfile = @"
FROM rockylinux:9

RUN dnf install -y rpm-build rpmdevtools && \
    rpmdev-setuptree

WORKDIR /build

COPY server /root/rpmbuild/SOURCES/server
COPY oop.service /root/rpmbuild/SOURCES/
COPY ollama-openai-proxy.spec /root/rpmbuild/SPECS/

ARG VERSION
RUN rpmbuild --define "version \$VERSION" -bb /root/rpmbuild/SPECS/ollama-openai-proxy.spec

CMD cp /root/rpmbuild/RPMS/*/*.rpm /output/
"@

$Dockerfile | Out-File -FilePath "$TempDir\Dockerfile" -Encoding utf8

# Build Docker image
docker build -t aigateway-rpm-builder --build-arg VERSION=$Version "$TempDir"
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to build Docker image"
    exit 1
}

# Create output directory
$DistDir = "$ProjectRoot\dist"
if (-not (Test-Path $DistDir)) { New-Item -ItemType Directory -Path $DistDir | Out-Null }

# Run container and extract RPM
docker run --rm -v "${DistDir}:/output" aigateway-rpm-builder
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to extract RPM from container"
    exit 1
}

# Cleanup
Remove-Item -Recurse -Force $TempDir
docker rmi aigateway-rpm-builder 2>$null

# Step 3: Report results
Write-Host "`n[3/3] Build complete!" -ForegroundColor Yellow

$RpmFile = Get-ChildItem "$DistDir\*.rpm" | Select-Object -First 1
if ($RpmFile) {
    $Size = [math]::Round($RpmFile.Length / 1MB, 2)
    
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host " RPM Package Ready!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  File: $($RpmFile.FullName)" -ForegroundColor White
    Write-Host "  Size: ${Size} MB" -ForegroundColor White
    Write-Host ""
    Write-Host "  Installation on Rocky Linux:" -ForegroundColor Cyan
    Write-Host "    scp $($RpmFile.Name) root@server:/tmp/" -ForegroundColor Gray
    Write-Host "    ssh root@server 'rpm -ivh /tmp/$($RpmFile.Name)'" -ForegroundColor Gray
    Write-Host ""
} else {
    Write-Error "RPM file not found in dist/"
    exit 1
}

