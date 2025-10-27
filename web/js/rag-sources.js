// RAG Data Sources Management
class RAGSourcesManager {
    constructor() {
        this.sources = [];
        this.currentSource = null;
        this.editMode = false;
    }

    async init() {
        // Check if RAG is enabled
        const ragEnabled = await api.isRAGEnabled();
        if (!ragEnabled) {
            this.showRAGDisabled();
            return;
        }

        this.setupEventListeners();
        await this.loadSources();
    }

    showRAGDisabled() {
        const container = document.querySelector('.dashboard-container');
        container.innerHTML = `
            <header class="page-header">
                <div>
                    <h1>RAG Data Sources</h1>
                    <p class="page-description">Manage your knowledge base sources</p>
                </div>
                <div style="display: flex; gap: 12px;">
                    <a href="/dashboard.html" class="btn btn-secondary">Back to Dashboard</a>
                </div>
            </header>
            
            <div style="background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 12px; padding: 48px 24px; min-height: calc(100vh - 280px); display: flex; align-items: center; justify-content: center;">
                <div style="text-align: center; max-width: 600px;">
                    <div style="font-size: 64px; margin-bottom: 24px;">🔒</div>
                    <h2 style="color: rgba(255, 255, 255, 0.9); margin-bottom: 16px;">RAG System Disabled</h2>
                    <p style="color: rgba(255, 255, 255, 0.6); font-size: 16px; line-height: 1.6; margin-bottom: 24px;">
                        The RAG (Retrieval-Augmented Generation) system is currently disabled by the administrator.
                        <br><br>
                        Contact your system administrator to enable RAG functionality.
                    </p>
                    <div style="background: rgba(255, 193, 7, 0.1); border: 1px solid rgba(255, 193, 7, 0.3); border-radius: 8px; padding: 16px; margin-top: 24px;">
                        <p style="color: rgba(255, 193, 7, 0.9); font-size: 14px; margin: 0;">
                            <strong>Administrator:</strong> Enable RAG in the server configuration file (<code style="background: rgba(0,0,0,0.3); padding: 2px 6px; border-radius: 4px;">rag.enabled: true</code>)
                        </p>
                    </div>
                </div>
            </div>
        `;
    }

    setupEventListeners() {
        // Create button
        document.getElementById('create-source-btn').addEventListener('click', () => {
            this.openModal();
        });

        // Modal controls
        document.getElementById('close-modal').addEventListener('click', () => {
            this.closeModal();
        });

        document.getElementById('cancel-btn').addEventListener('click', () => {
            this.closeModal();
        });

        // Form submission
        document.getElementById('source-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveSource();
        });

        // Test connection
        document.getElementById('test-connection-btn').addEventListener('click', () => {
            this.testConnection();
        });

        // Source type change
        document.getElementById('source-type').addEventListener('change', (e) => {
            this.handleSourceTypeChange(e.target.value);
        });

