# ═══════════════════════════════════════════════════════════════════════════
# RAG Infrastructure Quick Start Script (PowerShell)
# ═══════════════════════════════════════════════════════════════════════════

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$ComposeFile = Join-Path $ProjectRoot "docker-compose.rag-infrastructure.yml"
$EnvFile = Join-Path $ProjectRoot ".env"
$EnvExample = Join-Path $ScriptDir ".env.rag.example"

# Helper functions
function Print-Header {
    param([string]$Text)
    Write-Host "═══════════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
    Write-Host " $Text" -ForegroundColor Blue
    Write-Host "═══════════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
}

function Print-Success {
    param([string]$Text)
    Write-Host "✓ " -ForegroundColor Green -NoNewline
    Write-Host $Text
}

function Print-Warning {
    param([string]$Text)
    Write-Host "⚠ " -ForegroundColor Yellow -NoNewline
    Write-Host $Text
}

function Print-Error {
    param([string]$Text)
    Write-Host "✗ " -ForegroundColor Red -NoNewline
    Write-Host $Text
}

# Check Docker
function Check-Docker {
    try {
        $null = docker --version
        $null = docker compose version
        Print-Success "Docker установлен"
        return $true
    }
    catch {
        Print-Error "Docker не установлен или недоступен"
        Print-Error "Установите Docker Desktop: https://docs.docker.com/desktop/install/windows-install/"
        return $false
    }
}

# Check .env file
function Check-Env {
    if (-not (Test-Path $EnvFile)) {
        Print-Warning ".env файл не найден. Копирую из примера..."
        Copy-Item $EnvExample $EnvFile
        Print-Success ".env файл создан"
        Print-Warning "ВАЖНО: Отредактируйте $EnvFile и измените пароли!"
        Write-Host ""
        Read-Host "Нажмите Enter чтобы продолжить"
    }
    else {
        Print-Success ".env файл существует"
    }
}

# Show menu
function Show-Menu {
    Print-Header "RAG Infrastructure Quick Start"
    Write-Host ""
    Write-Host "Выберите профиль запуска:"
    Write-Host ""
    Write-Host "  1) Базовая конфигурация (PostgreSQL + Redis)"
    Write-Host "  2) + Qdrant (vector database)"
    Write-Host "  3) + MinIO (S3 storage)"
    Write-Host "  4) + Мониторинг (Prometheus + Grafana)"
    Write-Host "  5) Полный стек (все компоненты)"
    Write-Host ""
    Write-Host "  6) Остановить все"
    Write-Host "  7) Очистить данные (volumes)"
    Write-Host "  8) Показать статус"
    Write-Host "  9) Показать логи"
    Write-Host "  H) Health checks"
    Write-Host ""
    Write-Host "  0) Выход"
    Write-Host ""
}

# Start services
function Start-Services {
    param([string]$Profile)
    
    Print-Header "Запуск сервисов: $Profile"
    
    if ($Profile -eq "basic") {
        docker compose -f $ComposeFile up -d
    }
    else {
        docker compose -f $ComposeFile --profile $Profile up -d
    }
    
    Write-Host ""
    Print-Success "Сервисы запущены"
    Show-Urls
}

# Show URLs
function Show-Urls {
    Write-Host ""
    Print-Header "Доступные сервисы"
    
    $runningContainers = docker ps --format '{{.Names}}'
    
    if ($runningContainers -match 'rag-postgres') {
        Write-Host "PostgreSQL:    " -ForegroundColor Green -NoNewline
        Write-Host "postgres://proxy_user:***@localhost:5432/ollama_proxy"
    }
    
    if ($runningContainers -match 'rag-pgadmin') {
        Write-Host "pgAdmin:       " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:5050 (admin@admin.com)"
    }
    
    if ($runningContainers -match 'rag-redis') {
        Write-Host "Redis:         " -ForegroundColor Green -NoNewline
        Write-Host "redis://localhost:6379/0"
    }
    
    if ($runningContainers -match 'rag-qdrant') {
        Write-Host "Qdrant:        " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:6333"
        Write-Host "Qdrant UI:     " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:6333/dashboard"
    }
    
    if ($runningContainers -match 'rag-minio') {
        Write-Host "MinIO API:     " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:9000"
        Write-Host "MinIO Console: " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:9001"
    }
    
    if ($runningContainers -match 'rag-prometheus') {
        Write-Host "Prometheus:    " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:9090"
    }
    
    if ($runningContainers -match 'rag-grafana') {
        Write-Host "Grafana:       " -ForegroundColor Green -NoNewline
        Write-Host "http://localhost:3000 (admin/admin123)"
    }
    
    Write-Host ""
}

