# Simple CRUD converter for PostgreSQL
# Converts SQLite CRUD files to PostgreSQL with placeholder replacement

$sqliteDir = "E:\golang\my_projects\internal\storage\sqlite"
$postgresDir = "E:\golang\my_projects\internal\storage\postgresql"

# Files to convert
$files = @(
    "apikeys.go",
    "conversations.go",
    "audit.go",
    "rbac.go",
    "quotas.go",
    "model_registry.go"
)

foreach ($file in $files) {
    $sourcePath = Join-Path -Path $sqliteDir -ChildPath $file
    $targetPath = Join-Path -Path $postgresDir -ChildPath $file
    
    if (-not (Test-Path $sourcePath)) {
        Write-Host "Skip: $file not found" -ForegroundColor Yellow
        continue
    }
    
    Write-Host "Converting: $file" -ForegroundColor Cyan
    
    # Read file
    $content = Get-Content -Path $sourcePath -Raw
    
    # 1. Change package
    $content = $content -replace 'package sqlite', 'package postgresql'
    
    # 2. Change receiver
    $content = $content -replace '\(s \*SQLiteDB\)', '(db *PostgreSQLDB)'
    $content = $content -replace 's\.db', 'db.db'
    $content = $content -replace 's\.logger', 'db.logger'
    
    # 3. Replace CURRENT_TIMESTAMP
    $content = $content -replace 'CURRENT_TIMESTAMP', 'NOW()'
    $content = $content -replace 'DEFAULT 1([,\s\)])', 'DEFAULT TRUE$1'
    $content = $content -replace 'DEFAULT 0([,\s\)])', 'DEFAULT FALSE$1'
    $content = $content -replace '= 1([,\s])', '= TRUE$1'
    $content = $content -replace '= 0([,\s])', '= FALSE$1'
    
    # Write file
    Set-Content -Path $targetPath -Value $content -NoNewline
    
    Write-Host "  -> Created: $targetPath" -ForegroundColor Green
}

Write-Host "`nDone! Converted $($files.Count) files." -ForegroundColor Green
Write-Host "Note: You'll need to manually convert ? to numbered placeholders." -ForegroundColor Yellow

