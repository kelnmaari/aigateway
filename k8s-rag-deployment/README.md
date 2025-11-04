# RAG Infrastructure K8S Deployment - Решение проблемы Bitnami

## 🚨 Проблема

Bitnami (владелец Broadcom) с 28 августа 2025 года переместил старые Docker образы в **Bitnami Legacy Registry**. Старые образы больше не доступны по адресу `docker.io/bitnami/*` и не получают обновлений безопасности.

## ✅ Решение

Используем **официальные Docker образы** напрямую без Bitnami Helm чартов:
- **PostgreSQL + pgvector**: `pgvector/pgvector:pg16`
- **Redis**: `redis:7-alpine`

## 📦 Установленные компоненты

| Компонент | Образ | Внутренний порт | Внешний порт (через NGINX) |
|-----------|-------|-----------------|----------------------------|
| PostgreSQL + pgvector | pgvector/pgvector:pg16 | 5432 | 30432 → NodePort 32316 |
| Redis | redis:7-alpine | 6379 | 30379 → NodePort 30897 |

## 🔧 Установка

### 1. PostgreSQL + pgvector

```bash
kubectl apply -f postgres-statefulset.yaml
```

**Содержимое:**
- ConfigMap с init SQL скриптами (создание extension vector, таблиц, индексов)
- Secret с креденшалами
- StatefulSet с pgvector образом
- Services (headless + ClusterIP)

**Credentials:**
- User: `proxy_user`
- Password: `secure_password_change_me`
- Database: `ollama_proxy`

### 2. Redis

```bash
kubectl apply -f redis-statefulset.yaml
```

**Содержимое:**
- ConfigMap с redis.conf (persistence, memory management)
- StatefulSet с Redis Alpine образом
- Services (headless + ClusterIP)

**Features:**
- Persistence: AOF + RDB snapshots
- MaxMemory: 2GB с политикой allkeys-lru
- Без пароля (для совместимости с docker-compose)

### 3. NGINX Ingress TCP Services

```bash
# Обновить ConfigMap
kubectl apply -f nginx-ingress-tcp-patch.yaml

# Обновить Service (добавить порты)
kubectl patch svc ingress-nginx-1762145411-controller -n nginx-ingress --patch-file nginx-controller-service-patch.yaml
```

## 🌐 Доступ к сервисам

### Внутри кластера (из pods)

**PostgreSQL:**
```bash
postgres-service.rag-infrastructure.svc.cluster.local:5432
```

**Connection string:**
```
postgres://proxy_user:secure_password_change_me@postgres-service.rag-infrastructure.svc.cluster.local:5432/ollama_proxy?sslmode=disable
```

**Redis:**
```bash
redis-service.rag-infrastructure.svc.cluster.local:6379
```

### Извне кластера (через NGINX Ingress NodePort)

**PostgreSQL:**
```bash
# Через NodePort (замените <NODE_IP> на IP вашей ноды)
psql -h <NODE_IP> -p 32316 -U proxy_user -d ollama_proxy

# Пример
psql -h 192.168.1.101 -p 32316 -U proxy_user -d ollama_proxy
```

**Redis:**
```bash
# Через NodePort
redis-cli -h <NODE_IP> -p 30897

# Пример
redis-cli -h 192.168.1.101 -p 30897
```

### Port Forward (для локального тестирования)

**PostgreSQL:**
```bash
kubectl port-forward svc/postgres-service 5432:5432 -n rag-infrastructure
# Доступ: localhost:5432
```

**Redis:**
```bash
kubectl port-forward svc/redis-service 6379:6379 -n rag-infrastructure
# Доступ: localhost:6379
```

## 🔍 Проверка статуса

### Pods

```bash
kubectl get pods -n rag-infrastructure

# Должно быть:
# NAME         READY   STATUS    RESTARTS   AGE
# postgres-0   1/1     Running   0          xxm
# redis-0      1/1     Running   0          xxm
```

### Services

```bash
kubectl get svc -n rag-infrastructure

# NAME               TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
# postgres           ClusterIP   None            <none>        5432/TCP   xxm
# postgres-service   ClusterIP   10.96.222.50    <none>        5432/TCP   xxm
# redis              ClusterIP   None            <none>        6379/TCP   xxm
# redis-service      ClusterIP   10.101.35.211   <none>        6379/TCP   xxm
```

