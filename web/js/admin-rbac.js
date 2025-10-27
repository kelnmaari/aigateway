// RBAC Management JavaScript
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)

let selectedRole = null;
let allPermissions = [];
let allRoles = [];
let allUsers = [];
let allTenants = [];

// Initialize
$(document).ready(async function() {
    await checkAuth();
    initEventListeners();
    await loadInitialData();
});

async function checkAuth() {
    try {
        // Use api.js to verify authentication (will auto-redirect if failed)
        await api.getCurrentUser();
    } catch (error) {
        console.error('Auth check failed:', error);
        // api.js will handle redirect to login
    }
}

function initEventListeners() {
    $('#createRoleBtn').on('click', () => showRoleModal());
    $('#saveRoleBtn').on('click', saveRole);
    $('#editRoleBtn').on('click', () => showRoleModal(selectedRole));
    $('#deleteRoleBtn').on('click', deleteRole);
    $('#roleScope').on('change', handleScopeChange);
    $('#userSelect').on('change', handleUserSelect);
    $('#assignRoleBtn').on('click', assignRoleToUser);
    
    // Tab switch listeners
    $('#user-roles-tab').on('shown.bs.tab', loadUsersAndTenants);
    $('#permissions-tab').on('shown.bs.tab', loadPermissions);
}

// Load initial data
async function loadInitialData() {
    await Promise.all([
        loadAllPermissions(),
        loadRoles()
    ]);
}

// ============================================
// Permissions
// ============================================

async function loadAllPermissions() {
    try {
        const data = await api.getRBACPermissions();
        allPermissions = data.permissions || [];
        return allPermissions;
    } catch (error) {
        console.error('Error loading permissions:', error);
        showAlert('Failed to load permissions', 'danger');
        return [];
    }
}

