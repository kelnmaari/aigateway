// Performance Monitor (v1.9.3)
// Real-time performance metrics display using MoniGo API

class PerformanceMonitor {
    constructor() {
        this.updateInterval = null;
        this.UPDATE_FREQUENCY = 10000; // 10 seconds (reduced load)
        this.baseURL = '';
        
        // DOM elements
        this.cpuUsageEl = null;
        this.memoryUsageEl = null;
        this.goroutinesEl = null;
        this.healthEl = null;
        this.iframeEl = null;
    }

    init() {
        // Get DOM elements
        this.cpuUsageEl = document.getElementById('perf-cpu-usage');
        this.memoryUsageEl = document.getElementById('perf-memory-usage');
        this.goroutinesEl = document.getElementById('perf-goroutines');
        this.healthEl = document.getElementById('perf-health');
        this.iframeEl = document.getElementById('monigo-dashboard');

        // Listen for system tab activation
        const systemTab = document.querySelector('[data-tab="system"]');
        if (systemTab) {
            systemTab.addEventListener('click', () => {
                this.start();
            });
        }

        // Check if we're already on system tab
        const systemTabPane = document.getElementById('system-tab');
        if (systemTabPane && systemTabPane.classList.contains('active')) {
            this.start();
        }

        console.log('Performance Monitor initialized');
    }

    start() {
        if (this.updateInterval) {
            return; // Already running
        }

        console.log('Starting performance metrics updates');
        this.updateMetrics(); // Initial update
        this.updateInterval = setInterval(() => {
            this.updateMetrics();
        }, this.UPDATE_FREQUENCY);
    }

    stop() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
            console.log('Stopped performance metrics updates');
        }
    }

    async updateMetrics() {
        try {
            // MoniGo использует /monigo/api/v1/* пути (v1.9.3+)
            const response = await fetch('/admin/performance/monigo/api/v1/metrics', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('access_token') || ''}`,
                },
                credentials: 'include'
            });

            if (!response.ok) {
                throw new Error(`Failed to fetch metrics: ${response.status}`);
            }

            const metrics = await response.json();
            this.renderMetrics(metrics);
        } catch (error) {
            console.error('Error fetching performance metrics:', error);
            // Don't show error to user for polling failures
            // Just keep the old values
        }
    }

    renderMetrics(metrics) {
        // CPU Usage (service_cpu_load)
        // MoniGo returns strings like "1.93%" instead of numbers
        if (metrics.load_statistics && metrics.load_statistics.service_cpu_load !== undefined) {
            let cpuLoad = metrics.load_statistics.service_cpu_load;
            // Parse string "1.93%" to number 1.93
            if (typeof cpuLoad === 'string') {
                cpuLoad = parseFloat(cpuLoad.replace('%', ''));
            }
            if (this.cpuUsageEl) {
                this.cpuUsageEl.textContent = `${cpuLoad.toFixed(1)}%`;
                this.updateCardColor(this.cpuUsageEl.closest('.stat-card'), cpuLoad, 80, 90);
            }
        }

        // Memory Usage (service_memory_load)
        if (metrics.load_statistics && metrics.load_statistics.service_memory_load !== undefined) {
            let memLoad = metrics.load_statistics.service_memory_load;
            // Parse string "0.14%" to number 0.14
            if (typeof memLoad === 'string') {
                memLoad = parseFloat(memLoad.replace('%', ''));
            }
            if (this.memoryUsageEl) {
                this.memoryUsageEl.textContent = `${memLoad}%`;
                this.updateCardColor(this.memoryUsageEl.closest('.stat-card'), memLoad, 80, 90);
            }
        }

        // Goroutines Count (from core_statistics)
        if (metrics.core_statistics && metrics.core_statistics.goroutines !== undefined) {
            const goroutines = metrics.core_statistics.goroutines;
            if (this.goroutinesEl) {
                this.goroutinesEl.textContent = goroutines.toLocaleString();
                // Warning if > 1000, critical if > 10000
                this.updateCardColor(this.goroutinesEl.closest('.stat-card'), goroutines, 1000, 10000);
            }
        }

        // System Health (from health.service_health.percent)
        if (metrics.health && metrics.health.service_health && metrics.health.service_health.percent !== undefined) {
            const health = metrics.health.service_health.percent;
            if (this.healthEl) {
                this.healthEl.textContent = `${health.toFixed(1)}%`;
                // Inverted: low health = warning/critical
                this.updateCardColorInverted(this.healthEl.closest('.stat-card'), health, 70, 50);
            }
        }
    }

    updateCardColor(cardEl, value, warningThreshold, criticalThreshold) {
        if (!cardEl) return;

        cardEl.classList.remove('warning', 'critical', 'success');
        
        if (value >= criticalThreshold) {
            cardEl.classList.add('critical');
        } else if (value >= warningThreshold) {
            cardEl.classList.add('warning');
        } else {
            cardEl.classList.add('success');
        }
    }

    updateCardColorInverted(cardEl, value, warningThreshold, criticalThreshold) {
        if (!cardEl) return;

        cardEl.classList.remove('warning', 'critical', 'success');
        
        if (value <= criticalThreshold) {
            cardEl.classList.add('critical');
        } else if (value <= warningThreshold) {
            cardEl.classList.add('warning');
        } else {
            cardEl.classList.add('success');
        }
    }

    // Export metrics report (if MoniGo provides export API)
    async exportReport(format = 'excel') {
        try {
            const response = await fetch('/admin/performance/api/v1/reports', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${localStorage.getItem('access_token') || ''}`,
                },
                credentials: 'include',
                body: JSON.stringify({ format })
            });

            if (!response.ok) {
                throw new Error('Export failed');
            }

            // Download file
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `performance-report-${new Date().toISOString().split('T')[0]}.${format}`;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);

            notifications.showSuccess('Report exported successfully');
        } catch (error) {
            console.error('Export error:', error);
            notifications.showError('Failed to export report');
        }
    }
}

// Global instance
window.performanceMonitor = new PerformanceMonitor();

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.performanceMonitor.init();
    });
} else {
    window.performanceMonitor.init();
}

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (window.performanceMonitor) {
        window.performanceMonitor.stop();
    }
});