### Persistent Volume Claims

```bash
kubectl get pvc -n rag-infrastructure

# NAME                  STATUS   VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS
# postgres-data-postgres-0   Bound    pvc-xxx  20Gi       RWO            local-path
# redis-data-redis-0         Bound    pvc-yyy  8Gi        RWO            local-path
```

### NGINX Ingress TCP ConfigMap

```bash
kubectl get configmap tcp-services -n nginx-ingress -o yaml

# data:
#   "9000": nginx-ingress/ingress-nginx-1762145411-defaultbackend:80
#   "30379": rag-infrastructure/redis-service:6379
#   "30432": rag-infrastructure/postgres-service:5432
```

### NGINX Ingress Controller Ports

```bash
kubectl get svc ingress-nginx-1762145411-controller -n nginx-ingress

# TYPE           CLUSTER-IP    PORT(S)
# LoadBalancer   10.111.76.6   80:31917/TCP,443:31984/TCP,30432:32316/TCP,30379:30897/TCP
```

## 🧪 Тестирование подключений

### PostgreSQL

```bash
# 1. Запустить test pod
kubectl run postgres-test --rm -it --restart=Never \
  --image=postgres:16 \
  --namespace=rag-infrastructure \
  --command -- bash

# 2. Внутри pod'а подключиться к PostgreSQL
psql -h postgres-service -U proxy_user -d ollama_proxy

# 3. Проверить pgvector extension
\dx
SELECT * FROM pg_extension WHERE extname = 'vector';

# 4. Проверить таблицы
\dt
SELECT * FROM document_embeddings LIMIT 5;
```

### Redis

```bash
# 1. Запустить test pod
kubectl run redis-test --rm -it --restart=Never \
  --image=redis:7-alpine \
  --namespace=rag-infrastructure \
  --command -- sh

# 2. Внутри pod'а подключиться к Redis
redis-cli -h redis-service

# 3. Проверить Redis
PING
INFO server
DBSIZE
SET test:key "test-value"
GET test:key
```

## 📝 Логи

```bash
# PostgreSQL logs
kubectl logs -f postgres-0 -n rag-infrastructure

# Redis logs
kubectl logs -f redis-0 -n rag-infrastructure

# NGINX Ingress Controller logs
kubectl logs -f deployment/ingress-nginx-1762145411-controller -n nginx-ingress
```

## 🗑️ Удаление

### Удалить PostgreSQL

```bash
kubectl delete -f postgres-statefulset.yaml

# Удалить PVC (данные будут потеряны!)
kubectl delete pvc postgres-data-postgres-0 -n rag-infrastructure
```

### Удалить Redis

```bash
kubectl delete -f redis-statefulset.yaml

# Удалить PVC (данные будут потеряны!)
kubectl delete pvc redis-data-redis-0 -n rag-infrastructure
```

### Удалить namespace (всё!)

```bash
kubectl delete namespace rag-infrastructure
```

## 🔄 Обновление

### PostgreSQL

```bash
# Изменить postgres-statefulset.yaml (например, image tag)
# Применить изменения
kubectl apply -f postgres-statefulset.yaml

# Или patch напрямую
kubectl patch statefulset postgres -n rag-infrastructure \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"postgres","image":"pgvector/pgvector:pg17"}]}}}}'
```

### Redis

```bash
# Аналогично PostgreSQL
kubectl apply -f redis-statefulset.yaml
```

## 🔐 Изменение паролей

### PostgreSQL

```bash
# 1. Обновить Secret
kubectl edit secret postgres-secret -n rag-infrastructure

# 2. Restart StatefulSet
kubectl rollout restart statefulset/postgres -n rag-infrastructure
```

## 📊 Мониторинг

### Resource Usage

```bash
# CPU и Memory usage
kubectl top pods -n rag-infrastructure

# События
kubectl get events -n rag-infrastructure --sort-by='.lastTimestamp'
```

### Disk Usage

```bash
# PVC disk usage через df
kubectl exec -it postgres-0 -n rag-infrastructure -- df -h /var/lib/postgresql/data
kubectl exec -it redis-0 -n rag-infrastructure -- df -h /data
```

