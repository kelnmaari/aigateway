# ═══════════════════════════════════════════════════════════════════════════════
# RAG Infrastructure - Optional Components Deployment Script (PowerShell)
# ═══════════════════════════════════════════════════════════════════════════════
#
# Этот скрипт устанавливает опциональные компоненты RAG infrastructure:
#   - pgAdmin (PostgreSQL web UI)
#   - MinIO (S3-совместимое хранилище)
#   - Qdrant (альтернативный vector database)
#   - Grafana (визуализация метрик, подключается к Prometheus в lens-metrics)
#
# Использование:
#   .\deploy-optional.ps1 [all|pgadmin|minio|qdrant|grafana|status]
#
# ═══════════════════════════════════════════════════════════════════════════════

param(
    [Parameter(Position=0)]
    [string]$Component = "all"
)

$ErrorActionPreference = "Stop"

$NAMESPACE = "rag-infrastructure"
$SCRIPT_DIR = $PSScriptRoot

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Check-Namespace {
    $namespaceExists = kubectl get namespace $NAMESPACE 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Error-Custom "Namespace $NAMESPACE does not exist. Please deploy core components first."
        exit 1
    }
}

function Deploy-PgAdmin {
    Write-Info "Deploying pgAdmin..."
    kubectl apply -f "$SCRIPT_DIR\pgadmin-deployment.yaml"
    kubectl apply -f "$SCRIPT_DIR\pgadmin-ingress.yaml"
    
    Write-Info "Waiting for pgAdmin to be ready..."
    kubectl wait --for=condition=available --timeout=120s deployment/pgadmin -n $NAMESPACE
    
    Write-Info "✅ pgAdmin deployed successfully"
    Write-Info "Access via: kubectl port-forward svc/pgadmin 5050:80 -n $NAMESPACE"
    Write-Info "Or add '192.168.1.101 pgadmin.local' to C:\Windows\System32\drivers\etc\hosts"
    Write-Info "Login: admin@admin.com / admin123"
}

function Deploy-MinIO {
    Write-Info "Deploying MinIO..."
    kubectl apply -f "$SCRIPT_DIR\minio-statefulset.yaml"
    
    Write-Info "Waiting for MinIO to be ready..."
    kubectl wait --for=condition=ready --timeout=120s pod/minio-0 -n $NAMESPACE
    
    Write-Info "Creating MinIO buckets..."
    $jobComplete = kubectl wait --for=condition=complete --timeout=60s job/minio-setup-buckets -n $NAMESPACE 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "Bucket creation job timed out (may be still running)"
    }
    
    Write-Info "✅ MinIO deployed successfully"
    Write-Info "API endpoint: 192.168.1.101:30900 (via NGINX Ingress TCP)"
    Write-Info "Console: kubectl port-forward svc/minio-console 9001:9001 -n $NAMESPACE"
    Write-Info "Credentials: minioadmin / minioadmin123"
}

function Deploy-Qdrant {
    Write-Info "Deploying Qdrant..."
    kubectl apply -f "$SCRIPT_DIR\qdrant-statefulset.yaml"
    
    Write-Info "Waiting for Qdrant to be ready..."
    kubectl wait --for=condition=ready --timeout=120s pod/qdrant-0 -n $NAMESPACE
    
    Write-Info "✅ Qdrant deployed successfully"
    Write-Info "HTTP endpoint: 192.168.1.101:30633 (via NGINX Ingress TCP)"
    Write-Info "gRPC endpoint: 192.168.1.101:30634 (via NGINX Ingress TCP)"
}

function Deploy-Grafana {
    Write-Info "Deploying Grafana..."
    kubectl apply -f "$SCRIPT_DIR\grafana-deployment.yaml"
    kubectl apply -f "$SCRIPT_DIR\grafana-ingress.yaml"
    
    Write-Info "Waiting for Grafana to be ready..."
    kubectl wait --for=condition=available --timeout=120s deployment/grafana -n $NAMESPACE
    
    Write-Info "✅ Grafana deployed successfully"
    Write-Info "Access via: kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    Write-Info "Or add '192.168.1.101 grafana.local' to C:\Windows\System32\drivers\etc\hosts"
    Write-Info "Login: admin / admin123"
    Write-Info "Prometheus datasource already configured (lens-metrics namespace)"
}

function Update-NginxTCP {
    Write-Info "Updating NGINX Ingress TCP ConfigMap..."
    kubectl apply -f "$SCRIPT_DIR\nginx-ingress-tcp-full.yaml"
    
    Write-Info "Updating NGINX Ingress Controller Service..."
    kubectl patch svc ingress-nginx-1762145411-controller -n nginx-ingress --patch-file "$SCRIPT_DIR\nginx-controller-service-full.yaml"
    
    Write-Info "✅ NGINX Ingress TCP services updated"
}

function Deploy-All {
    Write-Info "Deploying all optional components..."
    
    Update-NginxTCP
    Deploy-PgAdmin
    Deploy-MinIO
    Deploy-Qdrant
    Deploy-Grafana
    
    Write-Host ""
    Write-Info "═══════════════════════════════════════════════════════════════════"
    Write-Info "✅ All optional components deployed successfully!"
    Write-Info "═══════════════════════════════════════════════════════════════════"
    
    Write-Host ""
    Write-Info "📊 Deployment Summary:"
    Write-Host ""
    Write-Host "🔧 pgAdmin (PostgreSQL UI):"
    Write-Host "   - URL: http://pgadmin.local (add to hosts file)"
    Write-Host "   - Port Forward: kubectl port-forward svc/pgadmin 5050:80 -n $NAMESPACE"
    Write-Host "   - Login: admin@admin.com / admin123"
    Write-Host ""
    Write-Host "💾 MinIO (S3 Storage):"
    Write-Host "   - API: 192.168.1.101:30900"
    Write-Host "   - Console Port Forward: kubectl port-forward svc/minio-console 9001:9001 -n $NAMESPACE"
    Write-Host "   - Credentials: minioadmin / minioadmin123"
    Write-Host "   - Buckets: user-files, rag-documents, embeddings-cache"
    Write-Host ""
    Write-Host "🔍 Qdrant (Vector DB):"
    Write-Host "   - HTTP: 192.168.1.101:30633"
    Write-Host "   - gRPC: 192.168.1.101:30634"
    Write-Host ""
    Write-Host "📈 Grafana (Monitoring):"
    Write-Host "   - URL: http://grafana.local (add to hosts file)"
    Write-Host "   - Port Forward: kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    Write-Host "   - Login: admin / admin123"
    Write-Host ""
}

function Show-Status {
    Write-Info "Checking deployment status..."
    Write-Host ""
    
    kubectl get pods -n $NAMESPACE
    Write-Host ""
    kubectl get svc -n $NAMESPACE
    Write-Host ""
    kubectl get ingress -n $NAMESPACE
    Write-Host ""
    kubectl get configmap tcp-services -n nginx-ingress -o jsonpath='{.data}'
}

# Main
Check-Namespace

switch ($Component.ToLower()) {
    "pgadmin" {
        Deploy-PgAdmin
    }
    "minio" {
        Update-NginxTCP
        Deploy-MinIO
    }
    "qdrant" {
        Update-NginxTCP
        Deploy-Qdrant
    }
    "grafana" {
        Deploy-Grafana
    }
    "all" {
        Deploy-All
    }
    "status" {
        Show-Status
    }
    default {
        Write-Error-Custom "Unknown component: $Component"
        Write-Host "Usage: .\deploy-optional.ps1 [all|pgadmin|minio|qdrant|grafana|status]"
        exit 1
    }
}

