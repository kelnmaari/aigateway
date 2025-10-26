// GPU Monitor - manages NVIDIA GPU metrics display
class GPUMonitor {
    constructor() {
        this.container = document.getElementById('gpu-cards-container');
        this.section = document.getElementById('gpu-metrics-section');
        this.enabled = false;
        this.updateInterval = null;
    }

    async init() {
        try {
            // Пробуем загрузить GPU метрики
            const metrics = await this.fetchGPUMetrics();
            
            console.log('GPU Metrics Response:', metrics);
            
            if (metrics.enabled && metrics.data && metrics.data.device_count > 0) {
                this.enabled = true;
                this.section.style.display = 'block';
                this.renderGPUCards(metrics.data);
                
                // Запускаем автообновление каждые 5 секунд
                this.startAutoUpdate();
                console.log(`✅ GPU Monitor initialized: ${metrics.data.device_count} GPU(s) detected`);
            } else {
                console.warn('GPU monitoring disabled or no GPUs found:', metrics);
                this.section.style.display = 'none';
            }
        } catch (error) {
            console.error('GPU monitoring not available:', error);
            // Не показываем секцию если GPU недоступны
            if (this.section) {
                this.section.style.display = 'none';
            }
        }
    }

    async fetchGPUMetrics() {
        const response = await fetch('/api/gpu/metrics', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token') || ''}`,
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`Failed to fetch GPU metrics: ${response.status}`);
        }

