// profile-devices.js - Device Management for User Profile

class DeviceManager {
    constructor() {
        this.devices = [];
        this.currentUser = null;
    }

    // Initialize page
    async init() {
        try {
            // Load current user
            await this.loadUser();

            // Load devices
            await this.loadDevices();

            console.log('✅ Device Manager initialized');
        } catch (error) {
            console.error('Failed to initialize device manager:', error);
            if (error.message.includes('401') || error.message.includes('unauthorized')) {
                api.logout();
            }
        }
    }

    // Load current user
    async loadUser() {
        try {
            this.currentUser = await api.getCurrentUser();
        } catch (error) {
            console.error('Failed to load user:', error);
            throw error;
        }
    }

    // Load devices from API
    async loadDevices() {
        const loadingSpinner = document.getElementById('loadingSpinner');
        const emptyState = document.getElementById('emptyState');
        const devicesList = document.getElementById('devicesList');

        // Show loading
        loadingSpinner.style.display = 'block';
        emptyState.style.display = 'none';
        devicesList.innerHTML = '';

        try {
            const status = document.getElementById('statusFilter').value;
            const sort = document.getElementById('sortBy').value;
            const order = document.getElementById('sortOrder').value;

            const response = await api.request(`${api.baseURL}/api/auth/devices?status=${status}&sort=${sort}&order=${order}`);

            if (response.status === 401) {
                toast.error('Session expired. Please login again.');
                api.logout();
                return;
            }

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.devices = data.devices || [];

            // Update stats
            this.updateStats(data);

            // Hide loading
            loadingSpinner.style.display = 'none';

            // Show devices or empty state
            if (this.devices.length === 0) {
                emptyState.style.display = 'block';
            } else {
                this.renderDevices(this.devices);
            }

        } catch (error) {
            console.error('Failed to load devices:', error);
            loadingSpinner.style.display = 'none';
            toast.error('Failed to load devices');
        }
    }

    // Update statistics
    updateStats(data) {
        document.getElementById('totalDevices').textContent = data.total || 0;
        document.getElementById('activeDevices').textContent = data.current_count || 0;
        
        const currentDevice = this.devices.find(d => d.is_current_device);
        document.getElementById('currentDevice').textContent = currentDevice ? 
            (currentDevice.device_name || 'This Device') : 'None';
    }

    // Render devices list
    renderDevices(devices) {
        const devicesList = document.getElementById('devicesList');
        devicesList.innerHTML = '';

        devices.forEach(device => {
            const deviceCard = this.createDeviceCard(device);
            devicesList.appendChild(deviceCard);
        });
    }

    // Create device card
    createDeviceCard(device) {
        const card = document.createElement('div');

        const isInactive = this.isDeviceInactive(device);
        const cardClasses = ['card', 'device-card'];
        
        if (device.is_current_device) {
            cardClasses.push('current-device');
        }
        if (isInactive) {
            cardClasses.push('inactive');
        }

        card.className = cardClasses.join(' ');
        card.innerHTML = `
            <div style="padding: 20px;">
                <div style="display: flex; gap: 16px;">
                    <!-- Device Icon -->
                    <div style="flex-shrink: 0;">
                        <span class="${this.getDeviceIcon(device.device_os)} device-icon"></span>
                    </div>

                    <!-- Device Info -->
                    <div style="flex: 1; min-width: 0;">
                        <div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 12px;">
                            <div style="flex: 1; min-width: 0;">
                                <h5 class="card-title" style="margin-bottom: 4px;">
                                    ${this.escapeHtml(device.device_name || 'Unknown Device')}
                                    ${device.is_current_device ? '<span class="badge badge-current" style="margin-left: 8px;">This Device</span>' : ''}
                                </h5>
                                <p class="card-text text-muted" style="margin: 0;">
                                    🖥️ ${this.escapeHtml(device.device_hostname || 'Unknown')}
                                </p>
                            </div>
                            <span class="badge ${this.getStatusBadgeClass(device.status, isInactive)}" style="margin-left: 8px;">
                                ${this.getStatusText(device.status, isInactive)}
                            </span>
                        </div>

                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 12px; font-size: 14px; color: var(--text-secondary);">
                            <div>💻 ${this.getOSName(device.device_os)}</div>
                            <div>🏷️ v${this.escapeHtml(device.device_version || '0.0.0')}</div>
                        </div>

                        <div class="last-seen ${this.getLastSeenClass(device.last_seen_at)}" style="font-size: 14px;">
                            🕒 ${this.formatLastSeen(device.last_seen_at)}
                        </div>

                        <div class="text-muted" style="font-size: 13px; margin-top: 4px;">
                            📅 Created: ${this.formatDate(device.created_at)}
                        </div>

                        ${device.expires_at ? `
                            <div class="text-muted" style="font-size: 13px; margin-top: 4px;">
                                ⏰ Expires: ${this.formatDate(device.expires_at)}
                            </div>
                        ` : ''}

                        <!-- Actions -->
                        <div style="margin-top: 16px; display: flex; gap: 8px;">
                            <button class="btn btn-sm btn-outline-primary" onclick="deviceManager.openRenameModal('${device.id}')">
                                ✏️ Rename
                            </button>
                            ${!device.is_current_device ? `
                                <button class="btn btn-sm btn-outline-danger" onclick="deviceManager.openDeleteModal('${device.id}')">
                                    🗑️ Remove
                                </button>
                            ` : ''}
                        </div>
                    </div>
                </div>
            </div>
        `;

        return card;
    }

