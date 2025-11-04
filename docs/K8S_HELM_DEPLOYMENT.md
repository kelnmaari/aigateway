# RAG Infrastructure Deployment в Kubernetes с Helm Charts

Документация по развертыванию RAG infrastructure стека в Kubernetes с использованием Helm чартов, основанная на `docker-compose.rag-infrastructure.yml`.

## 📋 Оглавление

1. [Обзор архитектуры](#обзор-архитектуры)
2. [Предварительные требования](#предварительные-требования)
3. [Подход 1: Автоматическая конвертация Docker Compose](#подход-1-автоматическая-конвертация-docker-compose)
4. [Подход 2: Использование официальных Helm чартов](#подход-2-использование-официальных-helm-чартов)
5. [Конфигурация компонентов](#конфигурация-компонентов)
6. [Мониторинг и управление](#мониторинг-и-управление)
7. [Troubleshooting](#troubleshooting)

---

## Обзор архитектуры

RAG Infrastructure Stack состоит из следующих компонентов:

| Компонент | Назначение | Порт(ы) |
|-----------|------------|---------|
| **PostgreSQL + pgvector** | Vector database для embeddings | 5432 |
| **pgAdmin** | Web UI для управления PostgreSQL | 5050 |
| **Redis** | Кеширование и очередь задач | 6379 |
| **Qdrant** (опционально) | Альтернативный vector store | 6333, 6334 |
| **MinIO** (опционально) | S3-совместимое файловое хранилище | 9000, 9001 |
| **Prometheus** (опционально) | Сбор метрик | 9090 |
| **Grafana** (опционально) | Визуализация метрик | 3000 |

---

## Предварительные требования

### 1. Установка необходимых инструментов

```bash
# Kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Helm 3
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Проверка установки
kubectl version --client
helm version
```

### 2. Доступ к Kubernetes кластеру

```bash
# Проверка подключения к кластеру
kubectl cluster-info
kubectl get nodes

# Создание namespace для RAG infrastructure
kubectl create namespace rag-infrastructure
```

### 3. Настройка Helm репозиториев

```bash
# Bitnami - официальные чарты для PostgreSQL, Redis, MinIO
helm repo add bitnami https://charts.bitnami.com/bitnami

# Prometheus community - мониторинг стек
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

# Qdrant - vector database
helm repo add qdrant https://qdrant.github.io/qdrant-helm

# Обновление репозиториев
helm repo update
```

---

## Подход 1: Автоматическая конвертация Docker Compose

Этот подход использует инструменты для автоматического преобразования `docker-compose.rag-infrastructure.yml` в Helm чарт.

### Вариант A: Kompose

**Установка Kompose:**

```bash
# Linux
curl -L https://github.com/kubernetes/kompose/releases/download/v1.31.2/kompose-linux-amd64 -o kompose
chmod +x kompose
sudo mv ./kompose /usr/local/bin/kompose

# macOS
brew install kompose

# Windows (через Chocolatey)
choco install kubernetes-kompose
```

**Конвертация в Helm чарт:**

```bash
# Перейти в директорию с docker-compose файлом
cd /path/to/my_projects

# Конвертация в Helm чарт
kompose convert -f docker-compose.rag-infrastructure.yml -c

# Будет создана директория с Helm чартом
# По умолчанию: ./docker-compose-rag-infrastructure/
```

**Настройка и установка:**

```bash
# Редактировать values.yaml при необходимости
nano docker-compose-rag-infrastructure/values.yaml

# Установка чарта
helm install rag-stack ./docker-compose-rag-infrastructure \
  --namespace rag-infrastructure \
  --create-namespace

# Проверка статуса
helm status rag-stack -n rag-infrastructure
kubectl get pods -n rag-infrastructure
```

### Вариант B: Katenary

**Установка Katenary:**

```bash
# Linux (через .deb пакет для Ubuntu/Debian)
wget https://github.com/metal3d/katenary/releases/latest/download/katenary_amd64.deb
sudo dpkg -i katenary_amd64.deb

# macOS
wget https://github.com/metal3d/katenary/releases/latest/download/katenary_darwin_amd64
chmod +x katenary_darwin_amd64
sudo mv katenary_darwin_amd64 /usr/local/bin/katenary

# Windows
# Скачать .exe файл с https://github.com/metal3d/katenary/releases/latest
```

**Конвертация:**

```bash
cd /path/to/my_projects

# Конвертация (создаст директорию ./chart)
katenary convert -f docker-compose.rag-infrastructure.yml

# Установка
helm install rag-stack ./chart \
  --namespace rag-infrastructure \
  --create-namespace
```

### Преимущества и недостатки автоконвертации

**✅ Преимущества:**
- Быстрое получение рабочего Helm чарта
- Сохранение всех настроек из docker-compose
- Минимум ручной работы

**❌ Недостатки:**
- Не всегда оптимальная конфигурация для production
- Ограниченная кастомизация
- Может не учитывать best practices Kubernetes
- Не использует официальные Helm чарты с их возможностями

---

## Подход 2: Использование официальных Helm чартов

Этот подход использует официальные, поддерживаемые community Helm чарты для каждого компонента. **Рекомендуется для production.**

### 1. PostgreSQL + pgvector

**Helm чарт:** `bitnami/postgresql`

**Установка:**

```bash
# Создание values файла для PostgreSQL
cat > postgres-values.yaml <<EOF
auth:
  username: proxy_user
  password: secure_password_change_me
  database: ollama_proxy
  
primary:
  # PostgreSQL с pgvector требует custom образ
  image:
    registry: docker.io
    repository: pgvector/pgvector
    tag: pg16
  
  # Инициализация pgvector расширения
  initdb:
    scripts:
      01-init.sql: |
        -- Enable pgvector extension
        CREATE EXTENSION IF NOT EXISTS vector;
        
        -- Создание таблиц для RAG (пример)
        CREATE TABLE IF NOT EXISTS document_embeddings (
          id SERIAL PRIMARY KEY,
          document_id VARCHAR(255) NOT NULL,
          chunk_index INTEGER NOT NULL,
          embedding vector(1536),  -- OpenAI ada-002 dimension
          content TEXT,
          metadata JSONB,
          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
          UNIQUE(document_id, chunk_index)
        );
        
        -- Индекс для быстрого поиска по векторам
        CREATE INDEX IF NOT EXISTS document_embeddings_vector_idx 
          ON document_embeddings 
          USING ivfflat (embedding vector_cosine_ops)
          WITH (lists = 100);
          
        -- Индекс для быстрого поиска по document_id
        CREATE INDEX IF NOT EXISTS document_embeddings_document_id_idx 
          ON document_embeddings(document_id);
      
      02-tuning.sql: |
        -- Performance tuning для RAG workload
        ALTER SYSTEM SET shared_buffers = '512MB';
        ALTER SYSTEM SET effective_cache_size = '2GB';
        ALTER SYSTEM SET maintenance_work_mem = '256MB';
        ALTER SYSTEM SET work_mem = '64MB';
        ALTER SYSTEM SET wal_buffers = '16MB';
        ALTER SYSTEM SET max_connections = '200';
  
  # Ресурсы
  resources:
    limits:
      memory: 4Gi
      cpu: 2000m
    requests:
      memory: 2Gi
      cpu: 1000m
  
  # Persistence
  persistence:
    enabled: true
    size: 20Gi
    storageClass: ""  # Использовать default storage class
  
  # Health checks
  livenessProbe:
    enabled: true
    initialDelaySeconds: 30
    periodSeconds: 10
  readinessProbe:
    enabled: true
    initialDelaySeconds: 5
    periodSeconds: 10

# Metrics для Prometheus
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
EOF

# Установка PostgreSQL
helm install postgres-pgvector bitnami/postgresql \
  -f postgres-values.yaml \
  --namespace rag-infrastructure \
  --version 13.2.24
```

**Connection string для приложения:**

```bash
postgres://proxy_user:secure_password_change_me@postgres-pgvector-postgresql.rag-infrastructure.svc.cluster.local:5432/ollama_proxy?sslmode=disable
```

### 2. pgAdmin

**Helm чарт:** `runix/pgadmin4`

```bash
# Добавление репозитория
helm repo add runix https://helm.runix.net
helm repo update

# Создание values файла
cat > pgadmin-values.yaml <<EOF
env:
  email: admin@admin.com
  password: admin123

# Автоматическая настройка подключения к PostgreSQL
serverDefinitions:
  enabled: true
  servers:
    rag-postgres:
      Name: "RAG PostgreSQL"
      Group: "Servers"
      Host: "postgres-pgvector-postgresql.rag-infrastructure.svc.cluster.local"
      Port: 5432
      MaintenanceDB: "ollama_proxy"
      Username: "proxy_user"
      SSLMode: "prefer"

# Persistence для настроек
persistentVolume:
  enabled: true
  size: 1Gi

# Service
service:
  type: ClusterIP
  port: 80

# Ingress (опционально)
ingress:
  enabled: false
  # Для доступа извне раскомментируйте:
  # enabled: true
  # hosts:
  #   - host: pgadmin.example.com
  #     paths:
  #       - path: /
  #         pathType: Prefix

resources:
  limits:
    memory: 512Mi
    cpu: 500m
  requests:
    memory: 256Mi
    cpu: 250m
EOF

# Установка pgAdmin
helm install pgadmin runix/pgadmin4 \
  -f pgadmin-values.yaml \
  --namespace rag-infrastructure \
  --version 1.25.3
```

**Port forward для локального доступа:**

```bash
kubectl port-forward svc/pgadmin-pgadmin4 5050:80 -n rag-infrastructure
# Доступ: http://localhost:5050
```

### 3. Redis

**Helm чарт:** `bitnami/redis`

```bash
# Создание values файла
cat > redis-values.yaml <<EOF
# Архитектура: standalone (для простоты) или replication
architecture: standalone

auth:
  enabled: false  # Для совместимости с docker-compose
  # Для production включите:
  # enabled: true
  # password: "redis_password"

# Master конфигурация
master:
  persistence:
    enabled: true
    size: 8Gi
  
  resources:
    limits:
      memory: 2Gi
      cpu: 1000m
    requests:
      memory: 512Mi
      cpu: 250m
  
  # Redis конфигурация
  configuration: |-
    # Persistence
    appendonly yes
    appendfsync everysec
    save 900 1
    save 300 10
    save 60 10000
    
    # Memory management
    maxmemory 2gb
    maxmemory-policy allkeys-lru
    
    # Network
    tcp-backlog 511
    timeout 0
    tcp-keepalive 300

# Metrics для Prometheus
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
EOF

# Установка Redis
helm install redis bitnami/redis \
  -f redis-values.yaml \
  --namespace rag-infrastructure \
  --version 18.4.0
```

**Connection string:**

```bash
redis://redis-master.rag-infrastructure.svc.cluster.local:6379/0
```

### 4. Qdrant (опционально)

**Helm чарт:** `qdrant/qdrant`

```bash
# Создание values файла
cat > qdrant-values.yaml <<EOF
# Replicas
replicaCount: 1

image:
  repository: qdrant/qdrant
  tag: latest

# Service
service:
  type: ClusterIP
  httpPort: 6333
  grpcPort: 6334

# Persistence
persistence:
  enabled: true
  size: 20Gi
  accessModes:
    - ReadWriteOnce

# Qdrant конфигурация
config:
  service:
    http_port: 6333
    grpc_port: 6334
  storage:
    storage_path: /qdrant/storage
    snapshots_path: /qdrant/snapshots
    optimizers:
      indexing_threshold_kb: 20000
      memmap_threshold_kb: 50000
  log_level: INFO

# Ресурсы
resources:
  limits:
    memory: 4Gi
    cpu: 2000m
  requests:
    memory: 1Gi
    cpu: 500m

# Health checks
livenessProbe:
  httpGet:
    path: /healthz
    port: 6333
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz
    port: 6333
  initialDelaySeconds: 10
  periodSeconds: 5
EOF

# Установка Qdrant
helm install qdrant qdrant/qdrant \
  -f qdrant-values.yaml \
  --namespace rag-infrastructure \
  --version 0.8.2
```

**Connection string:**

```bash
http://qdrant.rag-infrastructure.svc.cluster.local:6333
```

### 5. MinIO (опционально)

**Helm чарт:** `bitnami/minio`

```bash
# Создание values файла
cat > minio-values.yaml <<EOF
auth:
  rootUser: minioadmin
  rootPassword: minioadmin123

# Режим: standalone или distributed
mode: standalone

# Persistence
persistence:
  enabled: true
  size: 50Gi

# Default buckets (создаются автоматически)
defaultBuckets: "user-files,rag-documents,embeddings-cache"

# Service
service:
  type: ClusterIP
  ports:
    api: 9000
    console: 9001

# Ресурсы
resources:
  limits:
    memory: 2Gi
    cpu: 1000m
  requests:
    memory: 512Mi
    cpu: 250m

# Ingress для MinIO Console (опционально)
ingress:
  enabled: false
  # Для доступа извне:
  # enabled: true
  # hostname: minio-console.example.com

# Metrics
metrics:
  serviceMonitor:
    enabled: true
EOF

# Установка MinIO
helm install minio bitnami/minio \
  -f minio-values.yaml \
  --namespace rag-infrastructure \
  --version 12.13.2
```

**Connection для S3 API:**

```bash
# Endpoint
http://minio.rag-infrastructure.svc.cluster.local:9000

# Access Key: minioadmin
# Secret Key: minioadmin123
```

**Настройка MinIO Client (mc) для создания buckets вручную:**

```bash
# Port forward для доступа
kubectl port-forward svc/minio 9000:9000 -n rag-infrastructure

# Установка mc
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
sudo mv mc /usr/local/bin/

# Настройка алиаса
mc alias set k8s-minio http://localhost:9000 minioadmin minioadmin123

# Создание buckets
mc mb k8s-minio/user-files
mc mb k8s-minio/rag-documents
mc mb k8s-minio/embeddings-cache

# Установка публичного доступа для user-files
mc policy set download k8s-minio/user-files
```

### 6. Prometheus + Grafana Stack

**Helm чарт:** `prometheus-community/kube-prometheus-stack`

```bash
# Создание values файла (огромный файл, основные настройки)
cat > monitoring-values.yaml <<EOF
# Prometheus
prometheus:
  prometheusSpec:
    retention: 30d
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
    
    resources:
      limits:
        memory: 2Gi
        cpu: 1000m
      requests:
        memory: 512Mi
        cpu: 250m
    
    # Дополнительные scrape configs для наших сервисов
    additionalScrapeConfigs:
      - job_name: 'postgres-exporter'
        static_configs:
          - targets: ['postgres-pgvector-postgresql-metrics.rag-infrastructure.svc.cluster.local:9187']
      
      - job_name: 'redis-exporter'
        static_configs:
          - targets: ['redis-metrics.rag-infrastructure.svc.cluster.local:9121']
      
      - job_name: 'minio'
        static_configs:
          - targets: ['minio.rag-infrastructure.svc.cluster.local:9000']
        metrics_path: /minio/v2/metrics/cluster

# Grafana
grafana:
  adminPassword: admin123
  
  persistence:
    enabled: true
    size: 10Gi
  
  resources:
    limits:
      memory: 1Gi
      cpu: 500m
    requests:
      memory: 256Mi
      cpu: 100m
  
  # Автоматическая установка дашбордов
  dashboardProviders:
    dashboardproviders.yaml:
      apiVersion: 1
      providers:
        - name: 'rag-infrastructure'
          orgId: 1
          folder: 'RAG Infrastructure'
          type: file
          disableDeletion: false
          editable: true
          options:
            path: /var/lib/grafana/dashboards/rag-infrastructure
  
  # Предустановленные дашборды
  dashboards:
    rag-infrastructure:
      postgresql:
        gnetId: 9628
        revision: 7
        datasource: Prometheus
      redis:
        gnetId: 11835
        revision: 1
        datasource: Prometheus
      minio:
        gnetId: 13502
        revision: 10
        datasource: Prometheus
  
  # Service
  service:
    type: ClusterIP
    port: 80
  
  # Ingress (опционально)
  ingress:
    enabled: false
    # Для доступа извне:
    # enabled: true
    # hosts:
    #   - grafana.example.com

# Alertmanager
alertmanager:
  alertmanagerSpec:
    storage:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 10Gi

# Отключение компонентов, которые не нужны
kubeStateMetrics:
  enabled: true

nodeExporter:
  enabled: true

prometheusOperator:
  enabled: true
EOF

# Установка всего стека мониторинга
helm install monitoring prometheus-community/kube-prometheus-stack \
  -f monitoring-values.yaml \
  --namespace rag-infrastructure \
  --version 55.5.0
```

**Доступ к Grafana:**

```bash
# Port forward
kubectl port-forward svc/monitoring-grafana 3000:80 -n rag-infrastructure

# Открыть в браузере: http://localhost:3000
# Login: admin
# Password: admin123
```

---

## Конфигурация компонентов

### Создание ConfigMap с общими настройками

```yaml
# rag-infrastructure-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-infrastructure-config
  namespace: rag-infrastructure
data:
  # PostgreSQL settings
  POSTGRES_HOST: "postgres-pgvector-postgresql.rag-infrastructure.svc.cluster.local"
  POSTGRES_PORT: "5432"
  POSTGRES_DATABASE: "ollama_proxy"
  
  # Redis settings
  REDIS_HOST: "redis-master.rag-infrastructure.svc.cluster.local"
  REDIS_PORT: "6379"
  
  # Qdrant settings
  QDRANT_HOST: "qdrant.rag-infrastructure.svc.cluster.local"
  QDRANT_HTTP_PORT: "6333"
  QDRANT_GRPC_PORT: "6334"
  
  # MinIO settings
  MINIO_ENDPOINT: "minio.rag-infrastructure.svc.cluster.local:9000"
  MINIO_USE_SSL: "false"
  
  # Prometheus settings
  PROMETHEUS_URL: "http://monitoring-prometheus.rag-infrastructure.svc.cluster.local:9090"
```

```bash
kubectl apply -f rag-infrastructure-config.yaml
```

### Создание Secret для паролей

```yaml
# rag-infrastructure-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: rag-infrastructure-secrets
  namespace: rag-infrastructure
type: Opaque
stringData:
  # PostgreSQL credentials
  postgres-username: "proxy_user"
  postgres-password: "secure_password_change_me"
  
  # Redis password (если включен auth)
  redis-password: ""
  
  # MinIO credentials
  minio-root-user: "minioadmin"
  minio-root-password: "minioadmin123"
  
  # Grafana admin password
  grafana-admin-password: "admin123"
```

```bash
kubectl apply -f rag-infrastructure-secrets.yaml
```

---

## Мониторинг и управление

### Проверка статуса всех компонентов

```bash
# Список всех pods
kubectl get pods -n rag-infrastructure

# Список всех services
kubectl get svc -n rag-infrastructure

# Список всех PVC (persistent volumes)
kubectl get pvc -n rag-infrastructure

# Статус всех Helm releases
helm list -n rag-infrastructure

# Детальная информация о конкретном release
helm status postgres-pgvector -n rag-infrastructure
```

### Логи компонентов

```bash
# PostgreSQL logs
kubectl logs -f deployment/postgres-pgvector-postgresql -n rag-infrastructure

# Redis logs
kubectl logs -f deployment/redis-master -n rag-infrastructure

# Qdrant logs
kubectl logs -f deployment/qdrant -n rag-infrastructure

# MinIO logs
kubectl logs -f deployment/minio -n rag-infrastructure

# Логи за последние 1 час
kubectl logs --since=1h deployment/postgres-pgvector-postgresql -n rag-infrastructure
```

### Port forwarding для локального доступа

```bash
# PostgreSQL
kubectl port-forward svc/postgres-pgvector-postgresql 5432:5432 -n rag-infrastructure

# pgAdmin
kubectl port-forward svc/pgadmin-pgadmin4 5050:80 -n rag-infrastructure

# Redis
kubectl port-forward svc/redis-master 6379:6379 -n rag-infrastructure

# Qdrant
kubectl port-forward svc/qdrant 6333:6333 6334:6334 -n rag-infrastructure

# MinIO Console
kubectl port-forward svc/minio 9000:9000 9001:9001 -n rag-infrastructure

# Grafana
kubectl port-forward svc/monitoring-grafana 3000:80 -n rag-infrastructure

# Prometheus
kubectl port-forward svc/monitoring-prometheus 9090:9090 -n rag-infrastructure
```

### Обновление компонентов

```bash
# Обновление values для компонента
helm upgrade postgres-pgvector bitnami/postgresql \
  -f postgres-values.yaml \
  --namespace rag-infrastructure

# Rollback к предыдущей версии
helm rollback postgres-pgvector -n rag-infrastructure

# История изменений
helm history postgres-pgvector -n rag-infrastructure
```

### Масштабирование

```bash
# Масштабирование Redis replicas (если используется replication)
helm upgrade redis bitnami/redis \
  --set replica.replicaCount=3 \
  --namespace rag-infrastructure

# Масштабирование Qdrant
kubectl scale deployment qdrant --replicas=3 -n rag-infrastructure
```

---

## Troubleshooting

### 1. Pod не запускается

```bash
# Проверить статус pod
kubectl describe pod <pod-name> -n rag-infrastructure

# Проверить events
kubectl get events -n rag-infrastructure --sort-by='.lastTimestamp'

# Логи init containers
kubectl logs <pod-name> -c <init-container-name> -n rag-infrastructure
```

### 2. Проблемы с persistent volumes

```bash
# Проверить PVC
kubectl get pvc -n rag-infrastructure
kubectl describe pvc <pvc-name> -n rag-infrastructure

# Проверить storage class
kubectl get storageclass
kubectl describe storageclass <storage-class-name>

# Проверить доступные PV
kubectl get pv
```

### 3. Проблемы с сетью

```bash
# Проверить services
kubectl get svc -n rag-infrastructure
kubectl describe svc <service-name> -n rag-infrastructure

# Проверить endpoints
kubectl get endpoints -n rag-infrastructure

# Проверить сетевую политику
kubectl get networkpolicies -n rag-infrastructure

# Тест подключения между pods
kubectl run test-pod --image=busybox -n rag-infrastructure --rm -it -- sh
# Внутри pod:
# nslookup postgres-pgvector-postgresql.rag-infrastructure.svc.cluster.local
# wget -O- postgres-pgvector-postgresql.rag-infrastructure.svc.cluster.local:5432
```

### 4. PostgreSQL не инициализируется с pgvector

```bash
# Проверить логи инициализации
kubectl logs postgres-pgvector-postgresql-0 -n rag-infrastructure | grep -i "pgvector\|init"

# Подключиться к PostgreSQL и проверить расширение
kubectl exec -it postgres-pgvector-postgresql-0 -n rag-infrastructure -- psql -U proxy_user -d ollama_proxy

# Внутри psql:
# \dx  -- список расширений
# SELECT * FROM pg_extension WHERE extname = 'vector';
```

### 5. MinIO buckets не создаются

```bash
# Проверить логи MinIO
kubectl logs deployment/minio -n rag-infrastructure

# Создать buckets вручную через mc
kubectl port-forward svc/minio 9000:9000 -n rag-infrastructure
mc alias set k8s-minio http://localhost:9000 minioadmin minioadmin123
mc mb k8s-minio/user-files
mc mb k8s-minio/rag-documents
mc mb k8s-minio/embeddings-cache
```

### 6. Helm release в состоянии pending-install или failed

```bash
# Удалить failed release
helm uninstall <release-name> -n rag-infrastructure

# Если не удаляется, принудительно
helm uninstall <release-name> -n rag-infrastructure --no-hooks

# Очистить namespace (ВНИМАНИЕ: удалит все данные!)
kubectl delete namespace rag-infrastructure
```

---

## Backup и Restore

### PostgreSQL Backup

```bash
# Создание backup job
cat > postgres-backup-job.yaml <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: rag-infrastructure
spec:
  schedule: "0 2 * * *"  # Каждый день в 2:00 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16
            env:
            - name: PGHOST
              value: "postgres-pgvector-postgresql"
            - name: PGUSER
              value: "proxy_user"
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: rag-infrastructure-secrets
                  key: postgres-password
            - name: PGDATABASE
              value: "ollama_proxy"
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -Fc > /backup/postgres-\$(date +%Y%m%d-%H%M%S).dump
            volumeMounts:
            - name: backup
              mountPath: /backup
          restartPolicy: OnFailure
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: postgres-backup-pvc
EOF

kubectl apply -f postgres-backup-job.yaml
```

### MinIO Backup

MinIO уже хранит данные в persistent volume. Для backup можно использовать:

```bash
# Синхронизация с внешним S3
mc mirror k8s-minio/rag-documents s3-external/rag-documents-backup
```

---

## Production Checklist

### Безопасность

- [ ] Включить authentication для Redis (`auth.enabled: true`)
- [ ] Использовать сильные пароли для всех компонентов
- [ ] Настроить Network Policies для ограничения трафика
- [ ] Включить TLS для внешних подключений
- [ ] Использовать Secrets вместо plaintext паролей
- [ ] Настроить RBAC для доступа к namespace

### Надежность

- [ ] Настроить replication для PostgreSQL и Redis
- [ ] Использовать высокодоступные storage classes
- [ ] Настроить resource limits и requests для всех pods
- [ ] Настроить PodDisruptionBudgets
- [ ] Настроить health checks для всех компонентов
- [ ] Настроить автоматические backups

### Мониторинг

- [ ] Настроить ServiceMonitors для всех компонентов
- [ ] Создать дашборды в Grafana
- [ ] Настроить алерты в Alertmanager
- [ ] Настроить логирование в централизованную систему (ELK, Loki)

### Производительность

- [ ] Настроить HorizontalPodAutoscaler для масштабирования
- [ ] Использовать fast storage (SSD) для баз данных
- [ ] Настроить node affinity для размещения на мощных нодах
- [ ] Оптимизировать параметры PostgreSQL и Redis
- [ ] Использовать connection pooling (PgBouncer)

---

## Дополнительные ресурсы

### Официальная документация

- [Helm Documentation](https://helm.sh/docs/)
- [Bitnami Charts](https://github.com/bitnami/charts)
- [Prometheus Operator](https://prometheus-operator.dev/)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [MinIO Documentation](https://min.io/docs/minio/kubernetes/upstream/)

### Useful Tools

- **K9s**: Терминальный UI для управления Kubernetes
  ```bash
  brew install derailed/k9s/k9s
  k9s -n rag-infrastructure
  ```

- **Stern**: Мультипод логирование
  ```bash
  brew install stern
  stern postgres -n rag-infrastructure
  ```

- **Kubectx/Kubens**: Быстрое переключение contexts и namespaces
  ```bash
  brew install kubectx
  kubens rag-infrastructure
  ```

---

## Примеры использования в приложении

### Go приложение с подключением к RAG infrastructure

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-redis/redis/v8"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
    ctx := context.Background()
    
    // PostgreSQL connection
    pgConnStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        os.Getenv("POSTGRES_USER"),
        os.Getenv("POSTGRES_PASSWORD"),
        os.Getenv("POSTGRES_HOST"),
        os.Getenv("POSTGRES_PORT"),
        os.Getenv("POSTGRES_DATABASE"),
    )
    
    pgPool, err := pgxpool.New(ctx, pgConnStr)
    if err != nil {
        panic(err)
    }
    defer pgPool.Close()
    
    // Redis connection
    redisClient := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       0,
    })
    defer redisClient.Close()
    
    // MinIO connection
    minioClient, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
        Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
        Secure: false,
    })
    if err != nil {
        panic(err)
    }
    
    // Использование...
    fmt.Println("Connected to RAG infrastructure!")
}
```

### Deployment для приложения

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rag-app
  namespace: rag-infrastructure
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rag-app
  template:
    metadata:
      labels:
        app: rag-app
    spec:
      containers:
      - name: rag-app
        image: your-registry/rag-app:latest
        ports:
        - containerPort: 8080
        env:
        # PostgreSQL settings
        - name: POSTGRES_HOST
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: POSTGRES_HOST
        - name: POSTGRES_PORT
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: POSTGRES_PORT
        - name: POSTGRES_DATABASE
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: POSTGRES_DATABASE
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: rag-infrastructure-secrets
              key: postgres-username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: rag-infrastructure-secrets
              key: postgres-password
        
        # Redis settings
        - name: REDIS_HOST
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: REDIS_HOST
        - name: REDIS_PORT
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: REDIS_PORT
        
        # MinIO settings
        - name: MINIO_ENDPOINT
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: MINIO_ENDPOINT
        - name: MINIO_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: rag-infrastructure-secrets
              key: minio-root-user
        - name: MINIO_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: rag-infrastructure-secrets
              key: minio-root-password
        
        # Qdrant settings
        - name: QDRANT_HOST
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: QDRANT_HOST
        - name: QDRANT_HTTP_PORT
          valueFrom:
            configMapKeyRef:
              name: rag-infrastructure-config
              key: QDRANT_HTTP_PORT
        
        resources:
          limits:
            memory: 1Gi
            cpu: 500m
          requests:
            memory: 256Mi
            cpu: 100m
```

---

## Заключение

Развертывание RAG infrastructure в Kubernetes с использованием Helm чартов предоставляет множество преимуществ:

- **Масштабируемость**: Легкое горизонтальное и вертикальное масштабирование
- **Надежность**: HA конфигурации, автоматический рестарт, health checks
- **Управляемость**: Декларативная конфигурация, version control, rollbacks
- **Мониторинг**: Встроенная интеграция с Prometheus/Grafana
- **Безопасность**: Network policies, RBAC, secrets management

Выбирайте подход в зависимости от ваших требований:
- **Быстрый старт**: Используйте Kompose/Katenary для конвертации docker-compose
- **Production**: Используйте официальные Helm чарты с детальной настройкой

Удачи в развертывании! 🚀

