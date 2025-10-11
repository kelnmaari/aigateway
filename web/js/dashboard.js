// Dashboard Manager
class Dashboard {
    constructor() {
        this.user = null;
        this.stats = {
            conversations: 0,
            tenants: 0,
            apiKeys: 0,
            requests: 0
        };
    }

    // Initialize dashboard
    async init() {
        // Check authentication
        if (!this.isAuthenticated()) {
            window.location.href = '/web/login.html';
            return;
        }

        try {
            // Load user info
            await this.loadUser();
            
            // Load dashboard data
            await Promise.all([
                this.loadStats(),
                this.loadRecentConversations(),
                this.loadTenants()
            ]);
            
            // Setup event listeners
            this.setupEventListeners();
            
            console.log('✅ Dashboard initialized');
            
        } catch (error) {
            console.error('Failed to initialize dashboard:', error);
            if (error.message.includes('401') || error.message.includes('unauthorized')) {
                this.logout();
            }
        }
    }

    // Check if user is authenticated
    isAuthenticated() {
        return !!localStorage.getItem('access_token');
    }

    // Load current user
    async loadUser() {
        try {
            this.user = await api.getCurrentUser();
            
            // Navbar handles user info display now via navbar.js component

            // Update user avatar
            const avatars = document.querySelectorAll('.user-avatar');
            avatars.forEach(avatar => {
                avatar.textContent = (this.user.full_name || this.user.username).charAt(0).toUpperCase();
            });
            
            // Show/hide Admin link based on user role
            if (this.user.is_admin) {
                this.showAdminLink();
            }
        } catch (error) {
            console.error('Failed to load user:', error);
            throw error;
        }
    }
    
    showAdminLink() {
        // Check if admin link already exists
        if (document.querySelector('.nav-link[href="/admin.html"]')) {
            return;
        }
        
        // Find dashboard link and insert admin link after it
        const dashboardLink = document.querySelector('.nav-link[href="/dashboard.html"]');
        if (dashboardLink) {
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
            dashboardLink.parentNode.insertBefore(adminLink, dashboardLink.nextSibling);
            console.log('✅ Admin link added to navigation');
        }
    }

    // Load dashboard stats
    async loadStats() {
        try {
            // Load conversations count
            const conversationsResp = await api.getConversations();
            this.stats.conversations = conversationsResp.conversations?.length || 0;
            
            // Load tenants count
            const tenantsResp = await api.request(`${api.baseURL}/api/users/me/tenants`);
            if (tenantsResp.ok) {
                const tenantsData = await tenantsResp.json();
                this.stats.tenants = tenantsData.tenants?.length || 0;
            }
            
            // Load API keys count
            const keysResp = await api.request(`${api.baseURL}/api/users/me/api-keys`);
            if (keysResp.ok) {
                const keysData = await keysResp.json();
                this.stats.apiKeys = keysData.api_keys?.length || 0;
            }
            
            // TODO: Load API requests count from usage stats
            this.stats.requests = 0;
            
            // Update UI
            this.renderStats();
            
        } catch (error) {
            console.error('Failed to load stats:', error);
            this.renderStats(); // Render with defaults
        }
    }

    // Render stats cards
    renderStats() {
        document.getElementById('stat-conversations').textContent = this.stats.conversations;
        document.getElementById('stat-tenants').textContent = this.stats.tenants;
        document.getElementById('stat-apikeys').textContent = this.stats.apiKeys;
        document.getElementById('stat-requests').textContent = this.stats.requests.toLocaleString();
    }

