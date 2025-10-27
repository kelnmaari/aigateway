// Tenants Management Page
// Version 1.3.0

class TenantsManager {
    constructor() {
        this.tenants = [];
        this.currentTenant = null;
        this.currentUser = null;
        this.selectedUser = null; // For add member modal
        this.addMemberTenantId = null; // Save tenant ID for add member modal
        
        this.init();
    }

    async init() {
        try {
            // Load current user
            await this.loadUser();
            
            // Load tenants
            await this.loadTenants();
            
            // Setup UI
            this.setupEventListeners();
            
        } catch (error) {
            console.error('Failed to initialize tenants page:', error);
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

    // Load tenants list
    async loadTenants() {
        try {
            const response = await api.getUserTenants();
            this.tenants = response.tenants || [];
            
            this.renderTenants();
            
        } catch (error) {
            console.error('Failed to load tenants:', error);
            this.showError('Failed to load tenants');
            document.getElementById('tenants-grid').innerHTML = 
                '<div class="tenant-card error">Failed to load tenants</div>';
        }
    }

    // Render tenants grid
    renderTenants() {
        const grid = document.getElementById('tenants-grid');
        
        if (this.tenants.length === 0) {
            grid.innerHTML = `
                <div class="tenant-card empty">
                    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" stroke-width="2"/>
                        <circle cx="9" cy="7" r="4" stroke-width="2"/>
                        <path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" stroke-width="2"/>
                    </svg>
                    <h3>No Organizations</h3>
                    <p>Create your first organization to start collaborating with your team.</p>
                    <button class="btn btn-primary" onclick="tenantsManager.showCreateModal()">
                        Create Organization
                    </button>
                </div>
            `;
            return;
        }

        grid.innerHTML = this.tenants.map(tenant => this.renderTenantCard(tenant)).join('');
    }

    // Render single tenant card
    renderTenantCard(tenant) {
        const role = this.getTenantRole(tenant);
        const memberCount = tenant.member_count || 0;
        const typeIcon = tenant.type === 'personal' ? '👤' : '🏢';
        const statusBadge = tenant.status === 'active' 
            ? '<span class="badge badge-success">Active</span>'
            : '<span class="badge badge-secondary">Inactive</span>';

        return `
            <div class="tenant-card" data-tenant-id="${tenant.id}" onclick="tenantsManager.showTenantDetail('${tenant.id}')">
                <div class="tenant-card-header">
                    <div class="tenant-icon">${typeIcon}</div>
                    <div class="tenant-badge">${role}</div>
                </div>
                <h3>${this.escapeHtml(tenant.name)}</h3>
                <p class="tenant-description">${this.escapeHtml(tenant.description || 'No description')}</p>
                <div class="tenant-meta">
                    <span>👥 ${memberCount} member${memberCount !== 1 ? 's' : ''}</span>
                    ${statusBadge}
                </div>
            </div>
        `;
    }

    // Get user's role in tenant
    getTenantRole(tenant) {
        // Role comes from join with tenant_members table
        return tenant.role || 'member';
    }

    // Setup event listeners
    setupEventListeners() {
        // User dropdown and logout are now handled by navbar.js component
        // No need for duplicate event listeners here

        // Create tenant modal
        const createBtn = document.getElementById('create-tenant-btn');
        const createModal = document.getElementById('create-tenant-modal');
        const cancelBtn = document.getElementById('cancel-create-btn');
        const modalClose = document.getElementById('modal-close');
        const modalOverlay = document.getElementById('modal-overlay');
        const createForm = document.getElementById('create-tenant-form');

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
                this.createTenant();
            });
        }

        // Auto-generate slug from name
        const nameInput = document.getElementById('tenant-name');
        const slugInput = document.getElementById('tenant-slug');
        
        if (nameInput && slugInput) {
            nameInput.addEventListener('input', () => {
                if (!slugInput.value) {
                    slugInput.value = this.generateSlug(nameInput.value);
                }
            });
        }

