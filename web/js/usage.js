// Usage Statistics Page
// Version 1.3.0

class UsageManager {
    constructor() {
        this.currentUser = null;
        this.tenants = [];
        this.timePeriod = '7days';
        this.scope = 'personal';
        this.currentTenant = null;
        
        this.init();
    }

    async init() {
        try {
            // Load current user
            await this.loadUser();
            
            // Load tenants
            await this.loadTenants();
            
            // Load usage data
            await this.loadUsageData();
            
            // Setup UI
            this.setupEventListeners();
            
        } catch (error) {
            console.error('Failed to initialize usage page:', error);
            this.showError(error.message);
        }
    }

    // Load current user
    async loadUser() {
        try {
            this.currentUser = await api.getCurrentUser();
            
            // Update nav bar
            const navUserName = document.getElementById('nav-user-name');
            if (navUserName) {
                navUserName.textContent = this.currentUser.full_name || this.currentUser.username;
            }
            
            // Show admin link if user is admin
            if (this.currentUser.is_admin) {
                this.showAdminLink();
            }
        } catch (error) {
            console.error('Failed to load user:', error);
            throw error;
        }
    }
    
    showAdminLink() {
        const placeholder = document.getElementById('admin-link-placeholder');
        if (placeholder && !document.querySelector('.nav-link[href="/admin.html"]')) {
            const adminLink = document.createElement('a');
            adminLink.href = '/admin.html';
            adminLink.className = 'nav-link';
            adminLink.innerHTML = `
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                    <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" stroke-width="2"/>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" stroke-width="2"/>
                </svg>
                Admin
            `;
            placeholder.appendChild(adminLink);
            console.log('✅ Admin link added to navigation');
        }
    }

    // Load tenants
    async loadTenants() {
        try {
            const response = await api.getUserTenants();
            this.tenants = response.tenants || [];
            
            // Populate tenant selector
            this.populateTenantSelector();
            
        } catch (error) {
            console.error('Failed to load tenants:', error);
            this.tenants = [];
        }
    }

    // Populate tenant selector
    populateTenantSelector() {
        const selector = document.getElementById('tenant-selector');
        if (!selector) return;

        // Clear existing options except first
        while (selector.options.length > 1) {
            selector.remove(1);
        }

        // Filter out personal tenants
        const orgTenants = this.tenants.filter(t => t.type !== 'personal');

        // Add tenant options
        orgTenants.forEach(tenant => {
            const option = document.createElement('option');
            option.value = tenant.id;
            option.textContent = tenant.name;
            selector.appendChild(option);
        });
    }

    // Load usage data
    async loadUsageData() {
        try {
            let endpoint;
            
            if (this.scope === 'personal') {
                endpoint = `${api.baseURL}/api/usage/personal?period=${this.timePeriod}`;
            } else if (this.scope === 'tenant' && this.currentTenant) {
                endpoint = `${api.baseURL}/api/usage/tenant/${this.currentTenant}?period=${this.timePeriod}`;
            } else {
                // No tenant selected yet
                return;
            }

            const response = await api.request(endpoint);
            if (!response.ok) {
                throw new Error('Failed to load usage data');
            }

            const stats = await response.json();
            this.renderUsageData(stats);
            
        } catch (error) {
            console.error('Failed to load usage data:', error);
            this.showError('Failed to load usage statistics');
            // Fallback to showing no data
            this.renderEmptyState();
        }
    }

    // Render real usage data from API
    renderUsageData(stats) {
        if (!stats) {
            this.renderEmptyState();
            return;
        }

        // Summary stats
        document.getElementById('stat-total-requests').textContent = this.formatNumber(stats.total_requests || 0);
        document.getElementById('stat-avg-response').textContent = stats.avg_duration ? `${stats.avg_duration}ms` : '-';
        document.getElementById('stat-tokens').textContent = this.formatNumber(stats.total_tokens || 0);
        document.getElementById('stat-error-rate').textContent = stats.error_rate ? `${(stats.error_rate * 100).toFixed(1)}%` : '0%';

        // Model usage
        const modelUsage = stats.models || [];
        this.renderModelUsage(modelUsage);

        // API Key usage (if available)
        const apikeyUsage = stats.api_keys || [];
        this.renderAPIKeyUsage(apikeyUsage);

        // Recent requests
        const recentRequests = stats.recent_requests || [];
        this.renderRecentRequests(recentRequests);
    }

    renderModelUsage(models) {
        const modelTableBody = document.getElementById('model-usage-list');
        if (!models || models.length === 0) {
            modelTableBody.innerHTML = '<tr><td colspan="5" class="table-empty">No usage data available</td></tr>';
            return;
        }

        modelTableBody.innerHTML = models.map(m => `
            <tr>
                <td><strong>${this.escapeHtml(m.model)}</strong></td>
                <td>${this.formatNumber(m.requests)}</td>
                <td>${this.formatNumber(m.tokens)}</td>
                <td>${m.avg_duration ? m.avg_duration + 'ms' : '-'}</td>
                <td>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${(m.success_rate * 100).toFixed(0)}%;"></div>
                    </div>
                    <span style="font-size: 12px; color: var(--text-secondary);">${(m.success_rate * 100).toFixed(1)}%</span>
                </td>
            </tr>
        `).join('');
    }

