// admin-registry.js - Model Registry Admin UI

let providers = [];
let models = [];
let stats = {};

// Initialize page
document.addEventListener('DOMContentLoaded', async () => {
    await loadRegistryData();
    
    // Auto-refresh every 30 seconds
    setInterval(refreshProviders, 30000);
    setInterval(refreshStats, 30000);
});

// Load all registry data
async function loadRegistryData() {
    await Promise.all([
        loadStats(),
        loadProviders(),
        loadModels()
    ]);
}

// Load statistics
async function loadStats() {
    try {
        const response = await api.get('/api/admin/registry/stats');
        stats = response;
        renderStats();
    } catch (error) {
        console.error('Failed to load stats:', error);
        toast.error('Failed to load statistics');
    }
}

// Render statistics
function renderStats() {
    document.getElementById('stat-total-models').textContent = stats.total_models || 0;
    document.getElementById('stat-active-models').textContent = stats.active_models || 0;
    document.getElementById('stat-providers').textContent = stats.total_providers || 0;
    document.getElementById('stat-healthy-providers').textContent = stats.enabled_providers || 0;
}

// Load providers
async function loadProviders() {
    try {
        const response = await api.get('/api/admin/registry/providers');
        providers = response.providers || [];
        renderProviders();
        updateProviderFilters();
    } catch (error) {
        console.error('Failed to load providers:', error);
        toast.error('Failed to load providers');
        document.getElementById('providers-container').innerHTML = 
            '<p style="color: var(--error-color);">Failed to load providers. Please try again.</p>';
    }
}

// Render providers
function renderProviders() {
    const container = document.getElementById('providers-container');
    
    if (providers.length === 0) {
        container.innerHTML = `
            <div style="grid-column: 1 / -1; text-align: center; padding: 40px;">
                <p style="color: var(--text-secondary); font-size: 1.1rem;">No providers configured yet.</p>
                <button class="btn btn-primary" onclick="showCreateProviderModal()" style="margin-top: 15px;">
                    ➕ Add Your First Provider
                </button>
            </div>
        `;
        return;
    }
    
    container.innerHTML = providers.map(provider => `
        <div class="provider-card">
            <div class="provider-header">
                <div>
                    <div class="provider-name">${escapeHtml(provider.name)}</div>
                    <span class="provider-type ${provider.provider_type}">${provider.provider_type}</span>
                </div>
            </div>
            
            <div class="provider-status">
                <span class="status-indicator ${provider.health_status}"></span>
                <span style="font-weight: 500; color: var(--text-primary);">
                    ${getHealthLabel(provider.health_status)}
                </span>
            </div>
            
            <div class="provider-info">
                <span class="provider-info-label">URL:</span>
                <span>${escapeHtml(provider.base_url)}</span>
                
                <span class="provider-info-label">Priority:</span>
                <span>${provider.priority}</span>
                
                <span class="provider-info-label">Enabled:</span>
                <span>${provider.enabled ? '✅ Yes' : '❌ No'}</span>
                
                ${provider.last_health_check ? `
                    <span class="provider-info-label">Last Check:</span>
                    <span>${formatTimestamp(provider.last_health_check)}</span>
                ` : ''}
                
                ${provider.error_message ? `
                    <span class="provider-info-label" style="color: var(--error-color);">Error:</span>
                    <span style="color: var(--error-color);">${escapeHtml(provider.error_message)}</span>
                ` : ''}
            </div>
            
            <div class="provider-actions">
                <button class="btn btn-sm btn-secondary" onclick="checkProviderHealth('${provider.id}')">
                    🏥 Health Check
                </button>
                <button class="btn btn-sm btn-primary" onclick="editProvider('${provider.id}')">
                    ✏️ Edit
                </button>
                <button class="btn btn-sm btn-danger" onclick="deleteProvider('${provider.id}', '${escapeHtml(provider.name)}')">
                    🗑️ Delete
                </button>
            </div>
        </div>
    `).join('');
}

// Get health label
function getHealthLabel(status) {
    switch (status) {
        case 'healthy': return 'Healthy';
        case 'unhealthy': return 'Unhealthy';
        case 'unknown': return 'Unknown';
        default: return status;
    }
}