        // Detail modal
        const detailModalClose = document.getElementById('detail-modal-close');
        const detailModalOverlay = document.getElementById('detail-modal-overlay');
        const closeDetailBtn = document.getElementById('close-detail-btn');

        if (detailModalClose) {
            detailModalClose.addEventListener('click', () => this.hideDetailModal());
        }

        if (detailModalOverlay) {
            detailModalOverlay.addEventListener('click', () => this.hideDetailModal());
        }

        if (closeDetailBtn) {
            closeDetailBtn.addEventListener('click', () => this.hideDetailModal());
        }

        // Detail actions
        const editBtn = document.getElementById('edit-tenant-btn');
        const deleteBtn = document.getElementById('delete-tenant-btn');

        if (editBtn) {
            editBtn.addEventListener('click', () => this.editTenant());
        }

        if (deleteBtn) {
            deleteBtn.addEventListener('click', () => this.deleteTenant());
        }

        // Add member modal
        const addMemberBtn = document.getElementById('add-member-btn');
        const addMemberModalClose = document.getElementById('add-member-modal-close');
        const cancelAddMemberBtn = document.getElementById('cancel-add-member-btn');
        const addMemberForm = document.getElementById('add-member-form');
        const memberSearchInput = document.getElementById('member-search');

        if (addMemberBtn) {
            addMemberBtn.addEventListener('click', () => {
                console.log('Add Member button clicked, currentTenant:', this.currentTenant);
                this.showAddMemberModal();
            });
        }

        if (addMemberModalClose) {
            addMemberModalClose.addEventListener('click', () => this.hideAddMemberModal());
        }

        if (cancelAddMemberBtn) {
            cancelAddMemberBtn.addEventListener('click', () => this.hideAddMemberModal());
        }

