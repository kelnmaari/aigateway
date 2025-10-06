# WEBUI-02: Advanced Logs Features

**Приоритет:** HIGH  
**Версия:** 1.2.0  
**Оценка времени:** 3-4 часа  
**Зависимости:** Существующий Logs Viewer

---

## Цель

Улучшить функциональность просмотра логов в WebUI: добавить client-side фильтрацию, поиск и экспорт.

---

## Требования

### 1. Client-side Filtering (1 час)

- Кнопки фильтрации уже есть, но работают через server API
- Добавить client-side фильтрацию для мгновенного отклика
- Сохранять последний выбранный фильтр в localStorage

### 2. Text Search (1 час)

```html
<input type="text" id="logSearch" placeholder="Search logs..." 
       oninput="searchLogs(this.value)">
```

- Поиск по тексту (case-insensitive)
- Highlight найденных совпадений
- Показать количество результатов

### 3. Export Functionality (1 час)

- **Export to TXT** - plain text file
- **Export to JSON** - structured format
- Применять текущие фильтры при экспорте
- Download с timestamp в имени

### 4. Auto-refresh Control (0.5 часа)

```html
<label>
    <input type="checkbox" id="logsAutoRefresh" checked>
    Auto-refresh logs
</label>
```

- Toggle для остановки auto-refresh
- Useful когда ищешь что-то в логах

### 5. Performance (0.5 часа)

- Виртуальный scrolling для больших логов (>1000 lines)
- Lazy rendering

---

## Реализация

### JavaScript

```javascript
// Client-side filter
function filterLogsClientSide(level) {
    const logs = document.querySelectorAll('.log-entry');
    logs.forEach(log => {
        if (level === 'all' || log.classList.contains(level)) {
            log.style.display = '';
        } else {
            log.style.display = 'none';
        }
    });
}

// Search
function searchLogs(query) {
    if (!query) {
        // Clear highlights
        return;
    }
    
    const logs = document.querySelectorAll('.log-entry');
    let matchCount = 0;
    
    logs.forEach(log => {
        const text = log.textContent.toLowerCase();
        if (text.includes(query.toLowerCase())) {
            log.innerHTML = highlightText(log.textContent, query);
            log.style.display = '';
            matchCount++;
        } else {
            log.style.display = 'none';
        }
    });
    
    document.getElementById('searchResults').textContent = 
        `Found ${matchCount} matches`;
}

// Export
function exportLogsToTXT() {
    const logs = Array.from(document.querySelectorAll('.log-entry:not([style*="display: none"])'));
    const text = logs.map(log => log.textContent).join('\n');
    downloadFile(text, `logs-${Date.now()}.txt`, 'text/plain');
}

function exportLogsToJSON() {
    const logs = Array.from(document.querySelectorAll('.log-entry:not([style*="display: none"])'));
    const data = logs.map(log => ({
        level: detectLevel(log),
        message: log.textContent,
        timestamp: new Date().toISOString()
    }));
    downloadFile(JSON.stringify(data, null, 2), `logs-${Date.now()}.json`, 'application/json');
}
```

---

## UI Updates

```html
<!-- Add to logs view -->
<div class="logs-toolbar">
    <input type="text" id="logSearch" placeholder="🔍 Search logs..." />
    <span id="searchResults"></span>
    
    <div class="logs-controls">
        <button onclick="exportLogsToTXT()">📄 Export TXT</button>
        <button onclick="exportLogsToJSON()">📋 Export JSON</button>
        <label>
            <input type="checkbox" id="autoRefreshLogs" checked>
            Auto-refresh
        </label>
    </div>
</div>
```

---

## Testing

- [ ] Filter мгновенно работает
- [ ] Search highlights правильно
- [ ] Export включает только видимые логи
- [ ] Auto-refresh toggle работает
- [ ] Performance с 10,000+ logs OK

---

**Статус:** 📋 Ready  
**Complexity:** LOW
