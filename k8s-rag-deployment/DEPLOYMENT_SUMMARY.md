# RAG Infrastructure K8S Deployment - Итоги

## ✅ Что установлено

### 1. Namespace
```
rag-infrastructure
```

### 2. PostgreSQL + pgvector

**StatefulSet:** `postgres`
- **Image:** `pgvector/pgvector:pg16`
- **Replicas:** 1
- **Resources:**
  - Requests: 2Gi RAM, 1 CPU
  - Limits: 4Gi RAM, 2 CPU
- **Storage:** 20Gi PVC (local-path)
- **Services:**
  - `postgres` (ClusterIP None - headless)
  - `postgres-service` (ClusterIP: 10.96.222.50:5432)

**Features:**
- ✅ pgvector extension установлено
- ✅ Таблица `document_embeddings` создана с vector индексами
- ✅ Init SQL скрипты из ConfigMap
- ✅ Health checks (liveness + readiness)
- ✅ Persistence через PVC

**Credentials:**
```
User: proxy_user
Password: secure_password_change_me
Database: ollama_proxy
```

### 3. Redis

**StatefulSet:** `redis`
- **Image:** `redis:7-alpine`
- **Replicas:** 1
- **Resources:**
  - Requests: 512Mi RAM, 250m CPU
  - Limits: 2Gi RAM, 1 CPU
- **Storage:** 8Gi PVC (local-path)
- **Services:**
  - `redis` (ClusterIP None - headless)
  - `redis-service` (ClusterIP: 10.101.35.211:6379)

**Features:**
- ✅ AOF persistence + RDB snapshots
- ✅ MaxMemory 2GB с политикой allkeys-lru
- ✅ Custom redis.conf из ConfigMap
- ✅ Health checks (liveness + readiness)
- ✅ Persistence через PVC
- ✅ Без пароля (для совместимости)

### 4. NGINX Ingress TCP Services

**ConfigMap обновлен:** `tcp-services` в namespace `nginx-ingress`

**Маршрутизация:**
```yaml
"30432": rag-infrastructure/postgres-service:5432  # PostgreSQL
"30379": rag-infrastructure/redis-service:6379     # Redis
```

**Service обновлен:** `ingress-nginx-1762145411-controller`

**Порты:**
```
PORT(S)
80:31917/TCP      # HTTP
443:31984/TCP     # HTTPS
30432:32316/TCP   # PostgreSQL TCP
30379:30897/TCP   # Redis TCP
```

## 🌐 Доступ

### Внутри кластера (для ваших приложений)

**PostgreSQL:**
```
Host: postgres-service.rag-infrastructure.svc.cluster.local
Port: 5432

Connection String:
postgres://proxy_user:secure_password_change_me@postgres-service.rag-infrastructure.svc.cluster.local:5432/ollama_proxy?sslmode=disable
```

**Redis:**
```
Host: redis-service.rag-infrastructure.svc.cluster.local
Port: 6379

Connection String:
redis://redis-service.rag-infrastructure.svc.cluster.local:6379/0
```

### Извне кластера (через NGINX Ingress NodePort)

**PostgreSQL:**
```bash
# Через NodePort (IP вашей K8S ноды)
psql -h 192.168.1.101 -p 32316 -U proxy_user -d ollama_proxy

# Или используйте LoadBalancer External IP (если есть)
psql -h <EXTERNAL-IP> -p 30432 -U proxy_user -d ollama_proxy
```

**Redis:**
```bash
# Через NodePort
redis-cli -h 192.168.1.101 -p 30897

# Или через LoadBalancer External IP
redis-cli -h <EXTERNAL-IP> -p 30379
```

### Port Forward для локального тестирования

```bash
# PostgreSQL
kubectl port-forward svc/postgres-service 5432:5432 -n rag-infrastructure
# Доступ: localhost:5432

# Redis
kubectl port-forward svc/redis-service 6379:6379 -n rag-infrastructure
# Доступ: localhost:6379
```

## 📊 Текущий статус

```
NAME             READY   STATUS    RESTARTS   AGE
pod/postgres-0   1/1     Running   0          2m29s
pod/redis-0      1/1     Running   0          2m19s

NAME                       TYPE        CLUSTER-IP      PORT(S)
service/postgres           ClusterIP   None            5432/TCP
service/postgres-service   ClusterIP   10.96.222.50    5432/TCP
service/redis              ClusterIP   None            6379/TCP
service/redis-service      ClusterIP   10.101.35.211   6379/TCP

NAME                        READY   AGE
statefulset.apps/postgres   1/1     2m29s
statefulset.apps/redis      1/1     2m19s
```

## 🔧 Решенная проблема

### Bitnami Docker Images Migration

**Проблема:**
- Bitnami переместил старые образы в Legacy Registry (28 августа 2025)
- Старые образы больше не доступны по `docker.io/bitnami/*`
- Образы не получают обновлений безопасности
- Helm чарты `bitnami/postgresql` и `bitnami/redis` не работают с новыми образами

**Решение:**
- Используем **официальные Docker образы** напрямую
- Создали Kubernetes манифесты (StatefulSets) вместо Helm чартов
- PostgreSQL: `pgvector/pgvector:pg16`
- Redis: `redis:7-alpine`

**Преимущества:**
- ✅ Работающие образы без зависимости от Bitnami
- ✅ Актуальные версии с обновлениями безопасности
- ✅ Полный контроль над конфигурацией
- ✅ Легкая кастомизация через ConfigMaps
- ✅ Прозрачные манифесты без магии Helm чартов

## 📁 Структура файлов

