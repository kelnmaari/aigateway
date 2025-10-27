// Admin RAG Sources Management
class AdminRAGManager {
    constructor() {
        this.sources = [];
        this.currentPage = 1;
        this.pageSize = 20;
        this.totalSources = 0;
        this.filters = {
            type: '',
            status: '',
            user: ''
        };
        this.currentSourceId = null;
    }

    async init() {
        // Check admin permissions
        try {
            const user = await api.getCurrentUser();
            if (!user.is_admin) {
                this.showError('Access denied: Admin privileges required');
                setTimeout(() => {
                    window.location.href = '/dashboard.html';
                }, 2000);
                return;
            }
        } catch (error) {
            console.error('Failed to verify admin status:', error);
            // If authentication fails, api.js will handle redirect to login
            // Just show error message
            this.showError('Failed to verify admin status. Redirecting to login...');
            return;
        }

        // Check if RAG is enabled
        const ragEnabled = await api.isRAGEnabled();
        if (!ragEnabled) {
            this.showRAGDisabled();
            return;
        }

        this.setupEventListeners();
        await this.loadSources();
        this.updateStats();
    }

    showRAGDisabled() {
        const container = document.querySelector('.dashboard-container');
        container.innerHTML = `
            <header class="page-header">
                <div>
                    <h1>RAG Management (Admin)</h1>
                    <p class="page-description">Manage all RAG data sources across users</p>
                </div>
                <div style="display: flex; gap: 12px;">
                    <a href="/admin.html" class="btn btn-secondary">Back to Admin Panel</a>
                </div>
            </header>
            
            <div style="background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 12px; padding: 48px 24px; min-height: calc(100vh - 280px); display: flex; align-items: center; justify-content: center;">
                <div style="text-align: center; max-width: 600px;">
                    <div style="font-size: 64px; margin-bottom: 24px;">🔒</div>
                    <h2 style="color: rgba(255, 255, 255, 0.9); margin-bottom: 16px;">RAG System Disabled</h2>
                    <p style="color: rgba(255, 255, 255, 0.6); font-size: 16px; line-height: 1.6; margin-bottom: 24px;">
                        The RAG (Retrieval-Augmented Generation) system is currently disabled in the server configuration.
                        <br><br>
                        To enable RAG functionality, update the server configuration.
                    </p>
                    <div style="background: rgba(255, 193, 7, 0.1); border: 1px solid rgba(255, 193, 7, 0.3); border-radius: 8px; padding: 16px; margin-top: 24px;">
                        <p style="color: rgba(255, 193, 7, 0.9); font-size: 14px; margin: 0;">
                            <strong>Configuration:</strong> Set <code style="background: rgba(0,0,0,0.3); padding: 2px 6px; border-radius: 4px;">rag.enabled: true</code> in your config file and restart the server.
                        </p>
                    </div>
                </div>
            </div>
        `;
    }