// Check provider health
async function checkProviderHealth(providerId) {
    try {
        toast.info('Checking provider health...');
        const response = await api.get('/api/admin/registry/providers/health');
        toast.success('Health check completed');
        await loadProviders();
        await loadStats();
    } catch (error) {
        console.error('Health check failed:', error);
        toast.error('Health check failed');
    }
}

// Load models
async function loadModels() {
    try {
        const response = await api.get('/api/admin/registry/models');
        models = response.models || [];
        renderModels(models);
    } catch (error) {
        console.error('Failed to load models:', error);
        toast.error('Failed to load models');
        document.getElementById('models-table-body').innerHTML = 
            '<tr><td colspan="8" style="text-align: center; color: var(--error-color);">Failed to load models</td></tr>';
    }
}

// Render models
function renderModels(modelsToRender) {
    const tbody = document.getElementById('models-table-body');
    
    if (modelsToRender.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-secondary);">No models found. Try auto-discovery or register manually.</td></tr>';
        return;
    }
    
    tbody.innerHTML = modelsToRender.map(model => `
        <tr>
            <td><code>${escapeHtml(model.model_id)}</code></td>
            <td>${escapeHtml(model.model_name)}</td>
            <td>${getProviderName(model.provider_id)}</td>
            <td>
                ${(model.capabilities || []).map(cap => 
                    `<span class="capability-badge">${cap}</span>`
                ).join('')}
            </td>
            <td><span class="model-status ${model.status}">${model.status}</span></td>
            <td><span class="status-indicator ${model.health_status}"></span> ${model.health_status}</td>
            <td>${model.total_requests || 0}</td>
            <td>
                <button class="btn btn-sm btn-danger" onclick="deleteModel('${model.id}', '${escapeHtml(model.model_id)}')">
                    Delete
                </button>
            </td>
        </tr>
    `).join('');
}

// Get provider name by ID
function getProviderName(providerId) {
    const provider = providers.find(p => p.id === providerId);
    return provider ? escapeHtml(provider.name) : 'Unknown';
}

// Filter models
function filterModels() {
    const search = document.getElementById('filter-search').value.toLowerCase();
    const providerFilter = document.getElementById('filter-provider').value;
    const statusFilter = document.getElementById('filter-status').value;
    const healthFilter = document.getElementById('filter-health').value;
    
    const filtered = models.filter(model => {
        const matchesSearch = !search || 
            model.model_id.toLowerCase().includes(search) || 
            model.model_name.toLowerCase().includes(search);
        
        const matchesProvider = !providerFilter || model.provider_id === providerFilter;
        const matchesStatus = !statusFilter || model.status === statusFilter;
        const matchesHealth = !healthFilter || model.health_status === healthFilter;
        
        return matchesSearch && matchesProvider && matchesStatus && matchesHealth;
    });
    
    renderModels(filtered);
}

// Update provider filters
function updateProviderFilters() {
    const select = document.getElementById('filter-provider');
    const modelProviderSelect = document.getElementById('model-provider-select');
    
    const options = providers.map(p => 
        `<option value="${p.id}">${escapeHtml(p.name)} (${p.provider_type})</option>`
    ).join('');
    
    select.innerHTML = '<option value="">All Providers</option>' + options;
    modelProviderSelect.innerHTML = '<option value="">Select Provider...</option>' + options;
}

// Run discovery
async function runDiscovery() {
    const btn = document.getElementById('btn-discover');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span> Discovering...';
    
    try {
        toast.info('Discovering models from providers...');
        const response = await api.post('/api/admin/registry/discover', {});
        
        toast.success(`Discovery complete! Found ${response.discovered} new models`);
        
        await loadModels();
        await loadStats();
    } catch (error) {
        console.error('Discovery failed:', error);
        toast.error('Model discovery failed: ' + (error.message || 'Unknown error'));
    } finally {
        btn.disabled = false;
        btn.innerHTML = '🔍 Discover Models';
    }
}

// Refresh providers
async function refreshProviders() {
    await loadProviders();
}

