// Admin Panel Manager
class AdminPanel {
    constructor() {
        this.currentTab = 'dashboard';
        this.users = [];
        this.apiKeys = [];
        this.models = [];
        this.init();
    }

    async init() {
        console.log('Admin Panel: Initializing...');
        
        // Check if user is admin
        try {
            const user = await api.getCurrentUser();
            if (!user.is_admin) {
                alert('Access denied: Admin privileges required');
                window.location.href = '/dashboard.html';
                return;
            }
            
            // Update nav
            document.getElementById('nav-user-name').textContent = user.full_name || user.username;
            document.getElementById('nav-user-avatar').textContent = (user.full_name || user.username).charAt(0).toUpperCase();
        } catch (error) {
            console.error('Failed to verify admin status:', error);
            window.location.href = '/login.html';
            return;
        }

        // Setup event listeners
        this.setupEventListeners();
        
        // Load initial data
        await this.loadDashboard();
        
        console.log('✅ Admin Panel initialized');
    }

    setupEventListeners() {
        // Tab switching
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', () => this.switchTab(btn.dataset.tab));
        });

        // Logout
        document.getElementById('logout-btn').addEventListener('click', () => api.logout());

        // Create User button
        document.getElementById('create-user-btn').addEventListener('click', () => this.showCreateUserModal());

        // Create User form
        document.getElementById('create-user-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateUser(e.target);
        });

        // Create API Key button
        document.getElementById('create-apikey-btn').addEventListener('click', () => this.showCreateAPIKeyModal());

        // Create API Key form
        document.getElementById('create-apikey-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateAPIKey(e.target);
        });

        // Modal close buttons
        document.querySelectorAll('.modal-close, .cancel-btn').forEach(btn => {
            btn.addEventListener('click', () => this.closeModals());
        });

        // Close modal on outside click
        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModals();
            }
        });
    }

    async switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });

        // Update tab panes
        document.querySelectorAll('.tab-pane').forEach(pane => {
            pane.classList.remove('active');
        });
        document.getElementById(`${tabName}-tab`).classList.add('active');

        this.currentTab = tabName;

        // Load tab data
        switch(tabName) {
            case 'dashboard':
                await this.loadDashboard();
                break;
            case 'users':
                await this.loadUsers();
                break;
            case 'apikeys':
                await this.loadAPIKeys();
                break;
            case 'system':
                await this.loadSystem();
                break;
        }
    }

    async loadDashboard() {
        try {
            // Load users count
            const usersResp = await api.request(`${api.baseURL}/api/admin/users`);
            if (usersResp.ok) {
                const data = await usersResp.json();
                document.getElementById('stat-total-users').textContent = data.total || 0;
            }

            // Load API keys count
            const keysResp = await api.request(`${api.baseURL}/api/admin/keys`);
            if (keysResp.ok) {
                const data = await keysResp.json();
                // API returns total count in the response
                const keysCount = data.total || data.api_keys?.length || data.manager_stats?.total_keys || 0;
                document.getElementById('stat-total-keys').textContent = keysCount;
            }

            // Load models count
            const modelsResp = await api.request(`${api.baseURL}/api/admin/models`);
            if (modelsResp.ok) {
                const data = await modelsResp.json();
                document.getElementById('stat-total-models').textContent = data.data?.length || data.models?.length || 0;
            }

            // Load system stats
            const statsResp = await api.request(`${api.baseURL}/api/admin/stats`);
            if (statsResp.ok) {
                const stats = await statsResp.json();
                this.renderSystemHealth(stats);
            }

            document.getElementById('stat-total-requests').textContent = '0'; // TODO: implement
        } catch (error) {
            console.error('Failed to load dashboard:', error);
        }
    }

    renderSystemHealth(stats) {
        const tbody = document.getElementById('system-health-table');
        tbody.innerHTML = `
            <tr><td><strong>Status</strong></td><td><span class="badge badge-success">Healthy</span></td></tr>
            <tr><td><strong>API Keys</strong></td><td>${stats.manager_stats?.total_keys || 0} keys</td></tr>
            <tr><td><strong>Active Keys</strong></td><td>${stats.manager_stats?.active_keys || 0}</td></tr>
            <tr><td><strong>Rate Limits</strong></td><td>${stats.rate_limit_stats ? 'Enabled' : 'Disabled'}</td></tr>
        `;
    }

    async loadUsers() {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/users`);
            if (!response.ok) {
                throw new Error('Failed to load users');
            }

            const data = await response.json();
            this.users = data.users || [];
            this.renderUsers();
        } catch (error) {
            console.error('Failed to load users:', error);
            document.getElementById('users-table').innerHTML = 
                '<tr><td colspan="6" class="table-empty">Failed to load users</td></tr>';
        }
    }

    renderUsers() {
        const tbody = document.getElementById('users-table');
        
        if (this.users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="table-empty">No users found</td></tr>';
            return;
        }

        tbody.innerHTML = this.users.map(user => `
            <tr>
                <td>
                    <div style="font-weight: 500;">${this.escapeHtml(user.username)}</div>
                </td>
                <td>${this.escapeHtml(user.email)}</td>
                <td>
                    ${user.is_admin ? 
                        '<span class="badge badge-warning">Admin</span>' : 
                        '<span class="badge badge-secondary">User</span>'}
                </td>
                <td>
                    ${user.status === 'active' ? 
                        '<span class="badge badge-success">Active</span>' : 
                        `<span class="badge badge-error">${user.status}</span>`}
                </td>
                <td>${this.formatDate(user.created_at)}</td>
                <td>
                    <div class="table-actions">
                        <button class="btn btn-icon" onclick="adminPanel.editUser('${user.id}')" title="Edit">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke-width="2"/>
                                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke-width="2"/>
                            </svg>
                        </button>
                        ${!user.is_admin ? `
                            <button class="btn btn-icon btn-danger" onclick="adminPanel.deleteUser('${user.id}')" title="Delete">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                                </svg>
                            </button>
                        ` : ''}
                    </div>
                </td>
            </tr>
        `).join('');
    }

    async loadAPIKeys() {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/keys`);
            if (!response.ok) {
                throw new Error('Failed to load API keys');
            }

            const data = await response.json();
            // API returns api_keys array
            this.apiKeys = data.api_keys || data.keys || [];
            this.renderAPIKeys();
        } catch (error) {
            console.error('Failed to load API keys:', error);
            document.getElementById('apikeys-table').innerHTML = 
                '<tr><td colspan="6" class="table-empty">Failed to load API keys. Check console for details.</td></tr>';
        }
    }

    renderAPIKeys() {
        const tbody = document.getElementById('apikeys-table');
        
        if (this.apiKeys.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="table-empty">No API keys found</td></tr>';
            return;
        }

        tbody.innerHTML = this.apiKeys.map(key => `
            <tr>
                <td><strong>${this.escapeHtml(key.name)}</strong></td>
                <td><code>${key.id}</code></td>
                <td><span class="badge badge-info">${key.models?.length || 'All'} models</span></td>
                <td>${key.rate_limits?.requests_per_minute || 'Unlimited'} req/min</td>
                <td>${key.last_used_at ? this.formatDate(key.last_used_at) : 'Never'}</td>
                <td>
                    <div class="table-actions">
                        <button class="btn btn-icon btn-danger" onclick="adminPanel.deleteAPIKey('${key.id}')" title="Delete">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                            </svg>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    async loadSystem() {
        // Load models
        try {
            const response = await api.request(`${api.baseURL}/api/admin/models`);
            if (response.ok) {
                const data = await response.json();
                this.models = data.data || data.models || [];
                this.renderModels();
            }
        } catch (error) {
            console.error('Failed to load models:', error);
        }

        // Load logs
        try {
            const response = await api.request(`${api.baseURL}/api/admin/logs?limit=50`);
            if (response.ok) {
                const data = await response.json();
                document.getElementById('logs-display').textContent = data.logs || 'No logs available';
            }
        } catch (error) {
            console.error('Failed to load logs:', error);
            document.getElementById('logs-display').textContent = 'Failed to load logs';
        }
    }

    renderModels() {
        const tbody = document.getElementById('models-table');
        
        if (this.models.length === 0) {
            tbody.innerHTML = '<tr><td colspan="3" class="table-empty">No models found</td></tr>';
            return;
        }

        tbody.innerHTML = this.models.map(model => `
            <tr>
                <td><code>${this.escapeHtml(model.id || model.name)}</code></td>
                <td>${this.formatSize(model.size)}</td>
                <td>${this.formatDate(model.modified_at || model.created)}</td>
            </tr>
        `).join('');
    }

    // User Management
    showCreateUserModal() {
        document.getElementById('create-user-modal').classList.add('show');
    }

    showCreateAPIKeyModal() {
        document.getElementById('create-apikey-modal').classList.add('show');
    }

    async handleCreateUser(form) {
        const formData = new FormData(form);
        const userData = {
            username: formData.get('username'),
            email: formData.get('email'),
            password: formData.get('password'),
            full_name: formData.get('full_name') || '',
            is_admin: formData.get('is_admin') === 'on'
        };

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users`, {
                method: 'POST',
                body: JSON.stringify(userData)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to create user');
            }

            alert('User created successfully!');
            this.closeModals();
            form.reset();
            await this.loadUsers();
        } catch (error) {
            alert(`Error: ${error.message}`);
        }
    }

    async editUser(userId) {
        alert('Edit user feature coming soon!');
        // TODO: Implement edit user modal
    }

    async deleteUser(userId) {
        if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
            return;
        }

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete user');
            }

            alert('User deleted successfully!');
            await this.loadUsers();
        } catch (error) {
            alert(`Error: ${error.message}`);
        }
    }

    async deleteAPIKey(keyId) {
        if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
            return;
        }

        try {
            const response = await api.request(`${api.baseURL}/api/admin/keys/${keyId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete API key');
            }

            alert('API key deleted successfully!');
            await this.loadAPIKeys();
        } catch (error) {
            alert(`Error: ${error.message}`);
        }
    }

    async handleCreateAPIKey(form) {
        const formData = new FormData(form);
        
        // Parse models
        const modelsStr = formData.get('models');
        const models = modelsStr ? modelsStr.split(',').map(m => m.trim()).filter(m => m) : [];
        
        // Parse permissions
        const permissions = [];
        formData.getAll('permissions').forEach(p => permissions.push(p));
        
        const keyData = {
            name: formData.get('name'),
            description: formData.get('description') || '',
            models: models.length > 0 ? models : null, // null means all models
            permissions: permissions.length > 0 ? permissions : ['chat'],
            rate_limits: {
                requests_per_minute: parseInt(formData.get('rate_limit')) || 0
            }
        };

        try {
            const response = await api.request(`${api.baseURL}/api/admin/keys`, {
                method: 'POST',
                body: JSON.stringify(keyData)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error?.message || error.error || 'Failed to create API key');
            }

            const result = await response.json();
            
            // Show the actual API key to user (only shown once!)
            alert(`API Key created successfully!\n\nKey: ${result.plain_key}\n\nSave this key securely - it will not be shown again!`);
            
            this.closeModals();
            form.reset();
            await this.loadAPIKeys();
            await this.loadDashboard(); // Refresh stats
        } catch (error) {
            alert(`Error: ${error.message}`);
        }
    }

    closeModals() {
        document.querySelectorAll('.modal').forEach(modal => modal.classList.remove('show'));
    }

    // Utility methods
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', { 
            year: 'numeric', 
            month: 'short', 
            day: 'numeric' 
        });
    }

    formatSize(bytes) {
        if (!bytes) return 'N/A';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        let size = bytes;
        let unitIndex = 0;
        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024;
            unitIndex++;
        }
        return `${size.toFixed(2)} ${units[unitIndex]}`;
    }
}

// Initialize admin panel
let adminPanel;
document.addEventListener('DOMContentLoaded', () => {
    adminPanel = new AdminPanel();
});

