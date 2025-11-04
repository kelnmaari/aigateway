# Script to convert SQLite CRUD files to PostgreSQL syntax
# Converts ? placeholders to $1, $2, $3... and package names

$sqliteDir = "E:\golang\my_projects\internal\storage\sqlite"
$postgresDir = "E:\golang\my_projects\internal\storage\postgresql"

# Files to convert (exclude already converted and special files)
$filesToConvert = @(
    "apikeys.go",
    "audit.go",
    "changelogs.go",
    "conversations.go",
    "files.go",
    "invitations.go",
    "mcp.go",
    "model_configs.go",
    "model_registry.go",
    "quotas.go",
    "rbac.go",
    "usage.go"
)

function ConvertPlaceholders {
    param (
        [string]$content
    )
    
    # Convert placeholders ? to $1, $2, $3... in SQL queries
    # This regex finds SQL query strings and replaces ? with numbered placeholders
    
    $counter = 0
    $result = $content
    
    # Split by backticks to find SQL strings
    $parts = $result -split '`'
    $newParts = @()
    
    for ($i = 0; $i -lt $parts.Count; $i++) {
        if ($i % 2 -eq 1) {
            # Inside backticks - this is SQL code
            $sqlPart = $parts[$i]
            $placeholderCount = 1
            
            # Replace ? with numbered placeholders
            while ($sqlPart -match '\?') {
                $sqlPart = $sqlPart -replace '\?', "`$$placeholderCount", 1
                $placeholderCount++
            }
            
            $newParts += $sqlPart
        } else {
            $newParts += $parts[$i]
        }
    }
    
    $result = $newParts -join '`'
    
    return $result
}

foreach ($file in $filesToConvert) {
    $sourcePath = Join-Path -Path $sqliteDir -ChildPath $file
    $targetPath = Join-Path -Path $postgresDir -ChildPath $file
    
    if (-not (Test-Path $sourcePath)) {
        Write-Host "Skipping $file - source not found" -ForegroundColor Yellow
        continue
    }
    
    Write-Host "Converting $file..." -ForegroundColor Cyan
    
    # Read source file
    $content = Get-Content -Path $sourcePath -Raw
    
    # Replace package name
    $content = $content -replace 'package sqlite', 'package postgresql'
    
    # Replace struct receiver
    $content = $content -replace '\(s \*SQLiteDB\)', '(db *PostgreSQLDB)'
    $content = $content -replace '\bs\.db\b', 'db.db'
    $content = $content -replace '\bs\.logger\b', 'db.logger'
    
    # Convert ? placeholders to $1, $2, $3...
    $content = ConvertPlaceholders -content $content
    
    # Replace SQLite-specific functions
    $content = $content -replace 'CURRENT_TIMESTAMP', 'NOW()'
    $content = $content -replace '\bis_active = 1\b', 'is_active = TRUE'
    $content = $content -replace '\bis_active = 0\b', 'is_active = FALSE'
    $content = $content -replace '\bWHERE is_active = 1\b', 'WHERE is_active = TRUE'
    
    # Write to target file
    Set-Content -Path $targetPath -Value $content -NoNewline
    
    Write-Host "  Created: $targetPath" -ForegroundColor Green
}

Write-Host "`nConversion complete! Created $($filesToConvert.Count) CRUD files." -ForegroundColor Cyan
Write-Host "PostgreSQL storage directory: $postgresDir" -ForegroundColor Cyan

