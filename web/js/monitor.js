// Monitor Page JavaScript
// Handles real-time system monitoring with HTMX and additional metrics

document.addEventListener('DOMContentLoaded', function() {
    console.log('Monitor page loaded');
    
    // Initialize refresh timer countdown
    initRefreshTimer();
    
    // Load system metrics (non-GPU)
    loadSystemMetrics();
    
    // Poll system metrics every 2 seconds
    setInterval(loadSystemMetrics, 2000);
});

/**
 * Initialize refresh timer countdown for GPU metrics
 */
function initRefreshTimer() {
    const timerElement = document.getElementById('gpu-refresh-timer');
    if (!timerElement) return;
    
    let countdown = 5;
    
    setInterval(() => {
        countdown--;
        if (countdown <= 0) {
            countdown = 5;
        }
        timerElement.textContent = `Next update in ${countdown}s`;
    }, 1000);
}

/**
 * Load system metrics (CPU, Memory, Active Requests, etc.)
 */
async function loadSystemMetrics() {
    try {
        // Fetch from stats API
        const response = await fetch('/api/stats', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('jwt_token')}`
            }
        });
        
        if (!response.ok) {
            console.error('Failed to fetch stats:', response.statusText);
            return;
        }
        
        const stats = await response.json();
        
        // Update CPU (if available in stats)
        const cpuElement = document.getElementById('stat-cpu');
        if (cpuElement && stats.system) {
            const cpuUsage = stats.system.cpu_percent || 0;
            cpuElement.textContent = `${cpuUsage.toFixed(1)}%`;
            
            const cpuProgress = document.getElementById('cpu-progress');
            if (cpuProgress) {
                cpuProgress.style.width = `${cpuUsage}%`;
                
                // Change color based on usage
                if (cpuUsage > 80) {
                    cpuProgress.style.background = '#ef4444';
                } else if (cpuUsage > 60) {
                    cpuProgress.style.background = '#f59e0b';
                } else {
                    cpuProgress.style.background = 'linear-gradient(90deg, var(--accent-primary), var(--accent-secondary))';
                }
            }
        }
        
        // Update Memory (if available in stats)
        const memoryElement = document.getElementById('stat-memory');
        if (memoryElement && stats.system) {
            const memoryPercent = stats.system.memory_percent || 0;
            const memoryUsedGB = (stats.system.memory_used_mb / 1024).toFixed(2) || 0;
            const memoryTotalGB = (stats.system.memory_total_mb / 1024).toFixed(2) || 0;
            
            memoryElement.textContent = `${memoryUsedGB} GB / ${memoryTotalGB} GB`;
            
            const memoryProgress = document.getElementById('memory-progress');
            if (memoryProgress) {
                memoryProgress.style.width = `${memoryPercent}%`;
                
                // Change color based on usage
                if (memoryPercent > 85) {
                    memoryProgress.style.background = '#ef4444';
                } else if (memoryPercent > 70) {
                    memoryProgress.style.background = '#f59e0b';
                } else {
                    memoryProgress.style.background = 'linear-gradient(90deg, var(--accent-primary), var(--accent-secondary))';
                }
            }
        }
        
        // Update Active Requests
        const activeRequestsElement = document.getElementById('stat-active-requests');
        if (activeRequestsElement) {
            const activeRequests = stats.active_requests || 0;
            activeRequestsElement.textContent = activeRequests.toLocaleString();
        }
        
        // Update Uptime
        const uptimeElement = document.getElementById('stat-uptime');
        if (uptimeElement && stats.uptime_seconds) {
            uptimeElement.textContent = formatUptime(stats.uptime_seconds);
        }
        
        // Update Request Rate Stats
        updateRequestRateStats(stats);
        
    } catch (error) {
        console.error('Error loading system metrics:', error);
    }
}

/**
 * Update request rate statistics
 */
function updateRequestRateStats(stats) {
    // Requests per second (calculated from recent window)
    const rpsElement = document.getElementById('stat-rps');
    if (rpsElement && stats.requests_per_second !== undefined) {
        rpsElement.textContent = stats.requests_per_second.toFixed(2);
        
        // Update trend indicator
        const trendElement = document.getElementById('rps-trend');
        if (trendElement && stats.rps_change_percent !== undefined) {
            const change = stats.rps_change_percent;
            if (change > 0) {
                trendElement.className = 'stat-trend trend-up';
                trendElement.textContent = `+${change.toFixed(1)}%`;
            } else if (change < 0) {
                trendElement.className = 'stat-trend trend-down';
                trendElement.textContent = `${change.toFixed(1)}%`;
            } else {
                trendElement.className = 'stat-trend trend-neutral';
                trendElement.textContent = 'No change';
            }
        }
    }
    
    // Average Response Time
    const responseTimeElement = document.getElementById('stat-response-time');
    if (responseTimeElement && stats.avg_response_time_ms !== undefined) {
        responseTimeElement.textContent = `${stats.avg_response_time_ms.toFixed(0)}ms`;
    }
    
    // Success Rate
    const successRateElement = document.getElementById('stat-success-rate');
    if (successRateElement && stats.success_rate !== undefined) {
        const successRate = (stats.success_rate * 100).toFixed(2);
        successRateElement.textContent = `${successRate}%`;
    }
    
    // Error Rate
    const errorRateElement = document.getElementById('stat-error-rate');
    if (errorRateElement && stats.error_rate !== undefined) {
        const errorRate = (stats.error_rate * 100).toFixed(2);
        errorRateElement.textContent = `${errorRate}%`;
        
        // Color-code error rate
        if (stats.error_rate > 0.05) { // > 5%
            errorRateElement.style.color = '#ef4444';
        } else if (stats.error_rate > 0.01) { // > 1%
            errorRateElement.style.color = '#f59e0b';
        } else {
            errorRateElement.style.color = '#22c55e';
        }
    }
}

/**
 * Format uptime seconds to human-readable string
 */
function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) {
        return `${days}d ${hours}h ${minutes}m`;
    } else if (hours > 0) {
        return `${hours}h ${minutes}m`;
    } else {
        return `${minutes}m`;
    }
}

/**
 * Handle HTMX events for better UX
 */
document.body.addEventListener('htmx:beforeRequest', function(event) {
    // Optional: Show loading state
    const target = event.detail.target;
    if (target) {
        target.classList.add('htmx-loading');
    }
});

document.body.addEventListener('htmx:afterRequest', function(event) {
    // Remove loading state
    const target = event.detail.target;
    if (target) {
        target.classList.remove('htmx-loading');
    }
    
    // Log any errors
    if (!event.detail.successful) {
        console.error('HTMX request failed:', event.detail);
    }
});

document.body.addEventListener('htmx:responseError', function(event) {
    console.error('HTMX response error:', event.detail);
    showNotification('Failed to update metrics', 'error');
});

/**
 * Show notification (reusing from notifications.js if available)
 */
function showNotification(message, type = 'info') {
    if (typeof window.showNotification === 'function') {
        window.showNotification(message, type);
    } else {
        console.log(`[${type}] ${message}`);
    }
}