    setupEventListeners() {
        // Refresh button
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.loadSources();
        });

        // Apply filters
        document.getElementById('apply-filters-btn').addEventListener('click', () => {
            this.applyFilters();
        });

        // Pagination
        document.getElementById('prev-page-btn').addEventListener('click', () => {
            if (this.currentPage > 1) {
                this.currentPage--;
                this.loadSources();
            }
        });

        document.getElementById('next-page-btn').addEventListener('click', () => {
            const totalPages = Math.ceil(this.totalSources / this.pageSize);
            if (this.currentPage < totalPages) {
                this.currentPage++;
                this.loadSources();
            }
        });

        // Details modal
        document.getElementById('close-details-modal').addEventListener('click', () => {
            this.closeDetailsModal();
        });

        document.getElementById('close-details-btn').addEventListener('click', () => {
            this.closeDetailsModal();
        });

        // Delete modal
        document.getElementById('close-delete-modal').addEventListener('click', () => {
            this.closeDeleteModal();
        });

        document.getElementById('cancel-delete-btn').addEventListener('click', () => {
            this.closeDeleteModal();
        });

        document.getElementById('confirm-delete-btn').addEventListener('click', () => {
            this.confirmDelete();
        });

        // Close modals on outside click
        window.addEventListener('click', (e) => {
            const detailsModal = document.getElementById('details-modal');
            const deleteModal = document.getElementById('delete-modal');
            if (e.target === detailsModal) {
                this.closeDetailsModal();
            }
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            }
        });
    }

    applyFilters() {
        this.filters = {
            type: document.getElementById('filter-type').value,
            status: document.getElementById('filter-status').value,
            user: document.getElementById('filter-user').value
        };
        this.currentPage = 1;
        this.loadSources();
    }

    async loadSources() {
        try {
            const params = {
                limit: this.pageSize,
                offset: (this.currentPage - 1) * this.pageSize
            };

            if (this.filters.type) params.source_type = this.filters.type;
            if (this.filters.status) params.status = this.filters.status;

            const response = await api.getRAGSources(params);
            this.sources = response.sources || [];
            this.totalSources = response.total || 0;

            // Apply user filter on frontend (if backend doesn't support it)
            if (this.filters.user) {
                this.sources = this.sources.filter(s => 
                    s.user_id && s.user_id.toLowerCase().includes(this.filters.user.toLowerCase())
                );
            }

            this.renderSources();
            this.updatePagination();
            this.updateStats();
        } catch (error) {
            console.error('Failed to load sources:', error);
            this.showError('Failed to load RAG sources: ' + error.message);
            this.renderSources(); // Show empty state
        }
    }

    renderSources() {
        const tbody = document.getElementById('sources-table-body');
        tbody.innerHTML = '';

        if (this.sources.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" class="table-empty">No RAG sources found.</td></tr>';
            return;
        }

        this.sources.forEach(source => {
            const row = document.createElement('tr');
            
            const statusClass = source.status === 'active' ? 'status-success' : 
                              source.status === 'error' ? 'status-error' : 
                              source.status === 'syncing' ? 'status-warning' :
                              'status-inactive';
            
            const lastSync = source.last_sync_at ? 
                new Date(source.last_sync_at).toLocaleString() : 
                'Never';

            const shortId = source.id ? source.id.substring(0, 8) : 'N/A';
            const userName = source.user_id ? source.user_id.substring(0, 8) : 'System';

            row.innerHTML = `
                <td><code>${this.escapeHtml(shortId)}...</code></td>
                <td>
                    <strong>${this.escapeHtml(source.name)}</strong>
                    ${source.description ? `<br><small>${this.escapeHtml(source.description.substring(0, 50))}${source.description.length > 50 ? '...' : ''}</small>` : ''}
                </td>
                <td><code>${this.escapeHtml(userName)}</code></td>
                <td><span class="badge">${this.escapeHtml(source.source_type)}</span></td>
                <td><span class="badge ${statusClass}">${this.escapeHtml(source.status)}</span></td>
                <td>${lastSync}</td>
                <td>${source.total_chunks || 0}</td>
                <td>${this.formatTokens(source.total_tokens || 0)}</td>
                <td>${source.is_shared ? '✅ Yes' : '❌ No'}</td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick="adminRAGManager.viewDetails('${source.id}')" title="View Details">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" stroke-width="2"/>
                            <circle cx="12" cy="12" r="3" stroke-width="2"/>
                        </svg>
                    </button>
                    <button class="btn btn-sm btn-primary" onclick="adminRAGManager.syncSource('${source.id}')" title="Sync">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" stroke-width="2"/>
                        </svg>
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="adminRAGManager.deleteSource('${source.id}')" title="Delete">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                        </svg>
                    </button>
                </td>
            `;

            tbody.appendChild(row);
        });
    }

    updatePagination() {
        const totalPages = Math.ceil(this.totalSources / this.pageSize);
        const pagination = document.getElementById('pagination');
        const prevBtn = document.getElementById('prev-page-btn');
        const nextBtn = document.getElementById('next-page-btn');
        const pageInfo = document.getElementById('page-info');

        if (totalPages <= 1) {
            pagination.style.display = 'none';
            return;
        }

        pagination.style.display = 'flex';
        pageInfo.textContent = `Page ${this.currentPage} of ${totalPages}`;
        prevBtn.disabled = this.currentPage === 1;
        nextBtn.disabled = this.currentPage === totalPages;
    }

    updateStats() {
        const totalSources = this.sources.length;
        const activeSources = this.sources.filter(s => s.status === 'active').length;
        const errorSources = this.sources.filter(s => s.status === 'error').length;
        const totalChunks = this.sources.reduce((sum, s) => sum + (s.total_chunks || 0), 0);

        document.getElementById('stat-total-sources').textContent = totalSources;
        document.getElementById('stat-active-sources').textContent = activeSources;
        document.getElementById('stat-error-sources').textContent = errorSources;
        document.getElementById('stat-total-chunks').textContent = this.formatNumber(totalChunks);
    }

    async viewDetails(sourceId) {
        try {
            const source = await api.getRAGSource(sourceId);
            
            const detailsHtml = `
                <div style="display: grid; gap: 16px;">
                    <div>
                        <strong>ID:</strong> <code>${this.escapeHtml(source.id)}</code>
                    </div>
                    <div>
                        <strong>Name:</strong> ${this.escapeHtml(source.name)}
                    </div>
                    <div>
                        <strong>Description:</strong> ${this.escapeHtml(source.description || 'N/A')}
                    </div>
                    <div>
                        <strong>Type:</strong> <span class="badge">${this.escapeHtml(source.source_type)}</span>
                    </div>
                    <div>
                        <strong>Status:</strong> <span class="badge">${this.escapeHtml(source.status)}</span>
                    </div>
                    <div>
                        <strong>User ID:</strong> <code>${this.escapeHtml(source.user_id || 'N/A')}</code>
                    </div>
                    <div>
                        <strong>Tenant ID:</strong> <code>${this.escapeHtml(source.tenant_id || 'N/A')}</code>
                    </div>
                    <div>
                        <strong>Shared:</strong> ${source.is_shared ? '✅ Yes' : '❌ No'}
                    </div>
                    <div>
                        <strong>Total Chunks:</strong> ${source.total_chunks || 0}
                    </div>
                    <div>
                        <strong>Total Tokens:</strong> ${this.formatNumber(source.total_tokens || 0)}
                    </div>
                    <div>
                        <strong>Last Sync:</strong> ${source.last_sync_at ? new Date(source.last_sync_at).toLocaleString() : 'Never'}
                    </div>
                    <div>
                        <strong>Last Sync Status:</strong> ${source.last_sync_status || 'N/A'}
                    </div>
                    <div>
                        <strong>Created:</strong> ${new Date(source.created_at).toLocaleString()}
                    </div>
                    <div>
                        <strong>Updated:</strong> ${new Date(source.updated_at).toLocaleString()}
                    </div>
                    ${source.last_error ? `
                    <div style="background: rgba(239, 68, 68, 0.1); padding: 12px; border-radius: 4px;">
                        <strong style="color: #ef4444;">Last Error:</strong><br>
                        <code style="color: #ef4444;">${this.escapeHtml(source.last_error)}</code>
                    </div>
                    ` : ''}
                    <div>
                        <strong>Configuration:</strong>
                        <pre style="background: var(--bg-secondary); padding: 12px; border-radius: 4px; overflow-x: auto;">${JSON.stringify(source.config, null, 2)}</pre>
                    </div>
                    ${source.indexing_config ? `
                    <div>
                        <strong>Indexing Config:</strong>
                        <pre style="background: var(--bg-secondary); padding: 12px; border-radius: 4px; overflow-x: auto;">${JSON.stringify(source.indexing_config, null, 2)}</pre>
                    </div>
                    ` : ''}
                </div>
            `;

            document.getElementById('source-details-content').innerHTML = detailsHtml;
            document.getElementById('details-modal').style.display = 'flex';
        } catch (error) {
            console.error('Failed to load source details:', error);
            this.showError('Failed to load source details: ' + error.message);
        }
    }

    closeDetailsModal() {
        document.getElementById('details-modal').style.display = 'none';
    }

    async syncSource(sourceId) {
        try {
            await api.syncRAGSource(sourceId);
            this.showSuccess('Sync started successfully');
            await this.loadSources();
        } catch (error) {
            console.error('Failed to sync source:', error);
            this.showError('Failed to sync source: ' + error.message);
        }
    }

    deleteSource(sourceId) {
        this.currentSourceId = sourceId;
        const source = this.sources.find(s => s.id === sourceId);
        if (source) {
            document.getElementById('delete-confirm-text').textContent = 
                `Are you sure you want to delete "${source.name}"? This will remove all associated documents and chunks. This action cannot be undone.`;
            document.getElementById('delete-modal').style.display = 'flex';
        }
    }

    closeDeleteModal() {
        document.getElementById('delete-modal').style.display = 'none';
        this.currentSourceId = null;
    }

    async confirmDelete() {
        if (!this.currentSourceId) return;

        try {
            await api.deleteRAGSource(this.currentSourceId);
            this.showSuccess('RAG source deleted successfully');
            this.closeDeleteModal();
            await this.loadSources();
        } catch (error) {
            console.error('Failed to delete source:', error);
            this.showError('Failed to delete source: ' + error.message);
        }
    }

    showError(message) {
        const errorEl = document.getElementById('admin-rag-message');
        errorEl.textContent = message;
        errorEl.style.display = 'block';
        setTimeout(() => {
            errorEl.style.display = 'none';
        }, 5000);
    }

    showSuccess(message) {
        const successEl = document.getElementById('admin-rag-success');
        successEl.textContent = message;
        successEl.style.display = 'block';
        setTimeout(() => {
            successEl.style.display = 'none';
        }, 3000);
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatTokens(tokens) {
        return this.formatNumber(tokens);
    }

    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    }
}

// Initialize
const adminRAGManager = new AdminRAGManager();

// Wait for DOM to be ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        adminRAGManager.init();
    });
} else {
    adminRAGManager.init();
}