    // Load recent conversations
    async loadRecentConversations() {
        const tbody = document.getElementById('recent-conversations');
        tbody.innerHTML = '';
        
        try {
            const response = await api.getConversations();
            const conversations = response.conversations || [];
            
            if (conversations.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="table-empty">No conversations yet. <a href="/chat.html">Start chatting!</a></td></tr>';
                return;
            }
            
            // Show only first 5
            const recent = conversations.slice(0, 5);
            
            recent.forEach(conv => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>
                        <div style="font-weight: 500;">${this.escapeHtml(conv.title)}</div>
                    </td>
                    <td>
                        <span class="badge badge-info">${this.escapeHtml(conv.model)}</span>
                    </td>
                    <td>${this.formatDate(conv.created_at)}</td>
                    <td>
                        <div class="table-actions">
                            <button class="btn btn-icon" onclick="dashboard.openConversation('${conv.id}')" title="Open">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke-width="2"/>
                                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke-width="2"/>
                                </svg>
                            </button>
                            <button class="btn btn-icon" onclick="dashboard.deleteConversation('${conv.id}')" title="Delete">
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                                </svg>
                            </button>
                        </div>
                    </td>
                `;
                tbody.appendChild(row);
            });
            
        } catch (error) {
            console.error('Failed to load conversations:', error);
            tbody.innerHTML = '<tr><td colspan="4" class="table-empty">Failed to load conversations</td></tr>';
        }
    }

    // Load tenants
    async loadTenants() {
        try {
            const response = await api.request(`${api.baseURL}/api/users/me/tenants`);
            
            if (!response.ok) {
                throw new Error('Failed to load tenants');
            }
            
            const data = await response.json();
            const tenants = data.tenants || [];
            
            const container = document.getElementById('tenants-list');
            container.innerHTML = '';
            
            if (tenants.length === 0) {
                container.innerHTML = `
                    <div class="empty-state">
                        <div class="empty-state-icon">🏢</div>
                        <h3>No tenants yet</h3>
                        <p>Create your first organization to collaborate with your team</p>
                        <a href="/web/tenants.html" class="btn btn-primary">Create Organization</a>
                    </div>
                `;
                return;
            }
            
            tenants.forEach(tenant => {
                const card = this.createTenantCard(tenant);
                container.appendChild(card);
            });
            
        } catch (error) {
            console.error('Failed to load tenants:', error);
            document.getElementById('tenants-list').innerHTML = 
                '<div class="tenant-card loading">Failed to load tenants</div>';
        }
    }

    // Create tenant card element
    createTenantCard(tenant) {
        const card = document.createElement('div');
        card.className = 'tenant-card';
        card.style.cursor = 'pointer';
        card.onclick = () => window.location.href = `/web/tenants.html?id=${tenant.id}`;
        
        const icon = tenant.type === 'personal' ? '👤' : '🏢';
        const typeClass = tenant.type === 'personal' ? 'personal' : 'organization';
        
        card.innerHTML = `
            <div class="tenant-header">
                <div class="tenant-icon">${icon}</div>
                <div class="tenant-info">
                    <div class="tenant-name">${this.escapeHtml(tenant.name)}</div>
                    <div class="tenant-type ${typeClass}">${tenant.type}</div>
                </div>
            </div>
            ${tenant.description ? `<div class="tenant-description">${this.escapeHtml(tenant.description)}</div>` : ''}
            <div class="tenant-stats">
                <div class="tenant-stat">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke-width="2"/>
                        <circle cx="9" cy="7" r="4" stroke-width="2"/>
                    </svg>
                    <span>${tenant.member_count || 1} members</span>
                </div>
                <div class="tenant-stat">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2" stroke-width="2"/>
                        <path d="M7 11V7a5 5 0 0 1 10 0v4" stroke-width="2"/>
                    </svg>
                    <span>${tenant.api_key_count || 0} keys</span>
                </div>
            </div>
        `;
        
        return card;
    }

    // Open conversation in chat
    openConversation(id) {
        window.location.href = `/chat.html?conversation=${id}`;
    }

    // Delete conversation
    async deleteConversation(id) {
        const confirmed = await modal.danger(
            'Are you sure you want to delete this conversation?',
            'Delete Conversation'
        );
        if (!confirmed) return;
        
        try {
            await api.deleteConversation(id);
            await this.loadRecentConversations();
            await this.loadStats();
        } catch (error) {
            console.error('Failed to delete conversation:', error);
            toast.error('Failed to delete conversation');
        }
    }

    // Setup event listeners
    setupEventListeners() {
        // Navbar component (navbar.js) handles user dropdown and logout now
        // No need for duplicate event listeners here
    }

    // Logout
    logout() {
        api.logout();
    }

    // Helper: Escape HTML
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Helper: Format date
    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);
        
        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;
        
        return date.toLocaleDateString();
    }
}

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new Dashboard();
    dashboard.init();
});

