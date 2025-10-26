# Prometheus & Grafana Setup Guide

**Version:** 1.11.6+ (Enterprise Suite - Prometheus Metrics Export)

Данное руководство описывает настройку Prometheus и Grafana для мониторинга Ollama-OpenAI Proxy.

## Оглавление

- [Включение Metrics в Proxy](#включение-metrics-в-proxy)
- [Установка Prometheus](#установка-prometheus)
- [Установка Grafana](#установка-grafana)
- [Импорт Dashboard](#импорт-dashboard)
- [Docker Compose Setup](#docker-compose-setup)
- [Доступные Метрики](#доступные-метрики)
- [Примеры Prometheus Queries](#примеры-prometheus-queries)

## Включение Metrics в Proxy

### 1. Configuration

Включите экспорт метрик в `config.yaml`:

```yaml
metrics:
  enabled: true
  prometheus_path: "/metrics"
  
  collection:
    enabled: true
    buffer_size: 1000
    flush_interval: "10s"
```

### 2. Проверка Endpoint

Запустите Ollama-OpenAI Proxy и проверьте доступность метрик:

```bash
curl http://localhost:8080/metrics
```

Вы должны увидеть метрики в формате Prometheus:

```
# HELP ollama_proxy_http_requests_total Total number of HTTP requests
# TYPE ollama_proxy_http_requests_total counter
ollama_proxy_http_requests_total{method="GET",endpoint="/v1/models",status="200"} 42

# HELP ollama_proxy_http_request_duration_seconds HTTP request duration in seconds
# TYPE ollama_proxy_http_request_duration_seconds histogram
ollama_proxy_http_request_duration_seconds_bucket{method="POST",endpoint="/v1/chat/completions",le="0.001"} 5
...
```

## Установка Prometheus

### Option 1: Binary Installation (Linux/macOS)

1. **Скачайте Prometheus:**

```bash
# Linux
wget https://github.com/prometheus/prometheus/releases/download/v2.47.0/prometheus-2.47.0.linux-amd64.tar.gz
tar xvfz prometheus-*.tar.gz
cd prometheus-*

# macOS
brew install prometheus
```

2. **Создайте конфигурацию:**

Создайте файл `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

3. **Запустите Prometheus:**

```bash
./prometheus --config.file=prometheus.yml
```

Prometheus будет доступен на `http://localhost:9090`

### Option 2: Docker

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

### Option 3: Docker Compose (рекомендуется)

См. раздел [Docker Compose Setup](#docker-compose-setup) ниже.

## Установка Grafana

### Option 1: Binary Installation (Linux/macOS)

```bash
# Linux
wget https://dl.grafana.com/oss/release/grafana-10.1.5.linux-amd64.tar.gz
tar -zxvf grafana-*.tar.gz
cd grafana-*
./bin/grafana-server

# macOS
brew install grafana
brew services start grafana
```

Grafana будет доступна на `http://localhost:3000`

Default credentials: `admin` / `admin`

### Option 2: Docker

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

## Настройка Grafana

### 1. Добавьте Data Source

1. Откройте Grafana: `http://localhost:3000`
2. Войдите (admin/admin)
3. Navigate: **Configuration** → **Data Sources** → **Add data source**
4. Выберите **Prometheus**
5. URL: `http://localhost:9090` (или `http://prometheus:9090` если в Docker)
6. Click **Save & Test**

### 2. Импорт Dashboard

1. Navigate: **Dashboards** → **Import**
2. Upload `docs/grafana-dashboard.json` из репозитория
3. Выберите Prometheus data source
4. Click **Import**

### Альтернатива: Ручное создание

Если хотите создать dashboard вручную:

1. Navigate: **Dashboards** → **New Dashboard**
2. Add Panel
3. Используйте queries из раздела [Примеры Prometheus Queries](#примеры-prometheus-queries)

## Docker Compose Setup

Для полного stack с Prometheus, Grafana и Ollama-OpenAI Proxy:

Создайте `docker-compose-monitoring.yml`:

```yaml
version: '3.8'

services:
  # Ollama-OpenAI Proxy
  proxy:
    image: ollama-openai-proxy:latest
    ports:
      - "8080:8080"
    volumes:
      - ./configs/production.yaml:/app/config.yaml
      - ./data:/data
    environment:
      - GO_ENV=production

  # Prometheus
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  # Grafana
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./docs/grafana-dashboard.json:/etc/grafana/provisioning/dashboards/ollama-proxy.json
      - ./grafana-datasources.yml:/etc/grafana/provisioning/datasources/prometheus.yml
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false

volumes:
  prometheus_data:
  grafana_data:
```

Создайте `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['proxy:8080']
    metrics_path: '/metrics'
```

Создайте `grafana-datasources.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

Запустите stack:

```bash
docker-compose -f docker-compose-monitoring.yml up -d
```

## Доступные Метрики

### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_http_requests_total` | Counter | method, endpoint, status | Total HTTP requests |
| `ollama_proxy_http_request_duration_seconds` | Histogram | method, endpoint | Request duration |
| `ollama_proxy_http_response_size_bytes` | Histogram | method, endpoint | Response size |
| `ollama_proxy_http_active_connections` | Gauge | - | Active HTTP connections |

### API Usage Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_api_tokens_used_total` | Counter | api_key_id, model, type | Tokens used (prompt/completion) |
| `ollama_proxy_api_requests_total` | Counter | model, status | API requests (success/error) |
| `ollama_proxy_api_cost_total` | Counter | api_key_id, model | API cost (if enabled) |

### Model Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_model_request_duration_seconds` | Histogram | model | Model request latency |
| `ollama_proxy_models_loaded` | Gauge | - | Currently loaded models |
| `ollama_proxy_model_errors_total` | Counter | model, error_type | Model errors |

### System Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_goroutines` | Gauge | - | Number of goroutines |
| `ollama_proxy_memory_bytes` | Gauge | type | Memory usage (alloc, sys, heap_*) |
| `ollama_proxy_db_connections` | Gauge | - | Active DB connections |
| `ollama_proxy_api_keys_total` | Gauge | - | Total API keys |
| `ollama_proxy_users_total` | Gauge | - | Total users |
| `ollama_proxy_tenants_total` | Gauge | - | Total tenants |

### Authentication Metrics (v1.11+)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_auth_attempts_total` | Counter | type, status | Auth attempts (jwt, oidc, ldap, api_key) |
| `ollama_proxy_auth_duration_seconds` | Histogram | type | Auth duration by type |

### Quota Metrics (future)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ollama_proxy_quota_usage` | Gauge | target_id, type | Current quota usage |
| `ollama_proxy_quota_exceeded_total` | Counter | target_id, type | Quota exceeded events |
| `ollama_proxy_quota_limit` | Gauge | target_id, type | Quota limits |

## Примеры Prometheus Queries

### HTTP Performance

**Request Rate:**
```promql
rate(ollama_proxy_http_requests_total[5m])
```

**Error Rate:**
```promql
rate(ollama_proxy_http_requests_total{status=~"5.."}[5m])
```

**Request Duration p95:**
```promql
histogram_quantile(0.95, rate(ollama_proxy_http_request_duration_seconds_bucket[5m]))
```

**Request Duration by Endpoint (p99):**
```promql
histogram_quantile(0.99, sum(rate(ollama_proxy_http_request_duration_seconds_bucket[5m])) by (endpoint, le))
```

### API Usage

**Tokens per Second:**
```promql
rate(ollama_proxy_api_tokens_used_total[1m])
```

**Tokens by Model:**
```promql
sum(rate(ollama_proxy_api_tokens_used_total[5m])) by (model, type)
```

**Most Used Models (Top 5):**
```promql
topk(5, sum by (model) (rate(ollama_proxy_api_requests_total[5m])))
```

**Success Rate:**
```promql
sum(rate(ollama_proxy_api_requests_total{status="success"}[5m])) / sum(rate(ollama_proxy_api_requests_total[5m]))
```

### Model Performance

**Model Latency p95 by Model:**
```promql
histogram_quantile(0.95, sum(rate(ollama_proxy_model_request_duration_seconds_bucket[5m])) by (model, le))
```

**Model Error Rate:**
```promql
rate(ollama_proxy_model_errors_total[5m])
```

### System Health

**Memory Usage:**
```promql
ollama_proxy_memory_bytes{type="heap_alloc"}
```

**Goroutine Count:**
```promql
ollama_proxy_goroutines
```

**Active Connections:**
```promql
ollama_proxy_http_active_connections
```

## Alerting Examples

Добавьте в `prometheus.yml`:

```yaml
rule_files:
  - "alerts.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

Создайте `alerts.yml`:

```yaml
groups:
  - name: ollama_proxy
    interval: 10s
    rules:
      # High Error Rate
      - alert: HighErrorRate
        expr: rate(ollama_proxy_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"
      
      # High Latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(ollama_proxy_http_request_duration_seconds_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency"
          description: "p95 latency is {{ $value }}s"
      
      # High Memory Usage
      - alert: HighMemoryUsage
        expr: ollama_proxy_memory_bytes{type="heap_alloc"} > 2147483648
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High memory usage"
          description: "Heap allocation is {{ $value | humanize }}B"
      
      # Too Many Goroutines
      - alert: TooManyGoroutines
        expr: ollama_proxy_goroutines > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Too many goroutines"
          description: "Goroutine count is {{ $value }}"
```

## Troubleshooting

### Metrics не отображаются

1. **Проверьте что metrics включены:**
   ```bash
   curl http://localhost:8080/metrics
   ```

2. **Проверьте Prometheus targets:**
   ```
   http://localhost:9090/targets
   ```
   Status должен быть "UP"

3. **Проверьте логи Prometheus:**
   ```bash
   docker logs prometheus
   ```

### Grafana не подключается к Prometheus

1. Проверьте URL в data source настройках
2. Если используете Docker, убедитесь что сервисы в одной сети
3. Используйте имя сервиса в Docker Compose: `http://prometheus:9090`

### Dashboard не показывает данные

1. Убедитесь что выбран правильный data source
2. Проверьте time range (правый верхний угол)
3. Убедитесь что proxy генерирует метрики (делайте запросы)

## Production Recommendations

1. **Storage Retention:**
   ```bash
   --storage.tsdb.retention.time=30d
   --storage.tsdb.retention.size=50GB
   ```

2. **Scrape Interval:**
   - Development: 15s
   - Production: 30s или 1m

3. **High Availability:**
   - Используйте Prometheus Federation для multiple instances
   - Thanos или Cortex для long-term storage

4. **Security:**
   - Basic auth для `/metrics` endpoint
   - TLS для Prometheus → Proxy connection
   - Grafana authentication (не используйте admin/admin!)

5. **Label Cardinality:**
   - Избегайте высокой cardinality labels (user IDs, UUIDs)
   - Используйте `api_key_id` вместо actual key values
   - Ограничьте количество unique label values

## Дополнительные Ресурсы

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/)

---

**Версия:** 1.11.6+  
**Последнее обновление:** 2025-10-25

