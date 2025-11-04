# GC Benchmarking Script for Go 1.25 Standard vs GreenTea GC
#
# Usage:
#   .\scripts\benchmark_gc.ps1
#
# Output:
#   benchmarks/results/standard_gc.txt
#   benchmarks/results/greentea_gc.txt
#   benchmarks/results/comparison.txt (if benchstat available)

$ErrorActionPreference = "Stop"

Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Go 1.25 GC Benchmarking: Standard vs GreenTea" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

# Check Go version
$goVersion = go version
Write-Host "Go Version: $goVersion" -ForegroundColor Green

if ($goVersion -notmatch "go1\.25") {
    Write-Host "Warning: Go 1.25+ required for GreenTea GC" -ForegroundColor Yellow
}

# Create results directory
$resultsDir = "benchmarks\results"
if (!(Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$standardFile = "$resultsDir\standard_gc_$timestamp.txt"
$greenteaFile = "$resultsDir\greentea_gc_$timestamp.txt"

# Benchmark parameters
$benchTime = "10s"
$benchCount = 3

Write-Host "Benchmark Parameters:" -ForegroundColor Yellow
Write-Host "  Time per benchmark: $benchTime"
Write-Host "  Count: $benchCount"
Write-Host "  Output: $resultsDir"
Write-Host ""

# Run Standard GC Benchmarks
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Running Standard GC Benchmarks..." -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

go test -bench=BenchmarkGC_ -benchmem -benchtime=$benchTime -count=$benchCount ./benchmarks | Tee-Object -FilePath $standardFile

Write-Host ""
Write-Host "✅ Standard GC results saved to: $standardFile" -ForegroundColor Green
Write-Host ""

# Run GreenTea GC Benchmarks
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Running GreenTea GC Benchmarks (GOEXPERIMENT=greenteagc)..." -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$env:GOEXPERIMENT = "greenteagc"
go test -bench=BenchmarkGC_ -benchmem -benchtime=$benchTime -count=$benchCount ./benchmarks | Tee-Object -FilePath $greenteaFile
Remove-Item env:GOEXPERIMENT

Write-Host ""
Write-Host "✅ GreenTea GC results saved to: $greenteaFile" -ForegroundColor Green
Write-Host ""

# Try to run benchstat comparison
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Generating Comparison (benchstat)..." -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

$benchstatInstalled = Get-Command benchstat -ErrorAction SilentlyContinue

if ($benchstatInstalled) {
    $comparisonFile = "$resultsDir\comparison_$timestamp.txt"
    benchstat $standardFile $greenteaFile | Tee-Object -FilePath $comparisonFile

    Write-Host ""
    Write-Host "✅ Comparison saved to: $comparisonFile" -ForegroundColor Green
} else {
    Write-Host "⚠️  benchstat not installed. Install with:" -ForegroundColor Yellow
    Write-Host "    go install golang.org/x/perf/cmd/benchstat@latest" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Manual comparison:" -ForegroundColor Yellow
    Write-Host "    benchstat $standardFile $greenteaFile" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Benchmarking Complete!" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "Results:" -ForegroundColor Green
Write-Host "  Standard GC: $standardFile"
Write-Host "  GreenTea GC: $greenteaFile"
if ($benchstatInstalled) {
    Write-Host "  Comparison:  $comparisonFile"
}
Write-Host ""
Write-Host "Key Metrics to Compare:" -ForegroundColor Yellow
Write-Host "  • ns/op       - Nanoseconds per operation (lower is better)"
Write-Host "  • B/op        - Bytes allocated per operation (lower is better)"
Write-Host "  • allocs/op   - Number of allocations per operation (lower is better)"
Write-Host "  • GC pause    - GC pause time (check test output)"
Write-Host ""

