# Приоритет загрузки конфигурации

## 📋 Порядок приоритетов (от низкого к высокому)

1. **Значения по умолчанию** (hardcoded в коде)
2. **Конфигурационный файл** (`config.yaml`)
3. **Переменные окружения** ⚠️ **ПЕРЕОПРЕДЕЛЯЮТ ВСЁ**

## 🔧 Переменные окружения

Все переменные окружения используют префикс `PROXY_` и преобразуют точки в подчеркивания:

```yaml
# config.yaml
server:
  host: "192.168.1.100"
  port: 8080
```

Эквивалентные переменные окружения:

```bash
PROXY_SERVER_HOST=192.168.1.100
PROXY_SERVER_PORT=8080
```

## 🎯 Примеры конфликтов

### Проблема: Конфиг игнорируется

**Симптом:** Вы прописали в `config.yaml`:

```yaml
server:
  host: "192.168.1.100"
```

Но сервер все равно запускается на `0.0.0.0`

**Причина:** Установлена переменная окружения:

```bash
echo $PROXY_SERVER_HOST
# Вывод: 0.0.0.0
```

**Решение:**

```bash
# Удалить переменную
unset PROXY_SERVER_HOST

# Или установить правильное значение
export PROXY_SERVER_HOST=192.168.1.100
```

### Systemd конфликты

Если сервис запускается через systemd, проверить переменные:

```bash
# Посмотреть переменные окружения сервиса
systemctl show aigateway --property=Environment

# Пример вывода с проблемой:
# Environment=GIN_MODE=release PROXY_SERVER_HOST=0.0.0.0
```

Решение - отредактировать unit файл:

```bash
sudo nano /etc/systemd/system/aigateway.service
```

Удалить или изменить строку:

```ini
# Удалить эту строку если хотите использовать config.yaml:
# Environment="PROXY_SERVER_HOST=0.0.0.0"

# Или установить правильное значение:
Environment="PROXY_SERVER_HOST=192.168.1.100"
```

Перезагрузить:

```bash
sudo systemctl daemon-reload
sudo systemctl restart aigateway
```

## 📊 Полный список поддерживаемых переменных

### Server

```bash
PROXY_SERVER_HOST=0.0.0.0          # Адрес для прослушивания
PROXY_SERVER_PORT=8080             # Порт
PROXY_SERVER_READ_TIMEOUT=30s      # Таймаут чтения
PROXY_SERVER_WRITE_TIMEOUT=30s     # Таймаут записи
```

### Ollama

```bash
PROXY_OLLAMA_URL=http://localhost:11434
PROXY_OLLAMA_TIMEOUT=30s
PROXY_OLLAMA_RETRY_ATTEMPTS=3
```

### Database

```bash
PROXY_DATABASE_TYPE=sqlite
PROXY_DATABASE_SQLITE_PATH=/opt/aigateway/data/proxy.db
PROXY_DATABASE_POSTGRES_HOST=localhost
PROXY_DATABASE_POSTGRES_PORT=5432
PROXY_DATABASE_POSTGRES_USER=proxy
PROXY_DATABASE_POSTGRES_PASSWORD=secret
PROXY_DATABASE_POSTGRES_DATABASE=ollama_proxy
```

### Authentication

```bash
PROXY_AUTH_ENABLED=true
PROXY_AUTH_ADMIN_KEY=your-secure-admin-key
PROXY_AUTH_JWT_SECRET=your-jwt-secret
PROXY_AUTH_JWT_EXPIRATION=24h
```

### Logging

```bash
PROXY_LOGGING_LEVEL=info
PROXY_LOGGING_FORMAT=json
PROXY_LOGGING_FILE=/opt/aigateway/logs/proxy.log
```

## 🐛 Отладка конфигурации

С версии после этого коммита, сервер при запуске выводит откуда взята конфигурация:

```bash
# Запуск сервера
./bin/server -config configs/config.yaml

# Вывод:
✅ Server configuration loaded from: configs/config.yaml
📍 Server will bind to: 192.168.1.100:8080

# Если переопределено переменной окружения:
⚠️  SERVER HOST OVERRIDDEN by environment variable: PROXY_SERVER_HOST=0.0.0.0

# Если используется default:
⚠️  Using DEFAULT server configuration
📍 Server will bind to: 0.0.0.0:8080
```

## 💡 Рекомендации

### Development (разработка)

Используйте конфигурационный файл:

```bash
./bin/server -config configs/dev.yaml
```

### Production (продакшн)

**Вариант 1:** Конфигурационный файл (рекомендуется)

```bash
# config.yaml с защищенными правами
chmod 600 /opt/aigateway/configs/config.yaml
```

**Вариант 2:** Переменные окружения (для Docker/Kubernetes)

```bash
# В systemd unit файле
Environment="PROXY_AUTH_ADMIN_KEY=your-secret-key"
Environment="PROXY_AUTH_JWT_SECRET=your-jwt-secret"
```

**Вариант 3:** Смешанный подход

- Общие настройки в `config.yaml`
- Секреты через переменные окружения:

  ```bash
  PROXY_AUTH_ADMIN_KEY=secret
  PROXY_AUTH_JWT_SECRET=another-secret
  PROXY_DATABASE_POSTGRES_PASSWORD=db-password
  ```

## 🔒 Безопасность

### НЕ ИСПОЛЬЗУЙТЕ переменные окружения для секретов в systemd

❌ **Плохо:**

```ini
# Секреты видны через systemctl show
Environment="PROXY_AUTH_ADMIN_KEY=my-secret-key"
```

✅ **Хорошо:**

```ini
# Секреты в файле с ограниченными правами
# /opt/aigateway/configs/config.yaml (chmod 600)
```

Или используйте `EnvironmentFile`:

```ini
EnvironmentFile=/opt/aigateway/configs/secrets.env
# Файл secrets.env:
# PROXY_AUTH_ADMIN_KEY=my-secret-key
# chmod 600 /opt/aigateway/configs/secrets.env
```

## 📚 Проверка текущей конфигурации

Создайте тестовый эндпоинт или используйте логи при старте:

```bash
# Проверить какой хост используется
journalctl -u aigateway -n 20 | grep "Server will bind"

# Вывод покажет:
# 📍 Server will bind to: 192.168.1.100:8080
```

