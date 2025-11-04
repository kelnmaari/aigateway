# RAG Infrastructure - Optional Components Deployment Summary

## ✅ Установленные опциональные компоненты

### 1. pgAdmin (PostgreSQL Web UI)

**Deployment:** `pgadmin`
- **Image:** `dpage/pgadmin4:latest`
- **Status:** ✅ Running
- **Resources:**
  - Requests: 256Mi RAM, 100m CPU
  - Limits: 512Mi RAM, 500m CPU
- **Storage:** 1Gi PVC
- **Service:** pgadmin (ClusterIP: 10.98.137.214:80)

**Доступ:**

```bash
# 1. Через NGINX Ingress TCP (прямой доступ)
# URL: http://192.168.1.101:30523
# Открыть в браузере: http://192.168.1.101:30523

# 2. Port Forward (локальный доступ)
kubectl port-forward svc/pgadmin 5050:80 -n rag-infrastructure
# Открыть: http://localhost:5050

# 3. Через HTTP Ingress (добавить в hosts)
# Windows: C:\Windows\System32\drivers\etc\hosts
# Linux/Mac: /etc/hosts
192.168.1.101 pgadmin.local
# Открыть: http://pgadmin.local
```

**Credentials:**
- Email: `admin@admin.com`
- Password: `admin123`

**Преднастроенные подключения:**
- Server: RAG PostgreSQL
- Host: postgres-service
- Database: ollama_proxy
- User: proxy_user
- Password хранится в файле `/pgpass`

---

### 2. MinIO (S3-совместимое хранилище)

**StatefulSet:** `minio`
- **Image:** `minio/minio:latest`
- **Status:** ✅ Running
- **Resources:**
  - Requests: 512Mi RAM, 250m CPU
  - Limits: 2Gi RAM, 1 CPU
- **Storage:** 50Gi PVC
- **Services:**
  - `minio-api` (ClusterIP: 10.99.148.185:9000)
  - `minio-console` (ClusterIP: 10.106.91.141:9001)

**Доступ через NGINX Ingress TCP (NodePort):**

```bash
# MinIO API
# Endpoint: 192.168.1.101:30900 (NodePort: 30869)
# Внутренний endpoint: http://minio-api.rag-infrastructure.svc.cluster.local:9000

# MinIO Console Port Forward
kubectl port-forward svc/minio-console 9001:9001 -n rag-infrastructure
# Открыть: http://localhost:9001
```

**Credentials:**
- Root User: `minioadmin`
- Root Password: `minioadmin123`

**Созданные Buckets:**
- ✅ `user-files` (публичный download доступ)
- ✅ `rag-documents` (для RAG системы)
- ✅ `embeddings-cache` (для кеширования embeddings)

**Connection String для приложения:**

```go
// Go (minio-go)
minioClient, err := minio.New("minio-api.rag-infrastructure.svc.cluster.local:9000", &minio.Options{
    Creds:  credentials.NewStaticV4("minioadmin", "minioadmin123", ""),
    Secure: false,
})

// Или через NodePort (внешний доступ)
minioClient, err := minio.New("192.168.1.101:30869", &minio.Options{
    Creds:  credentials.NewStaticV4("minioadmin", "minioadmin123", ""),
    Secure: false,
})
```

---

### 3. Qdrant (Vector Database)

**StatefulSet:** `qdrant`
- **Image:** `qdrant/qdrant:latest`
- **Status:** ✅ Running
- **Resources:**
  - Requests: 1Gi RAM, 500m CPU
  - Limits: 4Gi RAM, 2 CPU
- **Storage:** 
  - storage: 20Gi PVC
  - snapshots: 10Gi PVC
- **Service:** `qdrant-service` (ClusterIP: 10.105.222.154)
  - HTTP port: 6333
  - gRPC port: 6334

**Доступ через NGINX Ingress TCP (NodePort):**

