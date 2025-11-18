// Settings Tab Management (v3.0.9)
// Phase 3: Selective editing with validation

// Global state
let editingSettingId = null;

// Load settings on tab switch
function loadSettings() {
    const loadingEl = document.getElementById('settings-loading');
    const contentEl = document.getElementById('settings-content');
    const errorEl = document.getElementById('settings-error');
    
    // Show loading using AG framework
    AG.loading.show(contentEl);
    loadingEl.style.display = 'block';
    contentEl.style.display = 'none';
    errorEl.style.display = 'none';
    
    const token = localStorage.getItem('access_token');
    if (!token) {
        // No token - user not logged in
        loadingEl.style.display = 'none';
        errorEl.style.display = 'block';
        document.getElementById('settings-error-message').textContent = 'You are not logged in. Please login first to access settings.';
        
        // Hide action button for this case
        const actionBtn = document.getElementById('settings-error-action');
        if (actionBtn) actionBtn.style.display = 'none';
        return;
    }
    
    // Use AG.http for API call
    AG.http.get('/api/admin/settings')
    .then(data => {
        renderSettings(data.settings);
        loadingEl.style.display = 'none';
        contentEl.style.display = 'block';
        AG.loading.hide(contentEl);
    })
    .catch(error => {
        console.error('Failed to load settings:', error);
        loadingEl.style.display = 'none';
        errorEl.style.display = 'block';
        
        let errorMessage = error.message;
        const actionBtn = document.getElementById('settings-error-action');
        
        // Check if it's authentication error
        if (error.message.includes('401')) {
            errorMessage = 'Authentication failed. Your session may have expired. Please use the Logout button in the navbar, then login again.';
        }
        
        document.getElementById('settings-error-message').textContent = errorMessage;
        
        // Show retry button
        if (actionBtn) {
            actionBtn.style.display = 'inline-block';
            actionBtn.textContent = 'Retry';
            actionBtn.onclick = () => loadSettings();
        }
    });
}

