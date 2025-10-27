# BUG-01: Fix Admin Logs Display

**Версия:** v1.5.1  
**Приоритет:** CRITICAL  
**Оценка:** 2-3 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Bug Fix

---

## 📋 Описание

**CRITICAL BUG:** В админке (System → Logs) логи отображаются как `[object Object]` вместо нормального текста. Это классический JavaScript баг неправильной сериализации объектов.

## 🐛 Bug Details

### Симптомы

**Текущее отображение:**

```
[object Object]
[object Object]
[object Object]
```

**Ожидаемое отображение:**

```
2025-10-10 12:34:56 INFO Server started on :8080
2025-10-10 12:35:01 INFO Request received: POST /api/chat/completions
2025-10-10 12:35:02 WARN Rate limit approaching for key: sk-xxx
```

### Root Cause

JavaScript пытается отобразить объект как строку без правильной сериализации.

**Проблемный код в `web/js/admin.js`:**

```javascript
// Где-то в renderLogs() или appendLog()
logElement.textContent = logEntry; // logEntry - это объект!
// или
innerHTML = `<div>${logEntry}</div>`; // toString() на объекте = "[object Object]"
```

## 🔍 Investigation

### Expected Log Structure

**Backend возвращает:**

```json
{
  "logs": [
    {
      "timestamp": "2025-10-10T12:34:56Z",
      "level": "INFO",
      "message": "Server started on :8080",
      "fields": {
        "version": "1.4.3",
        "port": 8080
      }
    },
    {
      "timestamp": "2025-10-10T12:35:01Z",
      "level": "INFO", 
      "message": "Request received",
      "fields": {
        "method": "POST",
        "path": "/api/chat/completions",
        "user_id": "uuid-123"
      }
    }
  ]
}
```

### Current JavaScript (Broken)

**Файл:** `web/js/admin.js`

```javascript
// Вероятный код (нужно проверить):
function renderLogs(logs) {
    const container = document.getElementById('logsContainer');
    container.innerHTML = logs.map(log => `
        <div class="log-entry">
            ${log}  <!-- ❌ Вот проблема! -->
        </div>
    `).join('');
}
```

## 🔧 Solution

### 1. Fix Log Rendering

**Файл:** `web/js/admin.js`

#### Вариант 1: Простой (рекомендуемый)

```javascript
function renderLogs(logs) {
    const container = document.getElementById('logsContainer');
    
    if (!logs || logs.length === 0) {
        container.innerHTML = '<p class="text-muted">No logs available</p>';
        return;
    }
    
    container.innerHTML = logs.map(log => {
        // Безопасная сериализация объекта лога
        const timestamp = log.timestamp || '';
        const level = log.level || 'INFO';
        const message = log.message || '';
        const fields = log.fields || {};
        
        // Форматирование timestamp
        const formattedTime = formatTimestamp(timestamp);
        
        // CSS класс для уровня
        const levelClass = getLevelClass(level);
        
        // Дополнительные поля (если есть)
        const fieldsStr = Object.keys(fields).length > 0 
            ? `<small class="text-muted ms-2">${formatFields(fields)}</small>`
            : '';
        
        return `
            <div class="log-entry log-${levelClass} mb-2 p-2 border-start border-3">
                <span class="log-timestamp text-muted">${formattedTime}</span>
                <span class="log-level badge bg-${levelClass} ms-2">${level}</span>
                <span class="log-message ms-2">${escapeHtml(message)}</span>
                ${fieldsStr}
            </div>
        `;
    }).join('');
}

// Helper: Форматирование timestamp
function formatTimestamp(timestamp) {
    if (!timestamp) return '';
    try {
        const date = new Date(timestamp);
        return date.toLocaleString('ru-RU', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch (e) {
        return timestamp;
    }
}

// Helper: CSS класс для уровня
function getLevelClass(level) {
    const levelMap = {
        'ERROR': 'danger',
        'WARN': 'warning',
        'WARNING': 'warning',
        'INFO': 'info',
        'DEBUG': 'secondary',
        'TRACE': 'light'
    };
    return levelMap[level.toUpperCase()] || 'secondary';
}

// Helper: Форматирование дополнительных полей
function formatFields(fields) {
    return Object.entries(fields)
        .map(([key, value]) => `${key}=${value}`)
        .join(' ');
}

// Helper: Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
```

#### Вариант 2: С детальным просмотром

```javascript
function renderLogs(logs) {
    const container = document.getElementById('logsContainer');
    
    container.innerHTML = logs.map((log, index) => {
        const formattedTime = formatTimestamp(log.timestamp);
        const levelClass = getLevelClass(log.level);
        const hasFields = log.fields && Object.keys(log.fields).length > 0;
        
        return `
            <div class="log-entry mb-2">
                <div class="log-header p-2 border-start border-3 border-${levelClass} bg-light" 
                     ${hasFields ? `onclick="toggleLogDetails(${index})" style="cursor: pointer;"` : ''}>
                    <span class="log-timestamp text-muted">${formattedTime}</span>
                    <span class="log-level badge bg-${levelClass} ms-2">${log.level}</span>
                    <span class="log-message ms-2">${escapeHtml(log.message)}</span>
                    ${hasFields ? '<i class="fas fa-chevron-down ms-2 toggle-icon"></i>' : ''}
                </div>
                ${hasFields ? `
                <div id="log-details-${index}" class="log-details p-2 bg-white border" style="display: none;">
                    <pre class="mb-0"><code>${JSON.stringify(log.fields, null, 2)}</code></pre>
                </div>
                ` : ''}
            </div>
        `;
    }).join('');
}

// Toggle log details
function toggleLogDetails(index) {
    const details = document.getElementById(`log-details-${index}`);
    const icon = details.previousElementSibling.querySelector('.toggle-icon');
    
    if (details.style.display === 'none') {
        details.style.display = 'block';
        icon.className = 'fas fa-chevron-up ms-2 toggle-icon';
    } else {
        details.style.display = 'none';
        icon.className = 'fas fa-chevron-down ms-2 toggle-icon';
    }
}
```

