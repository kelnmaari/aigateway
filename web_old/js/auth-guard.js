// Auth Guard - Protects pages from unauthorized access
// Include this script at the TOP of every protected page (before other scripts)

(function() {
    'use strict';
    
    // List of public pages that don't require authentication
    const PUBLIC_PAGES = [
        '/',
        '/login.html',
        '/register.html',
        '/bootstrap.html',
        '/web/login.html',      // Legacy paths for backward compatibility
        '/web/register.html',
        '/web/bootstrap.html'
    ];
    
    // Check if current page is public
    function isPublicPage() {
        const currentPath = window.location.pathname;
        // Exact match only - prevents false positives like /admin.html matching /
        return PUBLIC_PAGES.includes(currentPath);
    }
    
    // Check if user is authenticated
    function isAuthenticated() {
        const token = localStorage.getItem('access_token');
        console.log('Auth Guard: Checking token', { 
            hasToken: !!token, 
            tokenLength: token ? token.length : 0,
            currentPath: window.location.pathname 
        });
        return !!token;
    }
    
    // Redirect to login
    function redirectToLogin() {
        // Save intended destination
        const returnUrl = window.location.href;
        console.warn('Auth Guard: Redirecting to login', { 
            returnUrl, 
            currentPath: window.location.pathname 
        });
        localStorage.setItem('return_url', returnUrl);
        
        // Redirect to login
        window.location.href = '/login.html';
    }
    
    // Main auth guard logic
    function checkAuth() {
        const currentPath = window.location.pathname;
        const isPublic = isPublicPage();
        
        console.log('Auth Guard: Check started', { 
            currentPath, 
            isPublic,
            publicPages: PUBLIC_PAGES 
        });
        
        // Skip check for public pages
        if (isPublic) {
            console.log('Auth Guard: Public page, skipping check');
            return;
        }
        
        // Check authentication
        if (!isAuthenticated()) {
            console.warn('Auth Guard: Not authenticated, redirecting to login');
            redirectToLogin();
            return;
        }
        
        console.log('Auth Guard: Authentication check passed ✅');
    }
    
    // Run auth check immediately
    checkAuth();
    
    // Also check when page becomes visible (in case token was cleared in another tab)
    document.addEventListener('visibilitychange', function() {
        if (!document.hidden) {
            checkAuth();
        }
    });
    
    // Check token validity periodically (every 60 seconds)
    setInterval(function() {
        if (!isPublicPage() && !isAuthenticated()) {
            console.warn('Auth Guard: Token lost, redirecting to login');
            redirectToLogin();
        }
    }, 60000); // 60 seconds
    
})();

