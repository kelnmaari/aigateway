// Navigation Component - Global navigation bar
// Include this in every page after auth-guard.js

(function() {
    'use strict';
    
    // Create navigation HTML
    function createNavigation() {
        const nav = document.createElement('div');
        nav.className = 'global-nav';
        nav.innerHTML = `
            <style>
                .global-nav {
                    background: rgba(0, 0, 0, 0.2);
                    padding: 0;
                    margin: 0;
                    position: sticky;
                    top: 0;
                    z-index: 100;
                }
                .global-nav-inner {
                    max-width: 1400px;
                    margin: 0 auto;
                    display: flex;
                    align-items: center;
                    gap: 5px;
                    padding: 10px 20px;
                    flex-wrap: wrap;
                }
                .nav-link {
                    color: white;
                    text-decoration: none;
                    padding: 8px 16px;
                    border-radius: 6px;
                    font-size: 14px;
                    font-weight: 500;
                    transition: background 0.2s;
                    display: inline-flex;
                    align-items: center;
                    gap: 6px;
                }
                .nav-link:hover {
                    background: rgba(255, 255, 255, 0.1);
                }
                .nav-link.active {
                    background: rgba(255, 255, 255, 0.2);
                }
                .nav-spacer {
                    flex: 1;
                }
                .nav-user {
                    color: white;
                    font-size: 14px;
                    margin-right: 10px;
                }
                @media (max-width: 768px) {
                    .global-nav-inner {
                        padding: 10px;
                    }
                    .nav-link {
                        padding: 6px 12px;
                        font-size: 13px;
                    }
                }
            </style>
            <div class="global-nav-inner">
                <a href="/dashboard.html" class="nav-link" data-page="dashboard">📊 Dashboard</a>
                <a href="/chat.html" class="nav-link" data-page="chat">💬 Chat</a>
                <a href="/api-keys.html" class="nav-link" data-page="api-keys">🔑 API Keys</a>
                <a href="/files.html" class="nav-link" data-page="files">📁 Files</a>
                <a href="/tenants.html" class="nav-link" data-page="tenants">🏢 Tenants</a>
                <a href="/usage.html" class="nav-link" data-page="usage">📈 Usage</a>
                <a href="/mcp.html" class="nav-link" data-page="mcp">🔌 MCP</a>
                <a href="/about.html" class="nav-link" data-page="about">ℹ️ About</a>
                <div class="nav-spacer"></div>
                <span class="nav-user" id="nav-username">Loading...</span>
                <a href="/profile.html" class="nav-link" data-page="profile">👤 Profile</a>
                <a href="/admin.html" class="nav-link" data-page="admin" id="nav-admin" style="display:none;">⚙️ Admin</a>
            </div>
        `;
        
        // Insert at the top of body
        if (document.body.firstChild) {
            document.body.insertBefore(nav, document.body.firstChild);
        } else {
            document.body.appendChild(nav);
        }
        
        // Highlight current page
        const currentPage = window.location.pathname.split('/').pop().replace('.html', '');
        const activeLink = nav.querySelector(`[data-page="${currentPage}"]`);
        if (activeLink) {
            activeLink.classList.add('active');
        }
        
        // Load user info and show admin link if needed
        loadUserInfo();
    }
    
    async function loadUserInfo() {
        const token = localStorage.getItem('access_token');
        if (!token) return;
        
        try {
            const response = await fetch('/api/auth/me', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            
            if (response.ok) {
                const user = await response.json();
                const usernameEl = document.getElementById('nav-username');
                if (usernameEl) {
                    usernameEl.textContent = user.username;
                }
                
                // Show admin link if user is admin
                if (user.is_admin) {
                    const adminLink = document.getElementById('nav-admin');
                    if (adminLink) {
                        adminLink.style.display = 'inline-flex';
                    }
                }
            }
        } catch (error) {
            console.error('Failed to load user info:', error);
        }
    }
    
    // Don't show nav on login/register pages
    const publicPages = ['/login.html', '/register.html', '/bootstrap.html'];
    if (!publicPages.includes(window.location.pathname)) {
        // Wait for DOM to be ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', createNavigation);
        } else {
            createNavigation();
        }
    }
})();