        // Auth type change
        document.getElementById('api-auth-type').addEventListener('change', (e) => {
            this.handleAuthTypeChange(e.target.value);
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
            const sourceModal = document.getElementById('source-modal');
            const deleteModal = document.getElementById('delete-modal');
            if (e.target === sourceModal) {
                this.closeModal();
            }
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            }
        });
    }

    async loadSources() {
        try {
            const response = await api.getRAGSources();
            this.sources = response.sources || [];
            this.renderSources();
        } catch (error) {
            console.error('Failed to load sources:', error);
            this.showError('Failed to load data sources: ' + error.message);
            this.renderSources(); // Show empty state
        }
    }

    renderSources() {
        const tbody = document.getElementById('sources-list');
        tbody.innerHTML = '';

        if (this.sources.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="table-empty">No data sources yet. Click "Add Data Source" to get started.</td></tr>';
            return;
        }

        this.sources.forEach(source => {
            const row = document.createElement('tr');
            
            const statusClass = source.status === 'active' ? 'status-success' : 
                              source.status === 'error' ? 'status-error' : 
                              'status-warning';
            
            const lastSync = source.last_sync_at ? 
                new Date(source.last_sync_at).toLocaleString() : 
                'Never';

            row.innerHTML = `
                <td>
                    <strong>${this.escapeHtml(source.name)}</strong>
                    ${source.description ? `<br><small>${this.escapeHtml(source.description)}</small>` : ''}
                </td>
                <td><span class="badge">${this.escapeHtml(source.source_type)}</span></td>
                <td><span class="badge ${statusClass}">${this.escapeHtml(source.status)}</span></td>
                <td>${lastSync}</td>
                <td>${source.total_chunks || 0}</td>
                <td>${this.formatTokens(source.total_tokens || 0)}</td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick="ragSourcesManager.syncSource('${source.id}')">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" stroke-width="2"/>
                        </svg>
                    </button>
                    <button class="btn btn-sm btn-primary" onclick="ragSourcesManager.editSource('${source.id}')">
                        Edit
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="ragSourcesManager.deleteSource('${source.id}')">
                        Delete
                    </button>
                </td>
            `;

            tbody.appendChild(row);
        });
    }

    openModal(source = null) {
        this.editMode = !!source;
        this.currentSource = source;

        const modal = document.getElementById('source-modal');
        const title = document.getElementById('modal-title');
        const form = document.getElementById('source-form');

        title.textContent = this.editMode ? 'Edit Data Source' : 'Add Data Source';
        form.reset();

        if (this.editMode && source) {
            document.getElementById('source-name').value = source.name || '';
            document.getElementById('source-description').value = source.description || '';
            document.getElementById('source-type').value = source.source_type || '';
            this.handleSourceTypeChange(source.source_type);

            // Load config based on type
            if (source.config) {
                this.loadConfig(source.source_type, source.config);
            }

            if (source.indexing_config) {
                document.getElementById('chunk-size').value = source.indexing_config.chunk_size || 512;
                document.getElementById('chunk-overlap').value = source.indexing_config.chunk_overlap || 50;
            }

            document.getElementById('source-shared').checked = source.is_shared || false;
        }

        modal.style.display = 'flex';
    }

    closeModal() {
        document.getElementById('source-modal').style.display = 'none';
        document.getElementById('source-form').reset();
        this.currentSource = null;
        this.editMode = false;
    }

    handleSourceTypeChange(type) {
        // Hide all config sections
        document.querySelectorAll('.config-section').forEach(el => {
            el.style.display = 'none';
        });

        // Show relevant config section
        const configMap = {
            'api': 'api-config',
            'database': 'database-config',
            'file': 'file-config',
            'web': 'web-config'
        };

        const configId = configMap[type];
        if (configId) {
            document.getElementById(configId).style.display = 'block';
        }
    }

    handleAuthTypeChange(authType) {
        const tokenGroup = document.getElementById('api-token-group');
        const basicGroup = document.getElementById('api-basic-group');

        tokenGroup.style.display = 'none';
        basicGroup.style.display = 'none';

        if (authType === 'bearer' || authType === 'api_key') {
            tokenGroup.style.display = 'block';
        } else if (authType === 'basic') {
            basicGroup.style.display = 'block';
        }
    }

    loadConfig(sourceType, config) {
        if (sourceType === 'api' && config.url) {
            document.getElementById('api-url').value = config.url;
            document.getElementById('api-method').value = config.method || 'GET';
            document.getElementById('api-auth-type').value = config.auth_type || 'none';
            this.handleAuthTypeChange(config.auth_type || 'none');
        } else if (sourceType === 'database') {
            document.getElementById('db-host').value = config.host || '';
            document.getElementById('db-port').value = config.port || 5432;
            document.getElementById('db-name').value = config.database || '';
            document.getElementById('db-username').value = config.username || '';
            document.getElementById('db-query').value = config.query || '';
        } else if (sourceType === 'web' && config.url) {
            document.getElementById('web-url').value = config.url;
            document.getElementById('web-depth').value = config.max_depth || 1;
        }
    }

    async saveSource() {
        const sourceType = document.getElementById('source-type').value;
        if (!sourceType) {
            this.showError('Please select a source type');
            return;
        }

        const data = {
            name: document.getElementById('source-name').value,
            description: document.getElementById('source-description').value,
            source_type: sourceType,
            config: this.buildConfig(sourceType),
            indexing_config: {
                chunk_size: parseInt(document.getElementById('chunk-size').value),
                chunk_overlap: parseInt(document.getElementById('chunk-overlap').value),
                splitter_type: 'semantic'
            },
            is_shared: document.getElementById('source-shared').checked
        };

        // Build credentials
        const credentials = this.buildCredentials(sourceType);
        if (Object.keys(credentials).length > 0) {
            data.credentials = credentials;
        }

        try {
            if (this.editMode && this.currentSource) {
                await api.updateRAGSource(this.currentSource.id, data);
                this.showSuccess('Data source updated successfully');
            } else {
                await api.createRAGSource(data);
                this.showSuccess('Data source created successfully');
            }

            this.closeModal();
            await this.loadSources();
        } catch (error) {
            console.error('Failed to save source:', error);
            this.showError('Failed to save data source: ' + error.message);
        }
    }

    buildConfig(sourceType) {
        const config = {};

        if (sourceType === 'api') {
            config.url = document.getElementById('api-url').value;
            config.method = document.getElementById('api-method').value;
            config.auth_type = document.getElementById('api-auth-type').value;
            config.response_field = 'data'; // default
        } else if (sourceType === 'database') {
            config.host = document.getElementById('db-host').value;
            config.port = parseInt(document.getElementById('db-port').value);
            config.database = document.getElementById('db-name').value;
            config.username = document.getElementById('db-username').value;
            config.query = document.getElementById('db-query').value;
        } else if (sourceType === 'web') {
            config.url = document.getElementById('web-url').value;
            config.max_depth = parseInt(document.getElementById('web-depth').value);
        }

        return config;
    }

    buildCredentials(sourceType) {
        const credentials = {};

        if (sourceType === 'api') {
            const authType = document.getElementById('api-auth-type').value;
            if (authType === 'bearer' || authType === 'api_key') {
                const token = document.getElementById('api-token').value;
                if (token) {
                    credentials.token = token;
                }
            } else if (authType === 'basic') {
                credentials.username = document.getElementById('api-username').value;
                credentials.password = document.getElementById('api-password').value;
            }
        } else if (sourceType === 'database') {
            credentials.password = document.getElementById('db-password').value;
        }

        return credentials;
    }

    async testConnection() {
        const sourceType = document.getElementById('source-type').value;
        if (!sourceType) {
            this.showError('Please select a source type first');
            return;
        }

        const data = {
            source_type: sourceType,
            config: this.buildConfig(sourceType),
            credentials: this.buildCredentials(sourceType)
        };

        try {
            const result = await api.testRAGConnection(data);
            if (result.success) {
                this.showSuccess('Connection test successful!');
            } else {
                this.showError('Connection test failed: ' + (result.error || 'Unknown error'));
            }
        } catch (error) {
            console.error('Connection test failed:', error);
            this.showError('Connection test failed: ' + error.message);
        }
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

    async editSource(sourceId) {
        const source = this.sources.find(s => s.id === sourceId);
        if (source) {
            this.openModal(source);
        }
    }

    deleteSource(sourceId) {
        this.currentSource = this.sources.find(s => s.id === sourceId);
        if (this.currentSource) {
            document.getElementById('delete-modal').style.display = 'flex';
        }
    }

    closeDeleteModal() {
        document.getElementById('delete-modal').style.display = 'none';
        this.currentSource = null;
    }

    async confirmDelete() {
        if (!this.currentSource) return;

        try {
            await api.deleteRAGSource(this.currentSource.id);
            this.showSuccess('Data source deleted successfully');
            this.closeDeleteModal();
            await this.loadSources();
        } catch (error) {
            console.error('Failed to delete source:', error);
            this.showError('Failed to delete source: ' + error.message);
        }
    }

    showError(message) {
        const errorEl = document.getElementById('rag-message');
        errorEl.textContent = message;
        errorEl.style.display = 'block';
        setTimeout(() => {
            errorEl.style.display = 'none';
        }, 5000);
    }

    showSuccess(message) {
        const successEl = document.getElementById('rag-success');
        successEl.textContent = message;
        successEl.style.display = 'block';
        setTimeout(() => {
            successEl.style.display = 'none';
        }, 3000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatTokens(tokens) {
        if (tokens >= 1000000) {
            return (tokens / 1000000).toFixed(1) + 'M';
        } else if (tokens >= 1000) {
            return (tokens / 1000).toFixed(1) + 'K';
        }
        return tokens.toString();
    }
}

// Initialize
const ragSourcesManager = new RAGSourcesManager();

// Wait for DOM to be ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        ragSourcesManager.init();
    });
} else {
    ragSourcesManager.init();
}

