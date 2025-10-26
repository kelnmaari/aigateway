// MCP Servers Catalog
class MCPCatalog {
    constructor() {
        this.servers = [];
        this.categories = [];
        this.currentPage = 1;
        this.pageSize = 12;
        this.totalServers = 0;
        this.filters = {
            search: '',
            category: '',
            sortBy: 'created_at',
            sortOrder: 'desc'
        };

        this.init();
    }

    async init() {
        await this.loadCategories();
        await this.loadServers();
        this.setupEventListeners();
    }

    async loadCategories() {
        try {
            const response = await fetch(`${api.baseURL}/api/mcp/categories`);
            const data = await response.json();
            this.categories = data.categories || [];
            this.renderCategoryFilter();
        } catch (error) {
            console.error('Failed to load categories:', error);
        }
    }

    async loadServers() {
        const params = new URLSearchParams({
            limit: this.pageSize,
            offset: (this.currentPage - 1) * this.pageSize,
            sort_by: this.filters.sortBy,
            sort_order: this.filters.sortOrder,
            active_only: 'true'
        });

        if (this.filters.category) {
            params.append('category', this.filters.category);
        }

        if (this.filters.search) {
            params.append('search', this.filters.search);
        }

        try {
            const response = await fetch(`${api.baseURL}/api/mcp/servers?${params}`);
            const data = await response.json();
            
            this.servers = data.servers || [];
            this.totalServers = data.total || 0;
            
            this.renderServers();
            this.updateStats();
            this.updatePagination();
        } catch (error) {
            console.error('Failed to load servers:', error);
            this.showError('Failed to load MCP servers');
        }
    }

