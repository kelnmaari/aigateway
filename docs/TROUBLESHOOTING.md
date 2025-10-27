# 🔧 Troubleshooting Guide - AIGateway Platform v1.9.3

Решения распространенных проблем

---

## 📋 Содержание

1. [Server Issues](#server-issues)
2. [Ollama Connection](#ollama-connection)
3. [Authentication & JWT](#authentication--jwt)
4. [Chat Interface](#chat-interface)
5. [API Keys](#api-keys)
6. [Multi-Tenancy](#multi-tenancy)
7. [Performance Monitoring](#performance-monitoring)
8. [GPU Monitoring](#gpu-monitoring)
9. [Database Issues](#database-issues)
10. [Build & Deployment](#build--deployment)

---

## 🖥️ Server Issues

### Server не запускается

**Problem:** Server crashes сразу при старте

**Причина:** Порт 8080 уже занят

**Solution:**


```bash
# Проверьте порт
netstat -an | grep 8080    # Linux/macOS
netsh int ipv4 show tcp    # Windows

# Измените порт в конфигурации
# configs/dev.yaml
server:
  port: 9090  # Другой порт

# Или через environment variable
export PROXY_SERVER_PORT=9090
```

---

### "panic: runtime error"

**Problem:** Panic при запуске

**Причина:** Некорректная конфигурация или поврежденная БД


**Solution:**

```bash
# 1. Проверьте конфигурацию
cat configs/dev.yaml

# 2. Проверьте БД (если первый запуск - удалите)
rm data/proxy.db

# 3. Включите debug logging
# configs/dev.yaml:
logging:
  level: "debug"

# 4. Проверьте логи
tail -f logs/proxy-dev.log
```

---

## 🦙 Ollama Connection

### "Connection refused" to Ollama

**Problem:** `Failed to connect to Ollama server`

**Причина:** Ollama не запущен или недоступен


**Solution:**

```bash
# 1. Проверьте Ollama
curl http://localhost:11434/api/tags

# 2. Запустите если не работает
ollama serve

# 3. Проверьте процесс
ps aux | grep ollama    # Linux/macOS
tasklist | findstr ollama  # Windows

# 4. Если remote Ollama, обновите конфиг:
ollama:
  url: "http://192.168.1.100:11434"
```

---

### Models не обнаружены

**Problem:** `/v1/models` возвращает пустой список


**Причина:** Нет установленных моделей

**Solution:**

```bash
# Проверьте модели в Ollama
ollama list

# Установите модель
ollama pull llama3.2
ollama pull qwen2.5-coder:7b

# Проверьте API Ollama
curl http://localhost:11434/api/tags

# Refresh в WebUI
/admin → Models → [🔄 Refresh]
```

---

## 🔐 Authentication & JWT

### Bootstrap token expired

**Problem:** "Bootstrap token has expired or is invalid"


**Причина:** Bootstrap token действует 10 минут

**Solution:**

```bash
# ТОЛЬКО для первого запуска!
# 1. Остановите сервер
Ctrl+C

# 2. Удалите БД
rm data/proxy.db

# 3. Запустите снова
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml

# 4. Скопируйте новый bootstrap URL из логов
# 5. Откройте в браузере в течение 10 минут
```

---

### "Invalid JWT token"


**Problem:** JWT token не валиден при запросах

**Причина:** Token expired или jwt_secret изменен

**Solution:**

```bash
# 1. Logout из WebUI
# Кнопка Logout в правом верхнем углу

# 2. Login снова
# /login

# 3. Если не помогло, очистите localStorage
# Browser DevTools (F12) → Console:
localStorage.clear()
location.reload()

# 4. Проверьте jwt_secret в конфигурации
# configs/dev.yaml:
auth:

```

---

### "Unauthorized: missing token"


**Problem:** API requests возвращают 401

**Причина:** Authorization header отсутствует или некорректен

**Solution:**

```bash
# Правильный формат:
curl -H "Authorization: Bearer sk-your-api-key" \
  http://localhost:8080/v1/models

# НЕ:
curl -H "Authorization: sk-your-api-key"  # ❌ Без Bearer
curl -H "X-API-Key: sk-your-api-key"      # ❌ Неверный header
```

---

## 💬 Chat Interface


### Messages не отправляются

**Problem:** Кнопка Send не работает

**Причина 1:** Модель не выбрана

**Solution:**


```javascript
// Выберите модель из dropdown над input field
// Если dropdown пустой → проверьте Ollama
```

**Причина 2:** Empty message

**Solution:**

```javascript
// Введите текст перед отправкой
// Пустые сообщения блокируются
```


---

### Streaming не работает

**Problem:** Ответ появляется целиком, а не потоком

**Причина:** SSE (Server-Sent Events) блокируется

**Solution:**

```bash
# 1. Проверьте CORS в конфигурации
server:

    enabled: true
    allowed_origins: ["http://localhost:8080"]

# 2. Проверьте browser console (F12)
# Должны быть EventSource connections

# 3. Проверьте proxy/firewall
# SSE может блокироваться intermediate proxies
```


---

### Context tracking показывает 0%

**Problem:** Context window всегда 0/32000 (0%)

**Причина:** Token estimation не работает

**Solution:**

```bash
# 1. Проверьте API endpoint
curl http://localhost:8080/api/chat/estimate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"text": "Hello world"}'

# 2. Проверьте логи сервера
tail -f logs/proxy-dev.log | grep estimate

# 3. Refresh страницы
# Hard reload: Ctrl+Shift+R
```


---


## 🔑 API Keys

### Не могу создать API key

**Problem:** "Failed to create API key"

**Причина 1 (Personal Keys):** Database error

**Solution:**

```bash
# Проверьте логи

tail -f logs/proxy-dev.log


# Проверьте БД
ls -lh data/proxy.db

# Если БД поврежден:
# Backup → Restore или создайте новый
```

**Причина 2 (Tenant Keys):** Недостаточные права



```javascript
// Требуется роль Owner или Admin в tenant

// Проверьте в /tenants → Members
// Если Member или Viewer - попросите Owner добавить права
```

---

### API key не работает

**Problem:** `401 Unauthorized` при использовании key


**Причина 1:** Key disabled

**Solution:**

```javascript
// Проверьте в /api-keys
// Status должен быть 🟢 Active
// Если 🔴 Disabled - удалите и создайте новый
```


**Причина 2:** Model permissions

**Solution:**

```javascript
// Key имеет ограничения по моделям
// Проверьте в /api-keys → Models column
// Если "llama3" а вы используете "qwen" - будет 403

// Создайте key с "*" для всех моделей
```

**Причина 3:** Rate limit exceeded

**Solution:**


```javascript

// Проверьте Rate Limit column
// Подождите или создайте key с higher limit
```

---

## 👥 Multi-Tenancy

### Не вижу tenant после создания

**Problem:** Tenant создан но не отображается в списке

**Причина:** Page не обновилась автоматически

**Solution:**

```javascript

// 1. Refresh страницы
location.reload()

// 2. Если не помогло - проверьте через API
curl http://localhost:8080/api/tenants \
  -H "Authorization: Bearer $TOKEN"
```

---

### Не могу добавить member в tenant

**Problem:** "User not found" при search

**Причина:** User должен быть зарегистрирован

**Solution:**


```javascript

// 1. Пригласите пользователя зарегистрироваться
// /register

// 2. После регистрации найдите по email/username
// /tenants → Members → [➕ Add Member]
```

---

### "Permission denied" при доступе к tenant resources

**Problem:** 403 при попытке доступа

**Причина:** Недостаточная роль

**Solution:**

```
Действие              Требуется роль

-------------------   ----------------
View tenant           Member+
Create API key        Owner/Admin
Add members           Owner/Admin
Change member role    Owner/Admin

Delete tenant         Owner only
```

---

## 📊 Performance Monitoring

### MoniGo Dashboard 404

**Problem:** `http://localhost:9091` возвращает 404

**Причина:** MoniGo отключен или не запущен

**Solution:**

```yaml
# 1. Проверьте конфигурацию
# configs/dev.yaml:
performance:
  monigo:

    enabled: true
    port: 9091


# 2. Restart server
# Ctrl+C → ./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml

# 3. Проверьте логи
tail -f logs/proxy-dev.log | grep MoniGo
# Должно быть: "MoniGo server started on port 9091"
```

---

### Quick Stats Cards не обновляются

**Problem:** Метрики застыли в UI

**Причина:** JavaScript error или API недоступен

**Solution:**

```javascript
// 1. Откройте Browser Console (F12)

// Проверьте ошибки

// 2. Проверьте API endpoint
curl http://localhost:8080/admin/performance/monigo/api/v1/metrics \
  -H "Authorization: Bearer $TOKEN"

// 3. Hard reload
Ctrl+Shift+R
```

---


## 🎮 GPU Monitoring

### GPU Metrics пустые (Linux/macOS)

**Problem:** GPU cards не отображаются


**Причина 1:** nvidia-smi недоступен

**Solution:**

```bash
# Проверьте nvidia-smi
nvidia-smi

# Если не работает:
# Ubuntu/Debian
sudo apt-get install nvidia-driver-535

# Arch Linux
sudo pacman -S nvidia

# Проверьте установку

nvidia-smi
```

**Причина 2:** Permissions

**Solution:**

```bash
# Добавьте пользователя в video group
sudo usermod -a -G video $USER

# Logout и login снова
```

---

### GPU Monitoring на Windows

**Problem:** "GPU monitoring disabled"

**Причина:** Windows использует stub версию


**Solution:**

```
GPU monitoring через nvidia-smi работает только в Linux/macOS.

Windows: Используйте MoniGo Dashboard для CPU/Memory monitoring.

Для GPU на Windows:
- Используйте Task Manager (Ctrl+Shift+Esc)
- Или NVIDIA GeForce Experience
- Или GPU-Z
```

---

## 💾 Database Issues

### "database is locked"




**Причина:** Несколько процессов пытаются писать в БД

**Solution:**

```bash
# 1. Остановите все instances
pkill ollama-proxy

# 2. Проверьте lock файл
rm data/proxy.db-shm
rm data/proxy.db-wal

# 3. Запустите один instance
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml
```

---

### Migration failed

**Problem:** "Migration v25 failed"

**Причина:** Database schema corruption


**Solution:**

```bash
# 1. Backup текущей БД
cp data/proxy.db data/proxy.db.backup

# 2. Проверьте логи миграции
tail -f logs/proxy-dev.log | grep migration

# 3. Если критично - восстановите из backup
# /admin → Backup & Restore → Upload backup

# 4. Или пересоздайте БД (ПОТЕРЯ ДАННЫХ!)
# rm data/proxy.db
```

---

## 🏗️ Build & Deployment

### Build fails "go: module not found"

**Problem:** Dependencies не загружены


**Solution:**

```bash
# 1. Очистите module cache
go clean -modcache

# 2. Обновите зависимости
go mod tidy
go mod download

# 3. Проверьте go.mod
cat go.mod


./build.sh all
```

---


### CGO errors on Windows

**Problem:** `dlfcn.h: No such file or directory`

**П<https://github.com/yourusername/aigateway/issues>

**Solution:**

```bash
# Windows build БЕЗ GPU monitoring (stub)
# Build tags автоматически применяются

# Для pure Windows build:
GOOS=windows GOARCH=amd64 go build -tags=nogpu \
  -o dist/ollama-proxy-windows-amd64.exe \
  cmd/server/main.go

# Или используйте build.sh
./build.sh windows
```

---


### Docker build fails

**Problem:** Docker build ошибки


**Solution:**

```bash
# 1. Очистите Docker cache
docker system prune -a

# 2<https://github.com/yourusername/aigateway/issues>
docker compose --profile build up --build

# 3. Проверьте logs
docker compose logs -f

# 4. Если проблема с Go modules:
# Dockerfile должен иметь:
# RUN go mod download
```

---

## 🆘 Getting Help

### Не нашли решение?

1. **Check logs:**

   ```bash
   tail -f logs/proxy-dev.log
   ```

2. **Enable debug mode:**

   ```yaml
   logging:
     level: "debug"
   ```

3. **GitHub Issues:**
   <https://github.com/yourusername/aigateway/issues>

4. **Provide details:**
   - Version: `cat VERSION`
   - OS: `uname -a` (Linux/macOS) или `ver` (Windows)
   - Go version: `go version`
   - Ollama version: `ollama --version`
   - Error logs
   - Steps to reproduce

---

## 📚 Related Documentation

- [Quick Start](QUICK_START.md) - Быстрый старт
- [WebUI Guide](WEBUI_GUIDE.md) - WebUI мануал
- [Configuration](CONFIGURATION.md) - Конфигурация
- [API Documentation](API_DOCUMENTATION.md) - API reference

---

**Version:** 1.9.3  
**Last Updated:** 2025-10-14

