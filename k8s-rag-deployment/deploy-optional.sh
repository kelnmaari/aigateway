#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# RAG Infrastructure - Optional Components Deployment Script
# ═══════════════════════════════════════════════════════════════════════════════
#
# Этот скрипт устанавливает опциональные компоненты RAG infrastructure:
#   - pgAdmin (PostgreSQL web UI)
#   - MinIO (S3-совместимое хранилище)
#   - Qdrant (альтернативный vector database)
#   - Grafana (визуализация метрик, подключается к Prometheus в lens-metrics)
#
# Использование:
#   ./deploy-optional.sh [all|pgadmin|minio|qdrant|grafana]
#
# ═══════════════════════════════════════════════════════════════════════════════

set -e  # Exit on error

NAMESPACE="rag-infrastructure"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if namespace exists
check_namespace() {
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace $NAMESPACE does not exist. Please deploy core components first."
        exit 1
    fi
}

# Deploy pgAdmin
deploy_pgadmin() {
    log_info "Deploying pgAdmin..."
    kubectl apply -f "$SCRIPT_DIR/pgadmin-deployment.yaml"
    kubectl apply -f "$SCRIPT_DIR/pgadmin-ingress.yaml"
    
    log_info "Waiting for pgAdmin to be ready..."
    kubectl wait --for=condition=available --timeout=120s deployment/pgadmin -n $NAMESPACE
    
    log_info "✅ pgAdmin deployed successfully"
    log_info "Access via: kubectl port-forward svc/pgadmin 5050:80 -n $NAMESPACE"
    log_info "Or add '192.168.1.101 pgadmin.local' to /etc/hosts and access http://pgadmin.local"
    log_info "Login: admin@admin.com / admin123"
}

# Deploy MinIO
deploy_minio() {
    log_info "Deploying MinIO..."
    kubectl apply -f "$SCRIPT_DIR/minio-statefulset.yaml"
    
    log_info "Waiting for MinIO to be ready..."
    kubectl wait --for=condition=ready --timeout=120s pod/minio-0 -n $NAMESPACE
    
    log_info "Creating MinIO buckets..."
    # Wait for job to complete
    kubectl wait --for=condition=complete --timeout=60s job/minio-setup-buckets -n $NAMESPACE || log_warn "Bucket creation job timed out (may be still running)"
    
    log_info "✅ MinIO deployed successfully"
    log_info "API endpoint: 192.168.1.101:30900 (via NGINX Ingress TCP)"
    log_info "Console: kubectl port-forward svc/minio-console 9001:9001 -n $NAMESPACE"
    log_info "Credentials: minioadmin / minioadmin123"
}

# Deploy Qdrant
deploy_qdrant() {
    log_info "Deploying Qdrant..."
    kubectl apply -f "$SCRIPT_DIR/qdrant-statefulset.yaml"
    
    log_info "Waiting for Qdrant to be ready..."
    kubectl wait --for=condition=ready --timeout=120s pod/qdrant-0 -n $NAMESPACE
    
    log_info "✅ Qdrant deployed successfully"
    log_info "HTTP endpoint: 192.168.1.101:30633 (via NGINX Ingress TCP)"
    log_info "gRPC endpoint: 192.168.1.101:30634 (via NGINX Ingress TCP)"
    log_info "Or access via: kubectl port-forward svc/qdrant-service 6333:6333 6334:6334 -n $NAMESPACE"
}

# Deploy Grafana
deploy_grafana() {
    log_info "Deploying Grafana..."
    kubectl apply -f "$SCRIPT_DIR/grafana-deployment.yaml"
    kubectl apply -f "$SCRIPT_DIR/grafana-ingress.yaml"
    
    log_info "Waiting for Grafana to be ready..."
    kubectl wait --for=condition=available --timeout=120s deployment/grafana -n $NAMESPACE
    
    log_info "✅ Grafana deployed successfully"
    log_info "Access via: kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    log_info "Or add '192.168.1.101 grafana.local' to /etc/hosts and access http://grafana.local"
    log_info "Login: admin / admin123"
    log_info "Prometheus datasource already configured (lens-metrics namespace)"
}

# Update NGINX Ingress TCP ConfigMap
update_nginx_tcp() {
    log_info "Updating NGINX Ingress TCP ConfigMap..."
    kubectl apply -f "$SCRIPT_DIR/nginx-ingress-tcp-full.yaml"
    
    log_info "Updating NGINX Ingress Controller Service..."
    kubectl patch svc ingress-nginx-1762145411-controller -n nginx-ingress \
        --patch-file "$SCRIPT_DIR/nginx-controller-service-full.yaml"
    
    log_info "✅ NGINX Ingress TCP services updated"
}

# Deploy all components
deploy_all() {
    log_info "Deploying all optional components..."
    
    update_nginx_tcp
    deploy_pgadmin
    deploy_minio
    deploy_qdrant
    deploy_grafana
    
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info "✅ All optional components deployed successfully!"
    log_info "═══════════════════════════════════════════════════════════════════"
    
    # Print summary
    echo ""
    log_info "📊 Deployment Summary:"
    echo ""
    echo "🔧 pgAdmin (PostgreSQL UI):"
    echo "   - URL: http://pgadmin.local (add to /etc/hosts)"
    echo "   - Port Forward: kubectl port-forward svc/pgadmin 5050:80 -n $NAMESPACE"
    echo "   - Login: admin@admin.com / admin123"
    echo ""
    echo "💾 MinIO (S3 Storage):"
    echo "   - API: 192.168.1.101:30900"
    echo "   - Console Port Forward: kubectl port-forward svc/minio-console 9001:9001 -n $NAMESPACE"
    echo "   - Credentials: minioadmin / minioadmin123"
    echo "   - Buckets: user-files, rag-documents, embeddings-cache"
    echo ""
    echo "🔍 Qdrant (Vector DB):"
    echo "   - HTTP: 192.168.1.101:30633"
    echo "   - gRPC: 192.168.1.101:30634"
    echo "   - Port Forward: kubectl port-forward svc/qdrant-service 6333:6333 6334:6334 -n $NAMESPACE"
    echo ""
    echo "📈 Grafana (Monitoring):"
    echo "   - URL: http://grafana.local (add to /etc/hosts)"
    echo "   - Port Forward: kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    echo "   - Login: admin / admin123"
    echo "   - Prometheus: Connected to lens-metrics namespace"
    echo ""
}

# Show status of all components
show_status() {
    log_info "Checking deployment status..."
    echo ""
    
    kubectl get pods -n $NAMESPACE
    echo ""
    kubectl get svc -n $NAMESPACE
    echo ""
    kubectl get ingress -n $NAMESPACE
    echo ""
    kubectl get configmap tcp-services -n nginx-ingress -o jsonpath='{.data}' | jq .
}

# Main
main() {
    check_namespace
    
    COMPONENT="${1:-all}"
    
    case "$COMPONENT" in
        pgadmin)
            deploy_pgadmin
            ;;
        minio)
            update_nginx_tcp
            deploy_minio
            ;;
        qdrant)
            update_nginx_tcp
            deploy_qdrant
            ;;
        grafana)
            deploy_grafana
            ;;
        all)
            deploy_all
            ;;
        status)
            show_status
            ;;
        *)
            log_error "Unknown component: $COMPONENT"
            echo "Usage: $0 [all|pgadmin|minio|qdrant|grafana|status]"
            exit 1
            ;;
    esac
}

main "$@"