### 2. CSS Styling

**Файл:** `web/css/admin.css` или inline в `web/admin.html`

```css
/* Log entries styling */
.log-entry {
    font-family: 'Courier New', monospace;
    font-size: 0.9rem;
}

.log-entry.log-danger {
    background-color: #fff5f5;
}

.log-entry.log-warning {
    background-color: #fffbf0;
}

.log-entry.log-info {
    background-color: #f0f9ff;
}

.log-timestamp {
    font-size: 0.85rem;
}

.log-message {
    font-weight: 500;
}

.log-details pre {
    background-color: #f8f9fa;
    border-radius: 4px;
    padding: 8px;
    font-size: 0.85rem;
}

.log-header:hover {
    background-color: #e9ecef !important;
}

.toggle-icon {
    transition: transform 0.2s;
    font-size: 0.8rem;
}
```

### 3. WebSocket Real-time Updates (Optional Enhancement)

Если логи обновляются через WebSocket:

```javascript
// WebSocket connection для real-time logs
let logsSocket = null;

function connectLogsWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/admin/logs`;
    
    logsSocket = new WebSocket(wsUrl);
    
    logsSocket.onmessage = (event) => {
        const logEntry = JSON.parse(event.data);
        appendLogEntry(logEntry);
    };
    
    logsSocket.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    logsSocket.onclose = () => {
        console.log('WebSocket closed, reconnecting...');
        setTimeout(connectLogsWebSocket, 5000);
    };
}

function appendLogEntry(log) {
    const container = document.getElementById('logsContainer');
    const formattedTime = formatTimestamp(log.timestamp);
    const levelClass = getLevelClass(log.level);
    
    const logHTML = `
        <div class="log-entry log-${levelClass} mb-2 p-2 border-start border-3">
            <span class="log-timestamp text-muted">${formattedTime}</span>
            <span class="log-level badge bg-${levelClass} ms-2">${log.level}</span>
            <span class="log-message ms-2">${escapeHtml(log.message)}</span>
        </div>
    `;
    
    container.insertAdjacentHTML('afterbegin', logHTML);
    
    // Limit to 100 entries
    const entries = container.children;
    if (entries.length > 100) {
        container.removeChild(entries[entries.length - 1]);
    }
}
```

## 📝 Implementation Plan

### Phase 1: Investigation (15 мин)

1. Проверить `web/js/admin.js` - найти функцию renderLogs
2. Проверить API endpoint - структуру логов
3. Проверить WebSocket (если используется)

### Phase 2: Fix Rendering (45 мин)

1. Обновить renderLogs функцию
2. Добавить helper functions
3. Добавить CSS стили
4. Тестирование в браузере

### Phase 3: Enhancements (30 мин)

1. Добавить level filtering (если нет)
2. Добавить export (если нет)
3. Улучшить formatting

### Phase 4: Testing (30 мин)

1. Тестировать разные типы логов
2. Проверить на разных браузерах
3. Проверить real-time updates

## ✅ Acceptance Criteria

- [ ] Логи отображаются нормально (не `[object Object]`)
- [ ] Timestamp форматируется читаемо
- [ ] Level отображается с цветом (ERROR=red, WARN=yellow, INFO=blue)
- [ ] Message отображается корректно
- [ ] Дополнительные поля (fields) видны
- [ ] HTML экранируется (безопасность)
- [ ] CSS стилизация применена
- [ ] Работает на Chrome, Firefox, Safari
- [ ] Mobile responsive

## 🧪 Testing

### Manual Testing

1. **Открыть админку:**

   ```
   http://localhost:8080/admin.html
   ```

2. **Перейти в System → Logs**

3. **Проверить отображение:**
   - [ ] Логи видны и читаемы
   - [ ] Timestamp корректный
   - [ ] Уровни с правильными цветами
   - [ ] Messages без `[object Object]`

4. **Test different log levels:**
   - Trigger ERROR (неправильный запрос)
   - Trigger WARN (rate limit)
   - Trigger INFO (обычный запрос)

### Browser Testing

- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)

### API Testing

```bash
# Проверить формат логов от API
curl -H "Authorization: Bearer admin-key" \
     http://localhost:8080/api/admin/logs | jq

# Expected:
{
  "logs": [
    {
      "timestamp": "...",
      "level": "INFO",
      "message": "...",
      "fields": {...}
    }
  ]
}
```

## 📚 References

- [JavaScript Object.toString()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/toString)
- [JSON.stringify()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify)
- [Bootstrap Colors](https://getbootstrap.com/docs/5.0/utilities/colors/)

## 🔄 Follow-up Tasks

- [ ] Добавить log levels filter (если отсутствует)
- [ ] Добавить search по логам
- [ ] Добавить export в файл
- [ ] Real-time updates через WebSocket
- [ ] Log rotation visualization
- [ ] Performance optimization для больших логов (virtualization)

## 🔍 Debug Tips

Если проблема сохраняется:

1. **Проверить в Console:**

   ```javascript
   // В DevTools Console
   console.log(typeof logs[0]); // должно быть "object"
   console.log(logs[0]); // посмотреть структуру
   ```

2. **Проверить Network tab:**
   - Response от `/api/admin/logs`
   - Структура JSON

3. **Проверить код:**

   ```javascript
   // Найти где используется log
   // Плохо:
   element.innerHTML = log; // [object Object]
   
   // Хорошо:
   element.innerHTML = log.message; // реальное сообщение
   ```

