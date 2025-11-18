#!/usr/bin/env pwsh
# Add AGGateway UI Framework to all HTML pages

$webDir = "E:\golang\my_projects\web"
$htmlFiles = Get-ChildItem -Path $webDir -Filter "*.html" -Recurse

$frameworkTags = @"
    <!-- AIGateway UI Framework (v3.1.0) -->
    <script src="/framework/framework.js"></script>
    <link rel="stylesheet" href="/framework/framework.css">
"@

$pagesUpdated = 0
$pagesSkipped = 0

foreach ($file in $htmlFiles) {
    Write-Host "Processing: $($file.Name)" -ForegroundColor Cyan
    
    $content = Get-Content -Path $file.FullName -Raw
    
    # Skip if already has framework
    if ($content -match "framework\.js") {
        Write-Host "  ✓ Already has framework, skipping" -ForegroundColor Yellow
        $pagesSkipped++
        continue
    }
    
    # Find </body> tag
    if ($content -match "</body>") {
        # Insert framework before </body>
        $newContent = $content -replace "</body>", "$frameworkTags`n</body>"
        
        # Write back
        Set-Content -Path $file.FullName -Value $newContent -NoNewline
        Write-Host "  ✓ Updated" -ForegroundColor Green
        $pagesUpdated++
    } else {
        Write-Host "  ✗ No </body> tag found" -ForegroundColor Red
        $pagesSkipped++
    }
}

Write-Host "`n=====================================" -ForegroundColor Magenta
Write-Host "✅ Framework integration complete!" -ForegroundColor Green
Write-Host "   Pages updated: $pagesUpdated" -ForegroundColor Green
Write-Host "   Pages skipped: $pagesSkipped" -ForegroundColor Yellow
Write-Host "=====================================" -ForegroundColor Magenta

