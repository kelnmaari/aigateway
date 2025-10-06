// WebSocket client для real-time updates

class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.reconnectInterval = 3000; // 3 seconds
        this.reconnectTimer = null;
        this.isConnecting = false;
        this.eventHandlers = {};
        this.connectionStatusCallbacks = [];
    }

    // Подключение к WebSocket серверу
    connect() {
        if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
            return;
        }

        this.isConnecting = true;
        console.log('[WebSocket] Connecting to:', this.url);

        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                console.log('[WebSocket] Connected');
                this.isConnecting = false;
                this.notifyConnectionStatus(true);
                
                // Отменяем таймер переподключения
                if (this.reconnectTimer) {
                    clearTimeout(this.reconnectTimer);
                    this.reconnectTimer = null;
                }
            };

            this.ws.onmessage = (event) => {
                this.handleMessage(event.data);
            };

            this.ws.onerror = (error) => {
                console.error('[WebSocket] Error:', error);
                this.isConnecting = false;
            };

            this.ws.onclose = () => {
                console.log('[WebSocket] Disconnected');
                this.isConnecting = false;
                this.notifyConnectionStatus(false);
                this.scheduleReconnect();
            };

        } catch (error) {
            console.error('[WebSocket] Connection error:', error);
            this.isConnecting = false;
            this.scheduleReconnect();
        }
    }

    // Обработка входящих сообщений
    handleMessage(data) {
        try {
            const event = JSON.parse(data);
            console.log('[WebSocket] Received event:', event.type);

            // Вызываем обработчики для этого типа события
            if (this.eventHandlers[event.type]) {
                this.eventHandlers[event.type].forEach(handler => {
                    try {
                        handler(event);
                    } catch (error) {
                        console.error('[WebSocket] Handler error:', error);
                    }
                });
            }

            // Вызываем универсальный обработчик
            if (this.eventHandlers['*']) {
                this.eventHandlers['*'].forEach(handler => {
                    try {
                        handler(event);
                    } catch (error) {
                        console.error('[WebSocket] Universal handler error:', error);
                    }
                });
            }

        } catch (error) {
            console.error('[WebSocket] Failed to parse message:', error);
        }
    }

    // Регистрация обработчика события
    on(eventType, handler) {
        if (!this.eventHandlers[eventType]) {
            this.eventHandlers[eventType] = [];
        }
        this.eventHandlers[eventType].push(handler);
    }

    // Удаление обработчика события
    off(eventType, handler) {
        if (!this.eventHandlers[eventType]) {
            return;
        }

        const index = this.eventHandlers[eventType].indexOf(handler);
        if (index > -1) {
            this.eventHandlers[eventType].splice(index, 1);
        }
    }

    // Отправка сообщения на сервер
    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        } else {
            console.warn('[WebSocket] Cannot send message - not connected');
        }
    }

    // Планирование переподключения
    scheduleReconnect() {
        if (this.reconnectTimer) {
            return; // Уже запланировано
        }

        console.log(`[WebSocket] Reconnecting in ${this.reconnectInterval}ms...`);
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, this.reconnectInterval);
    }

    // Закрытие соединения
    disconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    // Регистрация callback для изменения статуса подключения
    onConnectionStatusChange(callback) {
        this.connectionStatusCallbacks.push(callback);
    }

    // Уведомление о изменении статуса подключения
    notifyConnectionStatus(isConnected) {
        this.connectionStatusCallbacks.forEach(callback => {
            try {
                callback(isConnected);
            } catch (error) {
                console.error('[WebSocket] Connection status callback error:', error);
            }
        });
    }

    // Проверка статуса подключения
    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}

// Создаем глобальный экземпляр WebSocket client
const wsClient = new WebSocketClient(`ws://${window.location.host}/ws`);

// Автоматическое подключение при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    console.log('[WebSocket] Initializing...');
    wsClient.connect();

    // Показываем индикатор подключения
    wsClient.onConnectionStatusChange((isConnected) => {
        const indicator = document.getElementById('ws-status-indicator');
        if (indicator) {
            if (isConnected) {
                indicator.classList.add('connected');
                indicator.classList.remove('disconnected');
                indicator.title = 'WebSocket: Connected';
            } else {
                indicator.classList.add('disconnected');
                indicator.classList.remove('connected');
                indicator.title = 'WebSocket: Disconnected';
            }
        }

        // Toast notification
        if (isConnected) {
            if (typeof Toast !== 'undefined') {
                Toast.success('Real-time updates connected');
            }
        } else {
            if (typeof Toast !== 'undefined') {
                Toast.warning('Real-time updates disconnected');
            }
        }
    });
});

// Обработчики событий для WebUI
wsClient.on('server_stats', (event) => {
    console.log('[WebSocket] Server stats updated');
    // Обновляем UI с новыми данными
    if (typeof updateDashboardFromWebSocket === 'function') {
        updateDashboardFromWebSocket(event.data);
    }
});

wsClient.on('metrics_update', (event) => {
    console.log('[WebSocket] Metrics updated');
    // Обновляем метрики на dashboard
    if (typeof updateMetricsFromWebSocket === 'function') {
        updateMetricsFromWebSocket(event.data);
    }
});

wsClient.on('api_key_created', (event) => {
    console.log('[WebSocket] API key created');
    // Обновляем список ключей
    if (typeof loadApiKeys === 'function') {
        loadApiKeys();
    }
    if (typeof Toast !== 'undefined') {
        Toast.info('API key created');
    }
});

wsClient.on('api_key_deleted', (event) => {
    console.log('[WebSocket] API key deleted');
    // Обновляем список ключей
    if (typeof loadApiKeys === 'function') {
        loadApiKeys();
    }
    if (typeof Toast !== 'undefined') {
        Toast.info('API key deleted');
    }
});

wsClient.on('new_log', (event) => {
    console.log('[WebSocket] New log entry');
    // Добавляем новую запись лога
    if (typeof appendLogEntry === 'function') {
        appendLogEntry(event.data);
    }
});

wsClient.on('heartbeat', (event) => {
    // Heartbeat для поддержания соединения
    console.debug('[WebSocket] Heartbeat received');
});

wsClient.on('error', (event) => {
    console.error('[WebSocket] Server error:', event.data);
    if (typeof Toast !== 'undefined') {
        Toast.error(event.data.error || 'Server error');
    }
});

// Экспортируем для использования в других скриптах
window.wsClient = wsClient;

