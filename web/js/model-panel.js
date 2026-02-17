// Model Panel Controller (v1.9.1)
// Manages model selection, parameters, localStorage persistence, and presets

class ModelPanelController {
    constructor() {
        this.STORAGE_KEY_MODEL = 'chat_selected_model';
        this.STORAGE_KEY_PARAMS = 'chat_model_params';
        
        // Presets matching backend ModelParameters
        this.PRESETS = {
            creative: {
                temperature: 1.2,
                top_p: 0.95,
                max_tokens: -1,
                num_ctx: 4096
            },
            balanced: {
                temperature: 0.7,
                top_p: 0.9,
                max_tokens: -1,
                num_ctx: 4096
            },
            precise: {
                temperature: 0.3,
                top_p: 0.8,
                max_tokens: 2048,
                num_ctx: 2048
            },
            coding: {
                temperature: 0.2,
                top_p: 0.95,
                max_tokens: 4096,
                num_ctx: 8192
            }
        };
    }

    // Initialize the panel
    init() {
        this.modelSelect = document.getElementById('chat-model-select');
        this.paramsToggle = document.getElementById('params-toggle');
        this.paramsPanel = document.getElementById('params-panel');
        
        // Parameter inputs
        this.tempSlider = document.getElementById('chat-temperature');
        this.tempValue = document.getElementById('chat-temperature-value');
        this.topPSlider = document.getElementById('chat-top-p');
        this.topPValue = document.getElementById('chat-top-p-value');
        this.maxTokensInput = document.getElementById('chat-max-tokens');
        this.numCtxInput = document.getElementById('chat-num-ctx');
        
        this.resetBtn = document.getElementById('reset-params-btn');
        
        this.setupEventListeners();
        this.loadModels();
        this.restoreSavedSettings();
    }

