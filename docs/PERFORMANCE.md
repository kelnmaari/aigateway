# 🚀 Performance Tuning Guide

Complete guide to optimizing Ollama-OpenAI Proxy performance.

## 📑 Table of Contents

- [Overview](#overview)
- [Hardware Recommendations](#hardware-recommendations)
- [GPU Optimization](#gpu-optimization)
- [Ollama Configuration](#ollama-configuration)
- [Proxy Configuration](#proxy-configuration)
- [Model Selection](#model-selection)
- [Connection Pool Tuning](#connection-pool-tuning)
- [Timeout Optimization](#timeout-optimization)
- [Caching Strategies](#caching-strategies)
- [Rate Limiting](#rate-limiting)
- [Monitoring Performance](#monitoring-performance)
- [Benchmarking](#benchmarking)
- [Common Issues](#common-issues)

---

## Overview

This guide helps you optimize Ollama-OpenAI Proxy for maximum performance based on your hardware and use case.

### Performance Metrics

Key metrics to monitor:

- **Request Latency** - Time from request to response
- **Token Generation Speed** - Tokens per second
- **Concurrent Requests** - Number of simultaneous requests
- **Memory Usage** - RAM and VRAM consumption
- **Error Rate** - Percentage of failed requests

### Optimization Goals

- ⚡ Minimize response latency
- 📈 Maximize throughput
- 💾 Optimize memory usage
- 🛡️ Maintain stability under load

---

## Hardware Recommendations

### Minimum Requirements

**For 7B models:**
- **GPU:** 8GB VRAM (NVIDIA GTX 1070 or better)
- **RAM:** 16GB
- **CPU:** 4+ cores
- **Storage:** SSD recommended

**For 13B models:**
- **GPU:** 12GB VRAM (NVIDIA RTX 3060 or better)
- **RAM:** 32GB
- **CPU:** 6+ cores
- **Storage:** NVMe SSD recommended

**For 30B+ models:**
- **GPU:** 24GB VRAM (NVIDIA RTX 4090, A5000)
- **RAM:** 64GB
- **CPU:** 8+ cores
- **Storage:** NVMe SSD

### Recommended Hardware

#### Entry Level
- **GPU:** NVIDIA RTX 3060 (12GB)
- **Models:** Up to 13B, quantized 30B
- **Concurrent Users:** 1-2
- **Cost:** ~$300-400

#### Mid Range
- **GPU:** NVIDIA RTX 4070 Ti (12GB) or RTX 4080 (16GB)
- **Models:** Up to 30B (Q4), 70B (Q2)
- **Concurrent Users:** 3-5
- **Cost:** ~$800-1200

#### High End
- **GPU:** NVIDIA RTX 4090 (24GB)
- **Models:** Up to 70B (Q4), 120B (Q2)
- **Concurrent Users:** 5-10
- **Cost:** ~$1600-2000

#### Professional
- **GPU:** NVIDIA A100 (40GB/80GB) or H100
- **Models:** Any size, multiple models
- **Concurrent Users:** 10-50+
- **Cost:** $10,000+

---

## GPU Optimization

### NVIDIA GPU Settings

#### Check GPU Status
```bash
# View GPU information
nvidia-smi

# Monitor GPU in real-time
watch -n 1 nvidia-smi
```

#### Optimize GPU Memory

**Enable persistence mode (reduces initialization time):**
```bash
sudo nvidia-smi -pm 1
```

**Set power limit (optional, for cooler/quieter operation):**
```bash
# RTX 4090 (default 450W)
sudo nvidia-smi -pl 400  # Reduce to 400W

# Check power limit
nvidia-smi -q -d POWER
```

#### GPU Clock Settings

**Lock GPU clocks for consistent performance:**
```bash
# RTX 4090
sudo nvidia-smi -lgc 2400  # Lock GPU clock to 2400 MHz
sudo nvidia-smi -lmc 1313  # Lock memory clock to max

# Reset to auto
sudo nvidia-smi -rgc
sudo nvidia-smi -rmc
```

### Multi-GPU Setup

**Run multiple Ollama instances:**

```bash
# GPU 0
CUDA_VISIBLE_DEVICES=0 ollama serve --port 11434

# GPU 1
CUDA_VISIBLE_DEVICES=1 ollama serve --port 11435
```

**Configure proxy for load balancing:**
```yaml
# Not yet implemented - planned feature
ollama:
  urls:
    - "http://localhost:11434"
    - "http://localhost:11435"
  load_balancing: "round_robin"
```

---

## Ollama Configuration

### Context Window Optimization

**For NVIDIA RTX 4090 (24GB VRAM):**

```bash
# Create Modelfile
cat > Modelfile << 'EOF'
FROM llama3.1

# Context window (higher = more memory, slower)
PARAMETER num_ctx 32768

# Max tokens to generate
PARAMETER num_predict 4096

# Other parameters
PARAMETER temperature 0.7
PARAMETER top_p 0.9
PARAMETER top_k 40
EOF

# Create optimized model
ollama create llama3.1-32k -f Modelfile
```

**Recommended context sizes by model:**

| Model Size | VRAM | Context Window | num_predict |
|------------|------|----------------|-------------|
| 7B (Q4)    | 6GB  | 32768          | 4096        |
| 13B (Q4)   | 10GB | 16384          | 2048        |
| 30B (Q4)   | 20GB | 16384          | 2048        |
| 70B (Q4)   | 40GB | 8192           | 1024        |

**Trade-offs:**
- **Larger context** = More memory, slower, better for long conversations
- **Smaller context** = Less memory, faster, better for short conversations

### Model Quantization

**Quantization levels (quality vs speed):**

```bash
# Highest quality, slowest
ollama pull llama3.1:70b-q8_0     # 8-bit

# Balanced
ollama pull llama3.1:70b-q4_0     # 4-bit (recommended)
ollama pull llama3.1:70b-q4_k_m   # 4-bit K-quant medium

# Fastest, lower quality
ollama pull llama3.1:70b-q2_k     # 2-bit
```

**Recommendations:**
- **Q8_0** - For <13B models (minimal quality loss)
- **Q4_K_M** - For >13B models (best balance)
- **Q4_0** - For >30B models (good balance)
- **Q2_K** - For testing or very large models

### Keep Models Loaded

**Configure keep-alive:**
```bash
# Keep models in VRAM indefinitely
OLLAMA_KEEP_ALIVE=-1 ollama serve

# Keep for 1 hour
OLLAMA_KEEP_ALIVE=1h ollama serve

# Default: 5 minutes
ollama serve
```

**In configuration:**
```bash
# systemd service
[Service]
Environment="OLLAMA_KEEP_ALIVE=24h"
```

### Batch Size

**For higher throughput:**
```bash
# Increase batch size (default: 512)
OLLAMA_NUM_BATCH=1024 ollama serve

# For RTX 4090
OLLAMA_NUM_BATCH=2048 ollama serve
```

---

## Proxy Configuration

### Optimal Settings for RTX 4090

```yaml
# configs/production.yaml

server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: "5m"     # High for large models
  write_timeout: "5m"    # High for streaming
  idle_timeout: "120s"

ollama:
  url: "http://localhost:11434"
  timeout: "5m"           # High for first load
  retry_attempts: 3
  retry_delay: "2s"
  connection_pool_size: 50  # High for concurrency
  keep_alive: true        # Reuse connections

auth:
  rate_limiting:
    default_requests_per_minute: 100  # High for single user
    default_requests_per_hour: 5000

logging:
  level: "info"           # Less verbose in production
  format: "json"
  output: "file"

models:
  cache:
    enabled: true
    ttl: "30m"            # Long TTL for stable setup
    refresh_interval: "10m"

tools:
  fallback_model: "llama3.1"
  optimizer:
    enabled: true
    simplify_system_message: true
    max_tools_per_request: 14

metrics:
  enabled: true           # Monitor performance

tui:
  refresh_rate: "2s"      # Less frequent updates
```

### High-Traffic Configuration

```yaml
server:
  read_timeout: "2m"      # Lower for faster fail
  write_timeout: "2m"

ollama:
  timeout: "2m"
  connection_pool_size: 100  # Very high
  keep_alive: true

auth:
  rate_limiting:
    enabled: true
    default_requests_per_minute: 30   # Moderate per-key
    default_requests_per_hour: 1000
```

### Low-Latency Configuration

```yaml
server:
  read_timeout: "30s"     # Fast fail
  write_timeout: "30s"

ollama:
  timeout: "30s"
  retry_attempts: 1       # Don't retry
  connection_pool_size: 20  # Lower concurrency

models:
  cache:
    ttl: "5m"             # Frequent refresh
    refresh_interval: "1m"
```

---

## Model Selection

### Model Performance Comparison

**Tokens per second on RTX 4090:**

| Model | Size | Quant | Tokens/sec | Latency | Quality |
|-------|------|-------|------------|---------|---------|
| llama3.2:3b | 3B | Q4_0 | 150-200 | ⚡ Very Low | ⭐⭐⭐ Good |
| qwen2.5-coder:7b | 7B | Q4_K_M | 100-150 | ⚡ Low | ⭐⭐⭐⭐ Very Good |
| llama3.1:latest | 8B | Q4_0 | 80-120 | ⚡ Low | ⭐⭐⭐⭐ Very Good |
| mistral:latest | 7B | Q4_0 | 90-130 | ⚡ Low | ⭐⭐⭐⭐ Very Good |
| qwen2.5-coder:14b | 14B | Q4_K_M | 60-90 | 🔶 Medium | ⭐⭐⭐⭐⭐ Excellent |
| qwen2.5-coder:30b | 30B | Q4_K_M | 30-50 | 🔶 Medium | ⭐⭐⭐⭐⭐ Excellent |
| llama3.1:70b | 70B | Q4_0 | 15-25 | 🔴 High | ⭐⭐⭐⭐⭐ Excellent |

### Model Selection Guide

**For coding (fast iteration):**
```yaml
# Recommended
- qwen2.5-coder:7b   # Best balance
- qwen2.5-coder:14b  # Better quality
- qwen2.5-coder:30b  # Highest quality (if VRAM allows)
```

**For chat/general use:**
```yaml
- llama3.2:3b        # Very fast
- llama3.1:latest    # Good balance (8B)
- mistral:latest     # Alternative
```

**For function calling:**
```yaml
- llama3.1:latest    # Best support
- llama3.2:latest    # Improved
- mistral:latest     # Good support
```

**For embeddings:**
```yaml
- nomic-embed-text   # Fast, good quality
- all-minilm         # Very fast, lower quality
```

### Pre-loading Models

```bash
# Start Ollama
ollama serve &

# Pre-load models
ollama run llama3.1:latest <<< "Hello"
ollama run qwen2.5-coder:7b <<< "Test"

# Keep them loaded
OLLAMA_KEEP_ALIVE=-1 ollama serve
```

---

## Connection Pool Tuning

### HTTP Client Configuration

**Default configuration:**
```go
&http.Transport{
    MaxIdleConns:       100,  // Total idle connections
    MaxIdleConnsPerHost: 10,  // Idle connections per Ollama instance
    IdleConnTimeout:    90 * time.Second,
    MaxConnsPerHost:    0,    // Unlimited (use connection_pool_size)
}
```

**High-traffic tuning:**
```yaml
ollama:
  connection_pool_size: 100  # Max concurrent requests
  keep_alive: true           # Essential for performance
```

**Low-latency tuning:**
```yaml
ollama:
  connection_pool_size: 20   # Lower concurrency
  keep_alive: true
```

**Memory-constrained:**
```yaml
ollama:
  connection_pool_size: 5    # Minimal concurrency
  keep_alive: true
```

### Connection Pool Monitoring

```bash
# Check Prometheus metrics
curl http://localhost:8080/metrics | grep ollama_connections_active

# Expected values:
# - Low traffic: 0-5
# - Medium traffic: 5-20
# - High traffic: 20-100
```

---

## Timeout Optimization

### Timeout Hierarchy

```
User Request
  ↓ read_timeout (server)
HTTP Handler
  ↓ ollama.timeout
Ollama Client
  ↓ HTTP Transport Timeout
Ollama Server
```

### Timeout Configuration by Use Case

#### Fast Models (< 10B)
```yaml
server:
  read_timeout: "30s"
  write_timeout: "30s"

ollama:
  timeout: "30s"
```

#### Large Models (30B+)
```yaml
server:
  read_timeout: "5m"
  write_timeout: "5m"

ollama:
  timeout: "5m"
```

#### SSH Tunnel / Remote Ollama
```yaml
server:
  read_timeout: "10m"
  write_timeout: "10m"

ollama:
  timeout: "10m"
```

#### Streaming with Large Context
```yaml
server:
  write_timeout: "15m"  # Very high

ollama:
  timeout: "15m"
```

### Timeout Best Practices

1. **write_timeout ≥ ollama.timeout**
   - Prevents premature connection close

2. **First request timeout higher**
   - Model loading takes extra time
   - Pre-load models to avoid this

3. **Monitor timeout errors**
   ```bash
   grep "timeout\|deadline" logs/proxy-dev.log
   ```

---

## Caching Strategies

### Model List Caching

**Configuration:**
```yaml
models:
  cache:
    enabled: true
    ttl: "30m"            # Cache duration
    refresh_interval: "10m"  # Background refresh
```

**Tuning:**
- **Frequent model changes:** ttl: "1m", refresh: "30s"
- **Stable setup:** ttl: "1h", refresh: "30m"
- **No caching:** enabled: false

### Response Caching (Future)

**Planned feature:**
```yaml
# Not yet implemented
caching:
  enabled: true
  backend: "redis"
  url: "redis://localhost:6379"
  ttl: "1h"
  
  # Cache identical requests
  cache_identical_requests: true
  
  # Cache key includes:
  # - Model name
  # - Messages (hash)
  # - Parameters
```

---

## Rate Limiting

### Optimal Rate Limits

**Single user (development):**
```yaml
auth:
  rate_limiting:
    default_requests_per_minute: 100
    default_requests_per_hour: 5000
```

**Multiple users (production):**
```yaml
auth:
  rate_limiting:
    default_requests_per_minute: 30
    default_requests_per_hour: 500
```

**Public API:**
```yaml
auth:
  rate_limiting:
    default_requests_per_minute: 10
    default_requests_per_hour: 100
```

### Rate Limit vs Model Performance

**For fast models (7B):**
- Can handle 100+ req/min per user

**For slow models (70B):**
- Limit to 10-20 req/min per user
- Adjust based on actual response times

### Monitoring Rate Limits

```bash
# Check rate limit metrics
curl http://localhost:8080/metrics | grep rate_limit_exceeded

# Low count = good
# High count = limits too restrictive or abuse
```

---

## Monitoring Performance

### Key Metrics to Watch

#### 1. Request Latency

```bash
# Prometheus query
ollama_proxy_http_request_duration_seconds

# Target:
# - P50: < 1s (fast models) or < 5s (large models)
# - P95: < 5s (fast models) or < 30s (large models)
# - P99: < 10s (fast models) or < 60s (large models)
```

#### 2. Ollama Latency

```bash
ollama_proxy_ollama_request_duration_seconds

# Should be similar to HTTP latency
# Large difference = proxy overhead
```

#### 3. Error Rate

```bash
ollama_proxy_ollama_errors_total / ollama_proxy_ollama_requests_total

# Target: < 1% (0.01)
# Above 5% = investigate issues
```

#### 4. Token Usage

```bash
ollama_proxy_api_key_tokens_used_total

# Monitor for:
# - Unexpected spikes
# - Heavy users
# - Billing/quotas
```

### TUI Monitoring

```bash
./bin/tui.exe

# Check screens:
# 1 - Dashboard: Avg latency, request counts
# 6 - Logs: Error messages
# 7 - Control: Server status, statistics
```

### Grafana Dashboard

**Example Prometheus queries:**

```promql
# Request rate
rate(ollama_proxy_http_requests_total[5m])

# Average latency
rate(ollama_proxy_http_request_duration_seconds_sum[5m]) / 
rate(ollama_proxy_http_request_duration_seconds_count[5m])

# Error rate
rate(ollama_proxy_http_requests_total{status_code=~"5.."}[5m]) /
rate(ollama_proxy_http_requests_total[5m])

# Active connections
ollama_proxy_ollama_connections_active
```

---

## Benchmarking

### Load Testing

**Using Apache Bench:**
```bash
# Simple test
ab -n 100 -c 10 \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -p request.json \
  http://localhost:8080/v1/chat/completions

# request.json:
{
  "model": "llama3.1",
  "messages": [{"role": "user", "content": "Hello"}],
  "max_tokens": 50
}
```

**Using k6:**
```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 10 },  // Ramp-up
    { duration: '3m', target: 10 },  // Steady
    { duration: '1m', target: 0 },   // Ramp-down
  ],
};

export default function () {
  const payload = JSON.stringify({
    model: 'llama3.1',
    messages: [{ role: 'user', content: 'Hello' }],
    max_tokens: 50,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sk-your-key',
    },
  };

  const res = http.post('http://localhost:8080/v1/chat/completions', payload, params);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 5s': (r) => r.timings.duration < 5000,
  });
}
```

**Run test:**
```bash
k6 run load-test.js
```

### Performance Baselines

**RTX 4090 + qwen2.5-coder:7b:**
- Single request: ~500ms
- 10 concurrent: ~1.5s per request
- 50 concurrent: ~5s per request

**RTX 4090 + qwen2.5-coder:30b:**
- Single request: ~2s
- 10 concurrent: ~10s per request
- 20+ concurrent: Queue builds up

---

## Common Issues

### Slow First Request

**Problem:** First request takes 30-60 seconds.

**Cause:** Model loading into VRAM.

**Solution:**
```bash
# Pre-load model
ollama run llama3.1 <<< "warmup"

# Keep loaded
OLLAMA_KEEP_ALIVE=-1 ollama serve
```

### Memory Errors

**Problem:** "out of memory" errors.

**Cause:** Model too large for available VRAM.

**Solutions:**
1. Use smaller model
2. Use higher quantization (Q4 → Q2)
3. Reduce context window
4. Reduce batch size

### High Latency Under Load

**Problem:** Latency increases with concurrent requests.

**Cause:** Model serializes requests (processes one at a time).

**Solutions:**
1. Increase `OLLAMA_NUM_BATCH`
2. Use smaller/faster model
3. Add more GPUs
4. Implement request queuing

### Connection Timeouts

**Problem:** Requests timeout frequently.

**Cause:** Timeouts too low for model speed.

**Solution:**
```yaml
server:
  write_timeout: "10m"
ollama:
  timeout: "10m"
```

---

## Summary

### Quick Wins

1. **Pre-load models** - Eliminates first-request delay
2. **Enable keep-alive** - Reduces connection overhead
3. **Use appropriate quantization** - Q4_K_M for best balance
4. **Increase connection pool** - For concurrent requests
5. **Monitor metrics** - Identify bottlenecks

### Optimization Checklist

- [ ] GPU persistence mode enabled
- [ ] Optimal context window for use case
- [ ] Models pre-loaded (keep-alive)
- [ ] Connection pool sized for traffic
- [ ] Timeouts appropriate for model size
- [ ] Model list caching enabled
- [ ] Rate limits set appropriately
- [ ] Monitoring enabled (Prometheus/TUI)
- [ ] Logs reviewed regularly
- [ ] Load testing performed

---

## Additional Resources

- **[README.md](../README.md)** - Project overview
- **[CONFIGURATION.md](CONFIGURATION.md)** - Configuration guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architecture details
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Problem solving

---

**Need help optimizing?** [Open a discussion on GitHub](https://github.com/yourusername/ollama-openai-proxy/discussions)

