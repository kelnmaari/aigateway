# SCALE-02: Load Balancing

**Приоритет:** MEDIUM  
**Версия:** 1.3.0  
**Оценка:** 10-12 часов  

---

## Цель

Балансировка запросов между несколькими Ollama серверами для high availability и масштабирования.

---

## Ключевые компоненты

### 1. Pool Manager (4 часа)

```go
type OllamaPool struct {
    servers   []*OllamaServer
    algorithm BalancingAlgorithm // round-robin, least-connections, weighted
}

type OllamaServer struct {
    URL          string
    Weight       int
    Active       bool
    Connections  atomic.Int32
    FailCount    int
}
```

### 2. Balancing Algorithms (3 часа)

- **Round Robin** - равномерное распределение
- **Least Connections** - на сервер с меньшей нагрузкой
- **Weighted** - с учетом capacity серверов
- **Sticky Sessions** - по API key (optional)

### 3. Configuration (1 час)

```yaml
ollama:
  servers:
    - url: http://ollama1:11434
      weight: 100
      max_connections: 50
    - url: http://ollama2:11434
      weight: 80
      max_connections: 40
  balancing:
    algorithm: least-connections
    health_check_interval: 10s
```

### 4. Metrics (1 час)

- Requests per server
- Active connections per server
- Server response times
- Failover events

### 5. Testing (3 часа)

- Load distribution verification
- Failover scenarios
- Performance benchmarks

---

**Dependencies:** SCALE-03 (Health Checks)  
**Priority in 1.3.0:** #1