    renderAPIKeyUsage(apikeys) {
        const apikeyTableBody = document.getElementById('apikey-usage-list');
        if (!apikeys || apikeys.length === 0) {
            apikeyTableBody.innerHTML = '<tr><td colspan="5" class="table-empty">No usage data available</td></tr>';
            return;
        }

        apikeyTableBody.innerHTML = apikeys.map(k => {
            const statusBadge = '<span class="badge badge-success">OK</span>'; // TODO: implement real status
            
            return `
                <tr>
                    <td><code class="key-display">${this.escapeHtml(k.key_id || 'N/A')}</code></td>
                    <td>${this.formatNumber(k.requests)}</td>
                    <td>${this.formatNumber(k.tokens)}</td>
                    <td>${k.last_used ? this.formatDateTime(k.last_used) : 'Never'}</td>
                    <td>${statusBadge}</td>
                </tr>
            `;
        }).join('');
    }

    renderRecentRequests(requests) {
        const recentTableBody = document.getElementById('recent-requests-list');
        if (!requests || requests.length === 0) {
            recentTableBody.innerHTML = '<tr><td colspan="6" class="table-empty">No recent requests</td></tr>';
            return;
        }

        recentTableBody.innerHTML = requests.map(r => {
            const statusBadge = r.success
                ? '<span class="badge badge-success">Success</span>'
                : '<span class="badge" style="background: var(--error-color); color: white;">Error</span>';
            
            return `
                <tr>
                    <td>${this.formatDateTime(r.timestamp)}</td>
                    <td>${this.escapeHtml(r.model)}</td>
                    <td><code class="key-display">${this.escapeHtml(r.key_id || 'N/A')}</code></td>
                    <td>${this.formatNumber(r.tokens)}</td>
                    <td>${r.duration ? r.duration + 'ms' : '-'}</td>
                    <td>${statusBadge}</td>
                </tr>
            `;
        }).join('');
    }

    renderEmptyState() {
        // Summary stats
        document.getElementById('stat-total-requests').textContent = '0';
        document.getElementById('stat-avg-response').textContent = '-';
        document.getElementById('stat-tokens').textContent = '0';
        document.getElementById('stat-error-rate').textContent = '0%';

        // Empty tables
        document.getElementById('model-usage-list').innerHTML = '<tr><td colspan="5" class="table-empty">No usage data available</td></tr>';
        document.getElementById('apikey-usage-list').innerHTML = '<tr><td colspan="5" class="table-empty">No usage data available</td></tr>';
        document.getElementById('recent-requests-list').innerHTML = '<tr><td colspan="6" class="table-empty">No recent requests</td></tr>';
    }

    // Setup event listeners
    setupEventListeners() {
        // User dropdown and logout are now handled by navbar.js component
        // No need for duplicate event listeners here

        // Time period selector
        const timePeriodSelect = document.getElementById('time-period');
        if (timePeriodSelect) {
            timePeriodSelect.addEventListener('change', (e) => {
                this.timePeriod = e.target.value;
                this.loadUsageData();
            });
        }

        // Scope selector
        const scopeSelect = document.getElementById('scope-selector');
        const tenantSelectorGroup = document.getElementById('tenant-selector-group');
        
        if (scopeSelect) {
            scopeSelect.addEventListener('change', (e) => {
                this.scope = e.target.value;
                
                if (tenantSelectorGroup) {
                    tenantSelectorGroup.style.display = this.scope === 'tenant' ? 'block' : 'none';
                }
                
                if (this.scope === 'personal') {
                    this.loadUsageData();
                }
            });
        }

        // Tenant selector
        const tenantSelector = document.getElementById('tenant-selector');
        if (tenantSelector) {
            tenantSelector.addEventListener('change', (e) => {
                this.currentTenant = e.target.value;
                if (this.currentTenant) {
                    this.loadUsageData();
                }
            });
        }
    }

    // Format number with commas
    formatNumber(num) {
        return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    }

    // Format date time
    formatDateTime(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;

        // Less than 1 minute
        if (diff < 60000) {
            return 'Just now';
        }

        // Less than 1 hour
        if (diff < 3600000) {
            const minutes = Math.floor(diff / 60000);
            return `${minutes}m ago`;
        }

        // Less than 24 hours
        if (diff < 86400000) {
            const hours = Math.floor(diff / 3600000);
            return `${hours}h ago`;
        }

        // Otherwise show date
        return date.toLocaleDateString('en-US', { 
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    // Show error message
    showError(message) {
        const errorDiv = document.getElementById('usage-message');
        if (errorDiv) {
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            errorDiv.className = 'error-message';
            
            setTimeout(() => {
                errorDiv.style.display = 'none';
            }, 5000);
        }
    }

    // Escape HTML
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Logout
    async logout() {
        try {
            await api.logout();
            localStorage.clear();
            window.location.href = '/login.html';
        } catch (error) {
            console.error('Logout failed:', error);
            // Force logout on client side
            localStorage.clear();
            window.location.href = '/login.html';
        }
    }
}

// Initialize on page load
let usageManager;
document.addEventListener('DOMContentLoaded', () => {
    usageManager = new UsageManager();
});



