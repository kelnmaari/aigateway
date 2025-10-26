// Admin Panel Manager
class AdminPanel {
    constructor() {
        this.currentTab = 'dashboard';
        this.users = [];
        this.apiKeys = [];
        this.models = [];
        this.mcpServers = [];
        this.init();
    }

    async init() {
        console.log('Admin Panel: Initializing...');
        
        // Check if user is admin
        try {
            const user = await api.getCurrentUser();
            if (!user.is_admin) {
                toast.error('Access denied: Admin privileges required');
                setTimeout(() => window.location.href = '/dashboard.html', 1000);
                return;
            }
            
            // Navbar component handles user display now
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

        // Navbar component handles logout now

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

        // Create Backup button
        document.getElementById('create-backup-btn').addEventListener('click', () => this.createBackup());

        // Edit User form
        document.getElementById('edit-user-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleEditUser(e.target);
        });

        // Reset Password form
        document.getElementById('reset-password-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleResetPassword(e.target);
        });

        // MCP Server button
        document.getElementById('create-mcp-btn').addEventListener('click', () => this.showCreateMCPModal());

        // MCP Server form
        document.getElementById('mcp-server-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleMCPServerSubmit(e.target);
        });

        // MCP Category filter
        document.getElementById('mcp-category-filter').addEventListener('change', (e) => {
            this.filterMCPServers(e.target.value);
        });

        // Refresh models button (v1.9.3+)
        const refreshModelsBtn = document.getElementById('refresh-models-btn');
        if (refreshModelsBtn) {
            refreshModelsBtn.addEventListener('click', () => this.refreshModels());
        }

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
            case 'models':
                await this.loadModels();
                break;
            case 'mcp':
                await this.loadMCPServers();
                break;
            case 'backups':
                await this.loadBackups();
                break;
            case 'audit':
                await this.loadAudit();
                break;
            case 'rbac':
                await this.loadRBAC();
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
                        <button class="btn btn-icon" onclick="adminPanel.resetPassword('${user.id}', '${this.escapeHtml(user.username)}')" title="Reset Password">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" stroke-width="2"/>
                                <path d="M9 11l2 2 4-4" stroke-width="2"/>
                            </svg>
                        </button>
                        ${user.status === 'active' ? `
                            <button class="btn btn-icon btn-warning" onclick="adminPanel.disableUser('${user.id}', '${this.escapeHtml(user.username)}')" title="Disable">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <circle cx="12" cy="12" r="10" stroke-width="2"/>
                                    <path d="M4.93 4.93l14.14 14.14" stroke-width="2"/>
                                </svg>
                            </button>
                        ` : `
                            <button class="btn btn-icon btn-success" onclick="adminPanel.enableUser('${user.id}', '${this.escapeHtml(user.username)}')" title="Enable">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" stroke-width="2"/>
                                    <path d="M22 4L12 14.01l-3-3" stroke-width="2"/>
                                </svg>
                            </button>
                        `}
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
                '<tr><td colspan="8" class="table-empty">Failed to load API keys. Check console for details.</td></tr>';
        }
    }

    renderAPIKeys() {
        const tbody = document.getElementById('apikeys-table');
        
        if (this.apiKeys.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No API keys found</td></tr>';
            return;
        }

        tbody.innerHTML = this.apiKeys.map(key => {
            // Determine owner display
            let ownerDisplay = '-';
            if (key.owner_username) {
                ownerDisplay = `<span class="badge badge-info">${this.escapeHtml(key.owner_username)}</span>`;
            } else if (key.scope === 'bootstrap') {
                ownerDisplay = '<span class="badge badge-warning">System</span>';
            }
            
            // Determine organization display
            let orgDisplay = '-';
            if (key.tenant_name) {
                orgDisplay = `<span class="badge badge-success">${this.escapeHtml(key.tenant_name)}</span>`;
            } else if (key.scope === 'personal') {
                orgDisplay = '<span class="badge badge-secondary">Personal</span>';
            }
            
            return `
                <tr>
                    <td><strong>${this.escapeHtml(key.name)}</strong></td>
                    <td>${ownerDisplay}</td>
                    <td>${orgDisplay}</td>
                    <td><code>${key.id}</code></td>
                    <td><span class="badge badge-info">${key.models?.length || 'All'} models</span></td>
                    <td>${key.rate_limits?.requests_per_minute || 'Unlimited'} req/min</td>
                    <td>${key.last_used_at ? this.formatDate(key.last_used_at) : 'Never'}</td>
                    <td>
                        <div class="table-actions">
                            ${key.scope !== 'bootstrap' ? `
                                <button class="btn btn-icon btn-danger" onclick="adminPanel.deleteAPIKey('${key.id}')" title="Delete">
                                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                        <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                                    </svg>
                                </button>
                            ` : '<span class="text-muted">Protected</span>'}
                        </div>
                    </td>
                </tr>
            `;
        }).join('');
    }

    // ==================== BACKUPS ====================
    
    async loadBackups() {
        const loadingEl = document.getElementById('backups-loading');
        const listEl = document.getElementById('backups-list');
        const noBackupsEl = document.getElementById('no-backups');

        loadingEl.style.display = 'block';
        listEl.style.display = 'none';
        noBackupsEl.style.display = 'none';

        try {
            const response = await api.request(`${api.baseURL}/api/admin/backups`);
            if (response.ok) {
                const data = await response.json();
                
                if (data.backups && data.backups.length > 0) {
                    this.renderBackups(data.backups);
                    listEl.style.display = 'block';
                } else {
                    noBackupsEl.style.display = 'block';
                }
            } else {
                throw new Error('Failed to load backups');
            }
        } catch (error) {
            console.error('Failed to load backups:', error);
            listEl.innerHTML = `<div class="error-message">Failed to load backups: ${error.message}</div>`;
            listEl.style.display = 'block';
        } finally {
            loadingEl.style.display = 'none';
        }
    }

    renderBackups(backups) {
        const listEl = document.getElementById('backups-list');
        
        // Sort by created date (newest first)
        backups.sort((a, b) => new Date(b.created) - new Date(a.created));

        const html = `
            <div class="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>📁 Filename</th>
                            <th>📅 Created</th>
                            <th>💾 Size</th>
                            <th>⚙️ Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${backups.map(backup => `
                            <tr>
                                <td><code>${backup.filename}</code></td>
                                <td>${new Date(backup.created).toLocaleString()}</td>
                                <td>${this.formatFileSize(backup.size)}</td>
                                <td>
                                    <button class="btn btn-sm btn-secondary" onclick="adminPanel.downloadBackup('${backup.filename}')">
                                        <i class="fas fa-download"></i> Download
                                    </button>
                                    <button class="btn btn-sm btn-warning" onclick="adminPanel.restoreBackup('${backup.filename}')">
                                        <i class="fas fa-undo"></i> Restore
                                    </button>
                                    <button class="btn btn-sm btn-danger" onclick="adminPanel.deleteBackup('${backup.filename}')">
                                        <i class="fas fa-trash"></i> Delete
                                    </button>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
        
        listEl.innerHTML = html;
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    }

    async createBackup() {
        const btn = document.getElementById('create-backup-btn');
        btn.disabled = true;
        btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Creating...';

        try {
            const response = await api.request(`${api.baseURL}/api/admin/backup`, {
                method: 'POST'
            });

            if (response.ok) {
                const data = await response.json();
                toast.success(`Backup created successfully!\n\nFilename: ${data.filename}\nSize: ${this.formatFileSize(data.size)}`);
                await this.loadBackups();
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to create backup');
            }
        } catch (error) {
            console.error('Failed to create backup:', error);
            toast.error(`Failed to create backup: ${error.message}`);
        } finally {
            btn.disabled = false;
            btn.innerHTML = '<i class="fas fa-plus"></i> Create Backup';
        }
    }

    async downloadBackup(filename) {
        try {
            const token = api.getToken();
            const url = `${api.baseURL}/api/admin/backup/${encodeURIComponent(filename)}`;
            
            // Create hidden link and trigger download
            const link = document.createElement('a');
            link.href = url;
            link.download = filename;
            link.setAttribute('target', '_blank');
            
            // Add auth header by using fetch and blob
            const response = await api.request(url);
            if (response.ok) {
                const blob = await response.blob();
                const blobUrl = window.URL.createObjectURL(blob);
                link.href = blobUrl;
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
                window.URL.revokeObjectURL(blobUrl);
            } else {
                throw new Error('Failed to download backup');
            }
        } catch (error) {
            console.error('Failed to download backup:', error);
            toast.error(`Failed to download backup: ${error.message}`);
        }
    }

    async restoreBackup(filename) {
        const confirmed = await modal.warning(
            `This will restore the database from backup!`,
            'Restore Database',
            `Backup: ${filename}\n\nCurrent data will be replaced. A safety backup will be created automatically.\nThe server will need to be restarted after restore.`
        );

        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/restore/${encodeURIComponent(filename)}`, {
                method: 'POST'
            });

            if (response.ok) {
                const data = await response.json();
                toast.success(`${data.message}\n\nPlease restart the server for changes to take effect.`);
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to restore backup');
            }
        } catch (error) {
            console.error('Failed to restore backup:', error);
            toast.error(`Failed to restore backup: ${error.message}`);
        }
    }

    async deleteBackup(filename) {
        const confirmed = await modal.danger(
            `Are you sure you want to delete this backup?`,
            'Delete Backup',
            `Filename: ${filename}\n\nThis action cannot be undone.`
        );

        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/backup/${encodeURIComponent(filename)}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                toast.success('Backup deleted successfully');
                await this.loadBackups();
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete backup');
            }
        } catch (error) {
            console.error('Failed to delete backup:', error);
            toast.error(`Failed to delete backup: ${error.message}`);
        }
    }

    // ==================== AUDIT ====================

    // Load Audit Events (v1.11.4+)
    async loadAudit() {
        const loadingEl = document.getElementById('audit-loading');
        const listEl = document.getElementById('audit-events-list');
        const noEventsEl = document.getElementById('no-audit-events');

        loadingEl.style.display = 'block';
        listEl.style.display = 'none';
        noEventsEl.style.display = 'none';

        try {
            // Load stats
            const statsResp = await api.request(`${api.baseURL}/api/admin/audit/stats`);
            if (statsResp.ok) {
                const statsData = await statsResp.json();
                document.getElementById('audit-stat-total').textContent = statsData.stats.total || 0;
                document.getElementById('audit-stat-critical').textContent = statsData.stats.by_severity.critical || 0;
                document.getElementById('audit-stat-warning').textContent = statsData.stats.by_severity.warning || 0;
                document.getElementById('audit-stat-failed-logins').textContent = statsData.stats.security_events.failed_logins || 0;
            }

            // Load recent events (last 20)
            const eventsResp = await api.request(`${api.baseURL}/api/admin/audit?limit=20&offset=0`);
            if (eventsResp.ok) {
                const eventsData = await eventsResp.json();
                
                if (eventsData.events && eventsData.events.length > 0) {
                    this.renderAuditEvents(eventsData.events);
                    listEl.style.display = 'block';
                } else {
                    noEventsEl.style.display = 'block';
                }
            } else {
                throw new Error('Failed to load audit events');
            }
        } catch (error) {
            console.error('Failed to load audit:', error);
            listEl.innerHTML = `<div class="error-message">Failed to load audit events: ${error.message}</div>`;
            listEl.style.display = 'block';
        } finally {
            loadingEl.style.display = 'none';
        }
    }

    renderAuditEvents(events) {
        const tbody = document.getElementById('audit-events-tbody');
        
        const getSeverityClass = (severity) => {
            switch(severity) {
                case 'critical': return 'severity-critical';
                case 'warning': return 'severity-warning';
                case 'info': return 'severity-info';
                default: return '';
            }
        };

        const getStatusClass = (status) => {
            return status === 'success' ? 'status-success' : 'status-failure';
        };

        const html = events.map(event => `
            <tr>
                <td>${new Date(event.timestamp).toLocaleString()}</td>
                <td><span class="badge">${event.event_type}</span></td>
                <td><span class="badge ${getSeverityClass(event.severity)}">${event.severity}</span></td>
                <td title="${event.actor_id}"><code>${event.actor_id.substring(0, 8)}...</code></td>
                <td>${event.action}</td>
                <td><span class="badge ${getStatusClass(event.status)}">${event.status}</span></td>
            </tr>
        `).join('');
        
        tbody.innerHTML = html;
    }

    // ==================== RBAC ====================

    // Load RBAC Stats (v1.11.5+)
    async loadRBAC() {
        try {
            // Load roles
            const rolesResp = await api.request(`${api.baseURL}/api/admin/rbac/roles`);
            if (rolesResp.ok) {
                const rolesData = await rolesResp.json();
                const roles = rolesData.roles || [];
                const customRoles = roles.filter(r => r.type === 'custom');
                
                document.getElementById('rbac-stat-roles').textContent = roles.length;
                document.getElementById('rbac-stat-custom-roles').textContent = customRoles.length;
            }
            
            // Load permissions
            const permsResp = await api.request(`${api.baseURL}/api/admin/rbac/permissions`);
            if (permsResp.ok) {
                const permsData = await permsResp.json();
                document.getElementById('rbac-stat-permissions').textContent = (permsData.permissions || []).length;
            }
            
            // Count user assignments (approximate - total users * avg roles)
            const usersResp = await api.request(`${api.baseURL}/api/admin/users`);
            if (usersResp.ok) {
                const usersData = await usersResp.json();
                // For now, just show user count as proxy for assignments
                document.getElementById('rbac-stat-assignments').textContent = (usersData.users || []).length;
            }
        } catch (error) {
            console.error('Failed to load RBAC stats:', error);
            document.getElementById('rbac-stat-roles').textContent = 'Error';
            document.getElementById('rbac-stat-custom-roles').textContent = 'Error';
            document.getElementById('rbac-stat-permissions').textContent = 'Error';
            document.getElementById('rbac-stat-assignments').textContent = 'Error';
        }
    }

    // ==================== SYSTEM ====================

    // Load Models (v1.9.3+)
    async loadModels() {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/models`);
            if (response.ok) {
                const data = await response.json();
                this.models = data.data || data.models || [];
                this.renderModels();
            }
        } catch (error) {
            console.error('Failed to load models:', error);
            const container = document.getElementById('models-accordion');
            if (container) {
                container.innerHTML = '<div class="error-placeholder">Failed to load models</div>';
            }
        }
    }

    async loadSystem() {
        // System tab теперь только для Performance Monitoring (v1.9.3+)
        // Модели перенесены в Models Tab
        // Логи остались в Logs Tab
        
        // Performance Monitor автоматически запускается при активации System Tab
        // через performance.js event listeners
        console.log('System tab loaded - Performance Monitor will auto-start');
        
        // GPU Monitor инициализация (v1.9.3+)
        if (window.gpuMonitor) {
            await window.gpuMonitor.init();
        }
    }

    renderModels() {
        const container = document.getElementById('models-accordion');
        
        if (this.models.length === 0) {
            container.innerHTML = '<div class="loading-placeholder">No models found</div>';
            return;
        }

        container.innerHTML = this.models.map(model => {
            const modelName = model.id || model.name;
            const modelId = this.sanitizeId(modelName);
            
            return `
                <div class="accordion-item" data-model="${this.escapeHtml(modelName)}">
                    <div class="accordion-header" onclick="adminPanel.toggleAccordion('${modelId}')">
                        <div class="accordion-header-content">
                            <div class="model-item">
                                <code class="model-name">${this.escapeHtml(modelName)}</code>
                                <button 
                                    class="copy-model-btn"
                                    onclick="event.stopPropagation(); copyModelName('${this.escapeHtml(modelName)}')"
                                    aria-label="Copy model name"
                                    title="Copy model name">
                                    <i class="fas fa-copy"></i>
                                </button>
                            </div>
                            <div class="model-meta">
                                <span class="model-size">${this.formatSize(model.size)}</span>
                                <span class="model-modified">${this.formatDate(model.modified_at || model.created)}</span>
                            </div>
                        </div>
                        <div class="accordion-icon">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M6 9l6 6 6-6" stroke-width="2"/>
                            </svg>
                        </div>
                    </div>
                    <div class="accordion-body" id="accordion-${modelId}">
                        <div class="loading-details">Click to load details...</div>
                    </div>
                </div>
            `;
        }).join('');
    }

    async toggleAccordion(modelId) {
        const body = document.getElementById(`accordion-${modelId}`);
        const item = body.closest('.accordion-item');
        const modelName = item.dataset.model;
        
        // Закрыть если уже открыт
        if (item.classList.contains('active')) {
            item.classList.remove('active');
            return;
        }

        // Закрыть все другие
        document.querySelectorAll('.accordion-item.active').forEach(el => {
            if (el !== item) el.classList.remove('active');
        });

        // Открыть текущий
        item.classList.add('active');

        // Загрузить детали если еще не загружены
        if (body.querySelector('.loading-details')) {
            body.innerHTML = '<div class="loading-details"><i class="fas fa-spinner fa-spin"></i> Loading details...</div>';
            await this.loadModelDetails(modelName, body);
        }
    }

    async loadModelDetails(modelName, container) {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/models/${encodeURIComponent(modelName)}/details`);
            
            if (!response.ok) {
                throw new Error('Failed to load model details');
            }

            const details = await response.json();
            container.innerHTML = this.renderModelDetails(details);
        } catch (error) {
            console.error('Failed to load model details:', error);
            container.innerHTML = `
                <div class="error-message">
                    <i class="fas fa-exclamation-circle"></i>
                    Failed to load model details: ${error.message}
                </div>
            `;
        }
    }

    renderModelDetails(details) {
        const sections = [];

        // Basic Information
        const basicInfo = [];
        if (details.family) basicInfo.push(['Family', details.family]);
        if (details.format) basicInfo.push(['Format', details.format]);
        if (details.parameter_size) basicInfo.push(['Parameters', details.parameter_size]);
        if (details.quantization) basicInfo.push(['Quantization', details.quantization]);
        if (details.architecture) basicInfo.push(['Architecture', details.architecture]);

        if (basicInfo.length > 0) {
            sections.push(`
                <div class="details-section">
                    <h4>📋 Basic Information</h4>
                    <div class="details-grid">
                        ${basicInfo.map(([label, value]) => `
                            <div class="detail-item">
                                <span class="detail-label">${label}:</span>
                                <span class="detail-value">${this.escapeHtml(value)}</span>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `);
        }

        // Model Specifications
        const specs = [];
        if (details.context_length) specs.push(['Context Length', details.context_length.toLocaleString()]);
        if (details.embedding_size) specs.push(['Embedding Size', details.embedding_size.toLocaleString()]);
        if (details.layers) specs.push(['Layers', details.layers.toLocaleString()]);
        if (details.attention_heads) specs.push(['Attention Heads', details.attention_heads.toLocaleString()]);
        if (details.vocab_size) specs.push(['Vocabulary Size', details.vocab_size.toLocaleString()]);

        if (specs.length > 0) {
            sections.push(`
                <div class="details-section">
                    <h4>⚙️ Model Specifications</h4>
                    <div class="details-grid">
                        ${specs.map(([label, value]) => `
                            <div class="detail-item">
                                <span class="detail-label">${label}:</span>
                                <span class="detail-value">${value}</span>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `);
        }

        // Template
        if (details.template) {
            sections.push(`
                <div class="details-section">
                    <h4>📝 Prompt Template</h4>
                    <pre class="code-block">${this.escapeHtml(details.template)}</pre>
                </div>
            `);
        }

        // Modelfile
        if (details.modelfile) {
            // Удаляем LICENSE секцию из Modelfile для компактности
            const cleanModelfile = this.stripLicenseFromModelfile(details.modelfile);
            sections.push(`
                <div class="details-section">
                    <h4>📄 Modelfile</h4>
                    <pre class="code-block code-block-large">${this.escapeHtml(cleanModelfile)}</pre>
                </div>
            `);
        }

        // License
        if (details.license) {
            sections.push(`
                <div class="details-section">
                    <h4>📜 License</h4>
                    <pre class="code-block">${this.escapeHtml(details.license)}</pre>
                </div>
            `);
        }

        // Parameters
        if (details.parameters && Object.keys(details.parameters).length > 0) {
            sections.push(`
                <div class="details-section">
                    <h4>🔧 Parameters</h4>
                    <div class="details-grid">
                        ${Object.entries(details.parameters).map(([key, value]) => `
                            <div class="detail-item">
                                <span class="detail-label">${this.escapeHtml(key)}:</span>
                                <span class="detail-value">${this.escapeHtml(String(value))}</span>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `);
        }

        return sections.join('') || '<div class="no-details">No additional details available</div>';
    }

    sanitizeId(str) {
        return str.replace(/[^a-zA-Z0-9-_]/g, '-');
    }

    // Refresh models (v1.9.3+)
    async refreshModels() {
        const btn = document.getElementById('refresh-models-btn');
        const originalHTML = btn.innerHTML;
        
        try {
            // Disable button and show loading
            btn.disabled = true;
            btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Refreshing...';
            
            // Reload models
            const response = await api.request(`${api.baseURL}/api/admin/models`);
            if (response.ok) {
                const data = await response.json();
                this.models = data.data || data.models || [];
                this.renderModels();
                notifications.showSuccess('Models refreshed successfully');
            } else {
                throw new Error('Failed to refresh models');
            }
        } catch (error) {
            console.error('Failed to refresh models:', error);
            notifications.showError('Failed to refresh models');
        } finally {
            // Re-enable button
            btn.disabled = false;
            btn.innerHTML = originalHTML;
        }
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

            toast.success('User created successfully!');
            this.closeModals();
            form.reset();
            await this.loadUsers();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async editUser(userId) {
        try {
            // Load user data
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}`);
            if (!response.ok) {
                throw new Error('Failed to load user');
            }
            
            const user = await response.json();
            
            // Populate modal
            document.getElementById('edit-user-id').value = user.id;
            document.getElementById('edit-username').value = user.username;
            document.getElementById('edit-email').value = user.email;
            document.getElementById('edit-fullname').value = user.full_name || '';
            document.getElementById('edit-is-admin').checked = user.is_admin || false;
            document.getElementById('edit-status').value = user.status;
            
            // Show modal
            document.getElementById('edit-user-modal').classList.add('show');
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async handleEditUser(form) {
        const userId = document.getElementById('edit-user-id').value;
        const userData = {
            email: document.getElementById('edit-email').value,
            full_name: document.getElementById('edit-fullname').value || undefined,
            is_admin: document.getElementById('edit-is-admin').checked,
            status: document.getElementById('edit-status').value
        };

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}`, {
                method: 'PUT',
                body: JSON.stringify(userData)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to update user');
            }

            toast.success('User updated successfully!');
            this.closeModals();
            await this.loadUsers();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async resetPassword(userId, username) {
        // Show modal
        document.getElementById('reset-user-id').value = userId;
        document.getElementById('reset-user-name').textContent = username;
        document.getElementById('reset-password').value = '';
        document.getElementById('reset-password-confirm').value = '';
        document.getElementById('reset-password-modal').classList.add('show');
    }

    async handleResetPassword(form) {
        const userId = document.getElementById('reset-user-id').value;
        const password = document.getElementById('reset-password').value;
        const confirmPassword = document.getElementById('reset-password-confirm').value;

        if (password !== confirmPassword) {
            toast.error('Passwords do not match!');
            return;
        }

        if (password.length < 8) {
            toast.error('Password must be at least 8 characters!');
            return;
        }

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}/reset-password`, {
                method: 'POST',
                body: JSON.stringify({ new_password: password })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to reset password');
            }

            toast.success('Password reset successfully!');
            this.closeModals();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async disableUser(userId, username) {
        const confirmed = await modal.confirm(
            `Are you sure you want to disable user "${username}"?`,
            'Disable User'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}/disable`, {
                method: 'PATCH'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to disable user');
            }

            toast.success('User disabled successfully!');
            await this.loadUsers();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async enableUser(userId, username) {
        const confirmed = await modal.confirm(
            `Are you sure you want to enable user "${username}"?`,
            'Enable User'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}/enable`, {
                method: 'PATCH'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to enable user');
            }

            toast.success('User enabled successfully!');
            await this.loadUsers();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async deleteUser(userId) {
        const confirmed = await modal.danger(
            'Are you sure you want to delete this user? This action cannot be undone.',
            'Delete User'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/users/${userId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete user');
            }

            toast.success('User deleted successfully!');
            await this.loadUsers();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    async deleteAPIKey(keyId) {
        const confirmed = await modal.danger(
            'Are you sure you want to delete this API key? This action cannot be undone.',
            'Delete API Key'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/keys/${keyId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete API key');
            }

            toast.success('API key deleted successfully!');
            await this.loadAPIKeys();
        } catch (error) {
            toast.error(`Error: ${error.message}`);
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
            toast.success(`API Key created successfully!\n\nKey: ${result.plain_key}\n\nSave this key securely - it will not be shown again!`, 30000);
            
            this.closeModals();
            form.reset();
            await this.loadAPIKeys();
            await this.loadDashboard(); // Refresh stats
        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    closeModals() {
        document.querySelectorAll('.modal').forEach(modal => modal.classList.remove('show'));
    }

    // ==================== MCP Servers Management ====================

    async loadMCPServers() {
        try {
            // Admin sees ALL servers, including inactive ones
            const response = await api.request(`${api.baseURL}/api/admin/mcp/servers?active_only=false`);
            if (!response.ok) throw new Error('Failed to load MCP servers');
            
            const data = await response.json();
            this.mcpServers = data.servers || [];
            
            // Load categories for filter
            await this.loadMCPCategories();
            
            this.renderMCPServers();
        } catch (error) {
            console.error('Failed to load MCP servers:', error);
            document.getElementById('mcp-servers-table').innerHTML = 
                `<tr><td colspan="6" class="table-empty error">Failed to load MCP servers: ${error.message}</td></tr>`;
        }
    }

    async loadMCPCategories() {
        try {
            const response = await api.request(`${api.baseURL}/api/mcp/categories`);
            if (!response.ok) throw new Error('Failed to load categories');
            
            const data = await response.json();
            const select = document.getElementById('mcp-category-filter');
            
            select.innerHTML = '<option value="">All Categories</option>';
            data.categories.forEach(cat => {
                const option = document.createElement('option');
                option.value = cat;
                option.textContent = cat.charAt(0).toUpperCase() + cat.slice(1);
                select.appendChild(option);
            });
        } catch (error) {
            console.error('Failed to load categories:', error);
        }
    }

    renderMCPServers(servers = this.mcpServers) {
        const tbody = document.getElementById('mcp-servers-table');
        
        if (servers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="table-empty">No MCP servers found</td></tr>';
            return;
        }

        tbody.innerHTML = servers.map(server => {
            const tags = Array.isArray(server.tags) ? server.tags : [];
            const tagsHtml = tags.length > 0 
                ? tags.slice(0, 3).map(tag => `<span class="tag">${this.escapeHtml(tag)}</span>`).join(' ')
                : '<span class="text-muted">—</span>';
            
            const statusBadge = server.is_active 
                ? '<span class="badge badge-success">Active</span>'
                : '<span class="badge badge-secondary">Inactive</span>';

            const categoryBadge = `<span class="category-badge category-${server.category}">${server.category}</span>`;

            return `
                <tr>
                    <td><strong>${this.escapeHtml(server.name)}</strong></td>
                    <td>${categoryBadge}</td>
                    <td class="description-cell">${this.escapeHtml(server.description || '')}</td>
                    <td class="tags-cell">${tagsHtml}</td>
                    <td>${statusBadge}</td>
                    <td class="actions-cell">
                        <button class="btn-icon" onclick="adminPanel.editMCPServer('${server.id}')" title="Edit">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke-width="2"/>
                                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke-width="2"/>
                            </svg>
                        </button>
                        <button class="btn-icon btn-danger" onclick="adminPanel.deleteMCPServer('${server.id}')" title="Delete">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                            </svg>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');
    }

    filterMCPServers(category) {
        if (!category) {
            this.renderMCPServers();
            return;
        }
        
        const filtered = this.mcpServers.filter(server => server.category === category);
        this.renderMCPServers(filtered);
    }

    showCreateMCPModal() {
        document.getElementById('mcp-modal-title').textContent = 'Add MCP Server';
        document.getElementById('mcp-submit-btn').textContent = 'Create Server';
        document.getElementById('mcp-server-form').reset();
        document.getElementById('mcp-server-id').value = '';
        document.getElementById('mcp-active').checked = true;
        document.getElementById('mcp-server-modal').classList.add('show');
    }

    async editMCPServer(serverId) {
        const server = this.mcpServers.find(s => s.id === serverId);
        if (!server) return;

        document.getElementById('mcp-modal-title').textContent = 'Edit MCP Server';
        document.getElementById('mcp-submit-btn').textContent = 'Update Server';
        document.getElementById('mcp-server-id').value = server.id;
        document.getElementById('mcp-name').value = server.name;
        document.getElementById('mcp-category').value = server.category;
        document.getElementById('mcp-description').value = server.description || '';
        document.getElementById('mcp-installation').value = server.installation_guide || '';
        document.getElementById('mcp-website').value = server.website_url || '';
        document.getElementById('mcp-github').value = server.github_url || '';
        document.getElementById('mcp-tags').value = Array.isArray(server.tags) ? server.tags.join(', ') : '';
        document.getElementById('mcp-active').checked = server.is_active;
        
        document.getElementById('mcp-server-modal').classList.add('show');
    }

    async handleMCPServerSubmit(form) {
        const serverId = document.getElementById('mcp-server-id').value;
        const isEdit = !!serverId;

        const tags = document.getElementById('mcp-tags').value
            .split(',')
            .map(t => t.trim())
            .filter(t => t);

        const data = {
            name: document.getElementById('mcp-name').value,
            category: document.getElementById('mcp-category').value,
            description: document.getElementById('mcp-description').value,
            installation_guide: document.getElementById('mcp-installation').value,
            website_url: document.getElementById('mcp-website').value || undefined,
            github_url: document.getElementById('mcp-github').value || undefined,
            tags: tags.length > 0 ? tags : undefined,
            is_active: document.getElementById('mcp-active').checked
        };

        try {
            const url = isEdit 
                ? `${api.baseURL}/api/admin/mcp/servers/${serverId}`
                : `${api.baseURL}/api/admin/mcp/servers`;
            
            const response = await api.request(url, {
                method: isEdit ? 'PUT' : 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to save MCP server');
            }

            this.closeModals();
            await this.loadMCPServers();
            
            toast.success(`MCP server ${isEdit ? 'updated' : 'created'} successfully!`);
        } catch (error) {
            console.error('Failed to save MCP server:', error);
            toast.error(`Failed to save MCP server: ${error.message}`);
        }
    }

    async deleteMCPServer(serverId) {
        const server = this.mcpServers.find(s => s.id === serverId);
        if (!server) return;

        const confirmed = await modal.danger(
            `Are you sure you want to delete "${server.name}"?`,
            'Delete MCP Server'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/mcp/servers/${serverId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete MCP server');
            }

            await this.loadMCPServers();
            toast.success('MCP server deleted successfully!');
        } catch (error) {
            console.error('Failed to delete MCP server:', error);
            toast.error(`Failed to delete MCP server: ${error.message}`);
        }
    }

    // Utility methods
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    stripLicenseFromModelfile(modelfile) {
        if (!modelfile) return modelfile;
        
        // Ищем строку LICENSE """ или LICENSE """ и все что после нее
        const licenseMatch = modelfile.match(/^LICENSE\s+"""/mi);
        if (licenseMatch) {
            return modelfile.substring(0, licenseMatch.index).trim();
        }
        
        // Альтернативный вариант: просто LICENSE с многострочным текстом
        const licenseLine = modelfile.match(/^LICENSE\s*$/mi);
        if (licenseLine) {
            return modelfile.substring(0, licenseLine.index).trim();
        }
        
        return modelfile;
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