```bash
# Qdrant HTTP API
# Endpoint: http://192.168.1.101:30633 (NodePort: 32297)
# Внутренний endpoint: http://qdrant-service.rag-infrastructure.svc.cluster.local:6333

# Qdrant gRPC
# Endpoint: 192.168.1.101:30634 (NodePort: 32197)

# Port Forward
kubectl port-forward svc/qdrant-service 6333:6333 6334:6334 -n rag-infrastructure
```

**Connection String для приложения:**

```python
# Python (qdrant-client)
from qdrant_client import QdrantClient

# Внутри кластера
client = QdrantClient(
    host="qdrant-service.rag-infrastructure.svc.cluster.local",
    port=6333
)

# Внешний доступ через NodePort
client = QdrantClient(
    host="192.168.1.101",
    port=32297  # NodePort для HTTP
)
```

**Healthcheck:**

```bash
# Внутри кластера
curl http://qdrant-service.rag-infrastructure.svc.cluster.local:6333/healthz

# Через NodePort
curl http://192.168.1.101:32297/healthz
```

---

### 4. Grafana (Визуализация метрик)

**Deployment:** `grafana`
- **Image:** `grafana/grafana:latest`
- **Status:** ✅ Running
- **Resources:**
  - Requests: 256Mi RAM, 100m CPU
  - Limits: 1Gi RAM, 500m CPU
- **Storage:** 10Gi PVC
- **Service:** grafana (ClusterIP: 10.100.244.162:3000)

**Доступ:**

```bash
# 1. Через NGINX Ingress TCP (прямой доступ)
# URL: http://192.168.1.101:32027
# Открыть в браузере: http://192.168.1.101:32027

# 2. Port Forward (локальный доступ)
kubectl port-forward svc/grafana 3000:3000 -n rag-infrastructure
# Открыть: http://localhost:3000

# 3. Через HTTP Ingress (добавить в hosts)
# Windows: C:\Windows\System32\drivers\etc\hosts
# Linux/Mac: /etc/hosts
192.168.1.101 grafana.local
# Открыть: http://grafana.local
```

**Credentials:**
- Username: `admin`
- Password: `admin123`

**Преднастроенные Datasources:**
- ✅ Prometheus (подключен к `prometheus.lens-metrics.svc.cluster.local`)

**Рекомендуемые дашборды:**

```bash
# После входа в Grafana импортировать дашборды:
# 1. PostgreSQL - ID: 9628
# 2. Redis - ID: 11835
# 3. Kubernetes Pods - ID: 6417
```

---

## 🌐 Карта портов (NGINX Ingress TCP NodePort)

| Сервис | Внутренний порт | TCP порт | NodePort | IP |
|--------|----------------|----------|----------|-----|
| PostgreSQL | 5432 | 30432 | 32316 | 192.168.1.101 |
| Redis | 6379 | 30379 | 30897 | 192.168.1.101 |
| **pgAdmin** | 80 | **30050** | **30523** | **192.168.1.101** |
| MinIO API | 9000 | 30900 | 30869 | 192.168.1.101 |
| MinIO Console | 9001 | 30901 | 31477 | 192.168.1.101 |
| Qdrant HTTP | 6333 | 30633 | 32297 | 192.168.1.101 |
| Qdrant gRPC | 6334 | 30634 | 32197 | 192.168.1.101 |
| **Grafana** | 3000 | **30300** | **32027** | **192.168.1.101** |

## 📊 Обновленный dev.yaml конфиг

Конфигурация `configs/dev.yaml` обновлена для использования K8S сервисов:

### Redis (Rate Limiting)

```yaml
auth:
  rate_limiting:
    redis:
      enabled: true
      url: "redis://192.168.1.101:30897"
      key_prefix: "ollama_proxy_rl:"
```

### PostgreSQL (Database)

```yaml
database:
  postgresql:
    host: "192.168.1.101"
    port: 32316
    user: "proxy_user"
    password: "secure_password_change_me"
    database: "ollama_proxy"
```

### MinIO (File Storage)

