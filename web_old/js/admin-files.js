// Admin Files Management (v1.10.0)

class AdminFilesManager {
    constructor() {
        this.currentPage = 0;
        this.pageSize = 20;
        this.total = 0;
        this.filters = {
            search: '',
            type: '',
            status: ''
        };
    }

    init() {
        console.log('AdminFilesManager: Initializing...');
        this.setupEventListeners();
        this.loadStats();
        this.loadFiles();
    }

    setupEventListeners() {
        // Refresh button
        const refreshBtn = document.getElementById('files-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadStats();
                this.loadFiles();
            });
        }

        // Filters
        const searchInput = document.getElementById('admin-files-search');
        if (searchInput) {
            let searchTimeout;
            searchInput.addEventListener('input', (e) => {
                clearTimeout(searchTimeout);
                searchTimeout = setTimeout(() => {
                    this.filters.search = e.target.value;
                    this.currentPage = 0;
                    this.loadFiles();
                }, 500);
            });
        }

        const typeFilter = document.getElementById('admin-files-type');
        if (typeFilter) {
            typeFilter.addEventListener('change', (e) => {
                this.filters.type = e.target.value;
                this.currentPage = 0;
                this.loadFiles();
            });
        }

        const statusFilter = document.getElementById('admin-files-status');
        if (statusFilter) {
            statusFilter.addEventListener('change', (e) => {
                this.filters.status = e.target.value;
                this.currentPage = 0;
                this.loadFiles();
            });
        }
    }

    async loadStats() {
        try {
            const response = await api.request(`${api.baseURL}/api/admin/files/stats`);
            if (!response.ok) {
                throw new Error('Failed to load file stats');
            }

            const stats = await response.json();

            document.getElementById('admin-files-total').textContent = stats.total_files || 0;
            document.getElementById('admin-files-size').textContent = stats.total_size_human || '0 B';
            document.getElementById('admin-files-extracted').textContent = stats.extracted_count || 0;
            document.getElementById('admin-files-pending').textContent = stats.pending_count || 0;
        } catch (error) {
            console.error('Failed to load file stats:', error);
            toast.error('Failed to load file statistics');
        }
    }

    async loadFiles() {
        try {
            const params = new URLSearchParams({
                limit: this.pageSize,
                offset: this.currentPage * this.pageSize,
                sort: 'created_at',
                order: 'desc'
            });

            if (this.filters.type) {
                params.append('mime_type', this.filters.type);
            }
            if (this.filters.status) {
                params.append('extraction_status', this.filters.status);
            }

            const response = await api.request(`${api.baseURL}/api/admin/files?${params}`);
            if (!response.ok) {
                throw new Error('Failed to load files');
            }

            const data = await response.json();
            this.total = data.total || 0;

            this.renderFiles(data.files || []);
            this.renderPagination();
        } catch (error) {
            console.error('Failed to load files:', error);
            toast.error('Failed to load files');
            this.renderFiles([]);
        }
    }

    renderFiles(files) {
        const tbody = document.getElementById('admin-files-table');
        
        // Filter by search term client-side
        let filteredFiles = files;
        if (this.filters.search) {
            const search = this.filters.search.toLowerCase();
            filteredFiles = files.filter(f => 
                f.filename.toLowerCase().includes(search) ||
                (f.owner_email && f.owner_email.toLowerCase().includes(search)) ||
                (f.owner_username && f.owner_username.toLowerCase().includes(search)) ||
                f.user_id.toLowerCase().includes(search)
            );
        }

        if (filteredFiles.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="table-empty">No files found</td></tr>';
            return;
        }

        tbody.innerHTML = filteredFiles.map(file => `
            <tr>
                <td>
                    <div style="display: flex; align-items: center; gap: 0.5rem;">
                        <span>${this.getFileIcon(file.mime_type)}</span>
                        <span title="${file.filename}">${this.truncate(file.filename, 40)}</span>
                    </div>
                </td>
                <td>${this.formatOwner(file)}</td>
                <td><span class="badge badge-secondary">${this.formatMimeType(file.mime_type)}</span></td>
                <td>${this.formatBytes(file.size_bytes)}</td>
                <td>${this.getStatusBadge(file.extraction_status)}</td>
                <td>${file.word_count ? file.word_count.toLocaleString() : '-'}</td>
                <td>${new Date(file.created_at).toLocaleDateString()}</td>
                <td>
                    <button class="btn btn-sm btn-danger" onclick="adminFilesManager.deleteFile('${file.id}', '${file.filename}')" title="Delete file">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `).join('');
    }

    formatOwner(file) {
        if (file.owner_email) {
            return `<div style="font-size: 0.875rem;">
                        <div style="font-weight: 500;">${file.owner_email}</div>
                        ${file.owner_username ? `<div style="color: #666; font-size: 0.75rem;">@${file.owner_username}</div>` : ''}
                    </div>`;
        } else if (file.owner_username) {
            return `<span style="font-size: 0.875rem;">@${file.owner_username}</span>`;
        } else {
            return `<code style="font-size: 0.75rem; color: #999;">${this.truncate(file.user_id, 20)}</code>`;
        }
    }

    async deleteFile(fileId, filename) {
        const confirmed = await modal.danger(
            `Are you sure you want to delete "${filename}"? This action cannot be undone.`,
            'Delete File'
        );

        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/admin/files/${fileId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                throw new Error('Delete failed');
            }

            toast.success(`File "${filename}" deleted successfully`);
            this.loadStats();
            this.loadFiles();
        } catch (error) {
            console.error('Delete error:', error);
            toast.error('Failed to delete file');
        }
    }

    renderPagination() {
        const totalPages = Math.ceil(this.total / this.pageSize);
        const pagination = document.getElementById('admin-files-pagination');

        if (totalPages <= 1) {
            pagination.innerHTML = '';
            return;
        }

        pagination.innerHTML = `
            <button class="btn btn-secondary" ${this.currentPage === 0 ? 'disabled' : ''} onclick="adminFilesManager.previousPage()">
                Previous
            </button>
            <span style="display: flex; align-items: center; padding: 0 1rem;">
                Page ${this.currentPage + 1} of ${totalPages} (${this.total} total)
            </span>
            <button class="btn btn-secondary" ${this.currentPage >= totalPages - 1 ? 'disabled' : ''} onclick="adminFilesManager.nextPage()">
                Next
            </button>
        `;
    }

    previousPage() {
        if (this.currentPage > 0) {
            this.currentPage--;
            this.loadFiles();
        }
    }

    nextPage() {
        const totalPages = Math.ceil(this.total / this.pageSize);
        if (this.currentPage < totalPages - 1) {
            this.currentPage++;
            this.loadFiles();
        }
    }

    getFileIcon(mimeType) {
        if (mimeType.includes('pdf')) return '📕';
        if (mimeType.includes('word')) return '📘';
        if (mimeType.includes('csv')) return '📊';
        if (mimeType.includes('sheet')) return '📗';
        if (mimeType.includes('text')) return '📄';
        return '📎';
    }

    formatMimeType(mimeType) {
        const short = {
            'text/plain': 'TXT',
            'text/csv': 'CSV',
            'application/pdf': 'PDF',
            'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'DOCX',
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'XLSX'
        };
        return short[mimeType] || mimeType.split('/').pop().toUpperCase();
    }

    getStatusBadge(status) {
        const badges = {
            'completed': '<span class="badge badge-success">✓ Extracted</span>',
            'pending': '<span class="badge badge-warning">⏳ Pending</span>',
            'failed': '<span class="badge badge-danger">✗ Failed</span>'
        };
        return badges[status] || '<span class="badge badge-secondary">' + status + '</span>';
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    }

    truncate(str, length) {
        return str.length > length ? str.substring(0, length) + '...' : str;
    }
}

// Global instance
let adminFilesManager;

// Initialize when files tab is activated
document.addEventListener('DOMContentLoaded', () => {
    // Listen for tab changes
    document.querySelectorAll('.tab-btn').forEach(btn => {
        if (btn.dataset.tab === 'files') {
            btn.addEventListener('click', () => {
                if (!adminFilesManager) {
                    adminFilesManager = new AdminFilesManager();
                    adminFilesManager.init();
                }
            });
        }
    });
});

