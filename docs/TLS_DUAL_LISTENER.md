# Dual-Listener Configuration (v3.0.8)

## Overview

AIGateway теперь поддерживает одновременную работу HTTP и HTTPS серверов:

- **HTTP**: Стандартный порт (например, `:8085`)
- **HTTPS**: HTTP порт + 400 (например, `:8485`)

## TLS Certificate Auto-Generation

При первом запуске с `server.tls.enabled: true` автоматически генерируется самоподписной сертификат:

- **Алгоритм**: ECDSA P-256 (современный, быстрый)
- **Срок действия**: 1 год (365 дней)
- **Расположение**: `certs/server.crt` и `certs/server.key`
- **CN**: localhost
- **Hosts**: localhost, 127.0.0.1, ::1, aigateway.local

### Логи при запуске

```
🔐 Generating self-signed TLS certificate...
✅ Self-signed TLS certificate generated successfully
✅ TLS certificate found: certs/server.crt
🌐 HTTP сервер запущен на 0.0.0.0:8085
🔐 HTTPS сервер запущен на 0.0.0.0:8485
```

## Configuration Examples

### Minimal (auto-generated certificate)

```yaml
server:
  port: 8085
  host: "0.0.0.0"
  tls:
    enabled: true  # Автоматически генерирует сертификат
```

**Результат:**
- HTTP: `http://localhost:8085`
- HTTPS: `https://localhost:8485`

### Custom Certificate (BYO - Bring Your Own)

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  tls:
    enabled: true
    cert_file: "/path/to/your/cert.pem"
    key_file: "/path/to/your/key.pem"
```

**Результат:**
- HTTP: `http://localhost:8080`
- HTTPS: `https://localhost:8480` (с вашим сертификатом)

### HTTPS-Only (disable HTTP)

Чтобы оставить только HTTPS, установите `server.port: 0`:

```yaml
server:
  port: 0  # Отключить HTTP
  tls:
    enabled: true
```

**Результат:**
- HTTP: выключен
- HTTPS: `https://localhost:400` (только TLS)

## Certificate Management

### Regenerate Certificate

Удалите существующие файлы, и они будут пересозданы:

```bash
rm certs/server.crt certs/server.key
./bin/server.exe -config configs/dev.yaml
```

### Check Certificate Info

```bash
# OpenSSL
openssl x509 -in certs/server.crt -text -noout

# API endpoint (будущая фича)
curl http://localhost:8085/api/system/tls/info
```

### Certificate Validation

Сертификат автоматически проверяется при запуске:
- ✅ Not expired
- ✅ Valid format
- ✅ Readable by server

Если сертификат невалиден, он автоматически перегенерируется.

## Production Deployment

### With Let's Encrypt (Recommended)

Используйте внешний reverse proxy (Nginx, Caddy) с автоматическим Let's Encrypt:

```yaml
# AIGateway config (HTTP only)
server:
  port: 8085
  host: "127.0.0.1"  # Только localhost
  tls:
    enabled: false   # Nginx/Caddy управляет TLS
```

```nginx
# Nginx config
server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    
    location / {
        proxy_pass http://127.0.0.1:8085;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Self-Signed for Internal Use

Для internal/corporate сетей самоподписной сертификат допустим:

```yaml
server:
  port: 8085
  host: "0.0.0.0"
  tls:
    enabled: true  # Auto-generated self-signed cert
```

**Trust certificate** на клиентских машинах:

```bash
# Linux
sudo cp certs/server.crt /usr/local/share/ca-certificates/aigateway.crt
sudo update-ca-certificates

# macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/server.crt

# Windows
Import-Certificate -FilePath certs\server.crt -CertStoreLocation Cert:\LocalMachine\Root
```

## API Client Examples

### cURL (allow self-signed)

```bash
# HTTP
curl http://localhost:8085/health

# HTTPS (skip certificate verification)
curl -k https://localhost:8485/health

# HTTPS (with certificate verification)
curl --cacert certs/server.crt https://localhost:8485/health
```

### Python (requests)

```python
import requests

# HTTP
requests.get("http://localhost:8085/health")

# HTTPS (skip verification)
requests.get("https://localhost:8485/health", verify=False)

# HTTPS (with certificate)
requests.get("https://localhost:8485/health", verify="certs/server.crt")
```

### JavaScript (fetch)

```javascript
// HTTP
fetch("http://localhost:8085/health")

// HTTPS (browser trusts cert)
fetch("https://localhost:8485/health")

// Node.js HTTPS (custom cert)
const https = require('https');
const fs = require('fs');

const agent = new https.Agent({
  ca: fs.readFileSync('certs/server.crt')
});

fetch("https://localhost:8485/health", { agent })
```

## Kubernetes Deployment

### Service Configuration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: aigateway
spec:
  type: LoadBalancer
  ports:
    - name: http
      port: 80
      targetPort: 8085
      protocol: TCP
    - name: https
      port: 443
      targetPort: 8485
      protocol: TCP
  selector:
    app: aigateway
```

### Ingress (Terminate TLS at Ingress)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aigateway
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
    - hosts:
        - api.yourdomain.com
      secretName: aigateway-tls
  rules:
    - host: api.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: aigateway
                port:
                  number: 8085  # HTTP only to pod
```

## Security Considerations

### Self-Signed Certificate Warnings

⚠️ **Browser Warning**: Браузеры покажут "Your connection is not private"
- **Development**: Нормально, нажмите "Advanced" → "Proceed"
- **Production**: Используйте Let's Encrypt или корпоративный CA

### HSTS (HTTP Strict Transport Security)

Для production включите HSTS в reverse proxy:

```nginx
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

### Certificate Pinning

Для критичных приложений используйте certificate pinning на клиенте.

## Performance

### ECDSA vs RSA

AIGateway использует **ECDSA P-256**:
- ⚡ **Быстрее** RSA-2048 (генерация и проверка подписи)
- 📦 **Меньше** размер ключа/сертификата
- 🔒 **Эквивалентная** безопасность RSA-3072

### TLS 1.3

Go автоматически использует TLS 1.3 если поддерживается клиентом.

## Troubleshooting

### "Certificate has expired"

```bash
rm certs/server.crt certs/server.key
./bin/server.exe -config configs/dev.yaml  # Regenerate
```

### "Address already in use"

Другой процесс использует порт 8485:

```bash
# Windows
netstat -ano | findstr :8485
taskkill /PID <PID> /F

# Linux/macOS
lsof -ti:8485 | xargs kill -9
```

### "Permission denied" on port 443

Порты <1024 требуют root/admin:

```bash
# Linux: Use capabilities (no root needed)
sudo setcap CAP_NET_BIND_SERVICE=+eip /path/to/server

# или bind на порт >1024 и используй iptables redirect
sudo iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port 8485
```

## Related Documentation

- [Security Best Practices](SECURITY.md)
- [Kubernetes Deployment Guide](KUBERNETES.md)
- [Production Checklist](PRODUCTION_CHECKLIST.md)

