# Script to convert SQLite migrations to PostgreSQL syntax
# Converts BOOLEAN values, TIMESTAMP defaults, and JSON TEXT to JSONB

$sqliteDir = "E:\golang\my_projects\internal\storage\sqlite\migrations"
$postgresDir = "E:\golang\my_projects\internal\storage\postgresql\migrations"

# Ensure target directory exists
New-Item -ItemType Directory -Force -Path $postgresDir | Out-Null

# Get all .up.sql and .down.sql files
$files = Get-ChildItem -Path $sqliteDir -Filter "*.sql"

foreach ($file in $files) {
    Write-Host "Converting $($file.Name)..."
    
    $content = Get-Content -Path $file.FullName -Raw
    
    # Convert BOOLEAN defaults: 0 → FALSE, 1 → TRUE
    $content = $content -replace 'DEFAULT 0([,\s])', 'DEFAULT FALSE$1'
    $content = $content -replace 'DEFAULT 1([,\s])', 'DEFAULT TRUE$1'
    
    # Convert TIMESTAMP defaults: CURRENT_TIMESTAMP → NOW()
    $content = $content -replace 'DEFAULT CURRENT_TIMESTAMP', 'DEFAULT NOW()'
    
    # Convert JSON TEXT fields to JSONB
    # Match patterns like: preferences TEXT NOT NULL DEFAULT '{}' -- JSON
    $content = $content -replace "(\w+)\s+TEXT\s+NOT NULL\s+DEFAULT\s+'(\{.*?\})'(,?\s*)-- JSON", '$1 JSONB NOT NULL DEFAULT ''$2''$3'
    $content = $content -replace "(\w+)\s+TEXT(,?\s*)-- JSON", '$1 JSONB$2'
    
    # Convert TEXT to JSONB for known JSON fields
    $content = $content -replace 'preferences TEXT NOT NULL DEFAULT', 'preferences JSONB NOT NULL DEFAULT'
    $content = $content -replace 'metadata TEXT([,\s])', 'metadata JSONB$1'
    $content = $content -replace 'settings TEXT NOT NULL DEFAULT', 'settings JSONB NOT NULL DEFAULT'
    $content = $content -replace 'models TEXT NOT NULL DEFAULT', 'models JSONB NOT NULL DEFAULT'
    $content = $content -replace 'permissions TEXT NOT NULL DEFAULT', 'permissions JSONB NOT NULL DEFAULT'
    $content = $content -replace 'rate_limits TEXT NOT NULL DEFAULT', 'rate_limits JSONB NOT NULL DEFAULT'
    $content = $content -replace 'usage TEXT NOT NULL DEFAULT', 'usage JSONB NOT NULL DEFAULT'
    $content = $content -replace 'tool_calls TEXT([,\s])', 'tool_calls JSONB$1'
    $content = $content -replace 'tags TEXT([,\s])', 'tags JSONB$1'
    $content = $content -replace 'capabilities TEXT NOT NULL DEFAULT', 'capabilities JSONB NOT NULL DEFAULT'
    $content = $content -replace 'credentials TEXT([,\s])', 'credentials JSONB$1'
    $content = $content -replace 'config TEXT([,\s])', 'config JSONB$1'
    $content = $content -replace 'performance_metrics TEXT([,\s])', 'performance_metrics JSONB$1'
    
    # Write converted content to PostgreSQL migrations directory
    $targetFile = Join-Path -Path $postgresDir -ChildPath $file.Name
    Set-Content -Path $targetFile -Value $content -NoNewline
    
    Write-Host "  Created: $targetFile" -ForegroundColor Green
}

Write-Host "`nConversion complete! Created $($files.Count) migration files." -ForegroundColor Cyan
Write-Host "PostgreSQL migrations directory: $postgresDir" -ForegroundColor Cyan

