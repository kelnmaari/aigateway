// Universal Top Navigation Component
class TopNavbar {
    constructor() {
        this.currentUser = null;
        this.currentPage = this.detectCurrentPage();
    }

    detectCurrentPage() {
        const path = window.location.pathname;
        if (path.includes('chat.html')) return 'chat';
        if (path.includes('dashboard.html')) return 'dashboard';
        if (path.includes('files.html')) return 'files';
        if (path.includes('admin.html')) return 'admin';
        if (path.includes('profile.html')) return 'profile';
        if (path.includes('tenants.html')) return 'tenants';
        if (path.includes('api-keys.html')) return 'api-keys';
        if (path.includes('usage.html')) return 'usage';
        if (path.includes('mcp.html')) return 'mcp';
        if (path.includes('about.html')) return 'about';
        return 'unknown';
    }

    async init() {
        try {
            // Try to get current user
            const response = await api.request(`${api.baseURL}/api/users/me`);
            if (response.ok) {
                this.currentUser = await response.json();
            }
        } catch (error) {
            console.log('User not authenticated or error fetching user:', error);
        }

        this.render();
        this.attachEventListeners();
    }

    render() {
        const navHTML = `
            <nav class="top-nav">
                <div class="nav-brand">
                    <h1>🤖 Ollama Proxy</h1>
                </div>
                
                <div class="nav-menu">
                    <a href="/chat.html" class="nav-link ${this.currentPage === 'chat' ? 'active' : ''}">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" stroke-width="2"/>
                        </svg>
                        Chat
                    </a>
                    <a href="/dashboard.html" class="nav-link ${this.currentPage === 'dashboard' ? 'active' : ''}">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <rect x="3" y="3" width="7" height="7" stroke-width="2"/>
                            <rect x="14" y="3" width="7" height="7" stroke-width="2"/>
                            <rect x="14" y="14" width="7" height="7" stroke-width="2"/>
                            <rect x="3" y="14" width="7" height="7" stroke-width="2"/>
                        </svg>
                        Dashboard
                    </a>
                    <a href="/files.html" class="nav-link ${this.currentPage === 'files' ? 'active' : ''}">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z" stroke-width="2"/>
                            <path d="M13 2v7h7" stroke-width="2"/>
                        </svg>
                        Files
                    </a>
                    ${this.currentUser && this.currentUser.is_admin ? `
                        <a href="/admin.html" class="nav-link ${this.currentPage === 'admin' ? 'active' : ''}">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" stroke-width="2"/>
                                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" stroke-width="2"/>
                            </svg>
                            Admin
                        </a>
                    ` : ''}
                </div>

                ${this.currentUser ? `
                    <div class="nav-user">
                        <div class="user-dropdown">
                            <button class="user-button" id="navbar-user-button">
                                <div class="user-avatar">${(this.currentUser.full_name || this.currentUser.username || 'U').charAt(0).toUpperCase()}</div>
                                <span id="navbar-user-name">${this.currentUser.full_name || this.currentUser.username}</span>
                            </button>
                            <div class="dropdown-menu" id="navbar-user-dropdown">
                                <a href="/profile.html" class="dropdown-item">Profile Settings</a>
                                <a href="/usage.html" class="dropdown-item">Usage & Billing</a>
                                <a href="/about.html" class="dropdown-item">О Системе</a>
                                <hr class="dropdown-divider">
                                <button class="dropdown-item" id="navbar-logout-btn">Logout</button>
                            </div>
                        </div>
                    </div>
                ` : `
                    <div class="nav-user">
                        <a href="/login.html" class="btn btn-secondary">Login</a>
                    </div>
                `}
            </nav>
        `;

        // Insert navbar at the beginning of body
        document.body.insertAdjacentHTML('afterbegin', navHTML);
    }

    attachEventListeners() {
        // User dropdown toggle
        const userButton = document.getElementById('navbar-user-button');
        const userDropdown = document.getElementById('navbar-user-dropdown');
        
        if (userButton && userDropdown) {
            console.log('Navbar: Attaching user dropdown listeners');
            
            userButton.addEventListener('click', (e) => {
                e.stopPropagation();
                console.log('Navbar: User button clicked, toggling dropdown');
                userDropdown.classList.toggle('show');
            });

            // Close dropdown when clicking outside
            document.addEventListener('click', (e) => {
                if (!e.target.closest('.user-dropdown')) {
                    userDropdown.classList.remove('show');
                }
            });
        } else {
            console.warn('Navbar: User dropdown elements not found', {
                userButton: !!userButton,
                userDropdown: !!userDropdown
            });
        }

        // Logout button
        const logoutBtn = document.getElementById('navbar-logout-btn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', () => {
                api.logout();
            });
        }
    }
}

// Auto-initialize navbar on page load
document.addEventListener('DOMContentLoaded', async () => {
    const navbar = new TopNavbar();
    await navbar.init();
});

