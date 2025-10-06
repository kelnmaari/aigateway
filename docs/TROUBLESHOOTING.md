# 🔧 Troubleshooting Guide

Solutions to common problems with Ollama-OpenAI Proxy.

## 📑 Table of Contents

- [Connection Issues](#connection-issues)
- [Authentication Problems](#authentication-problems)
- [Model Issues](#model-issues)
- [Performance Problems](#performance-problems)
- [Streaming Issues](#streaming-issues)
- [Function Calling Problems](#function-calling-problems)
- [TUI Issues](#tui-issues)
- [Logging Problems](#logging-problems)
- [Rate Limiting](#rate-limiting)
- [Error Messages](#error-messages)
- [Diagnostic Tools](#diagnostic-tools)

---

## Connection Issues

### Problem: "Connection refused" to proxy server

**Symptoms:**
```
Error: connect ECONNREFUSED 127.0.0.1:8080
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

**Diagnosis:**
```bash
# Check if server is running
curl http://localhost:8080/health

# Check listening ports
netstat -an | grep 8080  # Linux
netstat -an | findstr 8080  # Windows

# Check server logs
tail -f logs/proxy-dev.log
```

**Solutions:**

1. **Start the server**
   ```bash
   ./bin/server.exe
   ```

2. **Check port availability**
   ```bash
   # Windows
   netstat -ano | findstr :8080
   
   # Linux
   lsof -i :8080
   ```

3. **Change port if occupied**
   ```yaml
   # configs/dev.yaml
   server:
     port: 8081  # Use different port
   ```

4. **Check firewall**
   ```bash
   # Windows
   netsh advfirewall firewall add rule name="Ollama Proxy" dir=in action=allow protocol=TCP localport=8080
   
   # Linux
   sudo ufw allow 8080
   ```

---

### Problem: "Connection refused" to Ollama

**Symptoms:**
```
level=error msg="Failed to connect to Ollama" error="connection refused"
502 Bad Gateway: Ollama connection error
```

**Diagnosis:**
```bash
# Test Ollama directly
curl http://localhost:11434/api/tags

# Check Ollama status
ps aux | grep ollama  # Linux
tasklist | findstr ollama  # Windows
```

**Solutions:**

1. **Start Ollama**
   ```bash
   ollama serve
   ```

2. **Check Ollama URL**
   ```yaml
   # configs/dev.yaml
   ollama:
     url: "http://localhost:11434"  # Correct URL?
   ```

3. **Test Ollama connectivity**
   ```bash
   # List models
   ollama list
   
   # Test API
   curl http://localhost:11434/api/tags
   ```

4. **Remote Ollama server**
   ```yaml
   ollama:
     url: "http://192.168.1.100:11434"
     timeout: "5m"  # Higher timeout for network
   ```

5. **Check Ollama logs**
   ```bash
   # Linux
   journalctl -u ollama -f
   
   # Manual start with logs
   ollama serve
   ```

---

### Problem: Timeout errors

**Symptoms:**
```
level=error msg="Ollama timeout" error="context deadline exceeded"
504 Gateway Timeout
```

**Diagnosis:**
Check logs for timeout patterns:
```bash
grep -i "timeout\|deadline" logs/proxy-dev.log
```

**Solutions:**

1. **Increase timeouts**
   ```yaml
   # configs/dev.yaml
   server:
     read_timeout: "5m"
     write_timeout: "5m"
   
   ollama:
     timeout: "5m"
   ```

2. **First model load takes time**
   ```bash
   # Pre-load model
   ollama run llama3.1
   # Wait for model to load, then exit with Ctrl+D
   ```

3. **Large models need more time**
   ```yaml
   # For 30B+ models
   ollama:
     timeout: "10m"
   ```

4. **Over SSH tunnel**
   ```yaml
   ollama:
     timeout: "15m"  # Very high for SSH
   ```

---

## Authentication Problems

### Problem: "Invalid API key"

**Symptoms:**
```json
{
  "error": {
    "message": "Invalid API key provided",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

**Diagnosis:**
```bash
# Check Authorization header
curl -v http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-key"

# Check logs
grep "API key" logs/proxy-dev.log
```

**Solutions:**

1. **Check Authorization header format**
   ```bash
   # ✅ Correct
   Authorization: Bearer sk-your-api-key
   
   # ❌ Wrong
   Authorization: sk-your-api-key  # Missing "Bearer"
   Authorization: Bearer: sk-your-api-key  # Extra colon
   ```

2. **Verify key exists**
   ```bash
   # Launch TUI
   ./bin/tui.exe
   # Press 4 to check API Keys
   ```

3. **Create new key**
   ```bash
   # Via TUI
   ./bin/tui.exe
   # Press 4 → n → Fill form → Enter → Copy key
   ```

4. **Use admin key**
   ```yaml
   # configs/dev.yaml
   auth:
     admin_key: "sk-admin-dev-key-12345"
   ```

5. **Check key is enabled**
   ```bash
   # In TUI, check Status column
   # If disabled, delete and create new one
   ```

---

### Problem: "API key not found"

**Symptoms:**
```
level=warning msg="API key not found" key_prefix="sk-..."
401 Unauthorized
```

**Solutions:**

1. **Key might be deleted**
   - Create new key via TUI

2. **Storage file missing**
   ```bash
   # Check file exists
   ls -la data/api_keys.json
   
   # If missing, keys were deleted
   # Create new keys via TUI
   ```

3. **Permissions issue**
   ```bash
   # Check file permissions
   chmod 644 data/api_keys.json  # Linux
   ```

---

### Problem: "Model not allowed"

**Symptoms:**
```json
{
  "error": {
    "message": "API key does not have permission to access model 'llama3.1'",
    "type": "insufficient_quota",
    "code": "model_not_allowed"
  }
}
```

**Solutions:**

1. **Check key permissions**
   ```bash
   # In TUI → API Keys screen
   # Check "Permissions" column
   ```

2. **Key has specific model permissions**
   - Create new key with `*` (all models)
   - Or add required model to permissions

3. **Use admin key**
   - Admin key has access to all models

---

## Model Issues

### Problem: "Model not found"

**Symptoms:**
```json
{
  "error": {
    "message": "Model 'nonexistent' not found",
    "type": "invalid_request_error",
    "code": "model_not_found"
  }
}
```

**Diagnosis:**
```bash
# List available models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-key"

# Or via Ollama directly
ollama list
```

**Solutions:**

1. **Pull the model**
   ```bash
   ollama pull llama3.1
   ollama pull qwen2.5-coder:7b
   ```

2. **Use correct model name**
   ```bash
   # ✅ Correct
   "model": "qwen2.5-coder:7b"
   "model": "llama3.1:latest"
   
   # ❌ Wrong
   "model": "qwen2.5-coder"  # Missing tag
   "model": "gpt-4"  # OpenAI model, not Ollama
   ```

3. **Check model list**
   ```bash
   ollama list
   ```

---

### Problem: Model loads too slowly

**Symptoms:**
```
First request takes 30+ seconds
Subsequent requests are fast
```

**Solutions:**

1. **Pre-load model**
   ```bash
   # Keep model in memory
   ollama run llama3.1
   # Type /bye to exit but keep model loaded
   ```

2. **Increase Ollama keep-alive**
   ```bash
   # Keep models loaded longer
   OLLAMA_KEEP_ALIVE=24h ollama serve
   ```

3. **Smaller models load faster**
   ```bash
   # 7B models: ~5s load time
   # 30B models: ~30s load time
   # 70B models: ~60s+ load time
   ```

---

## Performance Problems

### Problem: Slow responses

**Symptoms:**
```
Requests take 10+ seconds
Dashboard shows high latency
```

**Diagnosis:**
```bash
# Check metrics
curl http://localhost:8080/metrics | grep duration

# Check TUI
./bin/tui.exe
# Press 1 (Dashboard) → Check "Avg Ollama Latency"
```

**Solutions:**

1. **Check system resources**
   ```bash
   # CPU usage
   top  # Linux
   taskmgr  # Windows
   
   # GPU usage
   nvidia-smi  # NVIDIA
   ```

2. **Reduce concurrent requests**
   ```yaml
   # configs/dev.yaml
   ollama:
     connection_pool_size: 5  # Lower concurrency
   ```

3. **Use smaller models**
   ```bash
   # Fast: 7B models
   qwen2.5-coder:7b
   llama3.2:3b
   
   # Slow: 30B+ models
   qwen2.5-coder:30b
   ```

4. **Optimize Ollama**
   ```bash
   # Set context window (for specific model)
   ollama run llama3.1
   /set parameter num_ctx 4096  # Smaller = faster
   ```

5. **Check GPU memory**
   ```bash
   nvidia-smi
   # If VRAM full, models swap to RAM (slow)
   ```

See [PERFORMANCE.md](PERFORMANCE.md) for detailed tuning.

---

### Problem: High memory usage

**Symptoms:**
```
RAM usage > 20GB
System becomes slow
```

**Solutions:**

1. **Unload unused models**
   ```bash
   # Check loaded models
   curl http://localhost:11434/api/ps
   
   # Models auto-unload after keep-alive expires
   ```

2. **Reduce connection pool**
   ```yaml
   ollama:
     connection_pool_size: 10  # Lower = less memory
   ```

3. **Use quantized models**
   ```bash
   # Lower memory usage
   ollama pull qwen2.5-coder:7b-q4_0  # 4-bit quantization
   ```

---

## Streaming Issues

### Problem: Streaming not working

**Symptoms:**
```
No incremental responses
All text arrives at once
Empty response
```

**Diagnosis:**
```bash
# Test streaming
curl -N -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

**Solutions:**

1. **Use `--no-buffer` or `-N` with curl**
   ```bash
   curl -N ...  # Disable buffering
   curl --no-buffer ...
   ```

2. **Check client library**
   ```python
   # Python (correct)
   response = client.chat.completions.create(
       model="llama3.1",
       messages=[...],
       stream=True  # ← Important
   )
   
   for chunk in response:
       print(chunk.choices[0].delta.content, end="")
   ```

3. **Content-Type must be correct**
   ```http
   Content-Type: application/json
   ```

4. **Increase timeouts**
   ```yaml
   server:
     write_timeout: "10m"  # Long timeout for streaming
   ```

---

### Problem: Streaming stops mid-response

**Symptoms:**
```
Response starts, then stops
Incomplete response
Connection closed
```

**Solutions:**

1. **Check timeouts**
   ```yaml
   server:
     write_timeout: "10m"
   ollama:
     timeout: "10m"
   ```

2. **Network issues**
   - Check network stability
   - Reduce response length
   - Try non-streaming first

3. **Model crashed**
   ```bash
   # Check Ollama logs
   ollama serve  # Look for errors
   ```

---

## Function Calling Problems

### Problem: Model returns text instead of tool calls

**Symptoms:**
```
Expected: {"tool_calls": [...]}
Got: "I'll use the get_weather function..."
```

**Diagnosis:**
```bash
# Check logs
grep -i "tool" logs/proxy-dev.log

# Test with known good model
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1",
    "messages": [{"role": "user", "content": "What'\''s the weather?"}],
    "tools": [...]
  }'
```

**Solutions:**

1. **Use tool-capable model**
   ```yaml
   # configs/dev.yaml
   tools:
     fallback_model: "llama3.1"  # Good tool support
   ```
   
   **Recommended models:**
   - `llama3.1` ✅ Best
   - `llama3.2` ✅ Good
   - `mistral` ✅ Good
   - `qwen2.5` ⚠️ Experimental

2. **Enable tool-aware routing**
   ```yaml
   tools:
     fallback_model: "llama3.1"
   ```

3. **Check tool format**
   ```json
   {
     "tools": [{
       "type": "function",
       "function": {
         "name": "get_weather",
         "description": "Get weather",  // ← Important!
         "parameters": {
           "type": "object",
           "properties": {...},
           "required": [...]
         }
       }
     }]
   }
   ```

4. **Enable optimizer**
   ```yaml
   tools:
     optimizer:
       enabled: true
       simplify_system_message: true
   ```

---

### Problem: Tool calls not detected by IDE

**Symptoms:**
```
IDE doesn't execute tools
Shows tool call as text
"Function calling not working"
```

**Solutions:**

1. **Check IDE configuration**
   ```json
   // Zed settings.json
   {
     "language_models": {
       "ollama": {
         "api_url": "http://localhost:8080/v1",
         "api_key": "sk-your-key"
       }
     }
   }
   ```

2. **Enable tool routing**
   ```yaml
   tools:
     fallback_model: "llama3.1"
   ```

3. **Check logs for tool detection**
   ```bash
   grep "tool_calls" logs/proxy-dev.log
   ```

4. **Test with curl first**
   - Verify tool calls work via API
   - Then troubleshoot IDE integration

---

## TUI Issues

### Problem: TUI doesn't start

**Symptoms:**
```
Error: connection refused
Blank screen
Crashes on start
```

**Solutions:**

1. **Ensure server is running**
   ```bash
   # Start server first
   ./bin/server.exe
   
   # Then start TUI
   ./bin/tui.exe
   ```

2. **Check server URL**
   ```bash
   # Default
   ./bin/tui.exe

   # Custom
   ./bin/tui.exe --server-url=http://remote:8080
   ```

3. **Terminal compatibility**
   - Use modern terminal (Windows Terminal, iTerm2, Alacritty)
   - Minimum size: 80x24 characters
   - Enable 256-color support

---

### Problem: TUI shows "Connection refused"

**Symptoms:**
Red error banner at top of TUI

**Solutions:**

1. **Server not running**
   ```bash
   ./bin/server.exe
   ```

2. **Wrong server URL**
   - Check TUI is connecting to correct URL
   - Default: `http://localhost:8080`

3. **Firewall blocking**
   - Allow port 8080

---

### Problem: TUI data not updating

**Symptoms:**
```
Statistics don't change
Old data shown
"Last Update" timestamp frozen
```

**Solutions:**

1. **Manual refresh**
   ```
   Press 'r' to refresh
   ```

2. **Check refresh rate**
   ```yaml
   tui:
     refresh_rate: "1s"  # Lower = faster updates
   ```

3. **Server not responding**
   ```bash
   curl http://localhost:8080/api/stats
   ```

---

## Logging Problems

### Problem: No log file created

**Symptoms:**
```
Expected: logs/proxy-dev.log
Got: File not found
```

**Solutions:**

1. **Create logs directory**
   ```bash
   mkdir -p logs
   ```

2. **Check output setting**
   ```yaml
   logging:
     output: "file"  # or "both"
     file_path: "logs/proxy-dev.log"
   ```

3. **Check permissions**
   ```bash
   # Ensure write permissions
   chmod 755 logs  # Linux
   ```

---

### Problem: Log file too large

**Symptoms:**
```
Log file > 1GB
Disk space running out
```

**Solutions:**

1. **Configure rotation**
   ```yaml
   logging:
     max_size: 100  # MB
     max_backups: 5
     max_age: 7  # days
     compress: true
   ```

2. **Lower log level**
   ```yaml
   logging:
     level: "info"  # Less verbose than "debug"
   ```

3. **Manual cleanup**
   ```bash
   # Delete old logs
   find logs/ -name "*.log.gz" -mtime +30 -delete
   ```

---

## Rate Limiting

### Problem: "Rate limit exceeded"

**Symptoms:**
```json
{
  "error": {
    "message": "Rate limit exceeded: 30 requests per minute",
    "type": "rate_limit_exceeded",
    "code": "rate_limit_exceeded"
  }
}
```

**Solutions:**

1. **Wait for reset**
   ```http
   # Check response headers
   X-RateLimit-Reset: 1699999999  # Unix timestamp
   Retry-After: 60  # Seconds to wait
   ```

2. **Create key with higher limits**
   ```bash
   # Via TUI
   ./bin/tui.exe
   # Press 4 → n
   # Set Rate Limits: 1000,10000
   ```

3. **Use admin key**
   ```yaml
   # Admin key has no rate limits
   auth:
     admin_key: "sk-admin-dev-key-12345"
   ```

4. **Disable rate limiting (dev only)**
   ```yaml
   auth:
     rate_limiting:
       enabled: false
   ```

---

## Error Messages

### HTTP 400 - Bad Request

**Common causes:**
- Missing required field (`model`, `messages`)
- Invalid JSON
- Malformed request

**Fix:**
```json
{
  "model": "llama3.1",  // ← Required
  "messages": [...]     // ← Required
}
```

---

### HTTP 401 - Unauthorized

**Common causes:**
- Missing Authorization header
- Invalid API key
- Malformed header

**Fix:**
```http
Authorization: Bearer sk-your-api-key
```

---

### HTTP 403 - Forbidden

**Common causes:**
- Model not allowed for API key
- Disabled API key

**Fix:**
- Create new key with required model permissions
- Or use admin key

---

### HTTP 429 - Too Many Requests

**Common causes:**
- Rate limit exceeded

**Fix:**
- Wait for reset (check `Retry-After` header)
- Create key with higher limits
- Use admin key

---

### HTTP 500 - Internal Server Error

**Common causes:**
- Server bug
- Panic/crash
- Database error

**Fix:**
1. Check server logs
2. Report issue with logs
3. Restart server

---

### HTTP 502 - Bad Gateway

**Common causes:**
- Ollama not running
- Ollama connection refused
- Wrong Ollama URL

**Fix:**
```bash
# Start Ollama
ollama serve

# Check connection
curl http://localhost:11434/api/tags
```

---

### HTTP 504 - Gateway Timeout

**Common causes:**
- Ollama request timeout
- Model loading took too long
- Network issues

**Fix:**
```yaml
ollama:
  timeout: "10m"  # Increase timeout
```

---

## Diagnostic Tools

### Check Server Health

```bash
# Health check
curl http://localhost:8080/health

# Expected: {"status":"ok"}
```

### Check Server Statistics

```bash
# Get stats
curl http://localhost:8080/api/stats | jq

# Check:
# - server.status: "running"
# - ollama.connected: true
# - stats.error_requests: low number
```

### Check Prometheus Metrics

```bash
# Get metrics
curl http://localhost:8080/metrics

# Key metrics:
# - ollama_proxy_http_requests_total
# - ollama_proxy_ollama_errors_total
# - ollama_proxy_api_keys_active_total
```

### Test API Endpoints

```bash
# List models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-key"

# Chat completion
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Check Logs

```bash
# Tail logs
tail -f logs/proxy-dev.log

# Search for errors
grep -i "error\|warning\|fatal" logs/proxy-dev.log

# Filter by component
grep "API key" logs/proxy-dev.log
grep "Ollama" logs/proxy-dev.log
```

### Debug Mode

```yaml
# configs/dev.yaml
logging:
  level: "debug"  # Maximum verbosity

development:
  debug_mode: true
```

Then check logs for detailed information.

---

## Getting Help

If you can't solve your problem:

1. **Check logs** for detailed error messages
2. **Search existing issues** on GitHub
3. **Open a new issue** with:
   - Problem description
   - Steps to reproduce
   - Relevant logs
   - Configuration (redact secrets!)
   - Environment (OS, Go version, Ollama version)

**GitHub Issues**: https://github.com/yourusername/ollama-openai-proxy/issues

---

## Additional Resources

- **[README.md](../README.md)** - Project overview
- **[API_DOCUMENTATION.md](API_DOCUMENTATION.md)** - API reference
- **[CONFIGURATION.md](CONFIGURATION.md)** - Configuration guide
- **[PERFORMANCE.md](PERFORMANCE.md)** - Performance tuning
- **[TUI_GUIDE.md](TUI_GUIDE.md)** - Terminal UI guide

---

**Still having issues?** Open an issue with full details and logs!