# Stop services
function Stop-Services {
    Print-Header "Остановка сервисов"
    docker compose -f $ComposeFile down
    Print-Success "Все сервисы остановлены"
}

# Clean volumes
function Clean-Volumes {
    Print-Warning "Это удалит ВСЕ данные (volumes)!"
    $confirm = Read-Host "Продолжить? (yes/no)"
    
    if ($confirm -eq "yes") {
        Print-Header "Очистка данных"
        docker compose -f $ComposeFile down -v
        Print-Success "Все volumes удалены"
    }
    else {
        Print-Warning "Отменено"
    }
}

# Show status
function Show-Status {
    Print-Header "Статус сервисов"
    docker compose -f $ComposeFile ps
    Write-Host ""
    
    Print-Header "Использование volumes"
    docker volume ls | Select-String "rag_"
}

# Show logs
function Show-Logs {
    Write-Host "Выберите сервис:"
    Write-Host "  1) PostgreSQL"
    Write-Host "  2) Redis"
    Write-Host "  3) Qdrant"
    Write-Host "  4) MinIO"
    Write-Host "  5) Prometheus"
    Write-Host "  6) Grafana"
    Write-Host "  7) Все сервисы"
    Write-Host ""
    
    $choice = Read-Host "Ваш выбор"
    
    switch ($choice) {
        "1" { docker logs -f rag-postgres }
        "2" { docker logs -f rag-redis }
        "3" { docker logs -f rag-qdrant }
        "4" { docker logs -f rag-minio }
        "5" { docker logs -f rag-prometheus }
        "6" { docker logs -f rag-grafana }
        "7" { docker compose -f $ComposeFile logs -f }
        default { Print-Error "Неверный выбор" }
    }
}

# Health checks
function Health-Checks {
    Print-Header "Проверка здоровья сервисов"
    
    # PostgreSQL
    try {
        $null = docker exec rag-postgres pg_isready -U proxy_user -d ollama_proxy 2>&1
        Print-Success "PostgreSQL: OK"
    }
    catch {
        Print-Error "PostgreSQL: FAIL"
    }
    
    # Redis
    try {
        $ping = docker exec rag-redis redis-cli ping 2>&1
        if ($ping -match "PONG") {
            Print-Success "Redis: OK"
        }
        else {
            Print-Error "Redis: FAIL"
        }
    }
    catch {
        Print-Error "Redis: FAIL"
    }
    
    # Qdrant
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:6333/healthz" -UseBasicParsing -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            Print-Success "Qdrant: OK"
        }
        else {
            Print-Warning "Qdrant: не запущен или недоступен"
        }
    }
    catch {
        Print-Warning "Qdrant: не запущен или недоступен"
    }
    
    # MinIO
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:9000/minio/health/live" -UseBasicParsing -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            Print-Success "MinIO: OK"
        }
        else {
            Print-Warning "MinIO: не запущен или недоступен"
        }
    }
    catch {
        Print-Warning "MinIO: не запущен или недоступен"
    }
    
    Write-Host ""
}

# Main
function Main {
    if (-not (Check-Docker)) {
        exit 1
    }
    
    Check-Env
    
    while ($true) {
        Clear-Host
        Show-Menu
        
        $choice = Read-Host "Ваш выбор"
        Write-Host ""
        
        switch ($choice) {
            "1" { Start-Services "basic" }
            "2" { Start-Services "qdrant" }
            "3" { Start-Services "minio" }
            "4" { Start-Services "monitoring" }
            "5" { Start-Services "full" }
            "6" { Stop-Services }
            "7" { Clean-Volumes }
            "8" { Show-Status }
            "9" { Show-Logs }
            "H" { Health-Checks }
            "h" { Health-Checks }
            "0" {
                Print-Success "Выход"
                exit 0
            }
            default { Print-Error "Неверный выбор" }
        }
        
        Write-Host ""
        Read-Host "Нажмите Enter чтобы продолжить"
    }
}

# Run
Main

