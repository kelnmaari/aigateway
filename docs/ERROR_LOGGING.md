# Error Logging Configuration

## Overview

The system supports **separate error log files** in addition to the main log file. This allows you to:
- Quickly find errors and warnings without searching through all logs
- Keep error logs longer than regular logs
- Monitor critical issues more easily
- Set up separate alerting for error files

## How It Works

When enabled, the system uses a **Logrus Hook** that intercepts messages at specific levels:
- `ERROR` - Application errors
- `WARN` - Warnings that might indicate problems
- `FATAL` - Fatal errors (also in main log)
- `PANIC` - Panic events (also in main log)

These messages are written to **both**:
1. Main log file (if file logging is enabled)
2. Separate error log file

## Configuration

### Development (`configs/dev.yaml`)

```yaml
logging:
  level: "debug"
  format: "text"
  output: "both"  # Console + file
  file_path: "logs/proxy-dev.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true
  
  # Separate error log (errors and warnings only)
  error_log_enabled: true
  error_log_file_path: "logs/proxy-errors.log"
  error_log_max_size: 50      # MB (smaller than main log)
  error_log_max_backups: 10   # Keep more error backups
  error_log_max_age: 90       # days (keep errors longer)
  error_log_compress: true
```

### Production (`configs/production.yaml`)

```yaml
logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/app/logs/proxy.log"
  max_size: 100
  max_backups: 10
  max_age: 90
  compress: true
  
  # Separate error log (recommended for production)
  error_log_enabled: true
  error_log_file_path: "/app/logs/proxy-errors.log"
  error_log_max_size: 50
  error_log_max_backups: 20   # More backups in production
  error_log_max_age: 180      # 6 months
  error_log_compress: true
```

## Features

### 1. **Automatic Rotation**
Error log files rotate automatically using `lumberjack`:
- When file reaches `error_log_max_size`
- Old files are compressed if `error_log_compress: true`
- Keeps `error_log_max_backups` backups
- Deletes files older than `error_log_max_age` days

### 2. **Startup Rotation**
On server start, existing log files are renamed to `*-previous.log`:
```
proxy-errors.log          → proxy-errors-previous.log
proxy-errors.log (new)    → created fresh
```

### 3. **Defaults**
If error log settings are not specified, they inherit from main log config:
```go
error_log_max_size    = max_size    (if not set)
error_log_max_backups = max_backups (if not set)
error_log_max_age     = max_age     (if not set)
error_log_compress    = compress    (if not set)
```

### 4. **Same Format**
Error log uses the same format as main log (text or JSON):
```json
{
  "timestamp": "2025-10-27T04:50:00.000Z",
  "level": "error",
  "message": "Failed to connect to Ollama",
  "service": "ollama-openai-proxy",
  "version": "1.13.0",
  "error": "connection refused"
}
```

## Usage

### Enable Error Logging

```yaml
logging:
  error_log_enabled: true
  error_log_file_path: "logs/errors.log"
```

### Disable Error Logging

```yaml
logging:
  error_log_enabled: false
```

Or simply omit the `error_log_enabled` field (defaults to `false`).

## File Locations

### Development
```
logs/
├── proxy-dev.log              # All logs (debug, info, warn, error)
├── proxy-dev-previous.log     # Previous run
├── proxy-errors.log           # Errors & warnings only
├── proxy-errors-previous.log  # Previous errors
├── proxy-errors-1.log.gz      # Rotated backup
├── proxy-errors-2.log.gz
└── ...
```

### Production
```
/app/logs/
├── proxy.log                  # All logs (info, warn, error)
├── proxy-previous.log
├── proxy-errors.log           # Errors & warnings only
├── proxy-errors-previous.log
├── proxy-errors-1.log.gz
└── ...
```

## Monitoring & Alerting

### Watch for New Errors

```bash
# Development
tail -f logs/proxy-errors.log

# Production
tail -f /app/logs/proxy-errors.log
```

### Count Errors

```bash
# Errors in last hour
grep "level=error" logs/proxy-errors.log | grep "$(date -u '+%Y-%m-%d %H')" | wc -l

# Warnings in last hour
grep "level=warning" logs/proxy-errors.log | grep "$(date -u '+%Y-%m-%d %H')" | wc -l
```

### Alert on Errors (Production)

```bash
#!/bin/bash
# alert_on_errors.sh

ERROR_LOG="/app/logs/proxy-errors.log"
THRESHOLD=10
LAST_HOUR=$(date -u '+%Y-%m-%d %H')

ERROR_COUNT=$(grep "level=error" "$ERROR_LOG" | grep "$LAST_HOUR" | wc -l)

if [ "$ERROR_COUNT" -gt "$THRESHOLD" ]; then
    echo "ALERT: $ERROR_COUNT errors in the last hour!" | mail -s "Ollama Proxy Errors" admin@example.com
fi
```

### Logrotate Integration (Optional)

If you prefer `logrotate` over `lumberjack`:

```
/app/logs/proxy-errors.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 app app
    postrotate
        killall -SIGHUP ollama-proxy
    endscript
}
```

## Best Practices

### Development
- Keep `error_log_enabled: true` to catch issues early
- Review `proxy-errors.log` daily
- Fix warnings before they become errors

### Production
- **Always enable** error logging
- Set longer retention: `error_log_max_age: 180` (6 months)
- Keep more backups: `error_log_max_backups: 20`
- Set up monitoring/alerting
- Use JSON format for easier parsing

### Docker
```yaml
# docker-compose.yml
volumes:
  - ./logs:/app/logs

logging:
  error_log_enabled: true
  error_log_file_path: "/app/logs/proxy-errors.log"
```

### Kubernetes
```yaml
# ConfigMap
logging:
  error_log_enabled: true
  error_log_file_path: "/var/log/ollama-proxy/errors.log"

# Mount volume
volumeMounts:
  - name: logs
    mountPath: /var/log/ollama-proxy
```

## Troubleshooting

### Error log not created

**Check:**
1. `error_log_enabled: true` in config
2. `error_log_file_path` is set
3. Directory exists and is writable
4. Check main log for warnings:
   ```
   Failed to create error log directory
   Error log file path not specified
   ```

### Permission denied

```bash
# Fix permissions
mkdir -p logs
chmod 755 logs

# Or in Docker
docker run -v $(pwd)/logs:/app/logs -u $(id -u):$(id -g) ...
```

### No errors in error log

This is **good**! It means your system is healthy. 

You should still see warnings if any warnings were logged.

## Implementation Details

- **Hook**: `internal/logger/error_hook.go`
- **Setup**: `internal/logger/logger.go`
- **Config**: `internal/config/config.go`
- Uses `github.com/sirupsen/logrus` hooks
- Rotation via `gopkg.in/natefinch/lumberjack.v2`

## Example Output

### Main Log (`proxy-dev.log`)
```
time="2025-10-27 04:50:00" level=debug msg="Request received" method=POST path=/api/v1/chat/completions
time="2025-10-27 04:50:01" level=info msg="Request processed" duration=1.2s status=200
time="2025-10-27 04:50:05" level=warning msg="High latency detected" latency=3.5s
time="2025-10-27 04:50:10" level=error msg="Failed to connect to Ollama" error="connection refused"
```

### Error Log (`proxy-errors.log`)
```
time="2025-10-27 04:50:05" level=warning msg="High latency detected" latency=3.5s
time="2025-10-27 04:50:10" level=error msg="Failed to connect to Ollama" error="connection refused"
```

Notice: **Only warnings and errors** appear in the error log.

