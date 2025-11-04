# Check PostgreSQL migrations for common syntax errors

$migrationsDir = "E:\golang\my_projects\internal\storage\postgresql\migrations"
$errors = @()

Write-Host "Checking PostgreSQL migrations for syntax errors..." -ForegroundColor Cyan
Write-Host ""

Get-ChildItem "$migrationsDir\*.up.sql" | ForEach-Object {
    $file = $_
    $lineNum = 0
    
    Get-Content $file.FullName | ForEach-Object {
        $lineNum++
        $line = $_
        
        # Check 1: INTEGER DEFAULT FALSE/TRUE
        if ($line -match 'INTEGER.*DEFAULT\s+(FALSE|TRUE)') {
            $errors += [PSCustomObject]@{
                File = $file.Name
                Line = $lineNum
                Error = "INTEGER with DEFAULT FALSE/TRUE"
                Content = $line.Trim()
            }
        }
        
        # Check 2: BOOLEAN DEFAULT 0/1
        if ($line -match 'BOOLEAN.*DEFAULT\s+[01]') {
            $errors += [PSCustomObject]@{
                File = $file.Name
                Line = $lineNum
                Error = "BOOLEAN with DEFAULT 0/1"
                Content = $line.Trim()
            }
        }
        
        # Check 3: Comment without -- (after comma with text)
        if ($line -match ',\s+[a-zA-Z_]+\s*\(.*\)\s*$' -and $line -notmatch '--') {
            $errors += [PSCustomObject]@{
                File = $file.Name
                Line = $lineNum
                Error = "Possible uncommented text after comma"
                Content = $line.Trim()
            }
        }
        
        # Check 4: JSONB with invalid comment
        if ($line -match 'JSONB.*,\s+[a-zA-Z]' -and $line -notmatch '--') {
            $errors += [PSCustomObject]@{
                File = $file.Name
                Line = $lineNum
                Error = "JSONB with uncommented text"
                Content = $line.Trim()
            }
        }
        
        # Check 5: requires_gpu BOOLEAN DEFAULT 0
        if ($line -match 'requires_gpu\s+BOOLEAN\s+DEFAULT\s+0') {
            $errors += [PSCustomObject]@{
                File = $file.Name
                Line = $lineNum
                Error = "requires_gpu with DEFAULT 0 (should be FALSE)"
                Content = $line.Trim()
            }
        }
    }
}

if ($errors.Count -eq 0) {
    Write-Host "✅ No syntax errors found!" -ForegroundColor Green
} else {
    Write-Host "❌ Found $($errors.Count) potential errors:" -ForegroundColor Red
    Write-Host ""
    
    $errors | Format-Table -AutoSize
    
    Write-Host ""
    Write-Host "Summary by error type:" -ForegroundColor Yellow
    $errors | Group-Object Error | Select-Object Count, Name | Format-Table -AutoSize
}

Write-Host ""
Write-Host "Total files checked: $((Get-ChildItem "$migrationsDir\*.up.sql").Count)" -ForegroundColor Cyan

