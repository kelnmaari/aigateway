// API Keys Management Page
// Version 1.3.0

class APIKeysManager {
    constructor() {
        this.currentUser = null;
        this.tenants = [];
        this.models = [];
        this.personalKeys = [];
        this.tenantKeys = [];
        this.currentTenant = null;
        this.activeTab = 'personal';
        
        this.init();
    }

    async init() {
        try {
            // Load current user
            await this.loadUser();
            
            // Load tenants
            await this.loadTenants();
            
            // Load models
            await this.loadModels();
            
            // Load personal keys
            await this.loadPersonalKeys();
            
            // Setup UI
            this.setupEventListeners();
            
        } catch (error) {
            console.error('Failed to initialize API keys page:', error);
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
            
            // Populate tenant selectors
            this.populateTenantSelectors();
            
        } catch (error) {
            console.error('Failed to load tenants:', error);
            this.tenants = [];
        }
    }

    // Load models
    async loadModels() {
        try {
            this.models = await api.getModels();
        } catch (error) {
            console.error('Failed to load models:', error);
            this.models = [];
        }
    }

    // Load personal API keys
    async loadPersonalKeys() {
        try {
            const response = await api.getPersonalAPIKeys();
            this.personalKeys = response.api_keys || [];
            
            this.renderPersonalKeys();
            
        } catch (error) {
            console.error('Failed to load personal keys:', error);
            this.showError('Failed to load personal API keys');
            document.getElementById('personal-keys-list').innerHTML = 
                '<tr><td colspan="6" class="table-empty error">Failed to load API keys</td></tr>';
        }
    }

    // Load tenant API keys
    async loadTenantKeys(tenantId) {
        try {
            const response = await api.getTenantAPIKeys(tenantId);
            this.tenantKeys = response.api_keys || [];
            
            this.renderTenantKeys();
            
        } catch (error) {
            console.error('Failed to load tenant keys:', error);
            this.showError('Failed to load organization API keys');
            document.getElementById('tenant-keys-list').innerHTML = 
                '<tr><td colspan="6" class="table-empty error">Failed to load API keys</td></tr>';
        }
    }

    // Populate tenant selectors
    populateTenantSelectors() {
        const selectors = [
            document.getElementById('tenant-selector'),
            document.getElementById('key-tenant')
        ];

        // Filter out personal tenants
        const orgTenants = this.tenants.filter(t => t.type !== 'personal');

        selectors.forEach(selector => {
            if (!selector) return;
            
            // Clear existing options except first
            while (selector.options.length > 1) {
                selector.remove(1);
            }
            
            // Add tenant options
            orgTenants.forEach(tenant => {
                const option = document.createElement('option');
                option.value = tenant.id;
                option.textContent = tenant.name;
                selector.appendChild(option);
            });
        });
    }