```yaml
file_storage:
  s3:
    endpoint: "http://192.168.1.101:30900"
    bucket: "user-files"
    access_key: "minioadmin"
    secret_key: "minioadmin123"
```

### RAG System (pgvector + MinIO)

```yaml
rag:
  enabled: true
  
  vector_store:
    type: "pgvector"
    connection_string: "postgres://proxy_user:secure_password_change_me@192.168.1.101:32316/ollama_proxy?sslmode=disable"
    dimensions: 1024
  
  file_storage:
    type: "s3"
    s3_endpoint: "http://192.168.1.101:30900"
    s3_bucket: "rag-documents"
    s3_access_key: "minioadmin"
    s3_secret_key: "minioadmin123"
```

---

## 🚀 Использование deployment скриптов

### PowerShell (Windows)

```powershell
# Установить все компоненты
.\deploy-optional.ps1 all

# Установить отдельный компонент
.\deploy-optional.ps1 pgadmin
.\deploy-optional.ps1 minio
.\deploy-optional.ps1 qdrant
.\deploy-optional.ps1 grafana

# Проверить статус
.\deploy-optional.ps1 status
```

### Bash (Linux/Mac)

```bash
# Сделать скрипт исполняемым
chmod +x deploy-optional.sh

# Установить все компоненты
./deploy-optional.sh all

# Установить отдельный компонент
./deploy-optional.sh pgadmin
./deploy-optional.sh minio
./deploy-optional.sh qdrant
./deploy-optional.sh grafana

# Проверить статус
./deploy-optional.sh status
```

---

## 🧪 Тестирование подключений

### PostgreSQL + pgvector

```bash
# Через pgAdmin (web UI)
kubectl port-forward svc/pgadmin 5050:80 -n rag-infrastructure
# Открыть: http://localhost:5050
# Server: RAG PostgreSQL уже настроен

# Через psql (command line)
psql -h 192.168.1.101 -p 32316 -U proxy_user -d ollama_proxy

# Проверить pgvector extension
\dx
SELECT * FROM pg_extension WHERE extname = 'vector';
```

### Redis

```bash
# Через redis-cli (command line)
redis-cli -h 192.168.1.101 -p 30897

# Проверить подключение
PING
INFO server

# Тест записи/чтения
SET test:key "Hello from K8S"
GET test:key
```

### MinIO

```bash
# Через MinIO Console (web UI)
kubectl port-forward svc/minio-console 9001:9001 -n rag-infrastructure
# Открыть: http://localhost:9001
# Login: minioadmin / minioadmin123

# Через mc (MinIO client)
mc alias set k8s-minio http://192.168.1.101:30869 minioadmin minioadmin123

# Список buckets
mc ls k8s-minio

# Загрузить файл
mc cp test.txt k8s-minio/user-files/

# Скачать файл
mc cp k8s-minio/user-files/test.txt ./downloaded.txt
```

### Qdrant

```bash
# Через HTTP API
curl http://192.168.1.101:32297/

# Список коллекций
curl http://192.168.1.101:32297/collections

# Healthcheck
curl http://192.168.1.101:32297/healthz
```

### Grafana + Prometheus

```bash
# Открыть Grafana
kubectl port-forward svc/grafana 3000:3000 -n rag-infrastructure
# http://localhost:3000
# Login: admin / admin123

# Проверить Prometheus datasource:
# Configuration → Data Sources → Prometheus
# URL должен быть: http://prometheus.lens-metrics.svc.cluster.local
```

---

## 📝 Полезные команды

### Логи компонентов

```bash
# pgAdmin logs
kubectl logs -f deployment/pgadmin -n rag-infrastructure

# MinIO logs
kubectl logs -f minio-0 -n rag-infrastructure

# Qdrant logs
kubectl logs -f qdrant-0 -n rag-infrastructure

# Grafana logs
kubectl logs -f deployment/grafana -n rag-infrastructure
```

### Restart компонентов

