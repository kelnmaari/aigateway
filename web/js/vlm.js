// VLM (Vision Language Model) Support (v3.0.4+)
// Handles image attachment, preview, and multimodal message format

class VLMManager {
    constructor() {
        this.attachedImages = []; // Array of {file: File, base64: string, preview: string}
        this.maxImages = 5;
        this.maxImageSize = 10 * 1024 * 1024; // 10MB per image
    }

    init() {
        this.imageBtn = document.getElementById('image-btn');
        this.imageInput = document.getElementById('image-input');
        this.imagesPreview = document.getElementById('attached-images-preview');
        
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Image button click
        this.imageBtn?.addEventListener('click', () => {
            this.imageInput?.click();
        });

        // Image selection
        this.imageInput?.addEventListener('change', (e) => {
            this.handleImageSelection(e.target.files);
        });
    }

    async handleImageSelection(files) {
        if (!files || files.length === 0) return;

        // Check total count
        if (this.attachedImages.length + files.length > this.maxImages) {
            alert(`Maximum ${this.maxImages} images allowed`);
            return;
        }

        for (const file of files) {
            // Validate file type
            if (!file.type.startsWith('image/')) {
                console.warn('Skipping non-image file:', file.name);
                continue;
            }

            // Validate file size
            if (file.size > this.maxImageSize) {
                alert(`Image ${file.name} is too large. Max size: 10MB`);
                continue;
            }

            try {
                // Convert to base64
                const base64 = await this.fileToBase64(file);
                
                // Create preview URL
                const previewURL = URL.createObjectURL(file);

                // Add to attached images
                this.attachedImages.push({
                    file: file,
                    base64: base64,
                    preview: previewURL,
                    name: file.name
                });

                console.log(`Image attached: ${file.name} (${(file.size / 1024).toFixed(2)} KB)`);
            } catch (error) {
                console.error('Failed to process image:', error);
                alert(`Failed to process ${file.name}`);
            }
        }

        // Reset input
        this.imageInput.value = '';

        // Update preview
        this.updatePreview();
    }

    fileToBase64(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                // Extract base64 data (remove data:image/...;base64, prefix)
                const base64 = reader.result;
                resolve(base64);
            };
            reader.onerror = reject;
            reader.readAsDataURL(file);
        });
    }

    updatePreview() {
        if (this.attachedImages.length === 0) {
            this.imagesPreview.style.display = 'none';
            return;
        }

        this.imagesPreview.style.display = 'flex';
        this.imagesPreview.innerHTML = this.attachedImages.map((img, index) => `
            <div class="attached-image-item" data-index="${index}">
                <img src="${img.preview}" alt="${img.name}" class="image-preview-thumb">
                <div class="image-info">
                    <span class="image-name">${this.truncateName(img.name)}</span>
                    <span class="image-size">${this.formatFileSize(img.file.size)}</span>
                </div>
                <button type="button" class="image-remove-btn" onclick="window.vlmManager.removeImage(${index})" title="Remove image">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path d="M18 6L6 18M6 6l12 12" stroke-width="2" stroke-linecap="round"/>
                    </svg>
                </button>
            </div>
        `).join('');
    }

    removeImage(index) {
        if (index >= 0 && index < this.attachedImages.length) {
            // Revoke preview URL
            URL.revokeObjectURL(this.attachedImages[index].preview);
            
            // Remove from array
            this.attachedImages.splice(index, 1);
            
            // Update preview
            this.updatePreview();
            
            console.log(`Image removed. Remaining: ${this.attachedImages.length}`);
        }
    }

    clearImages() {
        // Revoke all preview URLs
        this.attachedImages.forEach(img => {
            URL.revokeObjectURL(img.preview);
        });
        
        this.attachedImages = [];
        this.updatePreview();
        
        console.log('All images cleared');
    }

    hasImages() {
        return this.attachedImages.length > 0;
    }

    getImages() {
        return this.attachedImages;
    }

    // Build multimodal content array for OpenAI API
    buildMultimodalContent(textPrompt) {
        const content = [];

        // Add text part
        if (textPrompt && textPrompt.trim()) {
            content.push({
                type: 'text',
                text: textPrompt.trim()
            });
        }

        // Add image parts
        this.attachedImages.forEach(img => {
            content.push({
                type: 'image_url',
                image_url: {
                    url: img.base64,
                    detail: 'auto' // auto, low, high
                }
            });
        });

        return content;
    }

    // Helper methods
    truncateName(name, maxLength = 20) {
        if (name.length <= maxLength) return name;
        const ext = name.split('.').pop();
        const nameWithoutExt = name.substring(0, name.lastIndexOf('.'));
        const truncated = nameWithoutExt.substring(0, maxLength - ext.length - 4) + '...';
        return truncated + '.' + ext;
    }

    formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    // Check if current provider supports VLM
    isVLMProvider() {
        const provider = window.modelPanel?.getSelectedProvider() || 'ollama';
        return provider === 'yzma'; // Only yzma supports VLM
    }

    // Validate before sending
    validateBeforeSend() {
        if (!this.hasImages()) {
            return { valid: true };
        }

        // Check provider
        if (!this.isVLMProvider()) {
            return {
                valid: false,
                error: 'Images are only supported with yzma provider. Switch to "yzma Local" provider.'
            };
        }

        // Check model is VLM
        const model = window.modelPanel?.getSelectedModel();
        if (!model) {
            return {
                valid: false,
                error: 'No model selected'
            };
        }

        return { valid: true };
    }
}

// Initialize global instance
window.vlmManager = new VLMManager();

// Auto-init when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.vlmManager.init();
    });
} else {
    window.vlmManager.init();
}

