# Fix INSERT OR REPLACE to PostgreSQL syntax

$migrationsDir = "E:\golang\my_projects\internal\storage\postgresql\migrations"
$fixed = 0

Write-Host "Converting INSERT OR REPLACE to PostgreSQL syntax..." -ForegroundColor Cyan

Get-ChildItem "$migrationsDir\*.up.sql" | ForEach-Object {
    $file = $_
    $content = Get-Content $file.FullName -Raw
    
    if ($content -match "INSERT OR REPLACE") {
        # Replace INSERT OR REPLACE with PostgreSQL syntax
        # For changelogs table with PRIMARY KEY (version)
        $newContent = $content -replace `
            'INSERT OR REPLACE INTO changelogs', `
            'INSERT INTO changelogs'
        
        # Add ON CONFLICT clause before semicolon or at end of statement
        $newContent = $newContent -replace `
            "(\s+content\s*\)\s*VALUES\s*\([^;]+\))([,;])", `
            "`$1`nON CONFLICT (version) DO UPDATE SET`n    release_date = EXCLUDED.release_date,`n    content = EXCLUDED.content`$2"
        
        if ($newContent -ne $content) {
            Set-Content -Path $file.FullName -Value $newContent -NoNewline
            Write-Host "[OK] $($file.Name)" -ForegroundColor Green
            $fixed++
        }
    }
}

Write-Host "`nTotal files fixed: $fixed" -ForegroundColor Cyan