    // Render personal keys
    renderPersonalKeys() {
        const tbody = document.getElementById('personal-keys-list');
        
        if (this.personalKeys.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" class="table-empty">
                        No personal API keys yet. Create your first key to get started.
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = this.personalKeys.map(key => this.renderKeyRow(key, 'personal')).join('');
    }

    // Render tenant keys
    renderTenantKeys() {
        const tbody = document.getElementById('tenant-keys-list');
        
        if (!this.currentTenant) {
            tbody.innerHTML = '<tr><td colspan="6" class="table-empty">Select an organization to view API keys</td></tr>';
            return;
        }

        if (this.tenantKeys.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" class="table-empty">
                        No API keys for this organization yet.
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = this.tenantKeys.map(key => this.renderKeyRow(key, 'tenant')).join('');
    }

    // Render single key row
    renderKeyRow(key, type) {
        const maskedKey = this.maskKey(key.key_hash || key.key);
        const lastUsed = key.last_used_at ? this.formatDate(key.last_used_at) : 'Never';
        
        // Check status field (backend returns "active", "disabled", "expired", "revoked")
        const isActive = key.status === 'active' && (!key.expires_at || new Date(key.expires_at) > new Date());
        const statusBadge = isActive 
            ? '<span class="badge badge-success">Active</span>'
            : '<span class="badge badge-secondary">Inactive</span>';

        return `
            <tr>
                <td>
                    <div style="display: flex; flex-direction: column;">
                        <strong>${this.escapeHtml(key.name)}</strong>
                        ${key.description ? `<small style="color: var(--text-secondary);">${this.escapeHtml(key.description)}</small>` : ''}
                    </div>
                </td>
                <td>
                    <code class="key-display">${maskedKey}</code>
                </td>
                <td>${this.formatDate(key.created_at)}</td>
                <td>${lastUsed}</td>
                <td>${statusBadge}</td>
                <td>
                    <button 
                        class="btn btn-sm btn-secondary" 
                        onclick="apiKeysManager.deleteKey('${key.id}', '${type}')"
                        title="Delete key"
                    >
                        🗑️
                    </button>
                </td>
            </tr>
        `;
    }

    // Mask API key for display
    maskKey(key) {
        if (!key) return '••••••••••••';
        if (key.length <= 8) return '••••••••';
        return key.substring(0, 7) + '•'.repeat(20) + key.substring(key.length - 4);
    }

    // Setup event listeners
    setupEventListeners() {
        // User dropdown and logout are now handled by navbar.js component
        // No need for duplicate event listeners here

        // Tabs
        const tabBtns = document.querySelectorAll('.tab-btn');
        tabBtns.forEach(btn => {
            btn.addEventListener('click', () => this.switchTab(btn.dataset.tab));
        });

        // Tenant selector
        const tenantSelector = document.getElementById('tenant-selector');
        if (tenantSelector) {
            tenantSelector.addEventListener('change', (e) => {
                this.currentTenant = e.target.value;
                if (this.currentTenant) {
                    this.loadTenantKeys(this.currentTenant);
                } else {
                    document.getElementById('tenant-keys-list').innerHTML = 
                        '<tr><td colspan="6" class="table-empty">Select an organization to view API keys</td></tr>';
                }
            });
        }

        // Create key modal
        const createBtn = document.getElementById('create-key-btn');
        const createModal = document.getElementById('create-key-modal');
        const cancelBtn = document.getElementById('cancel-create-btn');
        const modalClose = document.getElementById('modal-close');
        const modalOverlay = document.getElementById('modal-overlay');
        const createForm = document.getElementById('create-key-form');

        if (createBtn) {
            createBtn.addEventListener('click', () => this.showCreateModal());
        }

        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => this.hideCreateModal());
        }

        if (modalClose) {
            modalClose.addEventListener('click', () => this.hideCreateModal());
        }

        if (modalOverlay) {
            modalOverlay.addEventListener('click', () => this.hideCreateModal());
        }

        if (createForm) {
            createForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.createKey();
            });
        }

        // Key type selector
        const keyTypeSelect = document.getElementById('key-type');
        if (keyTypeSelect) {
            keyTypeSelect.addEventListener('change', (e) => {
                const tenantGroup = document.getElementById('tenant-select-group');
                if (tenantGroup) {
                    tenantGroup.style.display = e.target.value === 'tenant' ? 'block' : 'none';
                }
            });
        }

        // All models checkbox
        const allModelsCheckbox = document.getElementById('key-all-models');
        const modelsListGroup = document.getElementById('models-list-group');
        
        if (allModelsCheckbox && modelsListGroup) {
            allModelsCheckbox.addEventListener('change', (e) => {
                modelsListGroup.style.display = e.target.checked ? 'none' : 'block';
            });
        }

        // Key created modal
        const createdModalClose = document.getElementById('created-modal-close');
        const createdModalOverlay = document.getElementById('created-modal-overlay');
        const closeCreatedBtn = document.getElementById('close-created-modal-btn');
        const copyKeyBtn = document.getElementById('copy-key-btn');

        if (createdModalClose) {
            createdModalClose.addEventListener('click', () => this.hideCreatedModal());
        }

        if (createdModalOverlay) {
            createdModalOverlay.addEventListener('click', () => this.hideCreatedModal());
        }

        if (closeCreatedBtn) {
            closeCreatedBtn.addEventListener('click', () => this.hideCreatedModal());
        }

        if (copyKeyBtn) {
            copyKeyBtn.addEventListener('click', () => this.copyKeyToClipboard());
        }
    }

    // Switch tabs
    switchTab(tab) {
        this.activeTab = tab;

        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tab}-content`);
        });
    }

    // Show create modal
    showCreateModal() {
        const modal = document.getElementById('create-key-modal');
        if (modal) {
            // Set key type based on active tab
            const keyTypeSelect = document.getElementById('key-type');
            if (keyTypeSelect) {
                keyTypeSelect.value = this.activeTab === 'tenant' ? 'tenant' : 'personal';
                keyTypeSelect.dispatchEvent(new Event('change'));
            }

            // Populate models checkboxes
            this.populateModelsCheckboxes();

            modal.style.display = 'flex';
            document.getElementById('key-name').focus();
        }
    }

    // Hide create modal
    hideCreateModal() {
        const modal = document.getElementById('create-key-modal');
        if (modal) {
            modal.style.display = 'none';
            document.getElementById('create-key-form').reset();
            document.getElementById('key-all-models').checked = true;
            document.getElementById('models-list-group').style.display = 'none';
        }
    }

    // Populate models checkboxes
    populateModelsCheckboxes() {
        const container = document.getElementById('models-checkboxes');
        if (!container) return;

        if (this.models.length === 0) {
            container.innerHTML = '<p class="text-secondary">No models available</p>';
            return;
        }

        container.innerHTML = this.models.map(model => `
            <label class="checkbox-label">
                <input type="checkbox" name="models" value="${model.id}">
                ${this.escapeHtml(model.id)}
            </label>
        `).join('');
    }

    // Create API key
    async createKey() {
        const submitBtn = document.getElementById('submit-create-btn');
        submitBtn.disabled = true;
        submitBtn.textContent = 'Creating...';

        try {
            const keyType = document.getElementById('key-type').value;
            const name = document.getElementById('key-name').value.trim();
            const description = document.getElementById('key-description').value.trim();
            const rpm = document.getElementById('key-rpm').value;
            const rph = document.getElementById('key-rph').value;
            const allModels = document.getElementById('key-all-models').checked;

            // Build request data
            const data = { name };
            
            if (description) {
                data.description = description;
            }

            // Rate limits
            if (rpm || rph) {
                data.rate_limits = {};
                if (rpm) data.rate_limits.requests_per_minute = parseInt(rpm);
                if (rph) data.rate_limits.requests_per_hour = parseInt(rph);
            }

            // Model permissions
            if (!allModels) {
                const selectedModels = Array.from(
                    document.querySelectorAll('input[name="models"]:checked')
                ).map(cb => cb.value);
                
                if (selectedModels.length > 0) {
                    data.models = selectedModels;
                }
            }

            // Create key
            let result;
            if (keyType === 'personal') {
                result = await api.createPersonalAPIKey(data);
                await this.loadPersonalKeys();
                this.switchTab('personal');
            } else {
                const tenantId = document.getElementById('key-tenant').value;
                if (!tenantId) {
                    throw new Error('Please select an organization');
                }
                result = await api.createTenantAPIKey(tenantId, data);
                await this.loadTenantKeys(tenantId);
                this.switchTab('tenant');
                document.getElementById('tenant-selector').value = tenantId;
            }

            this.hideCreateModal();
            this.showCreatedModal(result.key, name); // Use plaintext key, not api_key object

        } catch (error) {
            console.error('Failed to create API key:', error);
            this.showError(error.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Create Key';
        }
    }

    // Show key created modal
    showCreatedModal(apiKey, keyName) {
        const modal = document.getElementById('key-created-modal');
        if (modal) {
            document.getElementById('created-key-value').value = apiKey;
            document.getElementById('created-key-name').textContent = keyName;
            modal.style.display = 'flex';
        }
    }

    // Hide key created modal
    hideCreatedModal() {
        const modal = document.getElementById('key-created-modal');
        if (modal) {
            modal.style.display = 'none';
        }
    }

    // Copy key to clipboard
    async copyKeyToClipboard() {
        const keyInput = document.getElementById('created-key-value');
        if (!keyInput) return;

        try {
            await navigator.clipboard.writeText(keyInput.value);
            
            const btn = document.getElementById('copy-key-btn');
            const originalText = btn.textContent;
            btn.textContent = '✅ Copied!';
            
            setTimeout(() => {
                btn.textContent = originalText;
            }, 2000);
            
        } catch (error) {
            console.error('Failed to copy:', error);
            // Fallback
            keyInput.select();
            document.execCommand('copy');
            this.showSuccess('Copied to clipboard');
        }
    }

    // Delete API key
    async deleteKey(keyId, type) {
        if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
            return;
        }

        try {
            if (type === 'personal') {
                await api.deletePersonalAPIKey(keyId);
                await this.loadPersonalKeys();
            } else {
                if (!this.currentTenant) {
                    throw new Error('No organization selected');
                }
                await api.deleteTenantAPIKey(this.currentTenant, keyId);
                await this.loadTenantKeys(this.currentTenant);
            }

            this.showSuccess('API key deleted successfully');

        } catch (error) {
            console.error('Failed to delete API key:', error);
            this.showError(error.message);
        }
    }

    // Show error message
    showError(message) {
        const errorDiv = document.getElementById('apikey-message');
        if (errorDiv) {
            errorDiv.textContent = message;
            errorDiv.style.display = 'block';
            errorDiv.className = 'error-message';
            
            setTimeout(() => {
                errorDiv.style.display = 'none';
            }, 5000);
        }
    }

    // Show success message
    showSuccess(message) {
        const successDiv = document.getElementById('apikey-success');
        if (successDiv) {
            successDiv.textContent = message;
            successDiv.style.display = 'block';
            successDiv.className = 'success-message';
            
            setTimeout(() => {
                successDiv.style.display = 'none';
            }, 3000);
        }
    }

    // Format date
    formatDate(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', { 
            year: 'numeric', 
            month: 'short', 
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
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
let apiKeysManager;
document.addEventListener('DOMContentLoaded', () => {
    apiKeysManager = new APIKeysManager();
});

