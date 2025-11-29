// Error Boundary Utility (v3.1.0: AJAX-05)
// Graceful degradation с retry механизмом

class ErrorBoundary {
    constructor(options = {}) {
        this.maxRetries = options.maxRetries || 3;
        this.retryDelay = options.retryDelay || 1000;
        this.onError = options.onError || this.defaultErrorHandler;
        this.onRetry = options.onRetry || null;
        this.onMaxRetriesReached = options.onMaxRetriesReached || null;
    }

    // Default error handler
    defaultErrorHandler(error, context) {
        console.error(`[ErrorBoundary] Error in ${context}:`, error);
        
        // Show user-friendly notification
        const message = this.getUserFriendlyMessage(error);
        this.showNotification(message, 'error');
    }

    // Get user-friendly error message
    getUserFriendlyMessage(error) {
        if (!navigator.onLine) {
            return 'Network connection lost. Please check your internet connection.';
        }
        
        if (error.message.includes('401') || error.message.includes('unauthorized')) {
            return 'Your session has expired. Please log in again.';
        }
        
        if (error.message.includes('403') || error.message.includes('forbidden')) {
            return 'You do not have permission to perform this action.';
        }
        
        if (error.message.includes('404') || error.message.includes('not found')) {
            return 'The requested resource was not found.';
        }
        
        if (error.message.includes('500')) {
            return 'Server error. Please try again later.';
        }
        
        return `An error occurred: ${error.message}`;
    }

    // Retry механизм с exponential backoff
    async retry(fn, context = 'operation', attempt = 1) {
        try {
            return await fn();
        } catch (error) {
            console.warn(`[ErrorBoundary] Attempt ${attempt}/${this.maxRetries} failed for ${context}`);
            
            if (attempt >= this.maxRetries) {
                // Max retries reached
                if (this.onMaxRetriesReached) {
                    this.onMaxRetriesReached(error, context);
                }
                this.onError(error, context);
                throw error;
            }
            
            // Calculate backoff delay (exponential: 1s, 2s, 4s, etc.)
            const delay = this.retryDelay * Math.pow(2, attempt - 1);
            console.log(`[ErrorBoundary] Retrying in ${delay}ms...`);
            
            if (this.onRetry) {
                this.onRetry(attempt, delay, context);
            }
            
            // Wait before retrying
            await new Promise(resolve => setTimeout(resolve, delay));
            
            // Recursive retry
            return this.retry(fn, context, attempt + 1);
        }
    }

    // Wrap async function with error boundary
    wrap(fn, context = 'operation') {
        return async (...args) => {
            return this.retry(() => fn(...args), context);
        };
    }

    // Show notification toast
    showNotification(message, type = 'error') {
        const notification = document.createElement('div');
        notification.className = `error-boundary-notification notification-${type}`;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${type === 'error' ? '#ef4444' : '#f59e0b'};
            color: white;
            padding: 16px 20px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            z-index: 10000;
            max-width: 400px;
            animation: slideIn 0.3s ease-out;
        `;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOut 0.3s ease-out';
            setTimeout(() => notification.remove(), 300);
        }, 5000);
    }
}

// Global error boundary instance
window.errorBoundary = new ErrorBoundary({
    maxRetries: 3,
    retryDelay: 1000,
    onRetry: (attempt, delay, context) => {
        console.log(`🔄 Retrying ${context} (${attempt}/3) in ${delay}ms...`);
    },
    onMaxRetriesReached: (error, context) => {
        console.error(`❌ Max retries reached for ${context}`, error);
    }
});

// CSS animations
const style = document.createElement('style');
style.textContent = `
@keyframes slideIn {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

@keyframes slideOut {
    from {
        transform: translateX(0);
        opacity: 1;
    }
    to {
        transform: translateX(100%);
        opacity: 0;
    }
}
`;
document.head.appendChild(style);

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ErrorBoundary;
}