## 🚀 Следующие шаги

### 1. Добавить pgAdmin (опционально)

```bash
# TODO: Создать pgadmin-deployment.yaml
# Презентовать через NGINX Ingress HTTP (не TCP)
```

### 2. Добавить MinIO (опционально)

```bash
# Использовать официальный образ minio/minio:latest
# TODO: Создать minio-statefulset.yaml
```

### 3. Добавить Qdrant (опционально)

```bash
# Использовать официальный образ qdrant/qdrant:latest
# TODO: Создать qdrant-statefulset.yaml
```

### 4. Мониторинг с Prometheus + Grafana

```bash
# Установить kube-prometheus-stack (не зависит от Bitnami)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install monitoring prometheus-community/kube-prometheus-stack \
  --namespace rag-infrastructure \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

## 📚 Справка

### Полезные команды

```bash
# Все ресурсы в namespace
kubectl get all -n rag-infrastructure

# Describe pod для troubleshooting
kubectl describe pod postgres-0 -n rag-infrastructure

# Shell в pod
kubectl exec -it postgres-0 -n rag-infrastructure -- bash
kubectl exec -it redis-0 -n rag-infrastructure -- sh

# Copy файлов из/в pod
kubectl cp my-file.sql rag-infrastructure/postgres-0:/tmp/my-file.sql
kubectl cp rag-infrastructure/redis-0:/data/dump.rdb ./dump.rdb
```

### Connection Strings для приложения

**Go (pgx):**
```go
connStr := "postgres://proxy_user:secure_password_change_me@postgres-service.rag-infrastructure.svc.cluster.local:5432/ollama_proxy?sslmode=disable"
```

**Go (go-redis):**
```go
redisClient := redis.NewClient(&redis.Options{
    Addr: "redis-service.rag-infrastructure.svc.cluster.local:6379",
})
```

**Python (psycopg2):**
```python
conn = psycopg2.connect(
    host="postgres-service.rag-infrastructure.svc.cluster.local",
    port=5432,
    database="ollama_proxy",
    user="proxy_user",
    password="secure_password_change_me"
)
```

**Python (redis-py):**
```python
r = redis.Redis(
    host='redis-service.rag-infrastructure.svc.cluster.local',
    port=6379,
    decode_responses=True
)
```

## ⚠️ Важные замечания

1. **Persistence**: Данные хранятся в PVC. При удалении StatefulSet данные сохраняются. При удалении PVC данные теряются.

2. **Passwords**: Пароли хранятся в Kubernetes Secrets. Для production используйте внешние системы управления секретами (Vault, Sealed Secrets).

3. **Backups**: Настройте регулярные бэкапы PostgreSQL и Redis через CronJobs.

4. **Security**: 
   - Включите TLS для PostgreSQL
   - Включите auth для Redis (установить пароль)
   - Используйте Network Policies для ограничения трафика

5. **Performance**: 
   - Используйте быстрые storage классы (SSD) для production
   - Настройте resource limits/requests под вашу нагрузку
   - Рассмотрите replication для HA

6. **Bitnami Alternatives**: 
   - PostgreSQL: официальный образ `postgres:16` + pgvector extension
   - Redis: официальный образ `redis:7-alpine`
   - MinIO: официальный образ `minio/minio:latest`
   - Qdrant: официальный образ `qdrant/qdrant:latest`

## 📞 Troubleshooting

### Pod не запускается

```bash
kubectl describe pod <pod-name> -n rag-infrastructure
kubectl logs <pod-name> -n rag-infrastructure
```

### PVC в состоянии Pending

```bash
kubectl describe pvc <pvc-name> -n rag-infrastructure
kubectl get storageclass
```

### Нет доступа через NGINX Ingress

```bash
# Проверить ConfigMap
kubectl get configmap tcp-services -n nginx-ingress -o yaml

# Проверить Service ports
kubectl get svc ingress-nginx-1762145411-controller -n nginx-ingress

# Проверить NGINX Ingress Controller logs
kubectl logs -f deployment/ingress-nginx-1762145411-controller -n nginx-ingress
```

---

**Автор**: AI Assistant  
**Дата**: 2025-11-03  
**Версия**: 1.0