// Render settings grouped by category
function renderSettings(settingsGrouped) {
    const contentEl = document.getElementById('settings-content');
    
    if (!settingsGrouped || Object.keys(settingsGrouped).length === 0) {
        contentEl.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-cog"></i>
                <h3>No Settings Found</h3>
                <p>Settings will appear here after initial configuration</p>
            </div>
        `;
        return;
    }
    
    // Category icons and labels
    const categoryMeta = {
        'server': { icon: '🖥️', label: 'Server Configuration' },
        'inference': { icon: '🤖', label: 'Inference Engine' },
        'auth': { icon: '🔐', label: 'Authentication & Security' },
        'database': { icon: '💾', label: 'Database Settings' },
        'logging': { icon: '📝', label: 'Logging Configuration' },
        'metrics': { icon: '📊', label: 'Metrics & Monitoring' },
        'tls': { icon: '🔒', label: 'TLS/SSL Configuration' },
        'rag': { icon: '🧠', label: 'RAG System' },
        'huggingface': { icon: '🤗', label: 'Hugging Face Integration' }
    };
    
    let html = '';
    
    Object.keys(settingsGrouped).sort().forEach(category => {
        const settings = settingsGrouped[category];
        const meta = categoryMeta[category] || { icon: '⚙️', label: category };
        
        html += `
            <div class="settings-category">
                <div class="settings-category-header" onclick="toggleSettingsCategory('${category}')">
                    <div>
                        <span class="settings-category-icon">${meta.icon}</span>
                        <span class="settings-category-title">${meta.label}</span>
                        <span class="settings-count">${settings.length} settings</span>
                    </div>
                    <svg class="settings-chevron" id="chevron-${category}" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <polyline points="6 9 12 15 18 9" stroke-width="2"/>
                    </svg>
                </div>
                <div class="settings-category-content" id="category-${category}" style="display: none;">
                    <div class="data-table-container">
                        <table class="data-table">
                            <thead>
                                <tr>
                                    <th style="width: 20%;">Setting</th>
                                    <th style="width: 25%;">Value</th>
                                    <th style="width: 10%;">Type</th>
                                    <th style="width: 30%;">Description</th>
                                    <th style="width: 15%;">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
        `;
        
        settings.forEach(setting => {
            const isMigrated = setting.is_migrated;
            const isEditable = setting.is_editable;
            
            const badge = isMigrated 
                ? '<span class="badge badge-success">DB</span>' 
                : '<span class="badge badge-secondary">YAML</span>';
            
            // Show "Restart Required" badge for non-editable migrated settings
            const restartBadge = isMigrated && !isEditable 
                ? '<span class="badge badge-warning" style="margin-left: 0.5rem;">Restart Required</span>'
                : '';
            
            // Phase 4: Show if setting requires restart when changed
            const requiresRestartInfo = isEditable && setting.requires_restart
                ? '<span class="badge badge-warning" style="margin-left: 0.5rem; font-size: 0.7rem;">⚠️ Restart after change</span>'
                : isEditable && !setting.requires_restart
                ? '<span class="badge badge-success" style="margin-left: 0.5rem; font-size: 0.7rem;">✓ Live reload</span>'
                : '';
            
            // Edit button only for editable settings
            const editButton = isEditable
                ? `<button class="btn-edit" onclick="editSetting('${setting.id}')" title="Edit this setting">
                     <i class="fas fa-edit"></i> Edit
                   </button>`
                : '<span style="color: var(--text-secondary); font-size: 0.85rem;">Read-only</span>';
            
            html += `
                <tr id="setting-row-${setting.id}">
                    <td>
                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                            <code style="font-size: 0.9rem;">${setting.key}</code>
                            ${badge}
                        </div>
                    </td>
                    <td id="value-cell-${setting.id}">
                        <code class="setting-value">${escapeHtml(setting.value)}</code>
                    </td>
                    <td>
                        <span class="badge badge-info">${setting.type}</span>
                    </td>
                    <td style="font-size: 0.85rem; color: var(--text-secondary);">
                        ${setting.description || '-'}
                        ${restartBadge}
                        ${requiresRestartInfo}
                    </td>
                    <td>
                        ${editButton}
                    </td>
                </tr>
            `;
        });
        
        html += `
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;
    });
    
    contentEl.innerHTML = html;
}

// Toggle category expansion
function toggleSettingsCategory(category) {
    const contentEl = document.getElementById(`category-${category}`);
    const chevronEl = document.getElementById(`chevron-${category}`);
    
    if (contentEl.style.display === 'none') {
        contentEl.style.display = 'block';
        chevronEl.style.transform = 'rotate(180deg)';
    } else {
        contentEl.style.display = 'none';
        chevronEl.style.transform = 'rotate(0deg)';
    }
}

