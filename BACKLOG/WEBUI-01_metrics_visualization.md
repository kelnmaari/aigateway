# WEBUI-01: Metrics Visualization

**Приоритет:** HIGH  
**Версия:** 1.2.0  
**Оценка времени:** 8-10 часов  
**Зависимости:** Phase 12.1 (Metrics Storage), Phase 12.2 (WebSocket)

---

## Цель

Добавить визуализацию метрик в WebUI с использованием графиков для real-time и historical данных.

---

## Требования

### Функциональные

1. **Real-time Charts**
   - Line chart для request latency (last 5 minutes)
   - Bar chart для request count per minute
   - Pie chart для status codes distribution
   - Area chart для active requests

2. **Historical Charts**
   - Latency trends (1h, 6h, 24h, 7d)
   - Request volume по времени
   - Error rate trends
   - Token usage trends

3. **Interactive Features**
   - Zoom in/out на графиках
   - Hover tooltips с детальной информацией
   - Time range selector
   - Auto-refresh toggle

4. **Chart Types**
   - Line charts (latency, trends)
   - Bar charts (counts, volumes)
   - Pie charts (distributions)
   - Area charts (stacked metrics)

### Технические

- Использовать **Chart.js** v4+ (легковесная, responsive)
- WebSocket integration для real-time updates
- Использовать существующие `/api/metrics/*` endpoints
- Responsive design (адаптация к размеру экрана)
- Performance: throttle updates (max 1 update/second)

---

## Дизайн

### Новая страница "Analytics"

```
┌─────────────────────────────────────────────┐
│  📈 Analytics                               │
├─────────────────────────────────────────────┤
│                                             │
│  [Time Range: 1h ▼] [Auto-refresh: ON]    │
│                                             │
│  ┌───────────────────────────────────────┐ │
│  │  Request Latency (ms)                 │ │
│  │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │ │
│  │      Line Chart with P50/P95/P99      │ │
│  └───────────────────────────────────────┘ │
│                                             │
│  ┌──────────────┐  ┌────────────────────┐  │
│  │ Requests/min │  │ Status Distribution│  │
│  │ Bar Chart    │  │ Pie Chart          │  │
│  └──────────────┘  └────────────────────┘  │
│                                             │
│  ┌───────────────────────────────────────┐ │
│  │  Token Usage by Model                 │ │
│  │  Stacked Area Chart                   │ │
│  └───────────────────────────────────────┘ │
│                                             │
└─────────────────────────────────────────────┘
```

---

## Реализация

### 1. Frontend (8 часов)

#### Chart.js Integration (2 часа)

```javascript
// app.js additions
let charts = {};

function initCharts() {
    // Latency chart
    charts.latency = new Chart(
        document.getElementById('latencyChart'),
        {
            type: 'line',
            data: { datasets: [...] },
            options: { responsive: true, ... }
        }
    );
    
    // Request count chart
    charts.requests = new Chart(...);
    
    // Status distribution chart
    charts.status = new Chart(...);
}

function updateCharts(data) {
    // Update from WebSocket events
    charts.latency.data.datasets[0].data.push({
        x: data.timestamp,
        y: data.latency
    });
    charts.latency.update('none'); // No animation for smooth updates
}
```

#### WebSocket Integration (2 часа)

```javascript
// Listen to metrics_update events
wsClient.on('metrics_update', (event) => {
    updateCharts(event.data);
});

// Fetch historical data on load
async function loadHistoricalMetrics() {
    const period = document.getElementById('timeRange').value;
    const response = await fetch(`/api/metrics/history?period=${period}&interval=1m`);
    const data = await response.json();
    populateCharts(data.time_series);
}
```

#### HTML/CSS (2 часа)

```html
<!-- index.html -->
<div id="view-analytics" class="view">
    <div class="analytics-controls">
        <select id="timeRange" onchange="changeTimeRange(this.value)">
            <option value="1h">Last Hour</option>
            <option value="6h">Last 6 Hours</option>
            <option value="24h">Last 24 Hours</option>
            <option value="7d">Last 7 Days</option>
        </select>
        <label>
            <input type="checkbox" id="autoRefresh" checked onchange="toggleAutoRefresh()">
            Auto-refresh
        </label>
    </div>
    
    <div class="chart-container">
        <canvas id="latencyChart"></canvas>
    </div>
    
    <div class="chart-row">
        <div class="chart-container-half">
            <canvas id="requestsChart"></canvas>
        </div>
        <div class="chart-container-half">
            <canvas id="statusChart"></canvas>
        </div>
    </div>
    
    <div class="chart-container">
        <canvas id="tokensChart"></canvas>
    </div>
</div>
```

#### Interactive Features (2 часа)

- Time range selector
- Auto-refresh toggle
- Chart zoom/pan
- Export to PNG

### 2. Backend (0 часов)

- Используем существующие endpoints
- Никаких изменений не требуется

---

## Метрики для визуализации

### Primary Metrics

1. **Request Latency**
   - Source: `GET /api/metrics/history?type=request_latency`
   - Chart: Line (P50, P95, P99)
   - Update: Real-time via WebSocket

2. **Request Count**
   - Source: `GET /api/metrics/history?type=request_count`
   - Chart: Bar
   - Aggregation: per minute/hour

3. **Error Rate**
   - Source: `GET /api/metrics/history?type=error_count`
   - Chart: Line
   - Formula: errors / total * 100

4. **Active Requests**
   - Source: `GET /api/metrics/history?type=active_requests`
   - Chart: Area
   - Update: Real-time

### Secondary Metrics

5. **Request/Response Sizes**
   - Charts: Dual-axis line chart
   - Useful для bandwidth monitoring

6. **Status Code Distribution**
   - Chart: Pie/Doughnut
   - Categories: 2xx, 4xx, 5xx

---

## Тестирование

### Manual Testing

- [ ] Графики отрисовываются корректно
- [ ] Real-time updates работают
- [ ] Time range selector меняет данные
- [ ] Auto-refresh работает
- [ ] Responsive на разных экранах
- [ ] No memory leaks при длительной работе

### Performance Testing

- [ ] Chart updates не блокируют UI
- [ ] Memory usage стабилен (<50MB)
- [ ] CPU usage минимален (<5%)

---

## Зависимости

### NPM (CDN)

```html
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
```

Или локально через Go embed:

- Скачать chart.js в `internal/web/static/js/`
- Embed через `//go:embed`

---

## Альтернативы

### Другие chart libraries

1. **Apache ECharts** - мощнее, но тяжелее
2. **Plotly.js** - интерактивнее, но больше overhead
3. **D3.js** - максимальная кастомизация, сложнее

**Выбор:** Chart.js - баланс простоты и функциональности

---

## Risks & Mitigation

**Risk:** Chart.js может быть тяжелым для мобильных  
**Mitigation:** Lazy loading, только на desktop показывать все графики

**Risk:** Too many updates могут тормозить  
**Mitigation:** Throttle updates до 1/second, buffering

**Risk:** Historical data может быть большим  
**Mitigation:** Server-side aggregation, pagination

---

## Success Criteria

- ✅ Все 4+ типа графиков работают
- ✅ Real-time updates без лагов
- ✅ Historical data loading < 1s
- ✅ Responsive design на всех экранах
- ✅ Export to PNG работает
- ✅ Memory leaks отсутствуют

---

**Статус:** 📋 Ready for Implementation  
**Assignee:** TBD  
**Sprint:** Version 1.2.0
