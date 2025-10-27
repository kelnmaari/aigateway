# 🚀 Quick Start Guide - AIGateway Platform v1.9.3

Начните работу за 5 минут!

---

## 📋 Предварительные требования

### 1. Установите Ollama

**Download:** https://ollama.ai/

```bash
# Проверьте установку
ollama --version

# Запустите сервер
ollama serve
```

### 2. Загрузите модель

```bash
# Рекомендуемые модели
ollama pull llama3.2        # 3B - быстрая
ollama pull qwen2.5-coder:7b # 7B - для кода
ollama pull llama3.1        # 8B - универсальная

# Проверьте
ollama list
```

---

## 🔧 Установка Proxy

### Option 1: Docker Compose (рекомендуется)

```bash
# Clone repository
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Build & Start
docker compose --profile build up --build

# Сервер: http://localhost:8080
# WebUI: http://localhost:8080/login
```

### Option 2: Pre-built Binaries

```bash
# Download latest release
# https://github.com/yourusername/aigateway/releases

# Linux/macOS
chmod +x ollama-proxy-linux-amd64
./ollama-proxy-linux-amd64 -config configs/dev.yaml

# Windows
ollama-proxy-windows-amd64.exe -config configs/dev.yaml
```

### Option 3: Build from Source

```bash
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Install dependencies
go mod tidy

# Build
./build.sh all

# Run
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml
```

---

## 👤 Первый запуск

### 1. Bootstrap Admin User

При первом запуске в логах увидите:

```
🔐 Bootstrap Token:
http://localhost:8080/bootstrap?token=abc123def456...

⏰ Token expires in 10 minutes
```

**Откройте ссылку в браузере** и создайте первого admin пользователя.

### 2. Регистрация

Заполните форму:
- **Username:** admin
- **Email:** admin@example.com
- **Password:** ******** (минимум 8 символов)

Нажмите **Register** → автоматический вход.

### 3. Откройте Chat

```
http://localhost:8080/chat
```

Начните общаться с локальными моделями!

---

## 💬 Первый Chat

### 1. Выберите модель

Dropdown над input field:
- llama3.2
- qwen2.5-coder:7b
- или другую установленную модель

### 2. Настройте параметры (опционально)

Нажмите **[⚙️ Parameters]**:
- Quick Preset: **Balanced** (по умолчанию)
- Temperature: 0.7
- Context Window: 32000

### 3. Отправьте сообщение

```
Explain quantum computing in simple terms
```

Нажмите **Send** или **Enter**.

### 4. Real-time response

Ответ появится потоком с поддержкой Markdown и code highlighting.

---

## 🔑 Создание API Key

### Для программного доступа

1. Откройте `/api-keys`
2. Tab: **Personal Keys**
3. Нажмите **[➕ Create New Key]**
4. Заполните:
   - Name: "Development Key"
   - Rate Limit: 60/min
   - Models: * (все модели)
   - Expires: 30 days
5. **Скопируйте ключ** - он показывается только один раз!

### Использование

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-key-here" \
  -d '{
    "model": "llama3.2",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

---

## 📊 Monitoring (Admin)

### Performance Dashboard

1. Откройте `/admin`
2. Tab: **System**
3. Просмотр:
   - **Quick Stats** (CPU, Memory, Goroutines)
   - **GPU Metrics** (если NVIDIA GPU на Linux/macOS)
   - **Advanced Dashboard** (MoniGo на :9091)

### Models Management

1. Tab: **Models**
2. Список доступных моделей
3. **[🔄 Refresh]** для обновления

### Real-time Logs

1. Tab: **Logs**
2. Выберите файл: `proxy-dev.log`
3. Фильтр по уровню: All/Error/Warn/Info
4. Live streaming через SSE

---

## 👥 Создание Организации

### Multi-Tenancy Setup

1. Откройте `/tenants`
2. Нажмите **[➕ Create Tenant]**
3. Заполните:
   - Name: "My Company"
   - Description: "Development team"
4. **[Create]**

### Добавление участников

1. Выберите tenant → **Members** tab
2. **[➕ Add Member]**
3. Найдите пользователя по email/username
4. Выберите роль: Admin/Member/Viewer
5. **[Add]**

### Tenant API Keys

1. `/api-keys` → Tab: **Tenant Keys**
2. Выберите tenant из dropdown
3. Создайте ключ (требуется роль Owner/Admin)

---

## 🎯 Quick Tips

### Используйте Quick Presets

В chat parameters панели:
- 🎨 **Creative** - для креативного текста
- 💻 **Coding** - для программирования
- 🎯 **Precise** - для точных ответов
- ⚖️ **Balanced** - универсальный

### Следите за Context

Индикатор под messages показывает использование:
- 🟢 < 70% - нормально
- 🟡 70-90% - скоро заполнится
- 🔴 > 90% - auto-summarization сработает

### Экспортируйте данные

- **Chat:** Сохраняйте важные conversations
- **Usage:** `/usage` → Export CSV/JSON
- **API Keys:** Резервное копирование списка

---

## 🔧 Troubleshooting

### "Connection refused" to Ollama

```bash
# Проверьте Ollama
curl http://localhost:11434/api/tags

# Если не работает, запустите
ollama serve
```

### Bootstrap token expired

```bash
# Остановите сервер
Ctrl+C

# Удалите БД (только для первого запуска!)
rm data/proxy.db

# Запустите снова
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml
```

### Models не загружаются

```bash
# Проверьте доступность Ollama API
curl http://localhost:11434/api/tags

# Установите модель если нет
ollama pull llama3.2
```

### GPU Metrics пустые

**Windows:** GPU monitoring не поддерживается (stub version)

**Linux/macOS:**
```bash
# Проверьте nvidia-smi
nvidia-smi

# Если не работает, установите NVIDIA drivers
```

---

## 📚 Дальнейшее чтение

- [WEBUI Guide](WEBUI_GUIDE.md) - Полный WebUI мануал
- [API Documentation](API_DOCUMENTATION.md) - REST API reference
- [Configuration](CONFIGURATION.md) - Все параметры конфигурации
- [Architecture](ARCHITECTURE.md) - Архитектура системы

---

## 🎉 Готово!

Теперь у вас запущен полноценный ChatGPT-like интерфейс для локальных Ollama моделей с enterprise возможностями!

**Next steps:**
1. Пригласите team members через `/tenants`
2. Создайте tenant API keys для приложений
3. Мониторьте usage через `/usage`
4. Изучите performance dashboard в `/admin`

**Need help?**
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [GitHub Issues](https://github.com/yourusername/aigateway/issues)

---

**Version:** 1.9.3  
**Last Updated:** 2025-10-14