        if (addMemberForm) {
            addMemberForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.submitAddMember();
            });
        }

        if (memberSearchInput) {
            let searchTimeout;
            memberSearchInput.addEventListener('input', (e) => {
                clearTimeout(searchTimeout);
                const query = e.target.value.trim();
                
                if (query.length < 2) {
                    this.hideSearchResults();
                    return;
                }
                
                searchTimeout = setTimeout(() => {
                    this.searchUsers(query);
                }, 500); // Debounce 500ms
            });
        }
    }

    // Show create modal
    showCreateModal() {
        const modal = document.getElementById('create-tenant-modal');
        if (modal) {
            modal.style.display = 'flex';
            document.getElementById('tenant-name').focus();
        }
    }

    // Hide create modal
    hideCreateModal() {
        const modal = document.getElementById('create-tenant-modal');
        if (modal) {
            modal.style.display = 'none';
            document.getElementById('create-tenant-form').reset();
        }
    }

    // Create tenant
    async createTenant() {
        const submitBtn = document.getElementById('submit-create-btn');
        submitBtn.disabled = true;
        submitBtn.textContent = 'Creating...';

        try {
            const name = document.getElementById('tenant-name').value.trim();
            const slug = document.getElementById('tenant-slug').value.trim();
            const description = document.getElementById('tenant-description').value.trim();

            const data = { name };
            if (slug) data.slug = slug;
            if (description) data.description = description;

            const tenant = await api.createTenant(data);

            this.showSuccess('Organization created successfully!');
            this.hideCreateModal();
            await this.loadTenants();

        } catch (error) {
            console.error('Failed to create tenant:', error);
            this.showError(error.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Create';
        }
    }

    // Show tenant detail modal
    async showTenantDetail(tenantId) {
        try {
            const response = await api.getTenant(tenantId);
            // Backend returns {tenant: {...}, role: "..."}
            this.currentTenant = response.tenant;
            const role = response.role || 'unknown';
            console.log('showTenantDetail - set currentTenant:', this.currentTenant, 'role:', role);

            // Update modal header
            document.getElementById('detail-tenant-name').textContent = response.tenant.name;

            // Update info
            document.getElementById('detail-name').textContent = response.tenant.name;
            document.getElementById('detail-slug').textContent = response.tenant.slug || '-';
            document.getElementById('detail-type').textContent = response.tenant.type;
            document.getElementById('detail-status').innerHTML = 
                response.tenant.status === 'active' 
                    ? '<span class="badge badge-success">Active</span>'
                    : '<span class="badge badge-secondary">Inactive</span>';
            document.getElementById('detail-created').textContent = this.formatDate(response.tenant.created_at);
            
            // User's role from backend response
            document.getElementById('detail-role').innerHTML = `<span class="badge">${role}</span>`;
            
            document.getElementById('detail-description').textContent = response.tenant.description || 'No description';

            // Load members
            await this.loadMembers(tenantId);

            // Show/hide actions based on role
            const isOwner = role === 'owner';
            const isAdmin = role === 'admin' || role === 'owner';
            
            document.getElementById('edit-tenant-btn').style.display = isAdmin ? 'block' : 'none';
            document.getElementById('delete-tenant-btn').style.display = isOwner ? 'block' : 'none';
            document.getElementById('add-member-btn').style.display = isAdmin ? 'block' : 'none';

            // Show modal
            document.getElementById('tenant-detail-modal').style.display = 'flex';

        } catch (error) {
            console.error('Failed to load tenant details:', error);
            this.showError('Failed to load tenant details');
        }
    }

    // Hide detail modal
    hideDetailModal() {
        const modal = document.getElementById('tenant-detail-modal');
        if (modal) {
            modal.style.display = 'none';
            this.currentTenant = null;
        }
    }

    // Load members
    async loadMembers(tenantId) {
        try {
            const response = await api.getTenantMembers(tenantId);
            const members = response.members || [];

            const tbody = document.getElementById('members-list');
            
            if (members.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="table-empty">No members</td></tr>';
                return;
            }

            tbody.innerHTML = members.map(member => `
                <tr>
                    <td>
                        <div style="display: flex; flex-direction: column;">
                            <strong>${this.escapeHtml(member.username)}</strong>
                            ${member.email ? `<small style="color: var(--text-secondary);">${this.escapeHtml(member.email)}</small>` : ''}
                        </div>
                    </td>
                    <td><span class="badge">${member.role}</span></td>
                    <td>${this.formatDate(member.joined_at)}</td>
                    <td>
                        ${member.role !== 'owner' ? `
                            <button class="btn btn-sm btn-secondary" 
                                onclick="tenantsManager.updateMemberRole('${tenantId}', '${member.user_id}', '${this.escapeHtml(member.username)}', '${member.role}')"
                                style="margin-right: 4px;">
                                Change Role
                            </button>
                            <button class="btn btn-sm btn-secondary" 
                                onclick="tenantsManager.removeMember('${tenantId}', '${member.user_id}', '${this.escapeHtml(member.username)}')"
                                style="color: var(--error-color); border-color: var(--error-color);">
                                Remove
                            </button>
                        ` : '-'}
                    </td>
                </tr>
            `).join('');

        } catch (error) {
            console.error('Failed to load members:', error);
            document.getElementById('members-list').innerHTML = 
                '<tr><td colspan="4" class="table-empty error">Failed to load members</td></tr>';
        }
    }

    // Edit tenant (stub for future implementation)
    editTenant() {
        toast.info('Edit tenant feature coming soon!');
    }

    // Delete tenant
    async deleteTenant() {
        if (!this.currentTenant) return;

        const confirmed = await modal.danger(
            `Are you sure you want to delete "${this.currentTenant.name}"? This action cannot be undone.`,
            'Delete Tenant'
        );
        if (!confirmed) return;

        try {
            await api.deleteTenant(this.currentTenant.id);
            
            this.showSuccess('Organization deleted successfully');
            this.hideDetailModal();
            await this.loadTenants();

        } catch (error) {
            console.error('Failed to delete tenant:', error);
            this.showError(error.message);
        }
    }

    // Remove member (stub for future implementation)
    async removeMember(userId) {
        const confirmed = await modal.confirm(
            'Remove this member from the organization?',
            'Remove Member'
        );
        if (!confirmed) return;

        toast.info('Remove member feature coming soon!');
    }

    // Generate slug from name
    generateSlug(name) {
        return name
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '');
    }

    // Show error message
    showError(message) {
        const errorDiv = document.getElementById('tenant-message');
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
        const successDiv = document.getElementById('tenant-success');
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
            day: 'numeric' 
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

    // ========================================
    // Add Member Modal Functions (v1.5.8)
    // ========================================

    showAddMemberModal() {
        // Save tenant ID for this modal session
        this.addMemberTenantId = this.currentTenant ? this.currentTenant.id : null;
        
        console.log('showAddMemberModal - tenant ID:', this.addMemberTenantId);
        
        this.selectedUser = null;
        document.getElementById('member-search').value = '';
        document.getElementById('member-role').value = '';
        document.getElementById('selected-user-group').style.display = 'none';
        document.getElementById('submit-add-member-btn').disabled = true;
        this.hideSearchResults();
        
        const modal = document.getElementById('add-member-modal');
        if (modal) {
            modal.style.display = 'flex';
            document.getElementById('member-search').focus();
        }
    }

    hideAddMemberModal() {
        const modal = document.getElementById('add-member-modal');
        if (modal) {
            modal.style.display = 'none';
            // Clear tenant ID
            this.addMemberTenantId = null;
        }
    }

    async searchUsers(query) {
        // Use saved tenant ID from modal session
        const tenantId = this.addMemberTenantId;
        
        if (!tenantId) {
            console.log('No tenant ID for add member modal');
            return;
        }

        console.log('Searching for user:', query, 'in tenant:', tenantId);

        try {
            const result = await api.searchTenantUsers(tenantId, query);
            console.log('Search result:', result);
            this.displaySearchResults([result]);
        } catch (error) {
            console.error('Failed to search users:', error);
            // Show error message to user
            const resultsDiv = document.getElementById('search-results');
            resultsDiv.innerHTML = '<div class="search-result-item" style="color: #dc3545;">User not found</div>';
            resultsDiv.style.display = 'block';
        }
    }

    displaySearchResults(results) {
        const resultsDiv = document.getElementById('search-results');
        
        console.log('displaySearchResults called with:', results);
        console.log('resultsDiv:', resultsDiv);
        
        if (!results || results.length === 0) {
            console.log('No results, hiding');
            resultsDiv.style.display = 'none';
            return;
        }

        // Check if results have user data
        const hasValidUser = results.some(r => r && r.user);
        if (!hasValidUser) {
            console.log('No valid user in results');
            resultsDiv.innerHTML = '<div class="search-result-item" style="color: #dc3545;">Invalid user data</div>';
            resultsDiv.style.display = 'block';
            return;
        }

        resultsDiv.innerHTML = results.map(result => {
            const user = result.user;
            const alreadyMember = result.already_member;
            
            return `
                <div class="search-result-item ${alreadyMember ? 'already-member' : ''}" 
                     onclick="${alreadyMember ? '' : `tenantsManager.selectUser(${JSON.stringify(user).replace(/"/g, '&quot;')})`}">
                    <div class="search-result-name">
                        ${this.escapeHtml(user.username)}
                        ${alreadyMember ? '<span class="search-result-badge">Already Member</span>' : ''}
                    </div>
                    <div class="search-result-email">${this.escapeHtml(user.email)}</div>
                </div>
            `;
        }).join('');

        console.log('Setting display to block, HTML:', resultsDiv.innerHTML);
        resultsDiv.style.display = 'block';
        
        // Verify it's visible
        setTimeout(() => {
            console.log('After 100ms, display:', resultsDiv.style.display);
        }, 100);
    }

    hideSearchResults() {
        console.log('hideSearchResults called');
        const resultsDiv = document.getElementById('search-results');
        if (resultsDiv) {
            resultsDiv.style.display = 'none';
        }
    }

    selectUser(user) {
        this.selectedUser = user;
        
        // Hide search results
        this.hideSearchResults();
        
        // Clear search input
        document.getElementById('member-search').value = '';
        
        // Show selected user card
        const selectedUserCard = document.getElementById('selected-user-card');
        selectedUserCard.innerHTML = `
            <div class="selected-user-info">
                <div class="selected-user-name">${this.escapeHtml(user.username)}</div>
                <div class="selected-user-email">${this.escapeHtml(user.email)}</div>
            </div>
            <div class="selected-user-remove" onclick="tenantsManager.deselectUser()">
                ✕
            </div>
        `;
        
        document.getElementById('selected-user-group').style.display = 'block';
        
        // Enable submit button if role is selected
        const roleSelect = document.getElementById('member-role');
        if (roleSelect.value) {
            document.getElementById('submit-add-member-btn').disabled = false;
        }
        
        // Update role select listener
        roleSelect.onchange = () => {
            document.getElementById('submit-add-member-btn').disabled = !roleSelect.value;
        };
    }

    deselectUser() {
        this.selectedUser = null;
        document.getElementById('selected-user-group').style.display = 'none';
        document.getElementById('submit-add-member-btn').disabled = true;
    }

    async submitAddMember() {
        const tenantId = this.addMemberTenantId;
        
        if (!this.selectedUser || !tenantId) {
            console.log('No selected user or tenant ID');
            return;
        }

        const role = document.getElementById('member-role').value;
        if (!role) {
            this.showError('Please select a role');
            return;
        }

        const submitBtn = document.getElementById('submit-add-member-btn');
        submitBtn.disabled = true;
        submitBtn.textContent = 'Adding...';

        try {
            await api.addTenantMember(
                tenantId,
                { user_id: this.selectedUser.id },
                role
            );
            
            this.showSuccess('Member added successfully');
            this.hideAddMemberModal();
            
            // Reload members list if tenant detail modal is still open
            if (this.currentTenant && this.currentTenant.id === tenantId) {
                await this.loadMembers(tenantId);
            }
            
        } catch (error) {
            console.error('Failed to add member:', error);
            this.showError(error.message || 'Failed to add member');
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Add Member';
        }
    }

    async removeMember(tenantId, userId, username) {
        const confirmed = await modal.confirm(
            `Remove ${username} from this tenant?`,
            'Remove Member'
        );
        if (!confirmed) return;

        try {
            await api.removeTenantMember(tenantId, userId);
            this.showSuccess('Member removed successfully');
            await this.loadMembers(tenantId);
        } catch (error) {
            console.error('Failed to remove member:', error);
            this.showError(error.message || 'Failed to remove member');
        }
    }

    async updateMemberRole(tenantId, userId, username, currentRole) {
        const newRole = prompt(`Change role for ${username}\nCurrent: ${currentRole}\n\nEnter new role (viewer, member, admin):`);
        
        if (!newRole || newRole === currentRole) {
            return;
        }

        if (!['viewer', 'member', 'admin'].includes(newRole.toLowerCase())) {
            this.showError('Invalid role. Must be viewer, member, or admin');
            return;
        }

        try {
            await api.updateTenantMember(tenantId, userId, newRole.toLowerCase());
            this.showSuccess('Member role updated successfully');
            await this.loadMembers(tenantId);
        } catch (error) {
            console.error('Failed to update member role:', error);
            this.showError(error.message || 'Failed to update role');
        }
    }
}

// Initialize on page load
let tenantsManager;
document.addEventListener('DOMContentLoaded', () => {
    tenantsManager = new TenantsManager();
});



