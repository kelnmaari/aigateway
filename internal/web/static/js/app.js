// ===== GLOBAL STATE =====
const state = {
    stats: null,
    config: null,
    models: null,
    apiKeys: null,
    metrics: null,
    currentView: 'dashboard',
    serverStatus: 'checking',
    refreshInterval: null,
    adminKey: null,  // Admin API key for auth
    isAuthenticated: false
};

// ===== INITIALIZATION =====
document.addEventListener('DOMContentLoaded', () => {
    console.log('🌐 WebUI Initialized');
    
    // Check for saved admin key
    checkAuth();
    
    // Setup navigation
    setupNavigation();
    
    // Initial data load
    refreshData();
    
    // Auto-refresh every 5 seconds
    state.refreshInterval = setInterval(refreshData, 5000);
    
    // Load initial view
    switchView('dashboard');
});

// ===== NAVIGATION =====
function setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const view = item.dataset.view;
            switchView(view);
        });
    });
}

function switchView(viewName) {
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.view === viewName);
    });
    
    // Update views
    document.querySelectorAll('.view').forEach(view => {
        view.classList.toggle('active', view.id === `view-${viewName}`);
    });
    
    // Update title
    const titles = {
        dashboard: 'Dashboard',
        analytics: 'Analytics',
        apikeys: 'API Keys Management',
        models: 'Models',
        logs: 'Logs',
        config: 'Configuration'
    };
    document.getElementById('page-title').textContent = titles[viewName] || viewName;
    
    state.currentView = viewName;
    
    // Load view-specific data
    loadViewData(viewName);
}

function loadViewData(viewName) {
    switch(viewName) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'apikeys':
            loadApiKeys();
            break;
        case 'models':
            loadModels();
            break;
        case 'logs':
            loadLogs();
            break;
        case 'config':
            loadConfig();
            break;
    }
}

// ===== DATA FETCHING =====
async function refreshData() {
    try {
        // Fetch stats
        const statsResponse = await fetch('/api/stats');
        if (statsResponse.ok) {
            state.stats = await statsResponse.json();
            updateServerStatus('online');
        } else {
            updateServerStatus('offline');
            return;
        }
        
        // Update last update time
        const now = new Date();
        document.getElementById('last-update-time').textContent = 
            now.toLocaleTimeString();
        
        // Update current view
        if (state.currentView === 'dashboard') {
            updateDashboard();
        }
        
    } catch (error) {
        console.error('Failed to fetch data:', error);
        updateServerStatus('offline');
        Toast.error('Failed to connect to server', 'Connection Error');
    }
}

function updateServerStatus(status) {
    state.serverStatus = status;
    const statusDot = document.getElementById('server-status');
    const statusText = document.getElementById('server-status-text');
    
    if (status === 'online') {
        statusDot.className = 'status-dot online';
        statusText.textContent = 'Connected';
    } else {
        statusDot.className = 'status-dot offline';
        statusText.textContent = 'Disconnected';
    }
}

// ===== DASHBOARD =====
function loadDashboard() {
    if (state.stats) {
        updateDashboard();
    }
}

function updateDashboard() {
    if (!state.stats) return;
    
    const stats = state.stats.stats || state.stats;
    
    // Update stat cards
    document.getElementById('stat-total-requests').textContent = 
        formatNumber(stats.total_requests || 0);
    
    const successRate = stats.total_requests > 0 
        ? ((stats.success_requests / stats.total_requests) * 100).toFixed(1) 
        : 0;
    document.getElementById('stat-success-rate').textContent = successRate + '%';
    
    document.getElementById('stat-avg-latency').textContent = 
        formatDuration(stats.average_latency || 0);
    
    document.getElementById('stat-active-keys').textContent = 
        formatNumber(stats.active_keys || 0);
    
    // Update server info
    document.getElementById('info-status').innerHTML = 
        '<span class="badge badge-success">Running</span>';
    document.getElementById('info-uptime').textContent = 
        formatUptime(stats.uptime || '0s');
    document.getElementById('info-address').textContent = 
        state.stats.server?.address || 'N/A';
    document.getElementById('info-version').textContent = 
        state.stats.server?.version || 'N/A';
    
    // Update Ollama info
    const ollama = state.stats.ollama || {};
    document.getElementById('ollama-status').innerHTML = 
        ollama.connected 
            ? '<span class="badge badge-success">Connected</span>'
            : '<span class="badge badge-error">Disconnected</span>';
    document.getElementById('ollama-url').textContent = ollama.url || 'N/A';
    document.getElementById('ollama-models').textContent = ollama.models_count || 0;
    document.getElementById('ollama-requests').textContent = 
        formatNumber(stats.ollama_requests || 0);
    
    // Update request stats table
    updateRequestStatsTable(stats);
}

function updateRequestStatsTable(stats) {
    const tbody = document.getElementById('request-stats-table');
    const total = stats.total_requests || 0;
    
    if (total === 0) {
        tbody.innerHTML = '<tr><td colspan="3" class="text-center text-muted">No requests yet</td></tr>';
        return;
    }
    
    const rows = [
        {
            metric: 'Success',
            count: stats.success_requests || 0,
            color: 'text-success'
        },
        {
            metric: 'Errors',
            count: stats.error_requests || 0,
            color: 'text-error'
        },
        {
            metric: 'Total',
            count: total,
            color: ''
        }
    ];
    
    tbody.innerHTML = rows.map(row => {
        const percentage = ((row.count / total) * 100).toFixed(1);
        return `
            <tr>
                <td class="${row.color}">${row.metric}</td>
                <td>${formatNumber(row.count)}</td>
                <td>${percentage}%</td>
            </tr>
        `;
    }).join('');
}

