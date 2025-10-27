# ⚙️ Configuration Guide - AIGateway Platform v1.9.3

Полное руководство по конфигурации

---

## 📋 Содержание

1. [Configuration Files](#configuration-files)
2. [Server Settings](#server-settings)
3. [Ollama Connection](#ollama-connection)
4. [Database](#database)
5. [Authentication & JWT](#authentication--jwt)
6. [Performance Monitoring](#performance-monitoring)
7. [Logging](#logging)
8. [Environment Variables](#environment-variables)
9. [Production Configuration](#production-configuration)

---

## 📁 Configuration Files

### Available Configs

```
configs/
├── dev.yaml                    # Development (default)
├── production.yaml.example     # Production template
├── mvp.yaml                    # MVP без auth (legacy)
└── mvp_noauth.yaml            # MVP без auth (legacy)
```

### Usage

```bash
# Development
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml

# Production
cp configs/production.yaml.example configs/production.yaml
# Edit production.yaml
./dist/ollama-proxy-linux-amd64 -config configs/production.yaml

# Default (если -config не указан, ищет dev.yaml)
./dist/ollama-proxy-linux-amd64
```

---

## 🖥️ Server Settings

### Basic Server Config

```yaml
server:
  host: "0.0.0.0"              # Bind address (0.0.0.0 = all interfaces)
  port: 8080                   # HTTP port
  read_timeout: "30s"          # Request read timeout
  write_timeout: "180s"        # Response write timeout (high for streaming)
  idle_timeout: "120s"         # Keep-alive idle timeout
  max_header_bytes: 1048576    # 1MB max header size
```

**Production recommendations:**
- `host: "0.0.0.0"` - для Docker/cloud
- `host: "127.0.0.1"` - только localhost
- `write_timeout: "300s"` - для больших моделей
- `read_timeout: "60s"` - для slow clients

### CORS Settings

```yaml
server:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:8080"
      - "http://localhost:3000"      # Frontend dev server
      - "https://yourdomain.com"     # Production domain
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:
      - "Content-Type"
      - "Authorization"
    exposed_headers:
      - "X-Request-ID"
    allow_credentials: true
    max_age: 3600  # Preflight cache (seconds)
```

**Development:** Разрешите `localhost:*`  
**Production:** Только конкретные домены!

---

## 🦙 Ollama Connection

### Basic Ollama Config

```yaml
ollama:
  url: "http://localhost:11434"    # Ollama API URL
  timeout: "180s"                  # Request timeout
  retry_attempts: 3                # Retry count on failure
  retry_delay: "2s"                # Delay between retries
  connection_pool_size: 100        # HTTP connection pool
  keep_alive: true                 # Reuse connections
```

### Remote Ollama

```yaml
ollama:
  url: "http://192.168.1.100:11434"  # Remote server
  timeout: "300s"                     # Higher timeout for network latency
```

### High-Traffic Setup

```yaml
ollama:
  connection_pool_size: 200    # Increase for concurrent requests
  timeout: "300s"              # Longer timeout for queue
  retry_attempts: 5            # More retries
```

---

## 💾 Database

### SQLite (Default)

```yaml
database:
  type: "sqlite"
  sqlite:
    path: "data/proxy.db"          # Database file
    max_open_conns: 10             # Max concurrent connections
    max_idle_conns: 5              # Idle connection pool
    conn_max_lifetime: "1h"        # Connection lifetime
```

**Pros:** Simple, no setup, good for small-medium deployments  
**Cons:** Single-writer bottleneck для high concurrency

### PostgreSQL (Production)

```yaml
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "ollama_proxy"
    password: "secure_password"
    database: "ollama_proxy_db"
    sslmode: "require"              # require, verify-full, disable
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: "1h"
```

**Setup PostgreSQL:**

```sql
CREATE DATABASE ollama_proxy_db;
CREATE USER ollama_proxy WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE ollama_proxy_db TO ollama_proxy;
```

**Pros:** Multi-writer, better concurrency, production-grade  
**Cons:** Requires PostgreSQL setup

---

## 🔐 Authentication & JWT

### JWT Configuration

```yaml
auth:
  jwt_secret: "change-me-in-production-min-32-chars!"  # ⚠️ CRITICAL!
  token_expiration: "24h"          # Access token lifetime
  refresh_expiration: "7d"         # Refresh token lifetime
  issuer: "aigateway"    # JWT issuer claim
```

**Security:**
- `jwt_secret` должен быть **минимум 32 символа**
- Используйте **random string** для production:
  ```bash
  openssl rand -hex 32
  ```
- **НЕ** коммитьте production секрет в git!

### Bootstrap Admin

```yaml
auth:
  bootstrap:
    enabled: true               # Включить bootstrap при первом запуске
    token_expiration: "10m"     # Bootstrap token lifetime
```

**Bootstrap flow:**
1. При первом запуске генерируется bootstrap URL
2. Token действует 10 минут
3. Используется для создания первого admin user
4. После использования или expiration - disabled

### Password Policy

```yaml
auth:
  password:
    min_length: 8               # Минимум символов
    require_uppercase: true     # Требовать заглавные
    require_lowercase: true     # Требовать строчные
    require_numbers: true       # Требовать цифры
    require_special: false      # Требовать спецсимволы
    bcrypt_cost: 12            # Bcrypt cost (10-14)
```

**Recommendations:**
- Development: cost=10 (быстрее)
- Production: cost=12-14 (безопаснее)

---

## 📊 Performance Monitoring

### MoniGo Dashboard

```yaml
performance:
  monigo:
    enabled: true                 # Enable MoniGo
    port: 9091                    # Dedicated port for dashboard
    collect_interval: "5s"        # Metrics collection interval
    retention: "1h"               # Metrics retention time
```

**Access:** `http://localhost:9091`  
**Requirements:** Admin JWT token

### GPU Monitoring

```yaml
performance:
  gpu_monitoring:
    enabled: true                 # Enable NVIDIA GPU monitoring
    refresh_interval: "5s"        # Refresh interval
    command: "nvidia-smi"         # CLI command (default)
```

**Platform support:**
- ✅ Linux with nvidia-smi
- ✅ macOS with nvidia-smi (если есть GPU)
- ❌ Windows (stub version, disabled)

### OpenTelemetry

```yaml
observability:
  enabled: true
  tracing:
    enabled: true
    exporter: "jaeger"                             # jaeger, zipkin, otlp
    jaeger_endpoint: "http://localhost:14268/api/traces"
    zipkin_endpoint: "http://localhost:9411/api/v2/spans"
    sampling_rate: 0.1                             # 10% sampling
    service_name: "aigateway"
```

**Exporters:**
- `jaeger` - Jaeger UI на :16686
- `zipkin` - Zipkin UI на :9411
- `otlp` - OpenTelemetry Protocol (для Grafana Tempo, etc.)

---

## 📝 Logging

### Basic Logging

```yaml
logging:
  level: "info"                   # debug, info, warn, error
  format: "text"                  # text or json
  output: "both"                  # stdout, file, or both
  file_path: "logs/proxy-dev.log" # Log file path
```

**Log Levels:**
- `debug` - Все детали (для разработки)
- `info` - Общая информация (production default)
- `warn` - Предупреждения
- `error` - Только ошибки

### Log Rotation

```yaml
logging:
  rotation:
    enabled: true
    max_size: 100          # MB per file
    max_backups: 5         # Number of old files to keep
    max_age: 30            # Days to keep old files
    compress: true         # Gzip old files
```

**Example rotation:**
```
logs/
├── proxy-dev.log          # Current
├── proxy-dev-2025-10-14.log.gz
├── proxy-dev-2025-10-13.log.gz
└── proxy-dev-2025-10-12.log.gz
```

### Structured Logging

```yaml
logging:
  format: "json"
  fields:
    service: "aigateway"
    environment: "production"
    version: "1.9.3"
```

**JSON output:**
```json
{
  "time": "2025-10-14T14:30:00Z",
  "level": "info",
  "service": "aigateway",
  "environment": "production",
  "version": "1.9.3",
  "msg": "Server started",
  "port": 8080
}
```

---

## 🌍 Environment Variables

### Priority

1. **Environment variables** (highest)
2. **Config file** (`-config`)
3. **Default values** (lowest)

### Available Variables

```bash
# Server
export PROXY_SERVER_HOST="0.0.0.0"
export PROXY_SERVER_PORT="8080"

# Ollama
export PROXY_OLLAMA_URL="http://localhost:11434"
export PROXY_OLLAMA_TIMEOUT="180s"

# Database
export PROXY_DATABASE_TYPE="postgresql"
export PROXY_POSTGRESQL_HOST="localhost"
export PROXY_POSTGRESQL_PORT="5432"
export PROXY_POSTGRESQL_USER="proxy"
export PROXY_POSTGRESQL_PASSWORD="secret"
export PROXY_POSTGRESQL_DATABASE="proxy_db"

# Auth
export PROXY_JWT_SECRET="your-secret-key-min-32-chars"
export PROXY_TOKEN_EXPIRATION="24h"

# Logging
export PROXY_LOGGING_LEVEL="debug"
export PROXY_LOGGING_FORMAT="json"

# Performance
export PROXY_MONIGO_ENABLED="true"
export PROXY_MONIGO_PORT="9091"
export PROXY_GPU_MONITORING_ENABLED="true"
```

### Docker Compose Example

```yaml
services:
  proxy:
    image: ollama-proxy:latest
    environment:
      - PROXY_SERVER_PORT=8080
      - PROXY_OLLAMA_URL=http://ollama:11434
      - PROXY_DATABASE_TYPE=postgresql
      - PROXY_POSTGRESQL_HOST=postgres
      - PROXY_JWT_SECRET=${JWT_SECRET}
    ports:
      - "8080:8080"
```

---

## 🚀 Production Configuration

### Production Template

```yaml
# configs/production.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "60s"
  write_timeout: "300s"
  cors:
    enabled: true
    allowed_origins:
      - "https://yourdomain.com"
    allow_credentials: true

ollama:
  url: "http://ollama:11434"      # Docker service name
  timeout: "300s"
  connection_pool_size: 200
  retry_attempts: 5

database:
  type: "postgresql"
  postgresql:
    host: "postgres"              # Docker service name
    port: 5432
    user: "ollama_proxy"
    password: "${DB_PASSWORD}"    # From env var
    database: "ollama_proxy_prod"
    sslmode: "require"
    max_open_conns: 100

auth:
  jwt_secret: "${JWT_SECRET}"     # From env var (32+ chars)
  token_expiration: "8h"          # Shorter для production
  refresh_expiration: "7d"
  password:
    bcrypt_cost: 14               # Higher security

performance:
  monigo:
    enabled: true
    port: 9091
  gpu_monitoring:
    enabled: true

observability:
  enabled: true
  tracing:
    enabled: true
    exporter: "jaeger"
    jaeger_endpoint: "http://jaeger:14268/api/traces"
    sampling_rate: 0.05           # 5% для production

logging:
  level: "info"                   # НЕ debug в production!
  format: "json"                  # Structured для log aggregation
  output: "both"
  rotation:
    enabled: true
    max_size: 500
    max_backups: 10
    compress: true
```

### Security Checklist

- [ ] Сменить `jwt_secret` на random 32+ символов
- [ ] Использовать PostgreSQL вместо SQLite
- [ ] Включить `sslmode: "require"` для PostgreSQL
- [ ] Настроить CORS только для production домена
- [ ] Использовать environment variables для секретов
- [ ] Включить HTTPS (через reverse proxy)
- [ ] Настроить firewall (только 8080, 9091 ports)
- [ ] Регулярный backup database
- [ ] Логирование в `json` формате
- [ ] Мониторинг через Prometheus + Grafana

### Docker Production Stack

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: ollama_proxy_prod
      POSTGRES_USER: ollama_proxy
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ollama_models:/root/.ollama

  proxy:
    image: ollama-proxy:1.9.3
    depends_on:
      - postgres
      - ollama
    environment:
      - PROXY_DATABASE_TYPE=postgresql
      - PROXY_POSTGRESQL_HOST=postgres
      - PROXY_POSTGRESQL_PASSWORD=${DB_PASSWORD}
      - PROXY_OLLAMA_URL=http://ollama:11434
      - PROXY_JWT_SECRET=${JWT_SECRET}
    ports:
      - "8080:8080"
      - "9091:9091"

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "14268:14268"  # Collector

volumes:
  postgres_data:
  ollama_models:
```

---

## 📚 Related Documentation

- [Quick Start](QUICK_START.md) - Быстрый старт
- [Troubleshooting](TROUBLESHOOTING.md) - Решение проблем
- [Architecture](ARCHITECTURE.md) - Архитектура
- [API Documentation](API_DOCUMENTATION.md) - API reference

---

**Version:** 1.9.3  
**Last Updated:** 2025-10-14

