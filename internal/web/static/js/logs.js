// Logs Viewer (v1.5.1)
class LogsViewer {
    constructor() {
        this.logFiles = [];
        this.currentFile = null;
        this.currentLevel = '';
        this.eventSource = null;
        this.isRealtime = false;
        this.allEntries = [];
    }

    async init() {
        this.setupEventListeners();
        await this.loadLogFiles();
    }

    setupEventListeners() {
        // File selector
        document.getElementById('log-file-selector').addEventListener('change', (e) => {
            this.selectLogFile(e.target.value);
        });

        // Download button
        document.getElementById('download-log-btn').addEventListener('click', () => {
            this.downloadCurrentLog();
        });

        // Refresh button
        document.getElementById('refresh-logs-btn').addEventListener('click', () => {
            if (this.currentFile) {
                this.loadLogFile(this.currentFile);
            } else {
                this.loadLogFiles();
            }
        });

        // Realtime toggle
        document.getElementById('realtime-toggle-btn').addEventListener('click', () => {
            this.toggleRealtime();
        });

        // Level filters
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const level = e.currentTarget.getAttribute('data-level');
                this.filterByLevel(level);
            });
        });
    }

    async loadLogFiles() {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/logs`);
            
            if (!response.ok) {
                throw new Error('Failed to load log files');
            }

            const data = await response.json();
            this.logFiles = data.files || [];
            
            this.renderFileSelector();
        } catch (error) {
            console.error('Error loading log files:', error);
            this.showError('Failed to load log files: ' + error.message);
        }
    }

    renderFileSelector() {
        const selector = document.getElementById('log-file-selector');
        selector.innerHTML = '<option value="">Select log file...</option>';

        this.logFiles.forEach(file => {
            const option = document.createElement('option');
            option.value = file.name;
            
            const sizeKB = (file.size / 1024).toFixed(2);
            const date = new Date(file.modified).toLocaleString();
            const isCurrent = file.is_current ? ' (current)' : '';
            
            option.textContent = `${file.name} - ${sizeKB} KB - ${date}${isCurrent}`;
            selector.appendChild(option);
        });

        // Auto-select current log
        const currentLog = this.logFiles.find(f => f.is_current);
        if (currentLog) {
            selector.value = currentLog.name;
            this.selectLogFile(currentLog.name);
        }
    }

    async selectLogFile(filename) {
        if (!filename) {
            this.currentFile = null;
            document.getElementById('download-log-btn').disabled = true;
            this.showInfo('Select a log file to view');
            return;
        }

        this.currentFile = filename;
        document.getElementById('download-log-btn').disabled = false;

        // Stop realtime if switching files
        if (this.isRealtime) {
            this.stopRealtime();
        }

        await this.loadLogFile(filename);
    }

    async loadLogFile(filename) {
        const container = document.getElementById('logs-container');
        container.innerHTML = '<div class="logs-info"><p>Loading logs...</p></div>';

        try {
            const url = `${api.baseURL}/api/admin/logs/${filename}?limit=500`;
            const response = await api.request(url);
            
            if (!response.ok) {
                throw new Error('Failed to load log file');
            }

            const data = await response.json();
            this.allEntries = data.entries || [];
            
            this.renderLogs();
        } catch (error) {
            console.error('Error loading log file:', error);
            this.showError('Failed to load log file: ' + error.message);
        }
    }

    renderLogs() {
        const container = document.getElementById('logs-container');
        container.innerHTML = '';

        if (this.allEntries.length === 0) {
            this.showInfo('No log entries found');
            return;
        }

        this.allEntries.forEach(entry => {
            const logEntry = this.createLogEntry(entry);
            container.appendChild(logEntry);
        });

        // Apply current filter
        this.applyFilter();

        // Auto-scroll to bottom
        container.scrollTop = container.scrollHeight;
    }

    createLogEntry(entry) {
        const div = document.createElement('div');
        const level = entry.level.toLowerCase();
        div.className = `log-entry level-${level}`;
        div.setAttribute('data-level', level);

        const timestamp = entry.timestamp ? `<span class="log-timestamp">${this.formatTimestamp(entry.timestamp)}</span>` : '';
        const levelBadge = `<span class="log-level level-${level}">${entry.level}</span>`;
        
        // Разделяем основное сообщение и context fields
        let messageHtml = this.escapeHtml(entry.message);
        
        // Если есть separator | то выделяем context fields
        if (messageHtml.includes(' | ')) {
            const parts = messageHtml.split(' | ');
            const mainMsg = parts[0];
            const contextFields = parts.slice(1).join(' | ');
            
            // Подсвечиваем key=value пары
            const highlightedContext = this.highlightContextFields(contextFields);
            messageHtml = `${mainMsg}<br><span style="color: #7c7c7c; font-size: 0.9em;">${highlightedContext}</span>`;
        }
        
        const message = `<span class="log-message">${messageHtml}</span>`;

        div.innerHTML = `${timestamp}${levelBadge}${message}`;

        return div;
    }

    formatTimestamp(timestamp) {
        // Форматируем timestamp для лучшей читаемости
        try {
            const date = new Date(timestamp);
            const hours = String(date.getHours()).padStart(2, '0');
            const minutes = String(date.getMinutes()).padStart(2, '0');
            const seconds = String(date.getSeconds()).padStart(2, '0');
            const ms = String(date.getMilliseconds()).padStart(3, '0');
            return `${hours}:${minutes}:${seconds}.${ms}`;
        } catch (e) {
            return timestamp;
        }
    }

    highlightContextFields(text) {
        // Подсвечиваем key=value пары разными цветами
        return text.replace(/(\w+)=([^\s]+)/g, (match, key, value) => {
            return `<span style="color: #569cd6;">${key}</span>=<span style="color: #ce9178;">${value}</span>`;
        });
    }

    filterByLevel(level) {
        this.currentLevel = level;

        // Update button states
        document.querySelectorAll('.filter-btn').forEach(btn => {
            const btnLevel = btn.getAttribute('data-level');
            if (btnLevel === level) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });

        this.applyFilter();
    }

    applyFilter() {
        const entries = document.querySelectorAll('.log-entry');
        
        entries.forEach(entry => {
            const entryLevel = entry.getAttribute('data-level');
            
            if (this.currentLevel === '' || entryLevel === this.currentLevel) {
                entry.classList.remove('hidden');
            } else {
                entry.classList.add('hidden');
            }
        });
    }

    toggleRealtime() {
        if (this.isRealtime) {
            this.stopRealtime();
        } else {
            this.startRealtime();
        }
    }

    startRealtime() {
        if (!this.currentFile) {
            alert('Please select a log file first');
            return;
        }

        if (this.eventSource) {
            this.eventSource.close();
        }

        // Получаем JWT токен из localStorage
        const token = localStorage.getItem('access_token');
        if (!token) {
            alert('Authentication required. Please login again.');
            return;
        }

        // Добавляем токен и имя файла как query параметры (SSE не поддерживает custom headers)
        const url = `${api.baseURL}/api/admin/logs/stream?token=${encodeURIComponent(token)}&file=${encodeURIComponent(this.currentFile)}`;
        this.eventSource = new EventSource(url);

        this.eventSource.addEventListener('log', (e) => {
            const entry = JSON.parse(e.data);
            this.addLogEntry(entry);
        });

        this.eventSource.addEventListener('error', (e) => {
            console.error('SSE error:', e);
            this.stopRealtime();
        });

        this.isRealtime = true;
        this.updateRealtimeButton();
    }

    stopRealtime() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }

        this.isRealtime = false;
        this.updateRealtimeButton();
    }

    updateRealtimeButton() {
        const btn = document.getElementById('realtime-toggle-btn');
        
        if (this.isRealtime) {
            btn.innerHTML = '<span class="realtime-indicator"></span><i class="fas fa-stop"></i> Stop';
            btn.classList.remove('btn-primary');
            btn.classList.add('btn-danger');
        } else {
            btn.innerHTML = '<i class="fas fa-play"></i> Real-time';
            btn.classList.remove('btn-danger');
            btn.classList.add('btn-primary');
        }
    }

    addLogEntry(entry) {
        const container = document.getElementById('logs-container');
        const logEntry = this.createLogEntry(entry);
        
        container.appendChild(logEntry);

        // Apply filter
        const level = entry.level.toLowerCase();
        if (this.currentLevel !== '' && level !== this.currentLevel) {
            logEntry.classList.add('hidden');
        }

        // Auto-scroll to bottom
        container.scrollTop = container.scrollHeight;

        // Keep only last 1000 entries
        const entries = container.querySelectorAll('.log-entry');
        if (entries.length > 1000) {
            entries[0].remove();
        }
    }

    downloadCurrentLog() {
        if (!this.currentFile) {
            return;
        }

        const url = `${api.baseURL}/api/admin/logs/${this.currentFile}/download`;
        window.open(url, '_blank');
    }

    showInfo(message) {
        const container = document.getElementById('logs-container');
        container.innerHTML = `<div class="logs-info"><p>${message}</p></div>`;
    }

    showError(message) {
        const container = document.getElementById('logs-container');
        container.innerHTML = `<div class="logs-info" style="color: #dc3545;"><p>❌ ${message}</p></div>`;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    destroy() {
        this.stopRealtime();
    }
}

// Global instance
let logsViewer = null;

// Initialize when Logs subtab is opened (System & Logs > Logs)
document.addEventListener('DOMContentLoaded', () => {
    const logsSubtab = document.querySelector('[data-subtab="system-logs"]');
    
    if (logsSubtab) {
        logsSubtab.addEventListener('click', () => {
            if (!logsViewer) {
                logsViewer = new LogsViewer();
                logsViewer.init();
            }
        });
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (logsViewer) {
        logsViewer.destroy();
    }
});



