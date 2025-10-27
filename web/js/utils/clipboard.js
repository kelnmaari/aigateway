// Clipboard utility for copying text to clipboard
// Supports modern Clipboard API with fallback for older browsers

const ClipboardUtils = {
    /**
     * Copy text to clipboard
     * @param {string} text - Text to copy
     * @param {Object} options - Options {onSuccess, onError}
     * @returns {Promise<boolean>} Success status
     */
    async copy(text, options = {}) {
        try {
            // Check if Clipboard API is available
            if (navigator.clipboard && navigator.clipboard.writeText) {
                await navigator.clipboard.writeText(text);
                if (options.onSuccess) options.onSuccess(text);
                return true;
            } else {
                // Clipboard API not available, use fallback
                return this.fallbackCopy(text, options);
            }
        } catch (err) {
            console.warn('Clipboard API failed, using fallback:', err);
            return this.fallbackCopy(text, options);
        }
    },
    
    /**
     * Fallback copy method using execCommand
     * @param {string} text - Text to copy
     * @param {Object} options - Options {onSuccess, onError}
     * @returns {boolean} Success status
     */
    fallbackCopy(text, options = {}) {
        const textArea = document.createElement("textarea");
        textArea.value = text;
        textArea.style.cssText = "position:fixed;left:-9999px;top:-9999px;";
        document.body.appendChild(textArea);
        textArea.select();
        
        let success = false;
        try {
            success = document.execCommand('copy');
            if (success && options.onSuccess) {
                options.onSuccess(text);
            } else if (!success && options.onError) {
                options.onError(new Error('execCommand failed'));
            }
        } catch (err) {
            if (options.onError) options.onError(err);
        } finally {
            document.body.removeChild(textArea);
        }
        return success;
    },
    
    /**
     * Show toast notification
     * @param {string} message - Message to display
     * @param {string} type - Type: 'success' or 'error'
     */
    showToast(message, type = 'success') {
        // Remove existing toast if any
        const existing = document.querySelector('.copy-toast');
        if (existing) {
            existing.remove();
        }
        
        // Create toast element
        const toast = document.createElement('div');
        toast.className = `copy-toast copy-toast-${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);
        
        // Animate in
        setTimeout(() => toast.classList.add('show'), 10);
        
        // Animate out and remove
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300);
        }, 2000);
    }
};

/**
 * Copy model name with visual feedback
 * @param {string} modelName - Model name to copy
 */
function copyModelName(modelName) {
    ClipboardUtils.copy(modelName, {
        onSuccess: () => {
            ClipboardUtils.showToast(`✓ Copied: ${modelName}`, 'success');
        },
        onError: (err) => {
            ClipboardUtils.showToast('✗ Failed to copy', 'error');
            console.error('Copy failed:', err);
        }
    });
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { ClipboardUtils, copyModelName };
}