    // Get device icon based on OS
    getDeviceIcon(os) {
        switch(os) {
            case 'windows': return 'icon-windows';
            case 'darwin': return 'icon-apple';
            case 'linux': return 'icon-linux';
            default: return 'icon-laptop';
        }
    }

    // Get OS name
    getOSName(os) {
        switch(os) {
            case 'windows': return 'Windows';
            case 'darwin': return 'macOS';
            case 'linux': return 'Linux';
            default: return 'Unknown';
        }
    }

    // Check if device is inactive (not seen in 30 days)
    isDeviceInactive(device) {
        if (!device.last_seen_at) return true;
        const lastSeen = new Date(device.last_seen_at);
        const daysSince = (new Date() - lastSeen) / (1000 * 60 * 60 * 24);
        return daysSince > 30;
    }

    // Get status badge class
    getStatusBadgeClass(status, isInactive) {
        if (isInactive) return 'badge-inactive';
        if (status === 'active') return 'badge-active';
        if (status === 'expired') return 'badge-inactive';
        return 'bg-secondary';
    }

    // Get status text
    getStatusText(status, isInactive) {
        if (isInactive) return 'Inactive';
        if (status === 'active') return 'Active';
        if (status === 'expired') return 'Expired';
        return status.charAt(0).toUpperCase() + status.slice(1);
    }

    // Get last seen class
    getLastSeenClass(lastSeen) {
        if (!lastSeen) return 'inactive';
        const diff = new Date() - new Date(lastSeen);
        const minutes = diff / (1000 * 60);
        if (minutes < 5) return 'recent';
        if (minutes > 43200) return 'inactive'; // 30 days
        return '';
    }

    // Format last seen
    formatLastSeen(lastSeen) {
        if (!lastSeen) return 'Never';
        
        const diff = new Date() - new Date(lastSeen);
        const minutes = Math.floor(diff / (1000 * 60));
        const hours = Math.floor(diff / (1000 * 60 * 60));
        const days = Math.floor(diff / (1000 * 60 * 60 * 24));

        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
        if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
        if (days < 30) return `${days} day${days > 1 ? 's' : ''} ago`;
        return this.formatDate(lastSeen);
    }

    // Format date
    formatDate(dateStr) {
        if (!dateStr) return 'N/A';
        const date = new Date(dateStr);
        return date.toLocaleDateString('en-US', { 
            year: 'numeric', 
            month: 'short', 
            day: 'numeric' 
        });
    }

    // Escape HTML
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Refresh devices
    refreshDevices() {
        toast.info('Refreshing devices...');
        this.loadDevices();
    }

    // Open rename modal
    async openRenameModal(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        const newName = prompt(`Enter new name for device:\n\nCurrent: ${device.device_name || 'Unknown Device'}`, device.device_name || '');
        
        if (newName === null) return; // Cancelled
        
        if (!newName.trim()) {
            toast.error('Device name cannot be empty');
            return;
        }

        try {
            const response = await api.request(`${api.baseURL}/api/auth/devices/${deviceId}`, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ device_name: newName.trim() })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || `HTTP ${response.status}`);
            }

            toast.success('Device renamed successfully');
            this.loadDevices();

        } catch (error) {
            console.error('Failed to rename device:', error);
            toast.error(`Failed to rename device: ${error.message}`);
        }
    }

    // Open delete modal
    async openDeleteModal(deviceId) {
        const device = this.devices.find(d => d.id === deviceId);
        if (!device) return;

        const confirmed = await modal.danger(
            `Are you sure you want to remove device "${device.device_name || 'Unknown'}"?`,
            'Remove Device'
        );

        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/auth/devices/${deviceId}`, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ reason: 'Removed by user' })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || `HTTP ${response.status}`);
            }

            const result = await response.json();
            toast.success(result.message || 'Device removed successfully');
            this.loadDevices();

        } catch (error) {
            console.error('Failed to remove device:', error);
            toast.error(`Failed to remove device: ${error.message}`);
        }
    }
}

// Global instance
let deviceManager;

// Initialize on page load
document.addEventListener('DOMContentLoaded', async () => {
    deviceManager = new DeviceManager();
    await deviceManager.init();
});

// Global functions for onclick handlers
function refreshDevices() {
    deviceManager.refreshDevices();
}