// Utility: Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Phase 3: Edit Setting (inline editing)
function editSetting(settingId) {
    if (editingSettingId) {
        alert('Please save or cancel the current edit first');
        return;
    }
    
    editingSettingId = settingId;
    
    // Get current value
    const valueCell = document.getElementById(`value-cell-${settingId}`);
    const currentValue = valueCell.querySelector('.setting-value').textContent;
    
    // Replace with input field
    valueCell.innerHTML = `
        <div style="display: flex; gap: 0.5rem; align-items: center;">
            <input type="text" 
                   id="edit-input-${settingId}" 
                   class="setting-edit-input" 
                   value="${escapeHtml(currentValue)}"
                   style="flex: 1; padding: 0.4rem; border: 1px solid var(--border-color); border-radius: 4px;">
            <button class="btn-save" onclick="saveSetting('${settingId}')" title="Save changes">
                <i class="fas fa-check"></i>
            </button>
            <button class="btn-cancel" onclick="cancelEdit('${settingId}', '${escapeHtml(currentValue)}')" title="Cancel">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;
    
    // Focus input
    document.getElementById(`edit-input-${settingId}`).focus();
}

// Cancel editing
function cancelEdit(settingId, originalValue) {
    const valueCell = document.getElementById(`value-cell-${settingId}`);
    valueCell.innerHTML = `<code class="setting-value">${originalValue}</code>`;
    editingSettingId = null;
}

// Save setting with confirmation modal
function saveSetting(settingId) {
    const input = document.getElementById(`edit-input-${settingId}`);
    const newValue = input.value.trim();
    
    if (!newValue) {
        alert('Value cannot be empty');
        return;
    }
    
    // Show confirmation modal
    showConfirmModal(settingId, newValue);
}

// Show confirmation modal
function showConfirmModal(settingId, newValue) {
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>Confirm Setting Change</h3>
            </div>
            <div class="modal-body">
                <p><strong>Setting ID:</strong> <code>${settingId}</code></p>
                <p><strong>New Value:</strong> <code>${escapeHtml(newValue)}</code></p>
                <p style="color: var(--text-secondary); margin-top: 1rem;">
                    Are you sure you want to change this setting?
                </p>
            </div>
            <div class="modal-footer">
                <button class="btn-secondary" onclick="closeModal()">Cancel</button>
                <button class="btn-primary" onclick="confirmSaveSetting('${settingId}', '${escapeHtml(newValue)}')">
                    Confirm & Save
                </button>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    modal.style.display = 'flex';
}

// Close modal
function closeModal() {
    const modal = document.querySelector('.modal-overlay');
    if (modal) {
        modal.remove();
    }
}

// Confirm and save setting (actual API call)
function confirmSaveSetting(settingId, newValue) {
    closeModal();
    
    const token = localStorage.getItem('access_token');
    if (!token) {
        alert('Authentication required. Please login again.');
        return;
    }
    
    const valueCell = document.getElementById(`value-cell-${settingId}`);
    const originalHTML = valueCell.innerHTML;
    
    // Optimistic UI update - показываем новое значение сразу (v3.1.0: AJAX-03)
    valueCell.innerHTML = `
        <code class="setting-value" style="opacity: 0.6;">
            ${escapeHtml(newValue)}
            <span style="margin-left: 8px; font-size: 0.8em; color: var(--text-secondary);">
                <div class="skeleton" style="width: 50px; height: 12px; display: inline-block;"></div>
            </span>
        </code>
    `;
    
    fetch(`/api/admin/settings/${settingId}`, {
        method: 'PUT',
        headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ value: newValue })
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => {
                throw new Error(err.error || `HTTP ${response.status}`);
            });
        }
        return response.json();
    })
    .then(data => {
        // Success - confirm optimistic update with solid state
        valueCell.innerHTML = `<code class="setting-value">${escapeHtml(newValue)}</code>`;
        editingSettingId = null;
        
        // Phase 4: Check if setting requires restart
        const requiresRestart = data.setting && data.setting.requires_restart;
        const message = requiresRestart 
            ? 'Setting updated. Server restart required for changes to take effect.'
            : 'Setting updated and applied successfully';
        const type = requiresRestart ? 'warning' : 'success';
        
        // Use AG.toast instead of showNotification
        AG.toast(message, type);
    })
    .catch(error => {
        console.error('Failed to update setting:', error);
        
        // Rollback optimistic update (v3.1.0: AJAX-03)
        valueCell.innerHTML = originalHTML;
        
        AG.toast(`Failed to update: ${error.message}`, 'error');
    });
}

// Show notification toast
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    // Show notification
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    // Auto-hide after 3 seconds
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => {
            notification.remove();
        }, 300);
    }, 3000);
}

// Initialize settings when tab is opened
document.addEventListener('DOMContentLoaded', () => {
    // Listen for settings tab activation
    const settingsTabBtn = document.querySelector('[data-tab="settings"]');
    if (settingsTabBtn) {
        settingsTabBtn.addEventListener('click', () => {
            // Load settings on first click
            const contentEl = document.getElementById('settings-content');
            const loadingEl = document.getElementById('settings-loading');
            
            // Check if already loaded (content div is visible)
            // Use display check instead of innerHTML to avoid HTML comment false positive
            if (contentEl.style.display !== 'block') {
                loadSettings();
            }
        });
    }
});

