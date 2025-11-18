/**
 * AIGateway UI Framework v3.1.0
 * Core JavaScript Framework
 */

(function(window) {
    'use strict';

    // Framework namespace
    const AG = window.AG = {
        version: '3.1.0',
        components: {},
        utils: {},
        state: {},
        router: null,
        http: null,
    };

    /**
     * Component Registry
     */
    AG.component = function(name, definition) {
        AG.components[name] = definition;
        console.log(`[AG] Component registered: ${name}`);
    };

    /**
     * Utilities
     */
    AG.utils = {
        // DOM helpers
        $: (selector, context = document) => context.querySelector(selector),
        $$: (selector, context = document) => context.querySelectorAll(selector),

        // Data binding
        bind: (selector, data) => {
            const el = typeof selector === 'string' ? AG.utils.$(selector) : selector;
            if (!el) return;

            Object.keys(data).forEach(key => {
                const target = el.querySelector(`[data-bind="${key}"]`);
                if (target) {
                    target.textContent = data[key];
                }
            });
        },

        // Event delegation
        on: (selector, event, handler) => {
            document.addEventListener(event, (e) => {
                if (e.target.matches(selector)) {
                    handler.call(e.target, e);
                }
            });
        },

        // Debounce
        debounce: (fn, delay = 300) => {
            let timeout;
            return function(...args) {
                clearTimeout(timeout);
                timeout = setTimeout(() => fn.apply(this, args), delay);
            };
        },

        // Throttle
        throttle: (fn, limit = 100) => {
            let inThrottle;
            return function(...args) {
                if (!inThrottle) {
                    fn.apply(this, args);
                    inThrottle = true;
                    setTimeout(() => inThrottle = false, limit);
                }
            };
        },

        // Format bytes
        formatBytes: (bytes) => {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        },

        // Format date
        formatDate: (date) => {
            const d = new Date(date);
            return d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
        },

        // Escape HTML
        escapeHtml: (text) => {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        },

        // Copy to clipboard
        copyToClipboard: async (text) => {
            try {
                await navigator.clipboard.writeText(text);
                return true;
            } catch (err) {
                console.error('Failed to copy:', err);
                return false;
            }
        },
    };

    /**
     * Simple State Management (Reactive)
     */
    AG.createStore = function(initialState = {}) {
        const state = { ...initialState };
        const listeners = new Set();

        return {
            getState: () => ({ ...state }),

            setState: (updates) => {
                Object.assign(state, updates);
                listeners.forEach(listener => listener(state));
            },

            subscribe: (listener) => {
                listeners.add(listener);
                return () => listeners.delete(listener);
            },
        };
    };

    /**
     * HTTP Client (wraps existing api.js)
     */
    AG.http = {
        get: (url, options = {}) => window.api.get(url, options),
        post: (url, data, options = {}) => window.api.post(url, data, options),
        put: (url, data, options = {}) => window.api.put(url, data, options),
        delete: (url, options = {}) => window.api.delete(url, options),

        // Batch requests
        batch: async (requests) => {
            return Promise.all(requests.map(req => {
                const method = req.method || 'GET';
                return AG.http[method.toLowerCase()](req.url, req.data, req.options);
            }));
        },
    };

    /**
     * Toast Notifications
     */
    AG.toast = function(message, type = 'info', duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `ag-toast ag-toast-${type}`;
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${type === 'error' ? '#ef4444' : type === 'success' ? '#10b981' : '#3b82f6'};
            color: white;
            padding: 16px 24px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            z-index: 10000;
            animation: slideIn 0.3s ease-out;
        `;

        document.body.appendChild(toast);

        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease-out';
            setTimeout(() => toast.remove(), 300);
        }, duration);
    };

    /**
     * Loading Indicator
     */
    AG.loading = {
        show: (target) => {
            const el = typeof target === 'string' ? AG.utils.$(target) : target;
            if (!el) return;

            el.classList.add('ag-loading');
            el.setAttribute('aria-busy', 'true');
        },

        hide: (target) => {
            const el = typeof target === 'string' ? AG.utils.$(target) : target;
            if (!el) return;

            el.classList.remove('ag-loading');
            el.removeAttribute('aria-busy');
        },
    };

    /**
     * Skeleton Loaders
     */
    AG.skeleton = {
        show: (target, count = 3) => {
            const el = typeof target === 'string' ? AG.utils.$(target) : target;
            if (!el) return;

            const skeletons = Array(count).fill(0).map(() => 
                `<div class="skeleton" style="height: 40px; margin: 8px 0;"></div>`
            ).join('');

            el.innerHTML = skeletons;
        },

        hide: (target) => {
            const el = typeof target === 'string' ? AG.utils.$(target) : target;
            if (!el) return;

            el.querySelectorAll('.skeleton').forEach(sk => sk.remove());
        },
    };

    /**
     * Form Validation
     */
    AG.validate = {
        required: (value) => !!value,
        email: (value) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value),
        minLength: (min) => (value) => value.length >= min,
        maxLength: (max) => (value) => value.length <= max,
        pattern: (regex) => (value) => regex.test(value),

        form: (formEl, rules) => {
            const errors = {};
            const formData = new FormData(formEl);

            for (const [field, validators] of Object.entries(rules)) {
                const value = formData.get(field);
                for (const validator of validators) {
                    if (!validator.fn(value)) {
                        errors[field] = validator.message;
                        break;
                    }
                }
            }

            return {
                valid: Object.keys(errors).length === 0,
                errors,
            };
        },
    };

    /**
     * Lazy Loading
     */
    AG.lazyLoad = function(selector) {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src;
                    img.classList.remove('lazy');
                    observer.unobserve(img);
                }
            });
        });

        document.querySelectorAll(selector).forEach(img => observer.observe(img));
    };

    console.log(`[AG] Framework v${AG.version} loaded`);

})(window);

