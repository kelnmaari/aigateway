# Скрипт для автоматической замены старых pointer helpers на utils.Ptr/Value
# Usage: .\scripts\refactor_pointer_helpers.ps1

$ErrorActionPreference = "Stop"

Write-Host "🔄 Refactoring pointer helpers..." -ForegroundColor Cyan

# Файлы для рефакторинга
$files = @(
    "internal\websocket\handler.go",
    "internal\services\rag\datasource_service_test.go",
    "internal\services\audit\logger.go",
    "internal\models\mapping.go",
    "internal\converter\simple_converter.go",
    "internal\api\middleware\advanced_rate_limit.go",
    "internal\api\handlers\model_warmup.go"
)

$replacements = @{
    # Простые замены функций
    "stringPtr\(" = "utils.Ptr("
    "intPtr\(" = "utils.Ptr("
    "int64Ptr\(" = "utils.Ptr("
    "float32Ptr\(" = "utils.Ptr("
    "float64Ptr\(" = "utils.Ptr("
    "boolPtr\(" = "utils.Ptr("
    
    # Замены функций с default values
    # Требует контекста - не автоматизируется
}

foreach ($file in $files) {
    $fullPath = Join-Path $PSScriptRoot "..\$file"
    
    if (-not (Test-Path $fullPath)) {
        Write-Host "⏭️  Skip: $file (not found)" -ForegroundColor Yellow
        continue
    }
    
    Write-Host "📝 Processing: $file" -ForegroundColor White
    
    $content = Get-Content $fullPath -Raw
    $originalContent = $content
    
    # Добавляем import если его нет
    if ($content -notmatch '"aigateway/internal/utils"') {
        # Найти последний import из internal пакетов
        if ($content -match '(import \([^)]+)(aigateway/internal/[^"]+")([^)]+\))') {
            $before = $matches[1]
            $lastImport = $matches[2]
            $after = $matches[3]
            
            $content = $content -replace [regex]::Escape($matches[0]),  ($before + $lastImport + "`n`t`"aigateway/internal/utils`"" + $after)
            Write-Host "  ✓ Added utils import" -ForegroundColor Green
        }
    }
    
    # Выполняем замены
    $changeCount = 0
    foreach ($pattern in $replacements.Keys) {
        $replacement = $replacements[$pattern]
        $beforeCount = ([regex]::Matches($content, $pattern)).Count
        $content = $content -replace $pattern, $replacement
        $afterCount = ([regex]::Matches($content, $pattern)).Count
        $changed = $beforeCount - $afterCount
        
        if ($changed -gt 0) {
            Write-Host "  ✓ Replaced $changed instances of $pattern" -ForegroundColor Green
            $changeCount += $changed
        }
    }
    
    # Удаляем определения старых функций
    $oldFunctions = @(
        'func stringPtr\(s string\) \*string \{[^}]+\}',
        'func intPtr\(i int\) \*int \{[^}]+\}',
        'func int64Ptr\(i int64\) \*int64 \{[^}]+\}',
        'func float32Ptr\(f float32\) \*float32 \{[^}]+\}',
        'func float64Ptr\(f float64\) \*float64 \{[^}]+\}',
        'func boolPtr\(b bool\) \*bool \{[^}]+\}'
    )
    
    foreach ($funcPattern in $oldFunctions) {
        if ($content -match $funcPattern) {
            $content = $content -replace $funcPattern, ""
            Write-Host "  ✓ Removed old function definition" -ForegroundColor Green
            $changeCount++
        }
    }
    
    if ($content -ne $originalContent) {
        Set-Content -Path $fullPath -Value $content -NoNewline
        Write-Host "  ✅ Saved changes ($changeCount modifications)" -ForegroundColor Green
    } else {
        Write-Host "  ⏭️  No changes needed" -ForegroundColor Gray
    }
}

Write-Host "`n✅ Refactoring complete!" -ForegroundColor Green
Write-Host "Next step: Run 'go test ./...' to verify changes" -ForegroundColor Cyan