```bash
# Restart pgAdmin
kubectl rollout restart deployment/pgadmin -n rag-infrastructure

# Restart MinIO (удалить pod, StatefulSet пересоздаст)
kubectl delete pod minio-0 -n rag-infrastructure

# Restart Qdrant
kubectl delete pod qdrant-0 -n rag-infrastructure

# Restart Grafana
kubectl rollout restart deployment/grafana -n rag-infrastructure
```

### Удаление компонентов

```bash
# Удалить pgAdmin
kubectl delete -f pgadmin-deployment.yaml
kubectl delete -f pgadmin-ingress.yaml
kubectl delete pvc pgadmin-data -n rag-infrastructure

# Удалить MinIO
kubectl delete -f minio-statefulset.yaml
kubectl delete pvc minio-data-minio-0 -n rag-infrastructure

# Удалить Qdrant
kubectl delete -f qdrant-statefulset.yaml
kubectl delete pvc qdrant-storage-qdrant-0 qdrant-snapshots-qdrant-0 -n rag-infrastructure

# Удалить Grafana
kubectl delete -f grafana-deployment.yaml
kubectl delete -f grafana-ingress.yaml
kubectl delete pvc grafana-data -n rag-infrastructure
```

---

## 🔍 Мониторинг в Grafana

### Импорт дашбордов

1. Открыть Grafana: http://localhost:3000 (port-forward)
2. Login: admin / admin123
3. Dashboards → Import

**PostgreSQL Dashboard (ID: 9628):**
- Connections, transactions, cache hits
- Table sizes, index usage
- Query performance

**Redis Dashboard (ID: 11835):**
- Memory usage, keyspace
- Commands per second
- Hit rate, evictions

**Kubernetes Pods Dashboard (ID: 6417):**
- CPU, Memory usage per pod
- Network I/O
- Restart count

### Создание custom дашборда для RAG infrastructure

```
# Prometheus queries для RAG метрик:

# PostgreSQL connections
sum(pg_stat_activity_count{datname="ollama_proxy"})

# Redis memory usage
redis_memory_used_bytes / redis_memory_max_bytes

# MinIO storage usage
minio_bucket_usage_total_bytes

# Qdrant collections count
qdrant_collections_total
```

---

## ⚠️ Production Checklist

Перед использованием в production:

### Безопасность

- [ ] Изменить все дефолтные пароли
- [ ] Включить TLS для всех сервисов
- [ ] Настроить Network Policies
- [ ] Использовать Sealed Secrets или Vault
- [ ] Ограничить RBAC доступы
- [ ] Включить аудит логирование

### Надежность

- [ ] Настроить replication для PostgreSQL
- [ ] Настроить Redis Sentinel или Cluster
- [ ] Использовать fast storage (SSD)
- [ ] Настроить PodDisruptionBudgets
- [ ] Настроить автоматические backups
- [ ] Настроить мониторинг и алерты

### Производительность

- [ ] Оптимизировать resource limits/requests
- [ ] Настроить HorizontalPodAutoscaler
- [ ] Использовать connection pooling (PgBouncer)
- [ ] Настроить caching стратегию
- [ ] Оптимизировать PostgreSQL параметры

---

## 📚 Дополнительная документация

- [README.md](README.md) - Основная документация по RAG infrastructure
- [DEPLOYMENT_SUMMARY.md](DEPLOYMENT_SUMMARY.md) - Core components (PostgreSQL, Redis)
- [K8S_HELM_DEPLOYMENT.md](../docs/K8S_HELM_DEPLOYMENT.md) - Общая документация по Helm
- [configs/dev.yaml](../configs/dev.yaml) - Обновленная конфигурация приложения

---

**Статус:** ✅ All optional components deployed and running  
**Дата:** 2025-11-03  
**K8S Cluster:** 192.168.1.101:6443  
**Namespace:** rag-infrastructure  
**Components:** PostgreSQL, Redis, pgAdmin, MinIO, Qdrant, Grafana  
**Access:** NGINX Ingress TCP (NodePort) + HTTP Ingress