// Refresh stats
async function refreshStats() {
    await loadStats();
}

// Refresh all data
async function refreshAll() {
    await Promise.all([
        loadStats(),
        loadProviders(),
        loadModels()
    ]);
}

// Show create provider modal
function showCreateProviderModal() {
    document.getElementById('create-provider-modal').classList.add('active');
}

// Close create provider modal
function closeCreateProviderModal() {
    document.getElementById('create-provider-modal').classList.remove('active');
    document.getElementById('create-provider-form').reset();
}

// Create provider
async function createProvider(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const data = {
        name: formData.get('name'),
        provider_type: formData.get('provider_type'),
        base_url: formData.get('base_url'),
        api_key: formData.get('api_key') || undefined,
        priority: parseInt(formData.get('priority')),
        enabled: formData.get('enabled') === 'on'
    };
    
    try {
        await api.post('/api/admin/registry/providers', data);
        toast.success('Provider created successfully');
        closeCreateProviderModal();
        await loadProviders();
        await loadStats();
    } catch (error) {
        console.error('Failed to create provider:', error);
        toast.error('Failed to create provider: ' + (error.message || 'Unknown error'));
    }
}

// Delete provider
async function deleteProvider(providerId, providerName) {
    if (!confirm(`Are you sure you want to delete provider "${providerName}"? This will also remove all associated models.`)) {
        return;
    }
    
    try {
        await api.delete(`/api/admin/registry/providers/${providerId}`);
        toast.success('Provider deleted successfully');
        await loadProviders();
        await loadModels();
        await loadStats();
    } catch (error) {
        console.error('Failed to delete provider:', error);
        toast.error('Failed to delete provider: ' + (error.message || 'Unknown error'));
    }
}

// Show create model modal
function showCreateModelModal() {
    document.getElementById('create-model-modal').classList.add('active');
}

// Close create model modal
function closeCreateModelModal() {
    document.getElementById('create-model-modal').classList.remove('active');
    document.getElementById('create-model-form').reset();
}

// Create model
async function createModel(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    
    // Parse capabilities
    const capabilitiesStr = formData.get('capabilities') || '';
    const capabilities = capabilitiesStr ? capabilitiesStr.split(',').map(c => c.trim()).filter(c => c) : [];
    
    // Parse tags
    const tagsStr = formData.get('tags') || '';
    const tags = tagsStr ? tagsStr.split(',').map(t => t.trim()).filter(t => t) : [];
    
    const data = {
        model_id: formData.get('model_id'),
        model_name: formData.get('model_name'),
        provider_id: formData.get('provider_id'),
        description: formData.get('description') || undefined,
        capabilities: capabilities,
        tags: tags
    };
    
    try {
        await api.post('/api/admin/registry/models', data);
        toast.success('Model registered successfully');
        closeCreateModelModal();
        await loadModels();
        await loadStats();
    } catch (error) {
        console.error('Failed to register model:', error);
        toast.error('Failed to register model: ' + (error.message || 'Unknown error'));
    }
}

// Delete model
async function deleteModel(modelId, modelName) {
    if (!confirm(`Are you sure you want to delete model "${modelName}"?`)) {
        return;
    }
    
    try {
        await api.delete(`/api/admin/registry/models/${modelId}`);
        toast.success('Model deleted successfully');
        await loadModels();
        await loadStats();
    } catch (error) {
        console.error('Failed to delete model:', error);
        toast.error('Failed to delete model: ' + (error.message || 'Unknown error'));
    }
}

// Edit provider (placeholder - can be expanded)
function editProvider(providerId) {
    toast.info('Edit provider feature coming soon!');
    // TODO: Implement edit modal similar to create
}

// Format timestamp
function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    // Less than 1 minute
    if (diff < 60000) {
        return 'Just now';
    }
    
    // Less than 1 hour
    if (diff < 3600000) {
        const mins = Math.floor(diff / 60000);
        return `${mins} minute${mins > 1 ? 's' : ''} ago`;
    }
    
    // Less than 1 day
    if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000);
        return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    }
    
    // Format as date
    return date.toLocaleString();
}

// Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