    renderCategoryFilter() {
        const select = document.getElementById('category-filter');
        const currentValue = select.value;
        
        select.innerHTML = '<option value="">All Categories</option>';
        
        this.categories.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat;
            option.textContent = this.formatCategory(cat);
            select.appendChild(option);
        });
        
        if (currentValue) {
            select.value = currentValue;
        }
    }

    renderServers() {
        const grid = document.getElementById('servers-grid');
        
        if (this.servers.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-inbox fa-3x"></i>
                    <p>No MCP servers found</p>
                </div>
            `;
            return;
        }

        grid.innerHTML = this.servers.map(server => `
            <div class="server-card" data-id="${server.id}">
                <div class="server-card-header">
                    <div class="server-icon">
                        <i class="fas ${this.getCategoryIcon(server.category)}"></i>
                    </div>
                    <span class="server-category">${this.formatCategory(server.category)}</span>
                </div>
                
                <h3 class="server-name">${this.escapeHtml(server.name)}</h3>
                <p class="server-description">${this.escapeHtml(server.description)}</p>
                
                <div class="server-tags">
                    ${(server.tags || []).slice(0, 3).map(tag => 
                        `<span class="tag">${this.escapeHtml(tag)}</span>`
                    ).join('')}
                    ${server.tags && server.tags.length > 3 ? 
                        `<span class="tag">+${server.tags.length - 3}</span>` : ''}
                </div>
                
                <div class="server-card-footer">
                    <button class="btn btn-primary btn-sm" onclick="mcpCatalog.showServerDetails('${server.id}')">
                        <i class="fas fa-info-circle"></i> Details
                    </button>
                    ${server.github_url ? 
                        `<a href="${server.github_url}" target="_blank" class="btn btn-icon" title="GitHub">
                            <i class="fab fa-github"></i>
                        </a>` : ''}
                    ${server.website_url ? 
                        `<a href="${server.website_url}" target="_blank" class="btn btn-icon" title="Website">
                            <i class="fas fa-external-link-alt"></i>
                        </a>` : ''}
                </div>
            </div>
        `).join('');
    }

    async showServerDetails(serverId) {
        try {
            const response = await fetch(`${api.baseURL}/api/mcp/servers/${serverId}`);
            const server = await response.json();
            
            document.getElementById('modal-title').textContent = server.name;
            document.getElementById('modal-body').innerHTML = `
                <div class="server-details">
                    <div class="detail-section">
                        <div class="detail-header">
                            <span class="server-category-badge ${server.category}">
                                <i class="fas ${this.getCategoryIcon(server.category)}"></i>
                                ${this.formatCategory(server.category)}
                            </span>
                        </div>
                    </div>

                    <div class="detail-section">
                        <h4><i class="fas fa-info-circle"></i> Description</h4>
                        <p>${this.escapeHtml(server.description)}</p>
                    </div>

                    <div class="detail-section">
                        <h4><i class="fas fa-download"></i> Installation Guide</h4>
                        <pre class="installation-guide">${this.escapeHtml(server.installation_guide)}</pre>
                    </div>

                    ${server.tags && server.tags.length > 0 ? `
                        <div class="detail-section">
                            <h4><i class="fas fa-tags"></i> Tags</h4>
                            <div class="server-tags">
                                ${server.tags.map(tag => 
                                    `<span class="tag">${this.escapeHtml(tag)}</span>`
                                ).join('')}
                            </div>
                        </div>
                    ` : ''}

                    <div class="detail-section">
                        <h4><i class="fas fa-link"></i> Links</h4>
                        <div class="links-container">
                            ${server.website_url ? 
                                `<a href="${server.website_url}" target="_blank" class="btn btn-secondary">
                                    <i class="fas fa-globe"></i> Website
                                </a>` : ''}
                            ${server.github_url ? 
                                `<a href="${server.github_url}" target="_blank" class="btn btn-secondary">
                                    <i class="fab fa-github"></i> GitHub
                                </a>` : ''}
                        </div>
                    </div>

                    <div class="detail-section detail-meta">
                        <small>Added: ${this.formatDate(server.created_at)}</small>
                    </div>
                </div>
            `;
            
            document.getElementById('server-modal').classList.add('show');
        } catch (error) {
            console.error('Failed to load server details:', error);
            this.showError('Failed to load server details');
        }
    }

    closeModal() {
        document.getElementById('server-modal').classList.remove('show');
    }

    setupEventListeners() {
        // Search
        const searchInput = document.getElementById('search-input');
        let searchTimeout;
        searchInput.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                this.filters.search = e.target.value;
                this.currentPage = 1;
                this.loadServers();
            }, 300);
        });

        // Category filter
        document.getElementById('category-filter').addEventListener('change', (e) => {
            this.filters.category = e.target.value;
            this.currentPage = 1;
            this.loadServers();
        });

        // Sort
        document.getElementById('sort-by').addEventListener('change', (e) => {
            this.filters.sortBy = e.target.value;
            this.currentPage = 1;
            this.loadServers();
        });

        // Clear filters
        document.getElementById('clear-filters').addEventListener('click', () => {
            this.filters = {
                search: '',
                category: '',
                sortBy: 'created_at',
                sortOrder: 'desc'
            };
            document.getElementById('search-input').value = '';
            document.getElementById('category-filter').value = '';
            document.getElementById('sort-by').value = 'created_at';
            this.currentPage = 1;
            this.loadServers();
        });

        // Pagination
        document.getElementById('prev-page').addEventListener('click', () => {
            if (this.currentPage > 1) {
                this.currentPage--;
                this.loadServers();
                window.scrollTo({ top: 0, behavior: 'smooth' });
            }
        });

        document.getElementById('next-page').addEventListener('click', () => {
            const totalPages = Math.ceil(this.totalServers / this.pageSize);
            if (this.currentPage < totalPages) {
                this.currentPage++;
                this.loadServers();
                window.scrollTo({ top: 0, behavior: 'smooth' });
            }
        });

        // Modal close on background click
        document.getElementById('server-modal').addEventListener('click', (e) => {
            if (e.target.id === 'server-modal') {
                this.closeModal();
            }
        });

        // Close modal on Escape
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeModal();
            }
        });
    }

    updateStats() {
        const count = document.getElementById('results-count');
        const start = (this.currentPage - 1) * this.pageSize + 1;
        const end = Math.min(this.currentPage * this.pageSize, this.totalServers);
        
        if (this.totalServers === 0) {
            count.textContent = 'No servers found';
        } else {
            count.textContent = `Showing ${start}-${end} of ${this.totalServers} servers`;
        }
    }

    updatePagination() {
        const pagination = document.getElementById('pagination');
        const totalPages = Math.ceil(this.totalServers / this.pageSize);
        
        if (totalPages <= 1) {
            pagination.style.display = 'none';
            return;
        }
        
        pagination.style.display = 'flex';
        
        const prevBtn = document.getElementById('prev-page');
        const nextBtn = document.getElementById('next-page');
        const pageInfo = document.getElementById('page-info');
        
        prevBtn.disabled = this.currentPage === 1;
        nextBtn.disabled = this.currentPage === totalPages;
        pageInfo.textContent = `Page ${this.currentPage} of ${totalPages}`;
    }

    getCategoryIcon(category) {
        const icons = {
            development: 'fa-code',
            productivity: 'fa-rocket',
            database: 'fa-database',
            cloud: 'fa-cloud',
            ai: 'fa-brain',
            other: 'fa-box'
        };
        return icons[category] || 'fa-box';
    }

    formatCategory(category) {
        return category.charAt(0).toUpperCase() + category.slice(1);
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', { 
            year: 'numeric', 
            month: 'short', 
            day: 'numeric' 
        });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    showError(message) {
        const grid = document.getElementById('servers-grid');
        grid.innerHTML = `
            <div class="error-state">
                <i class="fas fa-exclamation-circle fa-3x"></i>
                <p>${message}</p>
                <button class="btn btn-primary" onclick="mcpCatalog.loadServers()">
                    <i class="fas fa-redo"></i> Retry
                </button>
            </div>
        `;
    }
}

// Initialize catalog
const mcpCatalog = new MCPCatalog();