```
k8s-rag-deployment/
├── README.md                           # Документация по использованию
├── DEPLOYMENT_SUMMARY.md               # Этот файл
├── postgres-statefulset.yaml           # PostgreSQL + pgvector манифест
├── redis-statefulset.yaml              # Redis манифест
├── nginx-ingress-tcp-patch.yaml        # Патч TCP ConfigMap
├── nginx-controller-service-patch.yaml # Патч NGINX Service
├── nginx-controller-service-backup.yaml # Бэкап оригинального Service
│
├── postgres-values.yaml                # (Deprecated) Bitnami Helm values
└── redis-values.yaml                   # (Deprecated) Bitnami Helm values
```

## 🚀 Следующие шаги

### 1. Тестирование подключений

```bash
# PostgreSQL test
kubectl run postgres-test --rm -it --restart=Never \
  --image=postgres:16 \
  --namespace=rag-infrastructure \
  --command -- psql -h postgres-service -U proxy_user -d ollama_proxy

# Redis test
kubectl run redis-test --rm -it --restart=Never \
  --image=redis:7-alpine \
  --namespace=rag-infrastructure \
  --command -- redis-cli -h redis-service PING
```

### 2. Добавить pgAdmin (опционально)

```bash
# TODO: Создать pgadmin-deployment.yaml
# Использовать официальный образ dpage/pgadmin4
# Презентовать через NGINX Ingress HTTP (не TCP)
```

### 3. Добавить MinIO (опционально)

```bash
# TODO: Создать minio-statefulset.yaml
# Использовать официальный образ minio/minio:latest
# S3-совместимое хранилище для файлов
```

### 4. Добавить Qdrant (опционально)

```bash
# TODO: Создать qdrant-statefulset.yaml
# Использовать официальный образ qdrant/qdrant:latest
# Альтернативный vector database
```

### 5. Настроить мониторинг

```bash
# Prometheus + Grafana стек (не зависит от Bitnami)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install monitoring prometheus-community/kube-prometheus-stack \
  --namespace rag-infrastructure \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

### 6. Настроить бэкапы

```bash
# TODO: Создать CronJobs для регулярных бэкапов
# - PostgreSQL: pg_dump → S3/MinIO
# - Redis: BGSAVE → S3/MinIO
```

### 7. Production hardening

- [ ] Включить TLS для PostgreSQL
- [ ] Добавить пароль для Redis
- [ ] Настроить Network Policies
- [ ] Использовать External Secrets Operator для управления секретами
- [ ] Настроить replication для HA
- [ ] Использовать fast storage class (SSD)
- [ ] Настроить resource quotas и limits
- [ ] Добавить PodDisruptionBudgets

## 📝 Полезные команды

```bash
# Статус всего в namespace
kubectl get all -n rag-infrastructure

# Логи
kubectl logs -f postgres-0 -n rag-infrastructure
kubectl logs -f redis-0 -n rag-infrastructure

# Shell в контейнер
kubectl exec -it postgres-0 -n rag-infrastructure -- bash
kubectl exec -it redis-0 -n rag-infrastructure -- sh

# Describe для troubleshooting
kubectl describe pod postgres-0 -n rag-infrastructure
kubectl describe pvc postgres-data-postgres-0 -n rag-infrastructure

# Resource usage
kubectl top pods -n rag-infrastructure

# События
kubectl get events -n rag-infrastructure --sort-by='.lastTimestamp'
```

## 🗂️ Бэкапы и восстановление

### PostgreSQL Backup

```bash
# Manual backup
kubectl exec postgres-0 -n rag-infrastructure -- \
  pg_dump -U proxy_user -d ollama_proxy -Fc > backup-$(date +%Y%m%d).dump

# Restore
kubectl cp backup-20251103.dump rag-infrastructure/postgres-0:/tmp/
kubectl exec -it postgres-0 -n rag-infrastructure -- \
  pg_restore -U proxy_user -d ollama_proxy -c /tmp/backup-20251103.dump
```

### Redis Backup

```bash
# Redis автоматически создает RDB dumps в /data/dump.rdb
# Manual backup
kubectl exec redis-0 -n rag-infrastructure -- redis-cli BGSAVE
kubectl cp rag-infrastructure/redis-0:/data/dump.rdb ./redis-backup-$(date +%Y%m%d).rdb

# Restore (скопировать dump.rdb в PVC и перезапустить)
kubectl cp redis-backup-20251103.rdb rag-infrastructure/redis-0:/data/dump.rdb
kubectl delete pod redis-0 -n rag-infrastructure  # StatefulSet пересоздаст
```

## ⚠️ Важные замечания

1. **Без Helm**: Используем чистые Kubernetes манифесты вместо Bitnami Helm чартов
2. **Официальные образы**: Только проверенные и актуальные образы от upstream разработчиков
3. **Persistence**: Данные сохраняются в PVC даже при удалении pods
4. **Security**: Пароли в Kubernetes Secrets (для production используйте Vault/Sealed Secrets)
5. **NGINX Ingress**: TCP сервисы презентуются через NodePort'ы

## 📚 Документация

- [README.md](README.md) - Детальная документация по использованию
- [K8S_HELM_DEPLOYMENT.md](../docs/K8S_HELM_DEPLOYMENT.md) - Общая документация по Helm (background info)

---

**Статус:** ✅ Deployed и работает  
**Дата:** 2025-11-03  
**K8S Cluster:** 192.168.1.101:6443  
**Namespace:** rag-infrastructure  
**Components:** PostgreSQL + pgvector, Redis  
**Access:** NGINX Ingress TCP (NodePort)

