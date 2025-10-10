# Установка Ollama OpenAI Proxy как systemd сервис

## Требования

- Linux система с systemd
- Права root или sudo

## Шаг 1: Создание пользователя для сервиса

```bash
# Создать системного пользователя
sudo useradd -r -s /bin/false -d /opt/ollama-openai-proxy ollama-proxy

# Или, если хотите чтобы пользователь мог логиниться:
# sudo useradd -m -s /bin/bash -d /opt/ollama-openai-proxy ollama-proxy
```

## Шаг 2: Установка файлов

```bash
# Создать директорию для приложения
sudo mkdir -p /opt/ollama-openai-proxy

# Скопировать файлы (из директории с исходниками)
sudo cp -r bin /opt/ollama-openai-proxy/
sudo cp -r configs /opt/ollama-openai-proxy/
sudo cp -r web /opt/ollama-openai-proxy/

# Создать директории для данных и логов
sudo mkdir -p /opt/ollama-openai-proxy/data
sudo mkdir -p /opt/ollama-openai-proxy/logs

# Установить владельца
sudo chown -R ollama-proxy:ollama-proxy /opt/ollama-openai-proxy

# Установить права доступа
sudo chmod 755 /opt/ollama-openai-proxy/bin/server
sudo chmod 644 /opt/ollama-openai-proxy/configs/config.yaml
```

## Шаг 3: Настройка конфигурации

```bash
# Создать production конфигурацию
sudo cp /opt/ollama-openai-proxy/configs/production.yaml.example \
     /opt/ollama-openai-proxy/configs/config.yaml

# Отредактировать конфигурацию
sudo nano /opt/ollama-openai-proxy/configs/config.yaml
```

**Важные настройки в config.yaml:**

```yaml
server:
  host: 0.0.0.0
  port: 8080

database:
  type: sqlite
  sqlite:
    path: /opt/ollama-openai-proxy/data/proxy.db

auth:
  enabled: true
  admin_key: "your-secure-admin-key-here"  # ОБЯЗАТЕЛЬНО ИЗМЕНИТЕ!
  jwt:
    secret: "your-secure-jwt-secret-here"   # ОБЯЗАТЕЛЬНО ИЗМЕНИТЕ!
    expiration: 24h

ollama:
  url: http://localhost:11434
```

## Шаг 4: Установка systemd сервиса

```bash
# Скопировать unit файл
sudo cp ollama-openai-proxy.service /etc/systemd/system/

# Перезагрузить systemd
sudo systemctl daemon-reload

# Включить автозапуск
sudo systemctl enable ollama-openai-proxy

# Запустить сервис
sudo systemctl start ollama-openai-proxy
```

## Управление сервисом

### Проверить статус
```bash
sudo systemctl status ollama-openai-proxy
```

### Просмотр логов
```bash
# Последние логи
sudo journalctl -u ollama-openai-proxy -n 50

# Логи в реальном времени
sudo journalctl -u ollama-openai-proxy -f

# Логи за сегодня
sudo journalctl -u ollama-openai-proxy --since today

# Логи с детальной информацией
sudo journalctl -u ollama-openai-proxy -xe
```

### Перезапуск сервиса
```bash
sudo systemctl restart ollama-openai-proxy
```

### Остановка сервиса
```bash
sudo systemctl stop ollama-openai-proxy
```

### Отключить автозапуск
```bash
sudo systemctl disable ollama-openai-proxy
```

## Nginx reverse proxy (опционально)

Если хотите использовать Nginx перед прокси:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Безопасность

### Настройка файрвола (UFW)

```bash
# Разрешить доступ к порту (если нужен внешний доступ)
sudo ufw allow 8080/tcp

# Или только с localhost
sudo ufw deny 8080/tcp
```

### Изменение прав доступа к конфигурации

```bash
# Защитить конфигурацию от чтения другими пользователями
sudo chmod 600 /opt/ollama-openai-proxy/configs/config.yaml
sudo chown ollama-proxy:ollama-proxy /opt/ollama-openai-proxy/configs/config.yaml
```

## Обновление

```bash
# Остановить сервис
sudo systemctl stop ollama-openai-proxy

# Обновить бинарник
sudo cp bin/server /opt/ollama-openai-proxy/bin/

# Установить права
sudo chown ollama-proxy:ollama-proxy /opt/ollama-openai-proxy/bin/server
sudo chmod 755 /opt/ollama-openai-proxy/bin/server

# Запустить сервис
sudo systemctl start ollama-openai-proxy
```

## Автоматическое обновление (опционально)

Создать скрипт для обновления:

```bash
sudo nano /opt/ollama-openai-proxy/update.sh
```

```bash
#!/bin/bash
set -e

echo "Stopping service..."
systemctl stop ollama-openai-proxy

echo "Backing up current version..."
cp /opt/ollama-openai-proxy/bin/server /opt/ollama-openai-proxy/bin/server.backup

echo "Installing new version..."
cp /path/to/new/server /opt/ollama-openai-proxy/bin/server
chown ollama-proxy:ollama-proxy /opt/ollama-openai-proxy/bin/server
chmod 755 /opt/ollama-openai-proxy/bin/server

echo "Starting service..."
systemctl start ollama-openai-proxy

echo "Checking status..."
systemctl status ollama-openai-proxy
```

```bash
sudo chmod +x /opt/ollama-openai-proxy/update.sh
```

## Мониторинг

### Prometheus метрики (если включены)

Добавить в prometheus.yml:

```yaml
scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## Решение проблем

### Сервис не запускается

```bash
# Проверить логи
sudo journalctl -u ollama-openai-proxy -xe

# Проверить конфигурацию
sudo -u ollama-proxy /opt/ollama-openai-proxy/bin/server -config /opt/ollama-openai-proxy/configs/config.yaml

# Проверить права доступа
ls -la /opt/ollama-openai-proxy/
```

### Ошибка доступа к базе данных

```bash
# Проверить права на директорию data
sudo chown -R ollama-proxy:ollama-proxy /opt/ollama-openai-proxy/data
sudo chmod 755 /opt/ollama-openai-proxy/data
```

### Высокое использование памяти

Отредактировать systemd unit:

```bash
sudo nano /etc/systemd/system/ollama-openai-proxy.service
```

Изменить лимит памяти:
```
MemoryLimit=1G
```

Затем:
```bash
sudo systemctl daemon-reload
sudo systemctl restart ollama-openai-proxy
```

## Полное удаление

```bash
# Остановить и отключить сервис
sudo systemctl stop ollama-openai-proxy
sudo systemctl disable ollama-openai-proxy

# Удалить systemd unit
sudo rm /etc/systemd/system/ollama-openai-proxy.service
sudo systemctl daemon-reload

# Удалить файлы (ВНИМАНИЕ: удалит все данные!)
sudo rm -rf /opt/ollama-openai-proxy

# Удалить пользователя
sudo userdel ollama-proxy
```