// ===== API KEYS =====
async function loadApiKeys() {
    const tbody = document.getElementById('apikeys-table');
    tbody.innerHTML = '<tr><td colspan="7" class="text-center">Loading...</td></tr>';
    
    // Check auth first
    if (!requireAuth()) {
        tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">Please login to view API keys</td></tr>';
        return;
    }
    
    try {
        const response = await fetch('/api/admin/keys', {
            headers: {
                'Authorization': 'Bearer ' + state.adminKey
            }
        });
        
        if (!response.ok) {
            if (response.status === 401 || response.status === 403) {
                // Key is invalid, logout
                handleLogout();
                throw new Error('Invalid admin API key');
            }
            throw new Error('Failed to load API keys');
        }
        
        const data = await response.json();
        state.apiKeys = data.api_keys || [];
        
        if (state.apiKeys.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">No API keys found</td></tr>';
            return;
        }
        
        tbody.innerHTML = state.apiKeys.map(key => `
            <tr>
                <td><strong>${key.name}</strong></td>
                <td><code>${key.id}</code></td>
                <td>
                    ${key.status === 'active' 
                        ? '<span class="badge badge-success">Active</span>'
                        : key.status === 'revoked'
                        ? '<span class="badge badge-error">Revoked</span>'
                        : '<span class="badge badge-secondary">Disabled</span>'}
                </td>
                <td>${key.rate_limits?.rate_limit_per_minute || 'Unlimited'}/min</td>
                <td>${(key.models || ['*']).join(', ')}</td>
                <td>${new Date(key.created_at).toLocaleDateString()}</td>
                <td>
                    <button class="btn btn-danger btn-small" onclick="deleteApiKey('${key.id}')">
                        Delete
                    </button>
                </td>
            </tr>
        `).join('');
        
    } catch (error) {
        console.error('Failed to load API keys:', error);
        tbody.innerHTML = `<tr><td colspan="7" class="text-center text-error">Error: ${error.message}</td></tr>`;
    }
}

// ===== MODELS =====
async function loadModels() {
    const container = document.getElementById('models-grid');
    container.innerHTML = '<div class="text-center">Loading...</div>';
    
    try {
        const response = await fetch('/api/models');
        if (!response.ok) {
            throw new Error('Failed to load models');
        }
        
        const data = await response.json();
        state.models = data.data || [];
        
        if (state.models.length === 0) {
            container.innerHTML = '<div class="text-center text-muted">No models available</div>';
            return;
        }
        
        container.innerHTML = state.models.map(model => {
            const sizeGB = model.size ? (model.size / (1024 * 1024 * 1024)).toFixed(2) : 'N/A';
            const modifiedDate = model.modified_at 
                ? new Date(model.modified_at * 1000).toLocaleDateString()
                : new Date(model.created * 1000).toLocaleDateString();
            
            return `
                <div class="model-card-extended">
                    <div class="model-header">
                <h4>${model.id}</h4>
                        <span class="model-size">${sizeGB} GB</span>
            </div>
                    <div class="model-details">
                        ${model.parameter_size ? `
                            <div class="model-detail">
                                <span class="detail-label">Parameters:</span>
                                <span class="detail-value">${model.parameter_size}</span>
                            </div>
                        ` : ''}
                        ${model.family ? `
                            <div class="model-detail">
                                <span class="detail-label">Family:</span>
                                <span class="detail-value">${model.family}</span>
                            </div>
                        ` : ''}
                        ${model.quantization_level ? `
                            <div class="model-detail">
                                <span class="detail-label">Quantization:</span>
                                <span class="detail-value">${model.quantization_level}</span>
                            </div>
                        ` : ''}
                        ${model.format ? `
                            <div class="model-detail">
                                <span class="detail-label">Format:</span>
                                <span class="detail-value">${model.format.toUpperCase()}</span>
                            </div>
                        ` : ''}
                        <div class="model-detail">
                            <span class="detail-label">Modified:</span>
                            <span class="detail-value">${modifiedDate}</span>
                        </div>
                    </div>
                    ${model.digest ? `
                        <div class="model-digest" title="${model.digest}">
                            Digest: ${model.digest.substring(0, 16)}...
                        </div>
                    ` : ''}
                </div>
            `;
        }).join('');
        
    } catch (error) {
        console.error('Failed to load models:', error);
        container.innerHTML = `<div class="text-center text-error">Error: ${error.message}</div>`;
    }
}

// ===== LOGS (WEBUI-02 Enhanced) =====
let allLogs = []; // Store all fetched logs for client-side filtering
let currentLogLevel = 'all';
let currentSearchText = '';
let logsAutoRefreshInterval = null;

// Load logs from server
async function loadLogs(filterLevel = null) {
    const container = document.getElementById('logs-container');
    
    if (filterLevel) {
        currentLogLevel = filterLevel;
    }
    
    try {
        // Fetch all logs (server-side doesn't filter by level anymore)
        const response = await fetch('/api/logs?limit=500');
        if (!response.ok) {
            throw new Error('Failed to load logs');
        }
        
        const data = await response.json();
        allLogs = data.logs || [];
        
        // Apply client-side filtering and render
        renderFilteredLogs();
        
    } catch (error) {
        console.error('Failed to load logs:', error);
        container.innerHTML = `<div class="text-center text-error">Error: ${error.message}</div>`;
    }
}

// Render logs with current filters applied (WEBUI-02)
function renderFilteredLogs() {
    const container = document.getElementById('logs-container');
    
    // Apply level filter
    let filtered = allLogs;
    if (currentLogLevel !== 'all') {
        filtered = filtered.filter(log => log.level === currentLogLevel);
    }
    
    // Apply search filter (WEBUI-02)
    if (currentSearchText.trim()) {
        const searchLower = currentSearchText.toLowerCase();
        filtered = filtered.filter(log => 
            log.line.toLowerCase().includes(searchLower)
        );
    }
    
    if (filtered.length === 0) {
        container.innerHTML = '<div class="text-center text-muted">No logs found</div>';
        return;
    }
    
    // Render logs
    container.innerHTML = filtered.map(log => {
        const levelClass = `log-entry ${log.level}`;
        let line = escapeHtml(log.line);
        
        // Highlight search text (WEBUI-02)
        if (currentSearchText.trim()) {
            const regex = new RegExp(`(${escapeRegex(currentSearchText)})`, 'gi');
            line = line.replace(regex, '<mark>$1</mark>');
        }
        
        return `<div class="${levelClass}">${line}</div>`;
    }).join('');
    
    // Auto-scroll to bottom
    container.scrollTop = container.scrollHeight;
}