function loadPermissions() {
    const container = $('#permissionsListContainer');
    
    if (allPermissions.length === 0) {
        container.html('<p class="text-muted">No permissions available</p>');
        return;
    }
    
    // Group by resource
    const groupedPermissions = {};
    allPermissions.forEach(perm => {
        if (!groupedPermissions[perm.resource]) {
            groupedPermissions[perm.resource] = [];
        }
        groupedPermissions[perm.resource].push(perm);
    });
    
    let html = '';
    Object.keys(groupedPermissions).sort().forEach(resource => {
        html += `
            <div class="card mb-3">
                <div class="card-header">
                    <h6 class="mb-0"><i class="fas fa-cube"></i> ${resource}</h6>
                </div>
                <div class="card-body">
                    <table class="table table-sm table-hover">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Action</th>
                                <th>Scope</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
        `;
        
        groupedPermissions[resource].forEach(perm => {
            html += `
                <tr>
                    <td><code>${perm.name}</code></td>
                    <td><span class="badge bg-info">${perm.action}</span></td>
                    <td><span class="badge bg-secondary">${perm.scope}</span></td>
                    <td>${perm.description}</td>
                </tr>
            `;
        });
        
        html += `
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    });
    
    container.html(html);
}

// ============================================
// Roles
// ============================================

async function loadRoles() {
    try {
        const data = await api.getRBACRoles(true);
        allRoles = data.roles || [];
        displayRoles(allRoles);
    } catch (error) {
        console.error('Error loading roles:', error);
        showAlert('Failed to load roles', 'danger');
    }
}

function displayRoles(roles) {
    const container = $('#rolesListContainer');
    
    if (roles.length === 0) {
        container.html('<p class="text-muted">No roles found</p>');
        return;
    }
    
    let html = '';
    roles.forEach(role => {
        const typeClass = role.type === 'system' ? 'system-role-badge' : 'custom-role-badge';
        const permCount = role.permissions ? role.permissions.length : 0;
        
        html += `
            <div class="role-card card ${selectedRole && selectedRole.id === role.id ? 'selected' : ''}" 
                 data-role-id="${role.id}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h6 class="mb-1">${role.display_name}</h6>
                            <small class="text-muted">${role.name}</small>
                        </div>
                        <span class="badge ${typeClass}">${role.type}</span>
                    </div>
                    <div class="mt-2">
                        <small>
                            <i class="fas fa-shield-alt"></i> ${permCount} permissions
                            ${role.scope === 'tenant' ? '<i class="fas fa-building ms-2"></i> Tenant' : ''}
                        </small>
                    </div>
                </div>
            </div>
        `;
    });
    
    container.html(html);
    
    // Add click handlers
    $('.role-card').on('click', function() {
        const roleId = $(this).data('role-id');
        const role = allRoles.find(r => r.id === roleId);
        if (role) {
            selectRole(role);
        }
    });
}

function selectRole(role) {
    selectedRole = role;
    $('.role-card').removeClass('selected');
    $(`.role-card[data-role-id="${role.id}"]`).addClass('selected');
    
    displayRoleDetails(role);
    $('#roleSelectPrompt').hide();
    $('#roleDetailsCard').show();
}

function displayRoleDetails(role) {
    $('#roleDetailsTitle').text(role.display_name);
    
    const permissions = role.permissions || [];
    let permissionsHtml = '';
    
    if (permissions.length === 0) {
        permissionsHtml = '<p class="text-muted">No permissions assigned</p>';
    } else {
        permissionsHtml = '<div class="d-flex flex-wrap">';
        permissions.forEach(perm => {
            permissionsHtml += `
                <span class="badge bg-primary permission-badge">
                    ${perm.name}
                </span>
            `;
        });
        permissionsHtml += '</div>';
    }
    
    const html = `
        <div class="row">
            <div class="col-md-6">
                <p><strong>Name:</strong> <code>${role.name}</code></p>
            </div>
            <div class="col-md-6">
                <p><strong>Type:</strong> <span class="badge ${role.type === 'system' ? 'bg-secondary' : 'bg-success'}">${role.type}</span></p>
            </div>
        </div>
        <div class="row">
            <div class="col-md-6">
                <p><strong>Scope:</strong> <span class="badge bg-info">${role.scope}</span></p>
            </div>
            ${role.tenant_id ? `<div class="col-md-6"><p><strong>Tenant ID:</strong> ${role.tenant_id}</p></div>` : ''}
        </div>
        <div class="row">
            <div class="col-12">
                <p><strong>Description:</strong></p>
                <p>${role.description || '<em class="text-muted">No description</em>'}</p>
            </div>
        </div>
        <hr>
        <div class="row">
            <div class="col-12">
                <h6>Permissions (${permissions.length})</h6>
                ${permissionsHtml}
            </div>
        </div>
    `;
    
    $('#roleDetailsBody').html(html);
    
    // Disable delete for system roles
    if (role.type === 'system') {
        $('#deleteRoleBtn').prop('disabled', true).attr('title', 'Cannot delete system roles');
        $('#editRoleBtn').prop('disabled', true).attr('title', 'Cannot edit system roles');
    } else {
        $('#deleteRoleBtn').prop('disabled', false).attr('title', '');
        $('#editRoleBtn').prop('disabled', false).attr('title', '');
    }
}

// ============================================
// Create/Edit Role Modal
// ============================================

async function showRoleModal(role = null) {
    const modal = new bootstrap.Modal(document.getElementById('roleModal'));
    
    if (role) {
        // Edit mode
        $('#roleModalTitle').text('Edit Role');
        $('#roleId').val(role.id);
        $('#roleName').val(role.name).prop('disabled', true); // Name cannot be changed
        $('#roleDisplayName').val(role.display_name);
        $('#roleDescription').val(role.description);
        $('#roleScope').val(role.scope).prop('disabled', role.type === 'system');
        
        if (role.scope === 'tenant' && role.tenant_id) {
            $('#tenantFieldContainer').show();
            $('#roleTenant').val(role.tenant_id);
        }
    } else {
        // Create mode
        $('#roleModalTitle').text('Create Role');
        $('#roleForm')[0].reset();
        $('#roleId').val('');
        $('#roleName').prop('disabled', false);
        $('#roleScope').prop('disabled', false);
        $('#tenantFieldContainer').hide();
    }
    
    // Load permissions checkboxes
    await loadPermissionsGrid(role);
    
    modal.show();
}

async function loadPermissionsGrid(role = null) {
    if (allPermissions.length === 0) {
        await loadAllPermissions();
    }
    
    const grid = $('#permissionsGrid');
    const rolePermissionIds = role && role.permissions 
        ? role.permissions.map(p => p.id)
        : [];
    
    // Group by resource
    const groupedPermissions = {};
    allPermissions.forEach(perm => {
        if (!groupedPermissions[perm.resource]) {
            groupedPermissions[perm.resource] = [];
        }
        groupedPermissions[perm.resource].push(perm);
    });
    
    let html = '';
    Object.keys(groupedPermissions).sort().forEach(resource => {
        html += `<div class="col-12 mb-2"><strong>${resource}</strong></div>`;
        groupedPermissions[resource].forEach(perm => {
            const checked = rolePermissionIds.includes(perm.id) ? 'checked' : '';
            html += `
                <div class="permission-item">
                    <input type="checkbox" class="form-check-input permission-checkbox" 
                           id="perm_${perm.id}" value="${perm.id}" ${checked}>
                    <label class="form-check-label" for="perm_${perm.id}">
                        ${perm.action}
                    </label>
                </div>
            `;
        });
    });
    
    grid.html(html);
}

function handleScopeChange() {
    const scope = $('#roleScope').val();
    if (scope === 'tenant') {
        $('#tenantFieldContainer').show();
        loadTenantsForSelect();
    } else {
        $('#tenantFieldContainer').hide();
    }
}

async function saveRole() {
    const roleId = $('#roleId').val();
    const isEdit = !!roleId;
    
    if (isEdit) {
        await updateRole(roleId);
    } else {
        await createRole();
    }
}

async function createRole() {
    const selectedPermissions = $('.permission-checkbox:checked').map(function() {
        return $(this).val();
    }).get();
    
    const data = {
        name: $('#roleName').val(),
        display_name: $('#roleDisplayName').val(),
        description: $('#roleDescription').val(),
        scope: $('#roleScope').val(),
        permissions: selectedPermissions
    };
    
    if (data.scope === 'tenant') {
        const tenantId = $('#roleTenant').val();
        if (tenantId) {
            data.tenant_id = tenantId;
        }
    }
    
    try {
        await api.createRBACRole(data);
        showAlert('Role created successfully', 'success');
        bootstrap.Modal.getInstance(document.getElementById('roleModal')).hide();
        await loadRoles();
    } catch (error) {
        console.error('Error creating role:', error);
        showAlert(error.message, 'danger');
    }
}

async function updateRole(roleId) {
    const data = {
        display_name: $('#roleDisplayName').val(),
        description: $('#roleDescription').val()
    };
    
    try {
        await api.updateRBACRole(roleId, data);
        
        // Update permissions separately
        await updateRolePermissions(roleId);
        
        showAlert('Role updated successfully', 'success');
        bootstrap.Modal.getInstance(document.getElementById('roleModal')).hide();
        await loadRoles();
        
        // Refresh details if this was selected role
        if (selectedRole && selectedRole.id === roleId) {
            const updated = allRoles.find(r => r.id === roleId);
            if (updated) {
                selectRole(updated);
            }
        }
    } catch (error) {
        console.error('Error updating role:', error);
        showAlert(error.message, 'danger');
    }
}

async function updateRolePermissions(roleId) {
    // Get current and new permissions
    const currentPerms = selectedRole.permissions ? selectedRole.permissions.map(p => p.id) : [];
    const newPerms = $('.permission-checkbox:checked').map(function() {
        return $(this).val();
    }).get();
    
    // Determine what to add and remove
    const toAdd = newPerms.filter(id => !currentPerms.includes(id));
    const toRemove = currentPerms.filter(id => !newPerms.includes(id));
    
    // Add new permissions
    for (const permId of toAdd) {
        try {
            await api.addRolePermission(roleId, permId);
        } catch (error) {
            console.error(`Failed to add permission ${permId}:`, error);
        }
    }
    
    // Remove permissions
    for (const permId of toRemove) {
        try {
            await api.removeRolePermission(roleId, permId);
        } catch (error) {
            console.error(`Failed to remove permission ${permId}:`, error);
        }
    }
}

async function deleteRole() {
    if (!selectedRole) return;
    
    if (!confirm(`Are you sure you want to delete role "${selectedRole.display_name}"?`)) {
        return;
    }
    
    try {
        await api.deleteRBACRole(selectedRole.id);
        showAlert('Role deleted successfully', 'success');
        selectedRole = null;
        $('#roleDetailsCard').hide();
        $('#roleSelectPrompt').show();
        await loadRoles();
    } catch (error) {
        console.error('Error deleting role:', error);
        showAlert(error.message, 'danger');
    }
}

// ============================================
// User Roles Management
// ============================================

async function loadUsersAndTenants() {
    await Promise.all([
        loadUsers(),
        loadTenants(),
        loadRolesForSelect()
    ]);
}

async function loadUsers() {
    try {
        const data = await api.getAdminUsers();
        allUsers = data.users || [];
        
        // Populate select
        const select = $('#userSelect');
        select.empty().append('<option value="">-- Select User --</option>');
        allUsers.forEach(user => {
            select.append(`<option value="${user.id}">${user.username} (${user.email})</option>`);
        });
    } catch (error) {
        console.error('Error loading users:', error);
    }
}

async function loadTenants() {
    try {
        const data = await api.getAdminTenants();
        allTenants = data.tenants || [];
        
        // Populate select
        const select = $('#tenantForRole, #roleTenant');
        select.empty().append('<option value="">-- Global --</option>');
        allTenants.forEach(tenant => {
            select.append(`<option value="${tenant.id}">${tenant.name}</option>`);
        });
    } catch (error) {
        console.error('Error loading tenants:', error);
    }
}

async function loadRolesForSelect() {
    if (allRoles.length === 0) {
        await loadRoles();
    }
    
    const select = $('#roleToAssign');
    select.empty().append('<option value="">-- Select Role --</option>');
    allRoles.forEach(role => {
        select.append(`<option value="${role.id}">${role.display_name}</option>`);
    });
}

async function handleUserSelect() {
    const userId = $('#userSelect').val();
    if (!userId) {
        $('#userRolesContainer').hide();
        return;
    }
    
    await loadUserRoles(userId);
    $('#userRolesContainer').show();
}

async function loadUserRoles(userId) {
    try {
        const data = await api.getUserRoles(userId, true);
        const userRoles = data.roles || [];
        displayUserRoles(userId, userRoles);
    } catch (error) {
        console.error('Error loading user roles:', error);
        showAlert('Failed to load user roles', 'danger');
    }
}

function displayUserRoles(userId, userRoles) {
    const container = $('#currentRolesList');
    
    if (userRoles.length === 0) {
        container.html('<p class="text-muted">No roles assigned</p>');
        return;
    }
    
    let html = '<div class="list-group">';
    userRoles.forEach(item => {
        const role = item.role;
        if (!role) return;
        
        const tenantLabel = item.user_role.tenant_id 
            ? `<span class="badge bg-info">Tenant: ${item.user_role.tenant_id}</span>` 
            : '<span class="badge bg-secondary">Global</span>';
        
        html += `
            <div class="list-group-item d-flex justify-content-between align-items-center">
                <div>
                    <strong>${role.display_name}</strong>
                    <br>
                    <small class="text-muted">${role.name}</small>
                    ${tenantLabel}
                </div>
                <button class="btn btn-danger btn-sm" onclick="removeUserRole('${userId}', '${role.id}', ${item.user_role.tenant_id ? `'${item.user_role.tenant_id}'` : 'null'})">
                    <i class="fas fa-times"></i>
                </button>
            </div>
        `;
    });
    html += '</div>';
    
    container.html(html);
}

async function assignRoleToUser() {
    const userId = $('#userSelect').val();
    const roleId = $('#roleToAssign').val();
    const tenantId = $('#tenantForRole').val();
    
    if (!userId || !roleId) {
        showAlert('Please select user and role', 'warning');
        return;
    }
    
    const data = {
        role_id: roleId
    };
    
    if (tenantId) {
        data.tenant_id = tenantId;
    }
    
    try {
        await api.assignUserRole(userId, data);
        showAlert('Role assigned successfully', 'success');
        await loadUserRoles(userId);
        $('#roleToAssign').val('');
        $('#tenantForRole').val('');
    } catch (error) {
        console.error('Error assigning role:', error);
        showAlert(error.message, 'danger');
    }
}

async function removeUserRole(userId, roleId, tenantId) {
    if (!confirm('Are you sure you want to remove this role?')) {
        return;
    }
    
    try {
        await api.removeUserRole(userId, roleId, tenantId);
        showAlert('Role removed successfully', 'success');
        await loadUserRoles(userId);
    } catch (error) {
        console.error('Error removing role:', error);
        showAlert(error.message, 'danger');
    }
}

// ============================================
// Utility Functions
// ============================================

function loadTenantsForSelect() {
    if (allTenants.length === 0) {
        loadTenants();
    }
}

function showAlert(message, type = 'info') {
    const alertHtml = `
        <div class="alert alert-${type} alert-dismissible fade show position-fixed top-0 end-0 m-3" 
             role="alert" style="z-index: 9999;">
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;
    $('body').append(alertHtml);
    
    setTimeout(() => {
        $('.alert').fadeOut(function() {
            $(this).remove();
        });
    }, 5000);
}

