# 🚀 Quick Start - Build & Run

**Ollama-OpenAI Proxy v1.3.0** - Быстрый старт для сборки и запуска

---

## ⚡ Самый быстрый способ

### Windows (PowerShell)

```powershell
# 1. Собрать для Windows
.\build.ps1 windows

# 2. Запустить
.\dist\ollama-proxy-windows-amd64.exe
```

### Linux / macOS

```bash
# 1. Собрать для Linux
./build.sh linux

# 2. Запустить
./dist/ollama-proxy-linux-amd64
```

---

## 📦 Сборка для всех платформ

### Windows

```powershell
# Собрать для Linux + Windows
.\build.ps1 all

# Результат:
# dist\ollama-proxy-linux-amd64
# dist\ollama-proxy-linux-arm64
# dist\ollama-proxy-windows-amd64.exe
```

### Linux / macOS

```bash
# Собрать для Linux + Windows
./build.sh all

# Или с Make
make build-all

# Результат:
# dist/ollama-proxy-linux-amd64
# dist/ollama-proxy-linux-arm64  
# dist/ollama-proxy-windows-amd64.exe
```

---

## 📦 Создание distribution пакетов

```bash
# Windows
.\build.ps1 all
.\build.ps1 package

# Linux/macOS
./build.sh all
./build.sh package

# Или с Make
make release
```

**Результат:** Готовые архивы с configs и WebUI

```
dist/
├── ollama-proxy-1.3.0-linux-amd64.tar.gz
├── ollama-proxy-1.3.0-linux-arm64.tar.gz
├── ollama-proxy-1.3.0-windows-amd64.zip
└── checksums.txt
```

---

## 🎯 Makefile команды

```bash
make build-all       # Собрать все платформы
make build-linux     # Только Linux
make build-windows   # Только Windows
make package         # Создать пакеты
make release         # Full release (build + package + checksums)
make clean           # Очистить
make test            # Запустить тесты
make run             # Запустить сервер
make help            # Показать все команды
```

---

## 🔧 Первый запуск

### 1. Запустить Ollama

```bash
# Linux/macOS
ollama serve

# Windows
ollama serve
```

### 2. Запустить Proxy

```bash
# Linux/macOS
./dist/ollama-proxy-linux-amd64

# Windows
.\dist\ollama-proxy-windows-amd64.exe
```

### 3. Открыть WebUI

Перейти на: **<http://localhost:8080>**

При первом запуске откроется **Bootstrap Setup** для создания admin аккаунта.

---

## 🌐 После запуска

### WebUI доступен по адресу

- **Chat Interface:** <http://localhost:8080/>
- **Dashboard:** <http://localhost:8080/dashboard.html>
- **Profile:** <http://localhost:8080/profile.html>
- **Tenants:** <http://localhost:8080/tenants.html>
- **API Keys:** <http://localhost:8080/api-keys.html>
- **Usage Stats:** <http://localhost:8080/usage.html>

### API endpoints

- **Health:** <http://localhost:8080/health>
- **Models:** <http://localhost:8080/v1/models>
- **Chat:** `POST http://localhost:8080/v1/chat/completions`
- **Metrics:** <http://localhost:8080/metrics>

---

## 🔐 Bootstrap Setup (первый запуск)

1. Откройте <http://localhost:8080>
2. Автоматически откроется **Bootstrap Setup**
3. Введите **Admin Token** (из `configs/dev.yaml`: `admin_key`)
4. Создайте первый admin аккаунт (username, email, password)
5. Готово! Теперь можно войти

---

## 📝 Конфигурация

### Изменить порт

```yaml
# configs/dev.yaml
server:
  port: 8080  # Изменить на нужный
```

### Изменить Ollama URL

```yaml
ollama:
  url: http://localhost:11434  # URL вашего Ollama
```

### Изменить Admin Token

```yaml
auth:
  admin_key: sk-admin-YOUR-SECURE-TOKEN-HERE
```

---

## 🐛 Решение проблем

### "Port already in use"

```bash
# Найти и остановить процесс
# Windows:
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Linux/macOS:
lsof -ti:8080 | xargs kill -9
```

### "Cannot connect to Ollama"

```bash
# Проверить статус Ollama
curl http://localhost:11434/api/tags

# Запустить Ollama если не запущен
ollama serve
```

### "Database migration failed"

```bash
# Удалить старую базу и пересоздать
rm data/proxy.db
./ollama-proxy
```

---

## 📚 Дополнительная документация

- [BUILD.md](BUILD.md) - Подробные инструкции по сборке
- [WEBUI.md](WEBUI.md) - Документация WebUI
- [Roadmap.MD](Roadmap.MD) - Development roadmap
- [configs/production.yaml.example](configs/production.yaml.example) - Production config

---

## 💡 Полезные команды

```bash
# Показать версию
./ollama-proxy --version

# Запустить на другом порту
PORT=9090 ./ollama-proxy

# Запустить с production конфигом
./ollama-proxy --config configs/production.yaml

# Запустить с debug логами
LOG_LEVEL=debug ./ollama-proxy
```

---

## 🎉 Готово

Теперь у вас есть:

- ✅ ChatGPT-like интерфейс для локальных моделей
- ✅ Multi-user система с JWT authentication
- ✅ Multi-tenancy для командной работы
- ✅ Personal dashboard с аналитикой
- ✅ API key management
- ✅ Usage statistics

**Enjoy!** 🚀

---

**Version:** 1.3.0  
**Last Updated:** 2025-10-06
