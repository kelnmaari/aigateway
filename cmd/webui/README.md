# 🌐 Ollama-OpenAI Proxy WebUI

Веб-интерфейс для мониторинга и управления Ollama-OpenAI Proxy.

## 📋 Возможности

- **Dashboard** - Реальное время метрик и статистики
- **API Keys Management** - Создание, просмотр и удаление API ключей
- **Models Viewer** - Список доступных моделей Ollama
- **Configuration Viewer** - Просмотр конфигурации сервера
- **Logs Viewer** - Просмотр логов (планируется)

## 🚀 Запуск

### Базовый запуск

```bash
# Запустить WebUI (подключится к http://localhost:8080)
./bin/webui.exe
```

WebUI будет доступен по адресу: **http://localhost:8081**

### С параметрами

```bash
# Указать адрес основного сервера
./bin/webui.exe -server-url http://192.168.1.100:8080

# Изменить порт WebUI
./bin/webui.exe -port 9000

# Изменить host
./bin/webui.exe -host 127.0.0.1

# Все параметры вместе
./bin/webui.exe -server-url http://192.168.1.100:8080 -port 9000 -host 0.0.0.0
```

## ⚙️ Параметры командной строки

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| `-server-url` | `http://localhost:8080` | URL основного API сервера |
| `-port` | `8081` | Порт WebUI сервера |
| `-host` | `0.0.0.0` | Host для прослушивания |

## 🔧 Архитектура

WebUI работает как **отдельный веб-сервер**, который:

1. **Не зависит от основного сервера** - запускается отдельно
2. **Подключается через HTTP API** - использует те же эндпоинты что и TUI
3. **Встроенные статические файлы** - всё в одном binary, не требует Node.js
4. **Auto-refresh** - обновляет данные каждые 5 секунд

```
┌─────────────┐         HTTP API          ┌─────────────────┐
│   Browser   │ ◄────────────────────────► │  WebUI Server   │
│             │                            │   (port 8081)   │
└─────────────┘                            └────────┬────────┘
                                                    │
                                           HTTP API │
                                                    │
                                            ┌───────▼────────┐
                                            │  Proxy Server  │
                                            │  (port 8080)   │
                                            └────────────────┘
```

## 📊 Использование

### Dashboard

- **Total Requests** - Общее количество запросов
- **Success Rate** - Процент успешных запросов
- **Avg Latency** - Средняя задержка ответа
- **Active Keys** - Количество активных API ключей
- **Server Info** - Статус, uptime, адрес
- **Ollama Connection** - Статус подключения, модели

### API Keys Management

1. Нажмите **"Create New Key"**
2. Заполните форму:
   - **Key Name** - Имя ключа
   - **Rate Limit** - Лимит запросов в минуту
   - **Models** - Разрешенные модели (* для всех)
   - **Admin Key** - Ваш админ ключ для создания
3. Нажмите **"Create Key"**
4. **Сохраните ключ!** - он показывается только один раз

**Удаление ключа:**
- Нажмите кнопку "Delete" напротив ключа
- Подтвердите действие
- Введите админ ключ

### Models Viewer

Показывает все доступные модели из Ollama:
- ID модели
- Дата создания

### Configuration Viewer

Показывает текущую конфигурацию сервера в формате JSON (без чувствительных данных).

## 🎨 Особенности UI

- **Темная тема** - Современный dark mode интерфейс
- **Responsive design** - Адаптируется под размер экрана
- **Real-time updates** - Автоматическое обновление каждые 5 секунд
- **Color coding** - Цветовая индикация статусов (зеленый = OK, красный = ошибка)
- **Smooth animations** - Плавные переходы и анимации

## 🔒 Безопасность

**WebUI НЕ хранит API ключи!**

- Все запросы проксируются на основной сервер
- Аутентификация происходит на сервере
- WebUI не кэширует чувствительные данные
- Admin ключ запрашивается для каждой операции

**Для production:**
- Используйте `-host 127.0.0.1` для локального доступа
- Настройте reverse proxy (nginx) с HTTPS
- Используйте firewall для ограничения доступа

## 🐛 Troubleshooting

### WebUI не может подключиться к серверу

```
Error: Failed to connect to server
```

**Решение:**
1. Проверьте что основной сервер запущен: `curl http://localhost:8080/health`
2. Проверьте URL сервера: `./bin/webui.exe -server-url http://localhost:8080`
3. Проверьте firewall

### API Keys не загружаются

```
Error: Failed to load API keys
```

**Решение:**
1. Убедитесь что вы ввели правильный admin ключ
2. Проверьте что ключ имеет админские права
3. Проверьте логи основного сервера

### Статус "Disconnected"

Красный индикатор в sidebar означает что WebUI не может подключиться к серверу.

**Решение:**
1. Проверьте что сервер запущен
2. Проверьте URL: `/info` endpoint показывает `server_url`
3. Перезапустите WebUI

## 📝 API Endpoints (WebUI)

WebUI предоставляет следующие эндпоинты:

- `GET /` - Главная страница
- `GET /health` - Health check
- `GET /info` - Информация о WebUI
- `GET /api/stats` - Проксирует `/api/stats` основного сервера
- `GET /api/config` - Проксирует `/api/config`
- `GET /api/models` - Проксирует `/v1/models`
- `POST /api/admin/keys` - Создание API ключа
- `GET /api/admin/keys` - Список API ключей
- `DELETE /api/admin/keys/:id` - Удаление API ключа

## 🔮 Roadmap

- [ ] **Real-time logs streaming** через WebSocket
- [ ] **Request monitoring** - Live таблица активных запросов
- [ ] **Metrics charts** - Графики с историей метрик
- [ ] **Authentication** для WebUI
- [ ] **Themes** - Light/Dark mode переключение
- [ ] **API playground** - Тестирование запросов
- [ ] **WebSocket updates** вместо polling

## 📚 Технологии

- **Backend**: Go + Gin
- **Frontend**: Vanilla JavaScript (без фреймворков!)
- **Styling**: Custom CSS (темная тема)
- **Embed**: Go embed для встраивания статики
- **HTTP**: Стандартная библиотека Go

## 🤝 Contributing

WebUI - часть проекта Ollama-OpenAI Proxy.

См. основной [README.md](../../README.md) для информации о разработке.

## 📄 License

См. [LICENSE](../../LICENSE) в корне проекта.

