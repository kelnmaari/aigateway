// Toast Notifications System (v1.5.15)
class ToastNotification {
    constructor() {
        this.container = null;
        this.toasts = [];
        this.init();
    }

    init() {
        // Create toast container
        this.container = document.createElement('div');
        this.container.className = 'toast-container';
        document.body.appendChild(this.container);
    }

    /**
     * Show a toast notification
     * @param {string} message - Message to display
     * @param {string} type - Type: 'success', 'error', 'info', 'warning'
     * @param {number} duration - Duration in milliseconds (0 = permanent)
     */
    show(message, type = 'info', duration = null) {
        // Default durations
        if (duration === null) {
            duration = type === 'error' ? 60000 : 10000; // 60s for errors, 10s for others
        }

        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        
        // Icon based on type
        const icons = {
            success: '✅',
            error: '❌',
            warning: '⚠️',
            info: 'ℹ️'
        };
        
        const icon = icons[type] || icons.info;

        toast.innerHTML = `
            <div class="toast-icon">${icon}</div>
            <div class="toast-message">${this.escapeHtml(message)}</div>
            <button class="toast-close" aria-label="Close">&times;</button>
        `;

        // Close button handler
        const closeBtn = toast.querySelector('.toast-close');
        closeBtn.addEventListener('click', () => this.remove(toast));

        // Add to container
        this.container.appendChild(toast);
        this.toasts.push(toast);

        // Trigger animation
        setTimeout(() => toast.classList.add('toast-show'), 10);

        // Auto remove after duration
        if (duration > 0) {
            setTimeout(() => this.remove(toast), duration);
        }

        return toast;
    }

    remove(toast) {
        toast.classList.remove('toast-show');
        toast.classList.add('toast-hide');
        
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
            const index = this.toasts.indexOf(toast);
            if (index > -1) {
                this.toasts.splice(index, 1);
            }
        }, 300); // Match CSS transition duration
    }

    success(message, duration = 10000) {
        return this.show(message, 'success', duration);
    }

    error(message, duration = 60000) {
        return this.show(message, 'error', duration);
    }

    warning(message, duration = 10000) {
        return this.show(message, 'warning', duration);
    }

    info(message, duration = 10000) {
        return this.show(message, 'info', duration);
    }

    escapeHtml(text) {
        const map = {
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#039;'
        };
        return text.replace(/[&<>"']/g, m => map[m]);
    }

    // Clear all toasts
    clearAll() {
        this.toasts.forEach(toast => this.remove(toast));
    }
}

// Modal Confirmation System (v1.5.15)
class ModalConfirmation {
    constructor() {
        this.modal = null;
        this.resolveCallback = null;
        this.init();
    }

    init() {
        // Create modal HTML
        this.modal = document.createElement('div');
        this.modal.className = 'modal-overlay';
        this.modal.innerHTML = `
            <div class="modal-dialog modal-confirm">
                <div class="modal-header">
                    <h3 class="modal-title"></h3>
                    <button class="modal-close" aria-label="Close">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="modal-icon"></div>
                    <div class="modal-message"></div>
                    <div class="modal-details"></div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary modal-cancel">Cancel</button>
                    <button class="btn btn-primary modal-confirm-btn">Confirm</button>
                </div>
            </div>
        `;

        document.body.appendChild(this.modal);

        // Event listeners
        this.modal.querySelector('.modal-close').addEventListener('click', () => this.hide(false));
        this.modal.querySelector('.modal-cancel').addEventListener('click', () => this.hide(false));
        this.modal.querySelector('.modal-confirm-btn').addEventListener('click', () => this.hide(true));
        
        // Close on backdrop click
        this.modal.addEventListener('click', (e) => {
            if (e.target === this.modal) {
                this.hide(false);
            }
        });

        // Close on ESC key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.modal.classList.contains('modal-show')) {
                this.hide(false);
            }
        });
    }

    /**
     * Show confirmation modal
     * @param {Object} options - Configuration options
     * @returns {Promise<boolean>} - True if confirmed, false if cancelled
     */
    show(options = {}) {
        const {
            title = 'Confirm Action',
            message = 'Are you sure?',
            details = '',
            confirmText = 'Confirm',
            cancelText = 'Cancel',
            type = 'warning', // 'warning', 'danger', 'info'
            confirmButtonClass = 'btn-primary'
        } = options;

        return new Promise((resolve) => {
            this.resolveCallback = resolve;

            // Set content
            this.modal.querySelector('.modal-title').textContent = title;
            this.modal.querySelector('.modal-message').textContent = message;
            
            const detailsEl = this.modal.querySelector('.modal-details');
            if (details) {
                detailsEl.textContent = details;
                detailsEl.style.display = 'block';
            } else {
                detailsEl.style.display = 'none';
            }

            // Set icon
            const icons = {
                warning: '⚠️',
                danger: '🗑️',
                info: 'ℹ️',
                success: '✅'
            };
            this.modal.querySelector('.modal-icon').textContent = icons[type] || icons.warning;

            // Set button text
            this.modal.querySelector('.modal-cancel').textContent = cancelText;
            const confirmBtn = this.modal.querySelector('.modal-confirm-btn');
            confirmBtn.textContent = confirmText;
            
            // Update confirm button class
            confirmBtn.className = `btn ${confirmButtonClass} modal-confirm-btn`;

            // Show modal
            this.modal.classList.add('modal-show');
            document.body.style.overflow = 'hidden';
        });
    }

    hide(result) {
        this.modal.classList.remove('modal-show');
        document.body.style.overflow = '';

        if (this.resolveCallback) {
            this.resolveCallback(result);
            this.resolveCallback = null;
        }
    }

    // Shorthand methods
    confirm(message, title = 'Confirm Action') {
        return this.show({
            title,
            message,
            type: 'warning',
            confirmButtonClass: 'btn-primary'
        });
    }

    danger(message, title = 'Confirm Deletion') {
        return this.show({
            title,
            message,
            type: 'danger',
            confirmText: 'Delete',
            confirmButtonClass: 'btn-danger'
        });
    }

    warning(message, title = 'Warning', details = '') {
        return this.show({
            title,
            message,
            details,
            type: 'warning',
            confirmText: 'Continue',
            confirmButtonClass: 'btn-warning'
        });
    }
}

// Global instances
window.toast = new ToastNotification();
window.modal = new ModalConfirmation();

// Backward compatibility helpers
window.showToast = (message, type = 'info') => window.toast.show(message, type);
window.showSuccess = (message) => window.toast.success(message);
window.showError = (message) => window.toast.error(message);
window.showWarning = (message) => window.toast.warning(message);
window.showInfo = (message) => window.toast.info(message);
window.confirmAction = (message, title) => window.modal.confirm(message, title);

console.log('✅ Notifications system initialized (v1.5.15)');