    // Setup event listeners
    setupEventListeners() {
        // Toggle parameters panel
        this.paramsToggle?.addEventListener('click', () => {
            const isExpanded = this.paramsToggle.getAttribute('aria-expanded') === 'true';
            this.paramsToggle.setAttribute('aria-expanded', !isExpanded);
            this.paramsPanel.hidden = isExpanded;
        });

        // Sync sliders with number inputs
        this.syncSliderWithInput(this.tempSlider, this.tempValue);
        this.syncSliderWithInput(this.topPSlider, this.topPValue);
        
        // Model selection change
        this.modelSelect?.addEventListener('change', () => {
            this.saveSelectedModel();
        });

        // Parameter changes
        [this.tempSlider, this.tempValue, this.topPSlider, this.topPValue, 
         this.maxTokensInput, this.numCtxInput].forEach(input => {
            input?.addEventListener('change', () => this.saveCurrentParams());
        });

        // Preset buttons
        document.querySelectorAll('.preset-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const preset = btn.dataset.preset;
                this.applyPreset(preset);
            });
        });

        // Reset button
        this.resetBtn?.addEventListener('click', () => this.resetToDefaults());
    }

    // Sync slider with number input
    syncSliderWithInput(slider, input) {
        if (!slider || !input) return;

        slider.addEventListener('input', () => {
            input.value = slider.value;
        });

        input.addEventListener('input', () => {
            slider.value = input.value;
        });
    }

    // Load models from API
    async loadModels() {
        try {
            const token = localStorage.getItem('access_token');
            if (!token) {
                throw new Error('No authentication token');
            }

            // v3.0.5+: OpenAI-compatible endpoint (returns model aliases)
            const response = await fetch('/v1/models', {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (!response.ok) {
                throw new Error('Failed to fetch models');
            }

            const data = await response.json();
            this.populateModelSelect(data.data || []);
        } catch (error) {
            console.error('Failed to load models:', error);
            if (this.modelSelect) {
                this.modelSelect.innerHTML = '<option value="">Failed to load models</option>';
            }
        }
    }

    // Populate model select dropdown
    populateModelSelect(models) {
        if (!this.modelSelect) return;

        // Keep recommended models at top
        const recommendedModels = [
            'qwen2.5-coder-tuned:7b',
            'qwen3-coder-tuned:30b',
            'devstral-tuned:latest'
        ];

        let html = '<optgroup label="Recommended">';
        recommendedModels.forEach(modelId => {
            const model = models.find(m => m.id === modelId);
            if (model) {
                html += `<option value="${model.id}">${this.getModelIcon(model.id)} ${model.id}</option>`;
            }
        });
        html += '</optgroup><optgroup label="All Models">';

        models.forEach(model => {
            if (!recommendedModels.includes(model.id)) {
                html += `<option value="${model.id}">${this.getModelIcon(model.id)} ${model.id}</option>`;
            }
        });
        html += '</optgroup>';

        this.modelSelect.innerHTML = html;

        // Restore saved model after populating
        const savedModel = localStorage.getItem(this.STORAGE_KEY_MODEL);
        if (savedModel && this.modelSelect.querySelector(`option[value="${savedModel}"]`)) {
            this.modelSelect.value = savedModel;
        }
    }

    // Get icon for model
    getModelIcon(modelName) {
        if (modelName.includes('coder')) return '💻';
        if (modelName.includes('vision')) return '👁️';
        if (modelName.includes('llama')) return '🦙';
        return '🤖';
    }

    // Restore saved settings from localStorage
    restoreSavedSettings() {
        // Restore model
        const savedModel = localStorage.getItem(this.STORAGE_KEY_MODEL);
        if (savedModel && this.modelSelect?.querySelector(`option[value="${savedModel}"]`)) {
            this.modelSelect.value = savedModel;
        }

        // Restore parameters
        const savedParams = localStorage.getItem(this.STORAGE_KEY_PARAMS);
        if (savedParams) {
            try {
                const params = JSON.parse(savedParams);
                this.applyParamsToUI(params);
            } catch (error) {
                console.error('Failed to parse saved params:', error);
            }
        }
    }

    // Apply parameters to UI
    applyParamsToUI(params) {
        if (params.temperature !== undefined && this.tempSlider && this.tempValue) {
            this.tempSlider.value = params.temperature;
            this.tempValue.value = params.temperature;
        }
        if (params.top_p !== undefined && this.topPSlider && this.topPValue) {
            this.topPSlider.value = params.top_p;
            this.topPValue.value = params.top_p;
        }
        if (params.max_tokens !== undefined && this.maxTokensInput) {
            this.maxTokensInput.value = params.max_tokens;
        }
        if (params.num_ctx !== undefined && this.numCtxInput) {
            this.numCtxInput.value = params.num_ctx;
        }
    }

    // Save selected model
    saveSelectedModel() {
        if (this.modelSelect) {
            localStorage.setItem(this.STORAGE_KEY_MODEL, this.modelSelect.value);
            console.log(`Model saved: ${this.modelSelect.value}`);
        }
    }

    // Save current parameters
    saveCurrentParams() {
        const params = this.getCurrentParams();
        localStorage.setItem(this.STORAGE_KEY_PARAMS, JSON.stringify(params));
        console.log('Parameters saved:', params);
    }

    // Get current parameters
    getCurrentParams() {
        return {
            temperature: parseFloat(this.tempSlider?.value || 0.7),
            top_p: parseFloat(this.topPSlider?.value || 0.9),
            max_tokens: parseInt(this.maxTokensInput?.value || -1),
            num_ctx: parseInt(this.numCtxInput?.value || 4096)
        };
    }

    // Get selected model
    getSelectedModel() {
        return this.modelSelect?.value || null;
    }

    // Apply preset
    applyPreset(presetName) {
        const preset = this.PRESETS[presetName];
        if (!preset) {
            console.error('Unknown preset:', presetName);
            return;
        }

        this.applyParamsToUI(preset);
        this.saveCurrentParams();

        // Visual feedback
        const btn = document.querySelector(`.preset-btn[data-preset="${presetName}"]`);
        if (btn) {
            btn.style.transform = 'scale(0.95)';
            setTimeout(() => {
                btn.style.transform = '';
            }, 100);
        }

        console.log(`Applied "${presetName}" preset`);
    }

    // Reset to defaults (balanced)
    resetToDefaults() {
        this.applyPreset('balanced');
        console.log('Reset to balanced defaults');
    }

    // Get parameters for API request (convert to OpenAI format) (v3.0.0+ with provider)
    getRequestParams() {
        const params = this.getCurrentParams();
        
        // Convert to API format (OpenAI-compatible)
        return {
            model: this.getSelectedModel(),
            temperature: params.temperature,
            top_p: params.top_p,
            max_tokens: params.max_tokens === -1 ? null : params.max_tokens,
            options: {
                num_ctx: params.num_ctx
            }
        };
    }
}

// Initialize global instance
window.modelPanel = new ModelPanelController();

// Auto-init when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.modelPanel.init();
    });
} else {
    window.modelPanel.init();
}