        return await response.json();
    }

    renderGPUCards(data) {
        if (!data.devices || data.devices.length === 0) {
            this.section.style.display = 'none';
            return;
        }

        // Создаем единый блок для всех GPU
        const unifiedCard = this.createUnifiedGPUCard(data.devices);
        this.container.innerHTML = '';
        this.container.appendChild(unifiedCard);
    }

    createUnifiedGPUCard(devices) {
        const card = document.createElement('div');
        card.className = 'stat-card gpu-unified-card';
        
        // Проверяем максимальную температуру для gradient border
        const maxTemp = Math.max(...devices.map(d => d.temperature_c || 0));
        let borderGradient = 'linear-gradient(135deg, #4caf50, #66bb6a)';
        if (maxTemp > 80) {
            borderGradient = 'linear-gradient(135deg, #f44336, #e57373)';
        } else if (maxTemp > 70) {
            borderGradient = 'linear-gradient(135deg, #ff9800, #ffb74d)';
        }
        
        const gpuRows = devices.map((device, index) => {
            const temp = device.temperature_c || 0;
            const gpuUtil = device.utilization_gpu_percent || 0;
            const memUsed = device.memory_used_mb || 0;
            const memTotal = device.memory_total_mb || 1;
            const memUsagePercent = device.memory_usage || ((memUsed / memTotal) * 100);
            const powerUsage = device.power_usage_w || 0;
            const powerLimit = device.power_limit_w || 0;
            const powerPercent = powerLimit > 0 ? (powerUsage / powerLimit * 100) : 0;
            const clockGfx = device.clock_graphics_mhz || 0;
            const fanSpeed = device.fan_speed_percent || 0;
            const name = device.name || 'Unknown GPU';
            
            // Цвета для температуры
            let tempColor = '#4caf50';
            if (temp > 80) tempColor = '#f44336';
            else if (temp > 70) tempColor = '#ff9800';
            
            // Цвета для GPU util
            let gpuUtilColor = gpuUtil > 90 ? '#f44336' : gpuUtil > 70 ? '#ff9800' : '#4caf50';
            let gpuUtilGradient = gpuUtil > 90 
                ? 'linear-gradient(90deg, #f44336, #e57373)' 
                : gpuUtil > 70 
                    ? 'linear-gradient(90deg, #ff9800, #ffb74d)'
                    : 'linear-gradient(90deg, #4caf50, #66bb6a)';
            
            return `
                <div style="padding: 1rem; ${index > 0 ? 'border-top: 1px solid var(--border-color);' : ''} transition: background 0.2s;" onmouseover="this.style.background='var(--bg-tertiary)'" onmouseout="this.style.background='transparent'">
                    <!-- GPU Header Row -->
                    <div style="display: grid; grid-template-columns: 2fr 1fr 1fr 1fr 1fr; gap: 1rem; align-items: center; margin-bottom: 0.75rem;">
                        <!-- Name & Index -->
                        <div>
                            <div style="font-size: 0.7rem; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600;">
                                GPU ${index}
                            </div>
                            <div style="font-size: 0.9rem; font-weight: 600; color: var(--text-primary); margin-top: 0.25rem;">
                                ${name.replace('NVIDIA GeForce ', '')}
                            </div>
                        </div>
                        
                        <!-- Temperature -->
                        <div style="text-align: center;">
                            <div style="font-size: 0.65rem; color: var(--text-secondary); margin-bottom: 0.25rem;">TEMP</div>
                            <div style="font-size: 1.3rem; font-weight: 700; color: ${tempColor};">${temp}°C</div>
                        </div>
                        
                        <!-- Power -->
                        <div style="text-align: center;">
                            <div style="font-size: 0.65rem; color: var(--text-secondary); margin-bottom: 0.25rem;">POWER</div>
                            <div style="font-size: 1rem; font-weight: 700; color: var(--text-primary);">${powerUsage.toFixed(0)}W</div>
                            <div style="font-size: 0.6rem; color: var(--text-secondary);">${powerPercent.toFixed(0)}%</div>
                        </div>
                        
                        <!-- Clock -->
                        <div style="text-align: center;">
                            <div style="font-size: 0.65rem; color: var(--text-secondary); margin-bottom: 0.25rem;">CLOCK</div>
                            <div style="font-size: 1rem; font-weight: 700; color: var(--text-primary);">${clockGfx} MHz</div>
                        </div>
                        
                        ${fanSpeed > 0 ? `
                        <!-- Fan -->
                        <div style="text-align: center;">
                            <div style="font-size: 0.65rem; color: var(--text-secondary); margin-bottom: 0.25rem;">FAN</div>
                            <div style="font-size: 1rem; font-weight: 700; color: var(--text-primary);">${fanSpeed}%</div>
                        </div>
                        ` : '<div></div>'}
                    </div>
                    
                    <!-- Utilization Bars -->
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                        <!-- GPU Utilization -->
                        <div>
                            <div style="display: flex; justify-content: space-between; margin-bottom: 0.35rem;">
                                <span style="font-size: 0.7rem; color: var(--text-secondary); font-weight: 600;">GPU Load</span>
                                <span style="font-size: 0.8rem; font-weight: 700; color: ${gpuUtilColor};">${gpuUtil}%</span>
                            </div>
                            <div style="background: rgba(0,0,0,0.2); height: 6px; border-radius: 3px; overflow: hidden;">
                                <div style="background: ${gpuUtilGradient}; height: 100%; width: ${gpuUtil}%; transition: width 0.3s ease;"></div>
                            </div>
                        </div>
                        
                        <!-- Memory Utilization -->
                        <div>
                            <div style="display: flex; justify-content: space-between; margin-bottom: 0.35rem;">
                                <span style="font-size: 0.7rem; color: var(--text-secondary); font-weight: 600;">VRAM</span>
                                <span style="font-size: 0.8rem; font-weight: 700; color: #2196f3;">${(memUsed/1024).toFixed(1)} / ${(memTotal/1024).toFixed(1)} GB</span>
                            </div>
                            <div style="background: rgba(0,0,0,0.2); height: 6px; border-radius: 3px; overflow: hidden;">
                                <div style="background: linear-gradient(90deg, #2196f3, #03a9f4); height: 100%; width: ${memUsagePercent.toFixed(1)}%; transition: width 0.3s ease;"></div>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }).join('');
        
        card.innerHTML = `
            <div style="position: relative;">
                <!-- Gradient Border Top -->
                <div style="position: absolute; top: 0; left: 0; right: 0; height: 3px; background: ${borderGradient}; border-radius: 8px 8px 0 0;"></div>
                
                <!-- Content -->
                <div style="padding-top: 0.5rem;">
                    ${gpuRows}
                </div>
            </div>
        `;
        
        return card;
    }


    async updateMetrics() {
        if (!this.enabled) return;

        try {
            const metrics = await this.fetchGPUMetrics();
            
            if (metrics.enabled && metrics.data.devices) {
                this.renderGPUCards(metrics.data);
            }
        } catch (error) {
            console.error('Failed to update GPU metrics:', error);
        }
    }

    startAutoUpdate() {
        // Обновляем каждые 10 секунд (уменьшена нагрузка)
        this.updateInterval = setInterval(() => this.updateMetrics(), 10000);
    }

    stop() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
        }
    }
}

// Создаем глобальный экземпляр
window.gpuMonitor = new GPUMonitor();

