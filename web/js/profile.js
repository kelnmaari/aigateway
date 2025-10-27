// Profile Manager
class ProfileManager {
    constructor() {
        this.user = null;
    }

    // Initialize profile page
    async init() {
        // Check authentication
        if (!this.isAuthenticated()) {
            window.location.href = '/web/login.html';
            return;
        }

        try {
            // Load user data
            await this.loadUser();
            
            // Setup event listeners
            this.setupEventListeners();
            
            console.log('✅ Profile page initialized');
            
        } catch (error) {
            console.error('Failed to initialize profile:', error);
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
            this.populateForm();
            
            // Show admin link if user is admin
            if (this.user.is_admin) {
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

    // Populate form with user data
    populateForm() {
        // Nav user name
        const navUserName = document.getElementById('nav-user-name');
        if (navUserName) {
            navUserName.textContent = this.user.full_name || this.user.username;
        }

        // User avatar
        const avatars = document.querySelectorAll('.user-avatar');
        avatars.forEach(avatar => {
            avatar.textContent = (this.user.full_name || this.user.username).charAt(0).toUpperCase();
        });

        // Form fields
        document.getElementById('username').value = this.user.username || '';
        document.getElementById('email').value = this.user.email || '';
        document.getElementById('full_name').value = this.user.full_name || '';

        // Account info
        const status = this.user.is_active ? 'Active' : 'Inactive';
        const statusBadge = document.getElementById('account-status');
        statusBadge.textContent = status;
        statusBadge.className = this.user.is_active ? 'badge badge-success' : 'badge badge-error';

        if (this.user.created_at) {
            document.getElementById('member-since').textContent = new Date(this.user.created_at).toLocaleDateString();
        }

        if (this.user.last_login) {
            document.getElementById('last-login').textContent = this.formatDateTime(this.user.last_login);
        } else {
            document.getElementById('last-login').textContent = 'Never';
        }
    }

    // Setup event listeners
    setupEventListeners() {
        // User dropdown and logout are now handled by navbar.js component
        // No need for duplicate event listeners here

        // Profile form
        document.getElementById('profile-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.updateProfile();
        });

        // Password form
        document.getElementById('password-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.changePassword();
        });

        // Delete account button
        document.getElementById('delete-account-btn').addEventListener('click', () => {
            this.deleteAccount();
        });
    }

    // Update profile
    async updateProfile() {
        const messageDiv = document.getElementById('profile-message');
        const successDiv = document.getElementById('profile-success');
        const btn = document.getElementById('update-profile-btn');

        messageDiv.style.display = 'none';
        successDiv.style.display = 'none';

        btn.disabled = true;
        btn.textContent = 'Saving...';

        try {
            const email = document.getElementById('email').value;
            const full_name = document.getElementById('full_name').value;

            const response = await api.request(`${api.baseURL}/api/users/me`, {
                method: 'PUT',
                body: JSON.stringify({
                    email,
                    full_name
                })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to update profile');
            }

            this.user = await response.json();
            this.populateForm();

            successDiv.textContent = 'Profile updated successfully!';
            successDiv.style.display = 'block';

        } catch (error) {
            messageDiv.textContent = error.message;
            messageDiv.style.display = 'block';
        } finally {
            btn.disabled = false;
            btn.textContent = 'Save Changes';
        }
    }

    // Change password
    async changePassword() {
        const messageDiv = document.getElementById('profile-message');
        const successDiv = document.getElementById('profile-success');
        const btn = document.getElementById('change-password-btn');

        messageDiv.style.display = 'none';
        successDiv.style.display = 'none';

        const old_password = document.getElementById('old_password').value;
        const new_password = document.getElementById('new_password').value;
        const confirm_password = document.getElementById('confirm_password').value;

        // Validation
        if (new_password !== confirm_password) {
            messageDiv.textContent = 'New passwords do not match';
            messageDiv.style.display = 'block';
            return;
        }

        btn.disabled = true;
        btn.textContent = 'Changing...';

        try {
            const response = await api.request(`${api.baseURL}/api/users/me/password`, {
                method: 'POST',
                body: JSON.stringify({
                    old_password,
                    new_password
                })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to change password');
            }

            successDiv.textContent = 'Password changed successfully!';
            successDiv.style.display = 'block';

            // Clear password fields
            document.getElementById('password-form').reset();

        } catch (error) {
            messageDiv.textContent = error.message;
            messageDiv.style.display = 'block';
        } finally {
            btn.disabled = false;
            btn.textContent = 'Change Password';
        }
    }

    // Delete account
    async deleteAccount() {
        const password = prompt('⚠️ WARNING: This action cannot be undone!\n\nEnter your password to confirm account deletion:');
        
        if (!password) return;

        const confirmed = await modal.danger(
            'Are you absolutely sure? This will permanently delete your account and all associated data.',
            'Delete Account'
        );
        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/users/me`, {
                method: 'DELETE',
                body: JSON.stringify({ password })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete account');
            }

            toast.success('Your account has been deleted.');
            setTimeout(() => this.logout(), 1000);

        } catch (error) {
            toast.error(`Error: ${error.message}`);
        }
    }

    // Logout
    logout() {
        api.logout();
    }

    // Helper: Format date and time
    formatDateTime(dateString) {
        const date = new Date(dateString);
        return date.toLocaleString();
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.profileManager = new ProfileManager();
    profileManager.init();
});



