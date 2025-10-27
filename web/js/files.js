// Files Management (v1.10.0+)
class FilesManager {
    constructor() {
        this.currentPage = 0;
        this.pageSize = 12;
        this.totalFiles = 0;
        this.filters = {
            search: '',
            type: '',
            status: ''
        };

        this.init();
    }

    init() {
        this.setupUploadArea();
        this.setupFilters();
        this.setupModal();
        this.loadFiles();
    }

    setupUploadArea() {
        const uploadArea = document.getElementById('upload-area');
        const fileInput = document.getElementById('file-input');

        // Click to upload
        uploadArea.addEventListener('click', () => {
            fileInput.click();
        });

        // File selection
        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.uploadFiles(Array.from(e.target.files));
            }
        });

        // Drag and drop
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });

        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });

        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            
            const files = Array.from(e.dataTransfer.files);
            this.uploadFiles(files);
        });
    }

    setupFilters() {
        const searchInput = document.getElementById('search-input');
        const typeFilter = document.getElementById('type-filter');
        const statusFilter = document.getElementById('status-filter');
        const refreshBtn = document.getElementById('refresh-btn');

        searchInput.addEventListener('input', (e) => {
            this.filters.search = e.target.value;
            this.currentPage = 0;
            this.loadFiles();
        });

        typeFilter.addEventListener('change', (e) => {
            this.filters.type = e.target.value;
            this.currentPage = 0;
            this.loadFiles();
        });

        statusFilter.addEventListener('change', (e) => {
            this.filters.status = e.target.value;
            this.currentPage = 0;
            this.loadFiles();
        });

        refreshBtn.addEventListener('click', () => {
            this.loadFiles();
        });
    }

    setupModal() {
        const modal = document.getElementById('text-modal');
        const closeBtn = document.getElementById('close-modal');

        closeBtn.addEventListener('click', () => {
            modal.classList.remove('active');
        });

        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        });
    }

    async uploadFiles(files) {
        const validFiles = files.filter(file => {
            const ext = file.name.split('.').pop().toLowerCase();
            const validExtensions = ['pdf', 'docx', 'doc', 'txt', 'csv', 'xlsx', 'xls', 'md', 'rtf'];
            return validExtensions.includes(ext);
        });

        if (validFiles.length === 0) {
            toast.error('No valid files selected. Please select PDF, DOCX, TXT, CSV, or XLSX files.');
            return;
        }

        const uploadPromises = validFiles.map(file => this.uploadFile(file));
        
        try {
            await Promise.all(uploadPromises);
            toast.success(`Successfully uploaded ${validFiles.length} file(s)`);
            this.loadFiles();
        } catch (error) {
            console.error('Upload error:', error);
            toast.error('Some files failed to upload');
        }
    }

    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('extract', 'true');

        // For FormData, we need to override the Content-Type header behavior
        // api.request adds 'Content-Type: application/json' by default, but FormData needs multipart/form-data
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${api.baseURL}/api/files/upload`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            },
            body: formData
            // Don't set Content-Type - browser will automatically set multipart/form-data with boundary
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Upload failed');
        }

        return response.json();
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

            const response = await api.request(`${api.baseURL}/api/files?${params}`);
            
            if (!response.ok) {
                throw new Error('Failed to load files');
            }

            const data = await response.json();
            this.totalFiles = data.total || 0;
            this.renderFiles(data.files || []);
            this.renderPagination();
        } catch (error) {
            console.error('Failed to load files:', error);
            toast.error('Failed to load files');
            this.renderFiles([]);
        }
    }

    renderFiles(files) {
        const grid = document.getElementById('files-grid');

        // Filter by search term
        const filteredFiles = this.filters.search 
            ? files.filter(file => 
                file.filename.toLowerCase().includes(this.filters.search.toLowerCase())
            )
            : files;

        if (filteredFiles.length === 0) {
            grid.innerHTML = `
                <div class="empty-state" style="grid-column: 1 / -1;">
                    <div class="empty-icon">📁</div>
                    <h3>No files found</h3>
                    <p style="color: var(--text-muted);">Upload your first document to get started</p>
                </div>
            `;
            return;
        }

        grid.innerHTML = filteredFiles.map(file => this.createFileCard(file)).join('');
    }

    createFileCard(file) {
        const ext = file.filename.split('.').pop().toLowerCase();
        const icon = this.getFileIcon(ext);
        const iconClass = this.getFileIconClass(file.mime_type);
        const size = this.formatFileSize(file.size_bytes);
        const date = new Date(file.created_at).toLocaleDateString();
        const extractionBadge = this.getExtractionBadge(file.extraction_status);

        return `
            <div class="file-card" data-file-id="${file.id}">
                <div class="file-header">
                    <div class="file-icon ${iconClass}">${icon}</div>
                    <div class="file-info">
                        <div class="file-name" title="${file.filename}">${file.filename}</div>
                        <div class="file-meta">${size} • ${date}</div>
                    </div>
                </div>

                <div class="file-stats">
                    ${extractionBadge}
                    ${file.word_count ? `<div class="file-stat">📄 ${file.word_count.toLocaleString()} words</div>` : ''}
                    ${file.page_count ? `<div class="file-stat">📑 ${file.page_count} pages</div>` : ''}
                </div>

                <div class="file-actions">
                    <button class="btn-icon" onclick="filesManager.downloadFile('${file.id}', '${file.filename}')" title="Download">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" stroke-width="2"/>
                        </svg>
                    </button>
                    ${file.extraction_status === 'completed' ? `
                        <button class="btn-icon" onclick="filesManager.viewText('${file.id}')" title="View extracted text">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" stroke-width="2"/>
                                <circle cx="12" cy="12" r="3" stroke-width="2"/>
                            </svg>
                        </button>
                    ` : ''}
                    <button class="btn-icon" onclick="filesManager.viewDetails('${file.id}')" title="Details">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <circle cx="12" cy="12" r="10" stroke-width="2"/>
                            <path d="M12 16v-4M12 8h.01" stroke-width="2"/>
                        </svg>
                    </button>
                    <button class="btn-icon danger" onclick="filesManager.deleteFile('${file.id}')" title="Delete">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
    }

    getFileIcon(ext) {
        const icons = {
            'pdf': '📕',
            'docx': '📘',
            'doc': '📘',
            'txt': '📄',
            'csv': '📊',
            'xlsx': '📗',
            'xls': '📗',
            'md': '📝',
            'rtf': '📄'
        };
        return icons[ext] || '📎';
    }

    getFileIconClass(mimeType) {
        if (mimeType.includes('pdf')) return 'pdf';
        if (mimeType.includes('word')) return 'doc';
        if (mimeType.includes('text')) return 'txt';
        if (mimeType.includes('csv')) return 'csv';
        if (mimeType.includes('sheet')) return 'xlsx';
        return 'txt';
    }

    getExtractionBadge(status) {
        const badges = {
            'completed': '<span class="extraction-badge completed">✓ Extracted</span>',
            'pending': '<span class="extraction-badge pending">⏳ Processing</span>',
            'failed': '<span class="extraction-badge failed">✗ Failed</span>'
        };
        return badges[status] || '';
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
    }

    async downloadFile(fileId, filename) {
        try {
            const response = await api.request(`${api.baseURL}/api/files/${fileId}/download`);
            
            if (!response.ok) {
                throw new Error('Download failed');
            }

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            toast.success('File downloaded');
        } catch (error) {
            console.error('Download error:', error);
            toast.error('Failed to download file');
        }
    }

    async viewText(fileId) {
        try {
            const response = await api.request(`${api.baseURL}/api/files/${fileId}/text`);
            
            if (!response.ok) {
                throw new Error('Failed to load text');
            }

            const data = await response.json();
            this.showModal('Extracted Text', `
                <div class="file-stats" style="margin-bottom: 1rem;">
                    ${data.word_count ? `<div class="file-stat">Words: ${data.word_count.toLocaleString()}</div>` : ''}
                    ${data.page_count ? `<div class="file-stat">Pages: ${data.page_count}</div>` : ''}
                    ${data.language ? `<div class="file-stat">Language: ${data.language.toUpperCase()}</div>` : ''}
                </div>
                <div class="extracted-text">${data.text || 'No text extracted'}</div>
            `);
        } catch (error) {
            console.error('Failed to load text:', error);
            toast.error('Failed to load extracted text');
        }
    }

    async viewDetails(fileId) {
        try {
            const response = await api.request(`${api.baseURL}/api/files/${fileId}`);
            
            if (!response.ok) {
                throw new Error('Failed to load file details');
            }

            const file = await response.json();
            this.showModal('File Details', `
                <table style="width: 100%; border-collapse: collapse;">
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Filename:</td>
                        <td style="padding: 0.75rem;">${file.filename}</td>
                    </tr>
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Type:</td>
                        <td style="padding: 0.75rem;">${file.mime_type}</td>
                    </tr>
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Size:</td>
                        <td style="padding: 0.75rem;">${this.formatFileSize(file.size_bytes)}</td>
                    </tr>
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Storage:</td>
                        <td style="padding: 0.75rem;">${file.storage_backend.toUpperCase()}</td>
                    </tr>
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Extraction:</td>
                        <td style="padding: 0.75rem;">${this.getExtractionBadge(file.extraction_status)}</td>
                    </tr>
                    ${file.page_count ? `
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Pages:</td>
                        <td style="padding: 0.75rem;">${file.page_count}</td>
                    </tr>
                    ` : ''}
                    ${file.word_count ? `
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Words:</td>
                        <td style="padding: 0.75rem;">${file.word_count.toLocaleString()}</td>
                    </tr>
                    ` : ''}
                    <tr style="border-bottom: 1px solid var(--border-color);">
                        <td style="padding: 0.75rem; font-weight: 600;">Downloads:</td>
                        <td style="padding: 0.75rem;">${file.download_count}</td>
                    </tr>
                    <tr>
                        <td style="padding: 0.75rem; font-weight: 600;">Uploaded:</td>
                        <td style="padding: 0.75rem;">${new Date(file.created_at).toLocaleString()}</td>
                    </tr>
                </table>
            `);
        } catch (error) {
            console.error('Failed to load details:', error);
            toast.error('Failed to load file details');
        }
    }

    async deleteFile(fileId) {
        const confirmed = await modal.danger(
            'Are you sure you want to delete this file? This action cannot be undone.',
            'Delete File'
        );

        if (!confirmed) return;

        try {
            const response = await api.request(`${api.baseURL}/api/files/${fileId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                throw new Error('Delete failed');
            }

            toast.success('File deleted successfully');
            this.loadFiles();
        } catch (error) {
            console.error('Delete error:', error);
            toast.error('Failed to delete file');
        }
    }

    showModal(title, content) {
        document.getElementById('modal-title').textContent = title;
        document.getElementById('modal-body').innerHTML = content;
        document.getElementById('text-modal').classList.add('active');
    }

    renderPagination() {
        const totalPages = Math.ceil(this.totalFiles / this.pageSize);
        const pagination = document.getElementById('pagination');

        if (totalPages <= 1) {
            pagination.innerHTML = '';
            return;
        }

        pagination.innerHTML = `
            <button class="btn btn-secondary" ${this.currentPage === 0 ? 'disabled' : ''} onclick="filesManager.previousPage()">
                Previous
            </button>
            <span>Page ${this.currentPage + 1} of ${totalPages}</span>
            <button class="btn btn-secondary" ${this.currentPage >= totalPages - 1 ? 'disabled' : ''} onclick="filesManager.nextPage()">
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
        const totalPages = Math.ceil(this.totalFiles / this.pageSize);
        if (this.currentPage < totalPages - 1) {
            this.currentPage++;
            this.loadFiles();
        }
    }
}

// Initialize
let filesManager;
window.addEventListener('DOMContentLoaded', () => {
    filesManager = new FilesManager();
});



