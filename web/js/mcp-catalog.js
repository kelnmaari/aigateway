// MCP Servers Catalog - Version 1.12.4+
// Каталог популярных MCP серверов с возможностью установки

const MCP_CATALOG = [
    {
        id: 'filesystem',
        name: 'Filesystem',
        author: 'Anthropic (Official)',
        description: 'Direct filesystem access with read/write operations. Essential for file management tasks.',
        icon: 'fa-folder-open',
        category: 'filesystem',
        status: 'official',
        features: ['Read files', 'Write files', 'List directories', 'Search files'],
        command: 'npx -y @modelcontextprotocol/server-filesystem',
        args: ['/allowed/path1', '/allowed/path2'],
        stars: 1500,
        downloads: 50000,
        lastUpdated: '2025-10-20',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem',
    },
    {
        id: 'postgres',
        name: 'PostgreSQL',
        author: 'Anthropic (Official)',
        description: 'Query and manage PostgreSQL databases. Perfect for database administration and data analysis.',
        icon: 'fa-database',
        category: 'database',
        status: 'official',
        features: ['Execute queries', 'Schema inspection', 'Table management', 'Data export'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-postgres', 'postgresql://localhost/mydb'],
        stars: 1200,
        downloads: 35000,
        lastUpdated: '2025-10-18',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/postgres',
    },
    {
        id: 'brave-search',
        name: 'Brave Search',
        author: 'Anthropic (Official)',
        description: 'Web search powered by Brave Search API. Privacy-focused search with comprehensive results.',
        icon: 'fa-search',
        category: 'search',
        status: 'official',
        features: ['Web search', 'News search', 'Image search', 'Privacy-focused'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-brave-search'],
        env: {
            BRAVE_API_KEY: 'your-api-key-here'
        },
        stars: 980,
        downloads: 28000,
        lastUpdated: '2025-10-15',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search',
    },
    {
        id: 'github',
        name: 'GitHub',
        author: 'Anthropic (Official)',
        description: 'Interact with GitHub repositories, issues, and pull requests. Essential for code collaboration.',
        icon: 'fa-brands fa-github',
        category: 'api',
        status: 'official',
        features: ['Repository management', 'Issue tracking', 'PR operations', 'File operations'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-github'],
        env: {
            GITHUB_PERSONAL_ACCESS_TOKEN: 'your-token-here'
        },
        stars: 1800,
        downloads: 45000,
        lastUpdated: '2025-10-22',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/github',
    },
    {
        id: 'slack',
        name: 'Slack',
        author: 'Anthropic (Official)',
        description: 'Send messages and manage Slack workspaces. Automate team communication.',
        icon: 'fa-brands fa-slack',
        category: 'api',
        status: 'official',
        features: ['Send messages', 'Channel management', 'User lookup', 'File sharing'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-slack'],
        env: {
            SLACK_BOT_TOKEN: 'xoxb-your-token',
            SLACK_TEAM_ID: 'T01234567'
        },
        stars: 750,
        downloads: 22000,
        lastUpdated: '2025-10-10',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/slack',
    },
    {
        id: 'puppeteer',
        name: 'Puppeteer',
        author: 'Anthropic (Official)',
        description: 'Browser automation with screenshots, navigation, and DOM interaction.',
        icon: 'fa-chrome',
        category: 'tools',
        status: 'official',
        features: ['Page navigation', 'Screenshots', 'Click elements', 'Form filling'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-puppeteer'],
        stars: 1100,
        downloads: 31000,
        lastUpdated: '2025-10-12',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/puppeteer',
    },
    {
        id: 'sqlite',
        name: 'SQLite',
        author: 'Anthropic (Official)',
        description: 'Query and manage SQLite databases. Lightweight database operations.',
        icon: 'fa-database',
        category: 'database',
        status: 'official',
        features: ['Execute queries', 'Schema management', 'Data export', 'Transactions'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-sqlite', '/path/to/database.db'],
        stars: 850,
        downloads: 26000,
        lastUpdated: '2025-10-08',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/sqlite',
    },
    {
        id: 'git',
        name: 'Git',
        author: 'Anthropic (Official)',
        description: 'Read, search, and manipulate Git repositories. Essential for version control.',
        icon: 'fa-brands fa-git-alt',
        category: 'tools',
        status: 'official',
        features: ['Repository status', 'Commit history', 'Branch operations', 'Diff viewing'],
        command: 'npx',
        args: ['-y', '@modelcontextprotocol/server-git'],
        stars: 920,
        downloads: 27500,
        lastUpdated: '2025-10-16',
        documentation: 'https://github.com/modelcontextprotocol/servers/tree/main/src/git',
    },
    {
        id: 'google-drive',
        name: 'Google Drive',
        author: 'Community',
        description: 'Access and manage Google Drive files. Seamless cloud storage integration.',
        icon: 'fa-brands fa-google-drive',
        category: 'filesystem',
        status: 'community',
        features: ['File access', 'Upload/download', 'Share management', 'Search files'],
        command: 'npx',
        args: ['-y', 'mcp-server-google-drive'],
        env: {
            GOOGLE_CLIENT_ID: 'your-client-id',
            GOOGLE_CLIENT_SECRET: 'your-secret'
        },
        stars: 450,
        downloads: 12000,
        lastUpdated: '2025-10-05',
        documentation: 'https://github.com/community/mcp-server-google-drive',
    },
    {
        id: 'docker',
        name: 'Docker',
        author: 'Community',
        description: 'Manage Docker containers and images. Container orchestration made easy.',
        icon: 'fa-brands fa-docker',
        category: 'tools',
        status: 'community',
        features: ['Container management', 'Image operations', 'Logs access', 'Network management'],
        command: 'npx',
        args: ['-y', 'mcp-server-docker'],
        stars: 680,
        downloads: 18000,
        lastUpdated: '2025-10-14',
        documentation: 'https://github.com/community/mcp-server-docker',
    }
];

let currentFilter = 'all';
let currentSearch = '';
let currentSort = 'popular';

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadCatalog();
    setupEventListeners();
});

function setupEventListeners() {
    // Search
    document.getElementById('searchInput').addEventListener('input', (e) => {
        currentSearch = e.target.value.toLowerCase();
        filterAndRenderServers();
    });

    // Sort
    document.getElementById('sortSelect').addEventListener('change', (e) => {
        currentSort = e.target.value;
        filterAndRenderServers();
    });

    // Category filters
    document.querySelectorAll('.category-badge').forEach(badge => {
        badge.addEventListener('click', () => {
            document.querySelectorAll('.category-badge').forEach(b => b.classList.remove('active'));
            badge.classList.add('active');
            currentFilter = badge.dataset.category;
            filterAndRenderServers();
        });
    });
}

function loadCatalog() {
    showLoading();
    
    // Simulate API call
    setTimeout(() => {
        hideLoading();
        filterAndRenderServers();
    }, 500);
}

function filterAndRenderServers() {
    let servers = [...MCP_CATALOG];

    // Filter by category
    if (currentFilter !== 'all') {
        servers = servers.filter(s => s.category === currentFilter);
    }

    // Filter by search
    if (currentSearch) {
        servers = servers.filter(s => 
            s.name.toLowerCase().includes(currentSearch) ||
            s.description.toLowerCase().includes(currentSearch) ||
            s.features.some(f => f.toLowerCase().includes(currentSearch))
        );
    }

    // Sort
    servers.sort((a, b) => {
        switch(currentSort) {
            case 'popular':
                return b.stars - a.stars;
            case 'recent':
                return new Date(b.lastUpdated) - new Date(a.lastUpdated);
            case 'name':
                return a.name.localeCompare(b.name);
            default:
                return 0;
        }
    });

    renderServers(servers);
}

function renderServers(servers) {
    const grid = document.getElementById('serversGrid');
    const emptyState = document.getElementById('emptyState');

    if (servers.length === 0) {
        grid.style.display = 'none';
        emptyState.style.display = 'block';
        return;
    }

    grid.style.display = 'grid';
    emptyState.style.display = 'none';

    grid.innerHTML = servers.map(server => `
        <div class="server-card" data-server-id="${server.id}">
            <span class="status-badge status-${server.status}">${server.status}</span>
            
            <div class="server-header">
                <div class="server-icon">
                    <i class="fas ${server.icon}"></i>
                </div>
                <div class="server-title">
                    <h3>${server.name}</h3>
                    <div class="author">by ${server.author}</div>
                </div>
            </div>

            <div class="server-description">
                ${server.description}
            </div>

            <div class="server-meta">
                <span class="meta-tag">
                    <i class="fas fa-star"></i> ${formatNumber(server.stars)}
                </span>
                <span class="meta-tag">
                    <i class="fas fa-download"></i> ${formatNumber(server.downloads)}
                </span>
                <span class="meta-tag">
                    <i class="fas fa-clock"></i> ${formatDate(server.lastUpdated)}
                </span>
            </div>

            <div class="server-features">
                ${server.features.slice(0, 3).map(f => `
                    <span class="meta-tag"><i class="fas fa-check"></i> ${f}</span>
                `).join('')}
            </div>

            <div class="server-actions">
                <button class="btn btn-install" onclick="showInstallModal('${server.id}')">
                    <i class="fas fa-download"></i> Install
                </button>
                <button class="btn btn-details" onclick="window.open('${server.documentation}', '_blank')">
                    <i class="fas fa-book"></i> Docs
                </button>
            </div>
        </div>
    `).join('');
}

function showInstallModal(serverId) {
    const server = MCP_CATALOG.find(s => s.id === serverId);
    if (!server) return;

    const modal = new bootstrap.Modal(document.getElementById('installModal'));
    const content = document.getElementById('installContent');

    // Generate config JSON
    const config = {
        mcpServers: {
            [server.id]: {
                command: server.command,
                args: server.args,
                ...(server.env && { env: server.env })
            }
        }
    };

    content.innerHTML = `
        <h5>${server.name}</h5>
        <p class="text-muted">${server.description}</p>

        <h6 class="mt-4">Installation Steps:</h6>
        <ol>
            <li>Copy the configuration below</li>
            <li>Go to <a href="mcp.html">MCP Servers</a> page</li>
            <li>Click "Add Server" and paste the configuration</li>
            <li>Save and your server will be activated</li>
        </ol>

        <h6 class="mt-4">Configuration:</h6>
        <div class="config-preview">
            <pre>${JSON.stringify(config, null, 2)}</pre>
        </div>

        <div class="d-flex gap-2 mt-3">
            <button class="btn btn-primary flex-fill" onclick="copyConfig('${serverId}')">
                <i class="fas fa-copy"></i> Copy Configuration
            </button>
            <button class="btn btn-success flex-fill" onclick="installDirect('${serverId}')">
                <i class="fas fa-rocket"></i> Install Now
            </button>
        </div>

        ${server.env ? `
            <div class="alert alert-warning mt-3">
                <i class="fas fa-exclamation-triangle"></i>
                <strong>Note:</strong> This server requires environment variables. 
                Make sure to configure them before starting.
            </div>
        ` : ''}
    `;

    modal.show();
}

function copyConfig(serverId) {
    const server = MCP_CATALOG.find(s => s.id === serverId);
    const config = {
        mcpServers: {
            [server.id]: {
                command: server.command,
                args: server.args,
                ...(server.env && { env: server.env })
            }
        }
    };

    navigator.clipboard.writeText(JSON.stringify(config, null, 2)).then(() => {
        showNotification('Configuration copied to clipboard!', 'success');
    }).catch(err => {
        showNotification('Failed to copy configuration', 'error');
    });
}

async function installDirect(serverId) {
    const server = MCP_CATALOG.find(s => s.id === serverId);
    
    showNotification('Installing server...', 'info');

    try {
        const response = await fetch('/api/admin/mcp/servers', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${localStorage.getItem('token')}`
            },
            body: JSON.stringify({
                name: server.name,
                command: server.command,
                args: server.args,
                env: server.env || {}
            })
        });

        if (response.ok) {
            showNotification('Server installed successfully!', 'success');
            bootstrap.Modal.getInstance(document.getElementById('installModal')).hide();
            
            setTimeout(() => {
                window.location.href = 'mcp.html';
            }, 1500);
        } else {
            throw new Error('Installation failed');
        }
    } catch (error) {
        showNotification('Failed to install server. Please try manual installation.', 'error');
    }
}

function formatNumber(num) {
    if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'k';
    }
    return num.toString();
}

function formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diffDays = Math.floor((now - date) / (1000 * 60 * 60 * 24));
    
    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    return date.toLocaleDateString();
}

function showLoading() {
    document.getElementById('loadingState').style.display = 'block';
    document.getElementById('serversGrid').style.display = 'none';
    document.getElementById('emptyState').style.display = 'none';
}

function hideLoading() {
    document.getElementById('loadingState').style.display = 'none';
}

function showNotification(message, type = 'info') {
    // Create toast notification
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type === 'success' ? 'success' : type === 'error' ? 'danger' : 'info'} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${message}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;

    // Add to page
    let toastContainer = document.querySelector('.toast-container');
    if (!toastContainer) {
        toastContainer = document.createElement('div');
        toastContainer.className = 'toast-container position-fixed top-0 end-0 p-3';
        document.body.appendChild(toastContainer);
    }
    toastContainer.appendChild(toast);

    // Show toast
    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();

    // Remove after hidden
    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
}
