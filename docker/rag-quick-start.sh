#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════
# RAG Infrastructure Quick Start Script
# ═══════════════════════════════════════════════════════════════════════════

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.rag-infrastructure.yml"
ENV_FILE="$PROJECT_ROOT/.env"
ENV_EXAMPLE="$SCRIPT_DIR/.env.rag.example"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker не установлен. Установите Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! docker compose version &> /dev/null; then
        print_error "Docker Compose не установлен или устарел. Обновите Docker."
        exit 1
    fi
    
    print_success "Docker установлен"
}

# Check .env file
check_env() {
    if [ ! -f "$ENV_FILE" ]; then
        print_warning ".env файл не найден. Копирую из примера..."
        cp "$ENV_EXAMPLE" "$ENV_FILE"
        print_success ".env файл создан"
        print_warning "ВАЖНО: Отредактируйте $ENV_FILE и измените пароли!"
        echo ""
        read -p "Нажмите Enter чтобы продолжить..."
    else
        print_success ".env файл существует"
    fi
}

# Show menu
show_menu() {
    print_header "RAG Infrastructure Quick Start"
    echo ""
    echo "Выберите профиль запуска:"
    echo ""
    echo "  1) Базовая конфигурация (PostgreSQL + Redis)"
    echo "  2) + Qdrant (vector database)"
    echo "  3) + MinIO (S3 storage)"
    echo "  4) + Мониторинг (Prometheus + Grafana)"
    echo "  5) Полный стек (все компоненты)"
    echo ""
    echo "  6) Остановить все"
    echo "  7) Очистить данные (volumes)"
    echo "  8) Показать статус"
    echo "  9) Показать логи"
    echo ""
    echo "  0) Выход"
    echo ""
}

# Start services
start_services() {
    local profile=$1
    print_header "Запуск сервисов: $profile"
    
    if [ "$profile" = "basic" ]; then
        docker compose -f "$COMPOSE_FILE" up -d
    else
        docker compose -f "$COMPOSE_FILE" --profile "$profile" up -d
    fi
    
    echo ""
    print_success "Сервисы запущены"
    show_urls
}

# Show URLs
show_urls() {
    echo ""
    print_header "Доступные сервисы"
    
    # Check which services are running
    if docker ps --format '{{.Names}}' | grep -q 'rag-postgres'; then
        echo -e "${GREEN}PostgreSQL:${NC}    postgres://proxy_user:***@localhost:5432/ollama_proxy"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-pgadmin'; then
        echo -e "${GREEN}pgAdmin:${NC}       http://localhost:5050 (admin@admin.com)"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-redis'; then
        echo -e "${GREEN}Redis:${NC}         redis://localhost:6379/0"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-qdrant'; then
        echo -e "${GREEN}Qdrant:${NC}        http://localhost:6333"
        echo -e "${GREEN}Qdrant UI:${NC}     http://localhost:6333/dashboard"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-minio'; then
        echo -e "${GREEN}MinIO API:${NC}     http://localhost:9000"
        echo -e "${GREEN}MinIO Console:${NC} http://localhost:9001"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-prometheus'; then
        echo -e "${GREEN}Prometheus:${NC}    http://localhost:9090"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q 'rag-grafana'; then
        echo -e "${GREEN}Grafana:${NC}       http://localhost:3000 (admin/admin123)"
    fi
    
    echo ""
}

# Stop services
stop_services() {
    print_header "Остановка сервисов"
    docker compose -f "$COMPOSE_FILE" down
    print_success "Все сервисы остановлены"
}

# Clean volumes
clean_volumes() {
    print_warning "Это удалит ВСЕ данные (volumes)!"
    read -p "Продолжить? (yes/no): " confirm
    
    if [ "$confirm" = "yes" ]; then
        print_header "Очистка данных"
        docker compose -f "$COMPOSE_FILE" down -v
        print_success "Все volumes удалены"
    else
        print_warning "Отменено"
    fi
}

# Show status
show_status() {
    print_header "Статус сервисов"
    docker compose -f "$COMPOSE_FILE" ps
    echo ""
    
    print_header "Использование volumes"
    docker volume ls | grep rag_
}

# Show logs
show_logs() {
    echo "Выберите сервис:"
    echo "  1) PostgreSQL"
    echo "  2) Redis"
    echo "  3) Qdrant"
    echo "  4) MinIO"
    echo "  5) Prometheus"
    echo "  6) Grafana"
    echo "  7) Все сервисы"
    echo ""
    read -p "Ваш выбор: " choice
    
    case $choice in
        1) docker logs -f rag-postgres ;;
        2) docker logs -f rag-redis ;;
        3) docker logs -f rag-qdrant ;;
        4) docker logs -f rag-minio ;;
        5) docker logs -f rag-prometheus ;;
        6) docker logs -f rag-grafana ;;
        7) docker compose -f "$COMPOSE_FILE" logs -f ;;
        *) print_error "Неверный выбор" ;;
    esac
}

# Health checks
health_checks() {
    print_header "Проверка здоровья сервисов"
    
    # PostgreSQL
    if docker exec rag-postgres pg_isready -U proxy_user -d ollama_proxy &> /dev/null; then
        print_success "PostgreSQL: OK"
    else
        print_error "PostgreSQL: FAIL"
    fi
    
    # Redis
    if docker exec rag-redis redis-cli ping | grep -q PONG; then
        print_success "Redis: OK"
    else
        print_error "Redis: FAIL"
    fi
    
    # Qdrant
    if curl -sf http://localhost:6333/healthz &> /dev/null; then
        print_success "Qdrant: OK"
    else
        print_warning "Qdrant: не запущен или недоступен"
    fi
    
    # MinIO
    if curl -sf http://localhost:9000/minio/health/live &> /dev/null; then
        print_success "MinIO: OK"
    else
        print_warning "MinIO: не запущен или недоступен"
    fi
    
    echo ""
}

# Main
main() {
    check_docker
    check_env
    
    while true; do
        show_menu
        read -p "Ваш выбор: " choice
        echo ""
        
        case $choice in
            1) start_services "basic" ;;
            2) start_services "qdrant" ;;
            3) start_services "minio" ;;
            4) start_services "monitoring" ;;
            5) start_services "full" ;;
            6) stop_services ;;
            7) clean_volumes ;;
            8) show_status ;;
            9) show_logs ;;
            0) 
                print_success "Выход"
                exit 0
                ;;
            *)
                print_error "Неверный выбор"
                ;;
        esac
        
        echo ""
        read -p "Нажмите Enter чтобы продолжить..."
        clear
    done
}

# Run
main