// Filter logs by level (client-side)
function filterLogs(level) {
    currentLogLevel = level;
    
    // Update button states
    document.querySelectorAll('.log-filter-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-level="${level}"]`)?.classList.add('active');
    
    renderFilteredLogs();
}

// Search logs by text (WEBUI-02)
function searchLogs() {
    const input = document.getElementById('log-search-input');
    currentSearchText = input.value;
    renderFilteredLogs();
}

// Toggle auto-refresh for logs (WEBUI-02)
function toggleLogsAutoRefresh() {
    const checkbox = document.getElementById('logs-auto-refresh');
    
    if (checkbox.checked) {
        // Start auto-refresh every 5 seconds
        logsAutoRefreshInterval = setInterval(() => {
            if (state.currentView === 'logs') {
                loadLogs();
            }
        }, 5000);
        Toast.success('Logs auto-refresh enabled');
    } else {
        // Stop auto-refresh
        if (logsAutoRefreshInterval) {
            clearInterval(logsAutoRefreshInterval);
            logsAutoRefreshInterval = null;
        }
        Toast.info('Logs auto-refresh disabled');
    }
}

// Export logs to TXT (WEBUI-02)
function exportLogsToTXT() {
    if (allLogs.length === 0) {
        Toast.error('No logs available to export');
        return;
    }
    
    // Apply current filters
    let filtered = allLogs;
    if (currentLogLevel !== 'all') {
        filtered = filtered.filter(log => log.level === currentLogLevel);
    }
    if (currentSearchText.trim()) {
        const searchLower = currentSearchText.toLowerCase();
        filtered = filtered.filter(log => log.line.toLowerCase().includes(searchLower));
    }
    
    if (filtered.length === 0) {
        Toast.error('No logs match current filters');
        return;
    }
    
    // Generate TXT content
    const txtContent = filtered.map(log => log.line).join('\n');
    
    // Download
    const blob = new Blob([txtContent], { type: 'text/plain' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs_${new Date().toISOString().replace(/[:.]/g, '-')}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    Toast.success(`Exported ${filtered.length} log entries to TXT`);
}

// Export logs to JSON (WEBUI-02)
function exportLogsToJSON() {
    if (allLogs.length === 0) {
        Toast.error('No logs available to export');
        return;
    }
    
    // Apply current filters
    let filtered = allLogs;
    if (currentLogLevel !== 'all') {
        filtered = filtered.filter(log => log.level === currentLogLevel);
    }
    if (currentSearchText.trim()) {
        const searchLower = currentSearchText.toLowerCase();
        filtered = filtered.filter(log => log.line.toLowerCase().includes(searchLower));
    }
    
    if (filtered.length === 0) {
        Toast.error('No logs match current filters');
        return;
    }
    
    // Generate JSON content
    const jsonData = {
        exported_at: new Date().toISOString(),
        filters: {
            level: currentLogLevel,
            search: currentSearchText
        },
        total_logs: filtered.length,
        logs: filtered
    };
    
    const jsonContent = JSON.stringify(jsonData, null, 2);
    
    // Download
    const blob = new Blob([jsonContent], { type: 'application/json' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs_${new Date().toISOString().replace(/[:.]/g, '-')}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    Toast.success(`Exported ${filtered.length} log entries to JSON`);
}

// Escape regex special characters for search highlighting
function escapeRegex(text) {
    return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ===== EXPORT =====
function exportStatsToCSV() {
    if (!state.stats) {
        Toast.error('No data available to export');
        return;
    }
    
    const csvRows = [];
    
    // Header
    csvRows.push('Metric,Value');
    
    // Server stats
    csvRows.push(`Server Uptime,${formatUptime(state.stats.stats?.uptime || 0)}`);
    csvRows.push(`Total Requests,${state.stats.stats?.total_requests || 0}`);
    csvRows.push(`Success Requests,${state.stats.stats?.success_requests || 0}`);
    csvRows.push(`Error Requests,${state.stats.stats?.error_requests || 0}`);
    
    // Ollama stats
    csvRows.push(`Ollama Connected,${state.stats.ollama?.connected ? 'Yes' : 'No'}`);
    csvRows.push(`Ollama Models Count,${state.stats.ollama?.models_count || 0}`);
    
    // API Keys stats
    csvRows.push(`Total API Keys,${state.stats.api_keys?.total || 0}`);
    csvRows.push(`Active API Keys,${state.stats.api_keys?.active || 0}`);
    csvRows.push(`Disabled API Keys,${state.stats.api_keys?.disabled || 0}`);
    
    const csv = csvRows.join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `ollama-proxy-stats-${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    Toast.success('Stats exported to CSV');
}

function exportStatsToJSON() {
    if (!state.stats) {
        Toast.error('No data available to export');
        return;
    }
    
    const json = JSON.stringify(state.stats, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `ollama-proxy-stats-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    Toast.success('Stats exported to JSON');
}

function exportAPIKeysToCSV() {
    if (!state.apiKeys || state.apiKeys.length === 0) {
        Toast.error('No API keys to export');
        return;
    }
    
    const csvRows = [];
    
    // Header
    csvRows.push('Key ID,Name,Status,Rate Limit,Allowed Models,Created,Last Used');
    
    // Data
    state.apiKeys.forEach(key => {
        const rateLimitPerMinute = key.rate_limits?.rate_limit_per_minute || 'N/A';
        const allowedModels = key.allowed_models?.join(';') || '*';
        const created = new Date(key.created_at).toLocaleString();
        const lastUsed = key.last_used_at ? new Date(key.last_used_at).toLocaleString() : 'Never';
        
        csvRows.push(`${key.key_id},${key.name},${key.status},${rateLimitPerMinute},${allowedModels},${created},${lastUsed}`);
    });
    
    const csv = csvRows.join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `ollama-proxy-api-keys-${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    
    Toast.success('API Keys exported to CSV');
}

// ===== CONFIG =====
async function loadConfig() {
    const container = document.getElementById('config-container');
    container.innerHTML = '<div class="text-center">Loading...</div>';
    
    try {
        const response = await fetch('/api/config');
        if (!response.ok) {
            throw new Error('Failed to load config');
        }
        
        state.config = await response.json();
        
        container.innerHTML = `
            <pre style="background: var(--bg-main); padding: 20px; border-radius: 8px; overflow-x: auto;">
${JSON.stringify(state.config, null, 2)}
            </pre>
        `;
        
    } catch (error) {
        console.error('Failed to load config:', error);
        container.innerHTML = `<div class="text-center text-error">Error: ${error.message}</div>`;
    }
}

// ===== MODALS =====
function showCreateKeyModal() {
    document.getElementById('create-key-modal').classList.add('active');
}

function closeCreateKeyModal() {
    document.getElementById('create-key-modal').classList.remove('active');
    document.getElementById('create-key-form').reset();
}

function closeShowKeyModal() {
    document.getElementById('show-key-modal').classList.remove('active');
}

async function createApiKey(event) {
    event.preventDefault();
    
    // Check auth first
    if (!requireAuth()) {
        return;
    }
    
    const formData = {
        name: document.getElementById('key-name').value,
        rate_limit_per_minute: parseInt(document.getElementById('key-rate-limit').value),
        models: document.getElementById('key-models').value.split(',').map(m => m.trim()),
    };
    
    try {
        const response = await fetch('/api/admin/keys', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + state.adminKey
            },
            body: JSON.stringify(formData)
        });
        
        if (!response.ok) {
            if (response.status === 401 || response.status === 403) {
                handleLogout();
                throw new Error('Invalid admin API key');
            }
            const error = await response.json();
            throw new Error(error.error || 'Failed to create API key');
        }
        
        const data = await response.json();
        
        // Show the created key
        document.getElementById('created-key-value').textContent = data.plain_key;
        closeCreateKeyModal();
        document.getElementById('show-key-modal').classList.add('active');
        
        // Show success toast
        Toast.success('API key created successfully!', 'Key Created');
        
        // Reload keys list
        loadApiKeys();
        
    } catch (error) {
        Toast.error(error.message, 'Failed to Create Key');
    }
}

async function deleteApiKey(keyId) {
    if (!confirm('Are you sure you want to delete this API key?')) {
        return;
    }
    
    // Check auth first
    if (!requireAuth()) {
        return;
    }
    
    try {
        const response = await fetch(`/api/admin/keys/${keyId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': 'Bearer ' + state.adminKey
            }
        });
        
        if (!response.ok) {
            if (response.status === 401 || response.status === 403) {
                handleLogout();
                throw new Error('Invalid admin API key');
            }
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete API key');
        }
        
        Toast.success('API key deleted successfully', 'Key Deleted');
        
        // Reload keys list
        loadApiKeys();
        
    } catch (error) {
        Toast.error(error.message, 'Failed to Delete Key');
    }
}

function copyKey() {
    const keyValue = document.getElementById('created-key-value').textContent;
    navigator.clipboard.writeText(keyValue).then(() => {
        Toast.success('API key copied to clipboard!', 'Copied', 3000);
    }).catch(err => {
        console.error('Failed to copy:', err);
        Toast.error('Failed to copy to clipboard', 'Copy Failed');
    });
}

// ===== UTILITIES =====
function formatNumber(num) {
    return new Intl.NumberFormat().format(num);
}

function formatDuration(ms) {
    if (ms < 1000) {
        return ms.toFixed(0) + ' ms';
    }
    return (ms / 1000).toFixed(2) + ' s';
}

function formatUptime(duration) {
    // Parse duration string like "1h23m45s" or "5m30s"
    const matches = duration.match(/(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?/);
    if (!matches) return duration;
    
    const hours = parseInt(matches[1]) || 0;
    const minutes = parseInt(matches[2]) || 0;
    const seconds = parseInt(matches[3]) || 0;
    
    const parts = [];
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`);
    
    return parts.join(' ');
}

// Format date string to readable format
function formatDate(dateString) {
    if (!dateString) return 'N/A';
    
    try {
        const date = new Date(dateString);
        
        // Check if date is valid
        if (isNaN(date.getTime())) return dateString;
        
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);
        
        // Relative time for recent dates
        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        if (diffDays < 7) return `${diffDays}d ago`;
        
        // Otherwise format as date
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        
        return `${year}-${month}-${day} ${hours}:${minutes}`;
    } catch (error) {
        console.error('Error formatting date:', error);
        return dateString;
    }
}

// ===== AUTHENTICATION =====
function checkAuth() {
    // Check sessionStorage for saved admin key
    const savedKey = sessionStorage.getItem('adminKey');
    
    if (savedKey) {
        state.adminKey = savedKey;
        state.isAuthenticated = true;
        showLogoutButton();
    }
}

// Helper function for authenticated fetch requests (AUTH-04)
async function fetchWithAuth(url, options = {}) {
    // Check if authenticated
    if (!state.isAuthenticated || !state.adminKey) {
        showLoginModal();
        throw new Error('Authentication required');
    }
    
    // Add Authorization header
    const headers = {
        ...options.headers,
        'Authorization': 'Bearer ' + state.adminKey
    };
    
    return fetch(url, {
        ...options,
        headers
    });
}

function showLoginModal() {
    document.getElementById('login-modal').classList.add('active');
    // Focus on input
    setTimeout(() => {
        document.getElementById('login-admin-key').focus();
    }, 100);
}

function closeLoginModal() {
    document.getElementById('login-modal').classList.remove('active');
    document.getElementById('login-form').reset();
}

async function handleLogin(event) {
    event.preventDefault();
    
    const adminKey = document.getElementById('login-admin-key').value.trim();
    
    if (!adminKey) {
        Toast.error('Please enter an admin key', 'Login Failed');
        return;
    }
    
    // Test the key by trying to list API keys
    try {
        const response = await fetch('/api/admin/keys?limit=1', {
            headers: {
                'Authorization': 'Bearer ' + adminKey
            }
        });
        
        if (response.ok) {
            // Key is valid
            state.adminKey = adminKey;
            state.isAuthenticated = true;
            sessionStorage.setItem('adminKey', adminKey);
            
            closeLoginModal();
            showLogoutButton();
            Toast.success('Successfully authenticated', 'Login Successful', 3000);
            
            // Reload current view if it's API keys
            if (state.currentView === 'apikeys') {
                loadApiKeys();
            }
        } else {
            throw new Error('Invalid admin key');
        }
    } catch (error) {
        Toast.error('Invalid admin API key', 'Authentication Failed');
    }
}

function handleLogout() {
    state.adminKey = null;
    state.isAuthenticated = false;
    sessionStorage.removeItem('adminKey');
    hideLogoutButton();
    Toast.info('Logged out successfully', 'Logout', 3000);
    
    // If on API keys view, clear it
    if (state.currentView === 'apikeys') {
        switchView('dashboard');
    }
}

function showLogoutButton() {
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.style.display = 'inline-flex';
    }
}

function hideLogoutButton() {
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
        logoutBtn.style.display = 'none';
    }
}

function requireAuth() {
    if (!state.isAuthenticated || !state.adminKey) {
        showLoginModal();
        return false;
    }
    return true;
}

// ===== TOAST NOTIFICATIONS =====
const Toast = {
    container: null,
    
    init() {
        this.container = document.getElementById('toast-container');
    },
    
    show(message, type = 'info', title = null, duration = 5000) {
        if (!this.container) this.init();
        
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        
        const icons = {
            success: '✅',
            error: '❌',
            warning: '⚠️',
            info: 'ℹ️'
        };
        
        const titles = {
            success: title || 'Success',
            error: title || 'Error',
            warning: title || 'Warning',
            info: title || 'Info'
        };
        
        toast.innerHTML = `
            <div class="toast-icon">${icons[type]}</div>
            <div class="toast-content">
                <div class="toast-title">${titles[type]}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="Toast.dismiss(this.parentElement)">×</button>
        `;
        
        this.container.appendChild(toast);
        
        // Auto dismiss
        if (duration > 0) {
            setTimeout(() => this.dismiss(toast), duration);
        }
        
        return toast;
    },
    
    success(message, title = null, duration = 5000) {
        return this.show(message, 'success', title, duration);
    },
    
    error(message, title = null, duration = 7000) {
        return this.show(message, 'error', title, duration);
    },
    
    warning(message, title = null, duration = 6000) {
        return this.show(message, 'warning', title, duration);
    },
    
    info(message, title = null, duration = 5000) {
        return this.show(message, 'info', title, duration);
    },
    
    dismiss(toast) {
        if (!toast) return;
        toast.classList.add('removing');
        setTimeout(() => {
            if (toast.parentElement) {
                toast.parentElement.removeChild(toast);
            }
        }, 300);
    }
};

// ===== THEME TOGGLE =====
// Change theme
function changeTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
    
    const themeNames = {
        'dark': 'Dark',
        'light': 'Light',
        'colorful': 'Colorful',
        'nord': 'Nord',
        'monokai': 'Monokai',
        'dracula': 'Dracula'
    };
    
    Toast.success(`${themeNames[theme]} theme activated`, 'Theme Changed', 2000);
}

// Initialize theme from localStorage
function initTheme() {
    const savedTheme = localStorage.getItem('theme') || 'dark';
    const themeSelect = document.getElementById('theme-select');
    
    // Apply saved theme
    document.documentElement.setAttribute('data-theme', savedTheme);
    
    // Update select element
    if (themeSelect) {
        themeSelect.value = savedTheme;
    }
}

// Call initTheme on page load
document.addEventListener('DOMContentLoaded', initTheme);

// ===== ANALYTICS CHARTS =====
// Global chart instances
const charts = {
    latency: null,
    requests: null,
    status: null,
    active: null
};

// Chart configuration
const chartConfig = {
    currentTimeRange: '1h',
    autoRefresh: true,
    refreshTimer: null
};

// Initialize charts when analytics view is opened
function initChartsIfNeeded() {
    if (!charts.latency) {
        initCharts();
        loadHistoricalMetrics();
    }
}

// Initialize all charts
function initCharts() {
    console.log('📊 Initializing charts...');
    
    // Get theme colors
    const isDark = document.documentElement.getAttribute('data-theme') !== 'light';
    const textColor = isDark ? '#e5e7eb' : '#1f2937';
    const gridColor = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
    
    // Common chart options
    const commonOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: true,
                labels: {
                    color: textColor,
                    font: { size: 12 }
                }
            },
            tooltip: {
                mode: 'index',
                intersect: false
            }
        },
        scales: {
            x: {
                ticks: { color: textColor },
                grid: { color: gridColor }
            },
            y: {
                ticks: { color: textColor },
                grid: { color: gridColor }
            }
        }
    };
    
    // 1. Latency Line Chart
    const latencyCtx = document.getElementById('latency-chart');
    if (latencyCtx) {
        charts.latency = new Chart(latencyCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: 'P50 (median)',
                        data: [],
                        borderColor: 'rgb(34, 197, 94)',
                        backgroundColor: 'rgba(34, 197, 94, 0.1)',
                        tension: 0.4,
                        fill: true
                    },
                    {
                        label: 'P95',
                        data: [],
                        borderColor: 'rgb(249, 115, 22)',
                        backgroundColor: 'rgba(249, 115, 22, 0.1)',
                        tension: 0.4,
                        fill: true
                    },
                    {
                        label: 'P99',
                        data: [],
                        borderColor: 'rgb(239, 68, 68)',
                        backgroundColor: 'rgba(239, 68, 68, 0.1)',
                        tension: 0.4,
                        fill: true
                    }
                ]
            },
            options: {
                ...commonOptions,
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        title: {
                            display: true,
                            text: 'Latency (ms)',
                            color: textColor
                        },
                        beginAtZero: true
                    }
                }
            }
        });
    }
    
    // 2. Requests Bar Chart
    const requestsCtx = document.getElementById('requests-chart');
    if (requestsCtx) {
        charts.requests = new Chart(requestsCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Requests',
                    data: [],
                    backgroundColor: 'rgba(99, 102, 241, 0.7)',
                    borderColor: 'rgb(99, 102, 241)',
                    borderWidth: 1
                }]
            },
            options: {
                ...commonOptions,
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        title: {
                            display: true,
                            text: 'Requests',
                            color: textColor
                        },
                        beginAtZero: true
                    }
                }
            }
        });
    }
    
    // 3. Status Distribution Pie Chart
    const statusCtx = document.getElementById('status-chart');
    if (statusCtx) {
        charts.status = new Chart(statusCtx, {
            type: 'doughnut',
            data: {
                labels: ['2xx Success', '4xx Client Error', '5xx Server Error'],
                datasets: [{
                    data: [0, 0, 0],
                    backgroundColor: [
                        'rgba(34, 197, 94, 0.7)',
                        'rgba(251, 191, 36, 0.7)',
                        'rgba(239, 68, 68, 0.7)'
                    ],
                    borderColor: [
                        'rgb(34, 197, 94)',
                        'rgb(251, 191, 36)',
                        'rgb(239, 68, 68)'
                    ],
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: true,
                        position: 'bottom',
                        labels: {
                            color: textColor,
                            font: { size: 12 }
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed || 0;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                return `${label}: ${value} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }
    
    // 4. Active Requests Area Chart
    const activeCtx = document.getElementById('active-chart');
    if (activeCtx) {
        charts.active = new Chart(activeCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Active Requests',
                    data: [],
                    borderColor: 'rgb(168, 85, 247)',
                    backgroundColor: 'rgba(168, 85, 247, 0.2)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                ...commonOptions,
                scales: {
                    ...commonOptions.scales,
                    y: {
                        ...commonOptions.scales.y,
                        title: {
                            display: true,
                            text: 'Concurrent Requests',
                            color: textColor
                        },
                        beginAtZero: true,
                        ticks: {
                            ...commonOptions.scales.y.ticks,
                            stepSize: 1
                        }
                    }
                }
            }
        });
    }
    
    console.log('✅ Charts initialized');
}

// Load historical metrics data
async function loadHistoricalMetrics() {
    try {
        const period = chartConfig.currentTimeRange;
        console.log(`📊 Loading metrics for period: ${period}`);
        
        // Fetch latency data
        const latencyResponse = await fetch(`/api/metrics/history?type=request_latency&period=${period}`);
        if (latencyResponse.ok) {
            const latencyData = await latencyResponse.json();
            updateLatencyChart(latencyData);
        }
        
        // Fetch request count data
        const countResponse = await fetch(`/api/metrics/history?type=request_count&period=${period}`);
        if (countResponse.ok) {
            const countData = await countResponse.json();
            updateRequestsChart(countData);
        }
        
        // Update status distribution from current stats
        if (state.stats) {
            updateStatusChart();
            updateActiveChartFromStats();
        }
        
        // Update summary stats
        updateSummaryStats();
        
        console.log('✅ Metrics loaded');
    } catch (error) {
        console.error('❌ Error loading metrics:', error);
        Toast.error('Failed to load metrics data');
    }
}

// Update latency chart
function updateLatencyChart(data) {
    if (!charts.latency || !data || !data.time_series) return;
    
    const labels = data.time_series.map(p => formatChartTime(p.timestamp));
    const p50Data = data.time_series.map(p => p.stats?.median || 0);
    const p95Data = data.time_series.map(p => p.stats?.p95 || 0);
    const p99Data = data.time_series.map(p => p.stats?.p99 || 0);
    
    charts.latency.data.labels = labels;
    charts.latency.data.datasets[0].data = p50Data;
    charts.latency.data.datasets[1].data = p95Data;
    charts.latency.data.datasets[2].data = p99Data;
    charts.latency.update('none');
}

// Update requests chart
function updateRequestsChart(data) {
    if (!charts.requests || !data || !data.time_series) return;
    
    const labels = data.time_series.map(p => formatChartTime(p.timestamp));
    const values = data.time_series.map(p => p.count || 0);
    
    charts.requests.data.labels = labels;
    charts.requests.data.datasets[0].data = values;
    charts.requests.update('none');
}

// Update status distribution chart
function updateStatusChart() {
    if (!charts.status || !state.stats) return;
    
    const stats = state.stats.stats || {};
    const successCount = stats.success_requests || 0;
    const errorCount = stats.error_requests || 0;
    
    // Assuming 4xx errors are about 30% of errors, 5xx are 70%
    const clientErrors = Math.floor(errorCount * 0.3);
    const serverErrors = errorCount - clientErrors;
    
    charts.status.data.datasets[0].data = [
        successCount,
        clientErrors,
        serverErrors
    ];
    charts.status.update('none');
}

// Update active requests chart from stats (real-time)
function updateActiveChartFromStats() {
    if (!charts.active || !state.stats) return;
    
    const stats = state.stats.stats || {};
    const activeCount = stats.active_requests || 0;
    const now = new Date();
    
    // Keep last 60 data points (10 seconds * 6 = 1 minute history)
    const maxPoints = 60;
    
    // Get current data
    let labels = charts.active.data.labels || [];
    let values = charts.active.data.datasets[0].data || [];
    
    // Add new point
    labels.push(now.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' }));
    values.push(activeCount);
    
    // Remove old points if exceeds max
    if (labels.length > maxPoints) {
        labels = labels.slice(-maxPoints);
        values = values.slice(-maxPoints);
    }
    
    // Update chart
    charts.active.data.labels = labels;
    charts.active.data.datasets[0].data = values;
    charts.active.update('none');
}

// Update summary statistics
function updateSummaryStats() {
    if (!state.stats) return;
    
    const requests = state.stats.requests || {};
    const latency = state.stats.latency || {};
    
    // Update summary values
    document.getElementById('summary-avg-latency').textContent = 
        latency.avg ? `${latency.avg}ms` : '-';
    document.getElementById('summary-p95-latency').textContent = 
        latency.p95 ? `${latency.p95}ms` : '-';
    document.getElementById('summary-p99-latency').textContent = 
        latency.p99 ? `${latency.p99}ms` : '-';
    document.getElementById('summary-total-requests').textContent = 
        formatNumber(requests.total || 0);
    
    const total = requests.total || 0;
    const errors = requests.errors || 0;
    const success = requests.success || 0;
    
    const errorRate = total > 0 ? ((errors / total) * 100).toFixed(2) : '0.00';
    const successRate = total > 0 ? ((success / total) * 100).toFixed(2) : '0.00';
    
    document.getElementById('summary-error-rate').textContent = `${errorRate}%`;
    document.getElementById('summary-success-rate').textContent = `${successRate}%`;
}

// Format timestamp for chart labels
function formatChartTime(timestamp) {
    // Timestamp приходит в Unix секундах, умножаем на 1000 для JavaScript Date
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diff = (now - date) / 1000 / 60; // minutes
    
    if (diff < 60) {
        // Show time for last hour
        return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    } else if (diff < 1440) {
        // Show time for last day
        return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    } else {
        // Show date + time for older
        return date.toLocaleDateString('ru-RU', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    }
}

// Chart control functions
function changeTimeRange(range) {
    chartConfig.currentTimeRange = range;
    console.log(`📊 Time range changed to: ${range}`);
    loadHistoricalMetrics();
}

function toggleAutoRefreshCharts() {
    const checkbox = document.getElementById('auto-refresh-charts');
    chartConfig.autoRefresh = checkbox.checked;
    
    if (chartConfig.autoRefresh) {
        startChartAutoRefresh();
        Toast.success('Auto-refresh enabled');
    } else {
        stopChartAutoRefresh();
        Toast.info('Auto-refresh disabled');
    }
}

function refreshCharts() {
    console.log('🔄 Refreshing charts...');
    loadHistoricalMetrics();
    Toast.success('Charts refreshed', null, 1000);
}

function startChartAutoRefresh() {
    if (chartConfig.refreshTimer) return;
    
    chartConfig.refreshTimer = setInterval(() => {
        if (state.currentView === 'analytics' && chartConfig.autoRefresh) {
            loadHistoricalMetrics();
        }
    }, 10000); // Refresh every 10 seconds
}

function stopChartAutoRefresh() {
    if (chartConfig.refreshTimer) {
        clearInterval(chartConfig.refreshTimer);
        chartConfig.refreshTimer = null;
    }
}

// Update charts when view switches to analytics
const originalSwitchView = switchView;
switchView = function(viewName) {
    originalSwitchView(viewName);
    
    if (viewName === 'analytics') {
        // Initialize charts if needed
        setTimeout(() => {
            initChartsIfNeeded();
            startChartAutoRefresh();
        }, 100);
    } else {
        // Stop auto-refresh when leaving analytics
        if (chartConfig.autoRefresh) {
            stopChartAutoRefresh();
        }
    }
};

// Listen for WebSocket metrics updates
if (typeof wsClient !== 'undefined') {
    const originalOnMessage = wsClient.onMessage;
    wsClient.onMessage = function(event) {
        if (originalOnMessage) {
            originalOnMessage(event);
        }
        
        const data = JSON.parse(event.data);
        
        // Update charts on metrics update
        if (data.type === 'metrics_update' && state.currentView === 'analytics') {
            updateStatusChart();
            updateActiveChartFromStats();
            updateSummaryStats();
        }
    };
}

console.log('📊 Analytics module loaded');

// ========================================
// AUTH-04: Enhanced API Key Management
// ========================================

// Load API Keys on view switch
async function loadApiKeys() {
    try {
        const response = await fetchWithAuth('/api/admin/keys');
        
        if (!response.ok) {
            throw new Error(`Failed to load API keys: ${response.statusText}`);
        }

        const data = await response.json();
        renderAPIKeys(data.api_keys || []);
    } catch (error) {
        console.error('❌ Failed to load API keys:', error);
        Toast.error('Failed to load API keys: ' + error.message);
        document.getElementById('apikeys-table').innerHTML = 
            '<tr><td colspan="7" class="text-center">Failed to load keys</td></tr>';
    }
}

// Render API Keys table with status badges and action buttons
function renderAPIKeys(keys) {
    const tbody = document.getElementById('apikeys-table');
    
    if (!keys || keys.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="text-center">No API keys found</td></tr>';
        return;
    }

    tbody.innerHTML = keys.map(key => `
        <tr>
            <td><strong>${escapeHtml(key.name)}</strong></td>
            <td><code style="font-size: 12px;">${key.id}</code></td>
            <td>${getStatusBadge(key.status)}</td>
            <td>${key.rate_limits?.requests_per_minute || '-'}/min</td>
            <td><span title="${(key.models || ['*']).join(', ')}">${formatModels(key.models)}</span></td>
            <td>${formatDate(key.created_at)}</td>
            <td>
                <div style="display: flex; gap: 5px;">
                    <button class="btn-icon" onclick="editAPIKey('${key.id}')" title="Edit">
                        ✏️
                    </button>
                    ${getToggleButton(key)}
                    <button class="btn-icon" onclick="deleteAPIKey('${key.id}', '${escapeHtml(key.name)}')" title="Delete" style="color: var(--danger);">
                        🗑️
                    </button>
                </div>
            </td>
        </tr>
    `).join('');
}

// Get status badge HTML
function getStatusBadge(status) {
    const badges = {
        'active': '<span class="badge badge-success">Active</span>',
        'revoked': '<span class="badge badge-danger">Revoked</span>',
        'expired': '<span class="badge badge-warning">Expired</span>',
        'disabled': '<span class="badge badge-secondary">Disabled</span>'
    };
    return badges[status] || '<span class="badge badge-secondary">' + status + '</span>';
}

// Get toggle button (Revoke/Enable) based on status
function getToggleButton(key) {
    if (key.status === 'active') {
        return `<button class="btn-icon" onclick="revokeAPIKey('${key.id}', '${escapeHtml(key.name)}')" title="Revoke" style="color: var(--warning);">
            🚫
        </button>`;
    } else if (key.status === 'revoked') {
        return `<button class="btn-icon" onclick="enableAPIKey('${key.id}', '${escapeHtml(key.name)}')" title="Enable" style="color: var(--success);">
            ✅
        </button>`;
    }
    return '';
}

// Format models list
function formatModels(models) {
    if (!models || models.length === 0) return '*';
    if (models.includes('*')) return '*';
    if (models.length === 1) return models[0];
    return `${models[0]} +${models.length - 1}`;
}

// Edit API Key - open modal
async function editAPIKey(keyId) {
    try {
        const response = await fetchWithAuth(`/api/admin/keys/${keyId}`);
        
        if (!response.ok) {
            throw new Error('Failed to load key details');
        }

        const data = await response.json();
        const key = data.api_key;

        // Fill form
        document.getElementById('edit-key-id').value = key.id;
        document.getElementById('edit-key-name').value = key.name;
        document.getElementById('edit-key-description').value = key.description || '';
        document.getElementById('edit-key-models').value = (key.models || ['*']).join(', ');
        document.getElementById('edit-key-permissions').value = (key.permissions || ['chat', 'models']).join(', ');
        
        // Fill rate limits
        if (key.rate_limits) {
            document.getElementById('edit-key-rpm').value = key.rate_limits.requests_per_minute || '';
            document.getElementById('edit-key-rph').value = key.rate_limits.requests_per_hour || '';
        } else {
            document.getElementById('edit-key-rpm').value = '';
            document.getElementById('edit-key-rph').value = '';
        }

        // Show modal
        document.getElementById('edit-key-modal').classList.add('active');
    } catch (error) {
        console.error('❌ Failed to load key for editing:', error);
        Toast.error('Failed to load key: ' + error.message);
    }
}

// Update API Key - submit form
async function updateApiKey(event) {
    event.preventDefault();

    const keyId = document.getElementById('edit-key-id').value;
    const name = document.getElementById('edit-key-name').value;
    const description = document.getElementById('edit-key-description').value;
    const models = document.getElementById('edit-key-models').value
        .split(',')
        .map(m => m.trim())
        .filter(m => m);
    const permissions = document.getElementById('edit-key-permissions').value
        .split(',')
        .map(p => p.trim())
        .filter(p => p);

    // Get rate limits
    const rpm = document.getElementById('edit-key-rpm').value;
    const rph = document.getElementById('edit-key-rph').value;
    
    // Build request body
    const body = {
        name,
        description,
        models,
        permissions
    };
    
    // Add rate limits if provided
    if (rpm || rph) {
        body.rate_limits = {
            requests_per_minute: rpm ? parseInt(rpm, 10) : 30,
            requests_per_hour: rph ? parseInt(rph, 10) : 500
        };
    }

    try {
        const response = await fetchWithAuth(`/api/admin/keys/${keyId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to update key');
        }

        Toast.success('API Key updated successfully!');
        closeEditKeyModal();
        loadApiKeys(); // Reload list
    } catch (error) {
        console.error('❌ Failed to update key:', error);
        Toast.error('Failed to update key: ' + error.message);
    }
}

// Revoke API Key
async function revokeAPIKey(keyId, keyName) {
    if (!confirm(`Are you sure you want to REVOKE the key "${keyName}"?\n\nThis will make it inactive immediately.`)) {
        return;
    }

    try {
        const response = await fetchWithAuth(`/api/admin/keys/${keyId}/revoke`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ reason: 'Revoked via WebUI' })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to revoke key');
        }

        Toast.success('API Key revoked successfully!');
        loadApiKeys(); // Reload list
    } catch (error) {
        console.error('❌ Failed to revoke key:', error);
        Toast.error('Failed to revoke key: ' + error.message);
    }
}

// Enable API Key
async function enableAPIKey(keyId, keyName) {
    if (!confirm(`Are you sure you want to ENABLE the key "${keyName}"?\n\nThis will make it active again.`)) {
        return;
    }

    try {
        const response = await fetchWithAuth(`/api/admin/keys/${keyId}/enable`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' }
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to enable key');
        }

        Toast.success('API Key enabled successfully!');
        loadApiKeys(); // Reload list
    } catch (error) {
        console.error('❌ Failed to enable key:', error);
        Toast.error('Failed to enable key: ' + error.message);
    }
}

// Delete API Key (placeholder - not in AUTH-04 spec, but good to have)
async function deleteAPIKey(keyId, keyName) {
    if (!confirm(`Are you sure you want to DELETE the key "${keyName}"?\n\nThis action CANNOT be undone!`)) {
        return;
    }

    try {
        const response = await fetchWithAuth(`/api/admin/keys/${keyId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to delete key');
        }

        Toast.success('API Key deleted successfully!');
        loadApiKeys(); // Reload list
    } catch (error) {
        console.error('❌ Failed to delete key:', error);
        Toast.error('Failed to delete key: ' + error.message);
    }
}

// Close Edit Modal
function closeEditKeyModal() {
    document.getElementById('edit-key-modal').classList.remove('active');
    document.getElementById('edit-key-form').reset();
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

console.log('🔑 Enhanced API Key Management loaded (AUTH-04)');

