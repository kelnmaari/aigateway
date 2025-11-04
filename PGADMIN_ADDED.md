# ✅ pgAdmin добавлен в RAG Infrastructure

pgAdmin 4 успешно интегрирован в Docker Compose конфигурацию.

## 🎉 Что добавлено

### Docker Compose
- ✅ Сервис `pgadmin` в `docker-compose.rag-infrastructure.yml`
- ✅ Volume `pgadmin_data` для сохранения настроек
- ✅ Healthcheck для мониторинга состояния
- ✅ Автоматическое подключение к PostgreSQL через `depends_on`

### Конфигурация
- ✅ `docker/pgadmin-servers.json` - pre-configured сервер PostgreSQL
- ✅ Переменные окружения в комментариях docker-compose.yml

### Документация
- ✅ Обновлен `RAG_INFRASTRUCTURE_README.md`
- ✅ Обновлен `RAG_INFRASTRUCTURE_SETUP_COMPLETE.md`
- ✅ Обновлен `docker/README-RAG.md`
- ✅ Обновлены quick-start скрипты (`.ps1` и `.sh`)

## 🚀 Как использовать

### 1. Добавьте переменные окружения

Добавьте в `.env` файл (или создайте если его нет):
```env
# pgAdmin
PGADMIN_PORT=5050
PGADMIN_DEFAULT_EMAIL=admin@admin.com
PGADMIN_DEFAULT_PASSWORD=admin123
```

⚠️ **Измените email и пароль для production!**

### 2. Запустите сервисы

pgAdmin запускается автоматически с базовой конфигурацией:
```bash
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

### 3. Откройте pgAdmin

**URL:** http://localhost:5050

**Credentials:**
- Email: `admin@admin.com` (или из `.env`)
- Password: `admin123` (или из `.env`)

### 4. Подключитесь к PostgreSQL

pgAdmin уже настроен на подключение к PostgreSQL:

1. В левом меню найдите **"RAG PostgreSQL (pgvector)"**
2. Кликните на сервер
3. При первом подключении введите пароль PostgreSQL:
   - Пароль: значение `POSTGRES_PASSWORD` из `.env`
4. ✅ Готово! Можете выполнять SQL запросы

## 📊 Pre-configured Features

### Автоматическая конфигурация сервера
Файл `docker/pgadmin-servers.json` содержит:
- **Name:** RAG PostgreSQL (pgvector)
- **Host:** postgres-pgvector (docker network)
- **Port:** 5432
- **Database:** ollama_proxy
- **User:** proxy_user

### Полезные возможности pgAdmin
- ✅ Query Tool - выполнение SQL запросов
- ✅ Schema Browser - визуализация структуры БД
- ✅ Table Editor - редактирование данных
- ✅ Query History - история выполненных запросов
- ✅ ERD Tool - Entity Relationship Diagrams
- ✅ Data Import/Export - импорт/экспорт данных
- ✅ Explain Plans - анализ производительности запросов

## 🔍 Полезные SQL запросы в pgAdmin

### Проверка pgvector
```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
```

### RAG статистика
```sql
SELECT * FROM rag.statistics;
```

### Размер таблиц
```sql
SELECT * FROM rag.table_sizes;
```

### Использование индексов
```sql
SELECT * FROM rag.index_usage;
```

### Поиск похожих векторов
```sql
-- Замените [1,2,3...] на ваш embedding вектор
SELECT 
    chunk_text,
    1 - (embedding <=> '[1,2,3,...]'::vector) AS similarity
FROM rag.document_chunks
ORDER BY embedding <=> '[1,2,3,...]'::vector
LIMIT 5;
```

## 🛠️ Управление

### Остановить pgAdmin
```bash
docker stop rag-pgadmin
```

### Запустить pgAdmin
```bash
docker start rag-pgadmin
```

### Просмотр логов
```bash
docker logs -f rag-pgadmin
```

### Очистка данных pgAdmin
```bash
docker volume rm rag_pgadmin_data
```

## 📝 Технические детали

### Образ Docker
- **Image:** `dpage/pgadmin4:latest`
- **Официальный:** да
- **Поддержка:** активная
- **Документация:** https://www.pgadmin.org/docs/

### Порты
- **5050:80** - Web UI доступен на порту 5050 хоста

### Memory Limits
- **Limit:** 512MB
- **Reservation:** 256MB

### Volume
- `/var/lib/pgadmin` - данные pgAdmin (настройки, история запросов)

### Health Check
```bash
wget --no-verbose --tries=1 --spider http://localhost/misc/ping
```

## 🔒 Security

### В development
- По умолчанию: `admin@admin.com` / `admin123`
- Режим: single-user (Server Mode: False)
- Master password: отключен

### В production
1. Измените `PGADMIN_DEFAULT_EMAIL` и `PGADMIN_DEFAULT_PASSWORD`
2. Используйте сложные пароли
3. Включите SSL:
   ```yaml
   environment:
     PGADMIN_CONFIG_SERVER_MODE: "True"
     PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED: "True"
   ```
4. Ограничьте доступ через firewall/reverse proxy

## 🐛 Troubleshooting

### pgAdmin не запускается
```bash
# Проверить логи
docker logs rag-pgadmin

# Проверить порт
netstat -ano | findstr :5050  # Windows
lsof -i :5050                 # Linux/macOS
```

### Не могу подключиться к PostgreSQL
1. Проверьте что PostgreSQL запущен: `docker ps | grep rag-postgres`
2. Проверьте пароль в `.env`: должен совпадать с `POSTGRES_PASSWORD`
3. Проверьте сеть: оба контейнера в `rag_network`

### Забыл пароль pgAdmin
```bash
# Удалить volume и пересоздать
docker-compose -f docker-compose.rag-infrastructure.yml down
docker volume rm rag_pgadmin_data
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

## 📚 Дополнительные ресурсы

- [pgAdmin Documentation](https://www.pgadmin.org/docs/)
- [pgvector Extension Guide](https://github.com/pgvector/pgvector)
- [PostgreSQL Official Docs](https://www.postgresql.org/docs/)

## ✅ Готово!

pgAdmin полностью интегрирован и готов к использованию.

**Быстрый запуск:**
```bash
# 1. Добавь переменные в .env
# 2. Запусти
docker-compose -f docker-compose.rag-infrastructure.yml up -d
# 3. Открой http://localhost:5050
```

Удобного управления PostgreSQL! 🚀

