// Chat UI Manager
class ChatManager {
    constructor() {
        this.currentConversationId = null;
        this.currentModel = null;
        this.messages = [];
        this.isStreaming = false;
        this.attachedFiles = []; // FILE-STORAGE-01: Phase 4, v1.10.0+
        
        // Configure marked for markdown rendering
        marked.setOptions({
            highlight: function(code, lang) {
                if (lang && hljs.getLanguage(lang)) {
                    return hljs.highlight(code, { language: lang }).value;
                }
                return hljs.highlightAuto(code).value;
            },
            breaks: true,
            gfm: true
        });
    }

    // Initialize chat
    init() {
        this.messagesList = document.getElementById('messages-list');
        this.welcomeScreen = document.getElementById('welcome-screen');
        this.messageInput = document.getElementById('message-input');
        this.sendBtn = document.getElementById('send-btn');
        this.chatTitle = document.getElementById('current-chat-title');
        
        this.setupEventListeners();
    }

    // Setup event listeners
    setupEventListeners() {
        // Auto-resize textarea
        this.messageInput.addEventListener('input', () => {
            this.messageInput.style.height = 'auto';
            this.messageInput.style.height = this.messageInput.scrollHeight + 'px';
        });

        // Submit on Enter (but not Shift+Enter)
        this.messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                document.getElementById('chat-form').dispatchEvent(new Event('submit'));
            }
        });

        // File attachment handlers (FILE-STORAGE-01: Phase 4)
        const attachBtn = document.getElementById('attach-btn');
        const fileInput = document.getElementById('file-input');
        
        if (attachBtn && fileInput) {
            attachBtn.addEventListener('click', () => fileInput.click());
            fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
        }
    }

    // Start new conversation
    newConversation() {
        this.currentConversationId = null;
        this.messages = [];
        this.clearMessages();
        this.showWelcome();
        this.chatTitle.textContent = 'New Chat';
        this.messageInput.value = '';
        this.messageInput.focus();
        
        // Clear context manager (v1.9.1+)
        if (window.contextManager) {
            window.contextManager.clear();
        }

        // Clear attached files (FILE-STORAGE-01: Phase 4)
        this.clearAttachedFiles();
    }

    // Load conversation
    async loadConversation(conversationId) {
        try {
            const conversation = await api.getConversation(conversationId);
            const messages = await api.getMessages(conversationId);
            
            this.currentConversationId = conversationId;
            this.messages = messages.messages || [];
            this.currentModel = conversation.model;
            
            this.chatTitle.textContent = conversation.title;
            this.renderMessages();
            this.hideWelcome();
            
        } catch (error) {
            console.error('Failed to load conversation:', error);
            this.showError('Failed to load conversation');
        }
    }

    // Send message
    async sendMessage(content) {
        if (!content.trim() || this.isStreaming) return;
        
        // Get selected model from Model Panel (v1.9.1+)
        if (window.modelPanel) {
            this.currentModel = window.modelPanel.getSelectedModel() || this.currentModel;
            
            // Update context window size from model params
            const params = window.modelPanel.getCurrentParams();
            if (window.contextManager && params.num_ctx) {
                window.contextManager.updateContextWindow(params.num_ctx);
            }
        } else {
            // Fallback to old model-select
            const modelSelect = document.getElementById('model-select');
            this.currentModel = modelSelect?.value || this.currentModel;
        }
        
        if (!this.currentModel) {
            this.showError('Please select a model');
            return;
        }

        // Hide welcome screen
        this.hideWelcome();

        // Add user message
        const userMessage = {
            role: 'user',
            content: content,
            file_ids: this.attachedFiles.map(f => f.id) // FILE-STORAGE-01: Phase 4 (snake_case!)
        };
        
        this.messages.push(userMessage);
        this.renderMessage(userMessage);
        
        // Track in context manager (v1.9.1+)
        if (window.contextManager) {
            window.contextManager.addMessage('user', content);
        }
        
        // Clear input
        this.messageInput.value = '';
        this.messageInput.style.height = 'auto';

        // Scroll to bottom
        this.scrollToBottom();

        // Create or update conversation
        if (!this.currentConversationId) {
            try {
                const title = this.generateTitle(content);
                const conversation = await api.createConversation(title, this.currentModel);
                this.currentConversationId = conversation.id;
                this.chatTitle.textContent = title;
                
                // Save user message (with file_ids if any)
                const fileIds = this.attachedFiles.map(f => f.id);
                await api.createMessage(this.currentConversationId, 'user', content, this.currentModel, fileIds);
                
                // Clear attached files
                this.clearAttachedFiles();
                
                // Update conversations list
                if (window.app) {
                    await window.app.loadConversations();
                }
            } catch (error) {
                console.error('Failed to create conversation:', error);
                this.showError('Failed to save conversation');
            }
        } else {
            // Save user message (with file_ids if any)
            try {
                const fileIds = this.attachedFiles.map(f => f.id);
                await api.createMessage(this.currentConversationId, 'user', content, this.currentModel, fileIds);
                
                // Clear attached files
                this.clearAttachedFiles();
            } catch (error) {
                console.error('Failed to save message:', error);
            }
        }

        // Start streaming response
        await this.streamResponse();
    }

    // Stream assistant response
    async streamResponse() {
        this.isStreaming = true;
        this.sendBtn.disabled = true;

        // Create assistant message element
        const messageDiv = this.createMessageElement('assistant');
        const contentDiv = messageDiv.querySelector('.message-content');
        
        // Add typing indicator
        contentDiv.innerHTML = '<div class="typing-indicator"><div class="typing-dot"></div><div class="typing-dot"></div><div class="typing-dot"></div></div>';
        this.scrollToBottom();

        let fullResponse = '';

        try {
            // Get model parameters from Model Panel (v1.9.1+)
            const requestParams = window.modelPanel ? window.modelPanel.getRequestParams() : {
                model: this.currentModel
            };
            
            // Add RAG parameters if enabled (v1.13.0+)
            if (window.ragManager) {
                const ragParams = window.ragManager.getRAGParams();
                if (ragParams) {
                    Object.assign(requestParams, ragParams);
                }
            }
            
            // Stream from API with parameters
            const stream = api.streamChatMessage(this.messages, requestParams);
            
            // Remove typing indicator
            contentDiv.innerHTML = '';

            for await (const chunk of stream) {
                fullResponse += chunk;
                contentDiv.innerHTML = marked.parse(fullResponse);
                this.scrollToBottom();
            }

            // Save assistant message
            const assistantMessage = {
                role: 'assistant',
                content: fullResponse
            };
            
            this.messages.push(assistantMessage);
            
            // Track in context manager (v1.9.1+)
            if (window.contextManager) {
                window.contextManager.addMessage('assistant', fullResponse);
            }

            if (this.currentConversationId) {
                try {
                    await api.createMessage(this.currentConversationId, 'assistant', fullResponse, this.currentModel);
                } catch (error) {
                    console.error('Failed to save assistant message:', error);
                }
            }

        } catch (error) {
            console.error('Streaming error:', error);
            contentDiv.innerHTML = '<p style="color: var(--error-color);">Error: Failed to get response</p>';
        } finally {
            this.isStreaming = false;
            this.sendBtn.disabled = false;
            this.messageInput.focus();
        }
    }

    // Render all messages
    renderMessages() {
        this.clearMessages();
        this.messages.forEach(msg => this.renderMessage(msg));
        this.scrollToBottom();
    }

    // Render single message
    renderMessage(message) {
        const messageDiv = this.createMessageElement(message.role);
        const contentDiv = messageDiv.querySelector('.message-content');
        
        if (message.role === 'user') {
            contentDiv.textContent = message.content;
            
            // Show attached files if any (FILE-STORAGE-01: Phase 4)
            if (message.file_ids && message.file_ids.length > 0) {
                this.renderMessageFiles(messageDiv, message.file_ids);
            }
        } else {
            contentDiv.innerHTML = marked.parse(message.content);
        }
    }

    // Render files attached to message (FILE-STORAGE-01: Phase 4)
    renderMessageFiles(messageDiv, fileIds) {
        const filesContainer = document.createElement('div');
        filesContainer.className = 'message-files';
        filesContainer.style.cssText = 'margin-top: 8px; display: flex; flex-wrap: wrap; gap: 4px;';
        
        fileIds.forEach(fileId => {
            // Find file in attachedFiles or create a placeholder
            const file = this.attachedFiles.find(f => f.id === fileId) || { id: fileId, filename: 'Файл', mime_type: 'unknown' };
            
            const fileBadge = document.createElement('span');
            fileBadge.className = 'message-file-badge';
            fileBadge.style.cssText = 'display: inline-flex; align-items: center; gap: 4px; padding: 4px 8px; background: var(--bg-tertiary); border-radius: 12px; font-size: 0.75rem; cursor: pointer;';
            fileBadge.innerHTML = `
                <span>${this.getFileIcon(file.mime_type)}</span>
                <span>${file.filename}</span>
            `;
            fileBadge.title = 'Просмотреть содержимое';
            fileBadge.onclick = () => this.viewFileContent(file.id);
            
            filesContainer.appendChild(fileBadge);
        });
        
        messageDiv.appendChild(filesContainer);
    }

    // View file content in modal (FILE-STORAGE-01: Phase 4)
    async viewFileContent(fileId) {
        try {
            const response = await api.request(`${api.baseURL}/api/files/${fileId}/text`);
            if (response.ok) {
                const data = await response.json();
                const text = data.text || '[Текст не извлечен]';
                
                // Use existing modal or alert
                if (window.modal) {
                    await window.modal.info(text, 'Содержимое файла');
                } else {
                    alert(text);
                }
            } else {
                toast.error('Не удалось загрузить файл');
            }
        } catch (error) {
            console.error('Failed to load file:', error);
            toast.error('Ошибка загрузки файла');
        }
    }

    // Create message element
    createMessageElement(role) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        const avatar = document.createElement('div');
        avatar.className = 'message-avatar';
        avatar.textContent = role === 'user' ? 'U' : '🤖';
        
        const content = document.createElement('div');
        content.className = 'message-content';
        
        messageDiv.appendChild(avatar);
        messageDiv.appendChild(content);
        this.messagesList.appendChild(messageDiv);
        
        return messageDiv;
    }

    // Clear all messages
    clearMessages() {
        this.messagesList.innerHTML = '';
    }

    // Show/hide welcome screen
    showWelcome() {
        this.welcomeScreen.style.display = 'flex';
    }

    hideWelcome() {
        this.welcomeScreen.style.display = 'none';
    }

    // Scroll to bottom
    scrollToBottom() {
        const container = document.getElementById('messages-container');
        container.scrollTop = container.scrollHeight;
    }

    // Generate conversation title from first message
    generateTitle(content) {
        const maxLength = 50;
        if (content.length <= maxLength) return content;
        return content.substring(0, maxLength) + '...';
    }

    // Show error message
    showError(message) {
        // Could implement a toast notification here
        console.error(message);
        toast.error(message);
    }

    // ========================================
    // File Attachment Methods (FILE-STORAGE-01: Phase 4, v1.10.0+)
    // ========================================

    // Handle file selection
    async handleFileSelect(event) {
        const files = Array.from(event.target.files);
        if (files.length === 0) return;

        for (const file of files) {
            try {
                // Upload file
                const uploadedFile = await this.uploadFile(file);
                this.attachedFiles.push(uploadedFile);
                
                // Show preview
                this.renderAttachedFiles();
                toast.success(`Uploaded: ${file.name}`);
            } catch (error) {
                console.error('Failed to upload file:', error);
                toast.error(`Failed to upload ${file.name}`);
            }
        }

        // Clear file input
        event.target.value = '';
    }

    // Upload file to server
    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        // Use access_token (not 'token')
        const accessToken = localStorage.getItem('access_token');
        const response = await fetch('/api/files/upload', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${accessToken}`
            },
            body: formData
        });

        // Handle 401 - try to refresh token
        if (response.status === 401 && window.api) {
            const refreshed = await window.api.refreshAccessToken();
            if (refreshed) {
                // Retry upload with new token
                const newToken = localStorage.getItem('access_token');
                const retryResponse = await fetch('/api/files/upload', {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${newToken}`
                    },
                    body: formData
                });
                
                if (!retryResponse.ok) {
                    throw new Error('Upload failed after token refresh');
                }
                
                const data = await retryResponse.json();
                // API returns file object directly (not wrapped in 'file' key)
                return data;
            }
        }

        if (!response.ok) {
            throw new Error('Upload failed');
        }

        const data = await response.json();
        // API returns file object directly (not wrapped in 'file' key)
        return data;
    }

    // Render attached files preview
    renderAttachedFiles() {
        const preview = document.getElementById('attached-files-preview');
        
        if (this.attachedFiles.length === 0) {
            preview.style.display = 'none';
            return;
        }

        preview.style.display = 'block';
        preview.innerHTML = this.attachedFiles.map((file, index) => `
            <div class="attached-file-chip">
                <span class="file-icon">${this.getFileIcon(file.mime_type)}</span>
                <span class="file-name" title="${file.filename}">${file.filename}</span>
                <button class="remove-file" onclick="chatManager.removeAttachedFile(${index})" type="button">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                        <path d="M18 6L6 18M6 6l12 12" stroke-width="2"/>
                    </svg>
                </button>
            </div>
        `).join('');
    }

    // Remove attached file
    removeAttachedFile(index) {
        this.attachedFiles.splice(index, 1);
        this.renderAttachedFiles();
    }

    // Get file icon emoji
    getFileIcon(mimeType) {
        if (mimeType.includes('pdf')) return '📕';
        if (mimeType.includes('word')) return '📘';
        if (mimeType.includes('csv')) return '📊';
        if (mimeType.includes('sheet')) return '📗';
        if (mimeType.includes('text')) return '📄';
        return '📎';
    }

    // Clear attached files
    clearAttachedFiles() {
        this.attachedFiles = [];
        this.renderAttachedFiles();
    }
}

// RAG Manager (v1.13.0+)
class RAGManager {
    constructor() {
        this.enabled = false;
        this.selectedSources = [];
        this.sources = [];
    }

    async init() {
        // Check if RAG is enabled in system configuration
        const ragEnabled = await api.isRAGEnabled();
        if (!ragEnabled) {
            // Hide entire RAG section
            const ragSection = document.getElementById('rag-section');
            if (ragSection) {
                ragSection.style.display = 'none';
            }
            console.info('RAG system is disabled by administrator');
            return;
        }

        this.ragToggle = document.getElementById('rag-enabled');
        this.ragControls = document.getElementById('rag-controls');
        this.ragSourcesSelect = document.getElementById('rag-sources');
        this.ragTopKSlider = document.getElementById('rag-top-k');
        this.ragTopKValue = document.getElementById('rag-top-k-value');
        this.ragMinScoreSlider = document.getElementById('rag-min-score');
        this.ragMinScoreValue = document.getElementById('rag-min-score-value');
        this.ragRerankCheckbox = document.getElementById('rag-rerank');

        this.setupEventListeners();
        await this.loadSources();
    }

    setupEventListeners() {
        // Toggle RAG
        this.ragToggle.addEventListener('change', (e) => {
            this.enabled = e.target.checked;
            this.ragControls.style.display = this.enabled ? 'block' : 'none';
        });

        // Sync slider with number input
        this.ragTopKSlider.addEventListener('input', (e) => {
            this.ragTopKValue.value = e.target.value;
        });
        this.ragTopKValue.addEventListener('input', (e) => {
            this.ragTopKSlider.value = e.target.value;
        });

        this.ragMinScoreSlider.addEventListener('input', (e) => {
            this.ragMinScoreValue.value = e.target.value;
        });
        this.ragMinScoreValue.addEventListener('input', (e) => {
            this.ragMinScoreSlider.value = e.target.value;
        });
    }

    async loadSources() {
        try {
            const response = await api.getRAGSources();
            this.sources = response.sources || [];
            
            // Populate select
            this.ragSourcesSelect.innerHTML = '';
            
            if (this.sources.length === 0) {
                const option = document.createElement('option');
                option.value = '';
                option.textContent = 'No sources available';
                option.disabled = true;
                this.ragSourcesSelect.appendChild(option);
            } else {
                this.sources.forEach(source => {
                    const option = document.createElement('option');
                    option.value = source.id;
                    option.textContent = `${source.name} (${source.source_type})`;
                    this.ragSourcesSelect.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Failed to load RAG sources:', error);
        }
    }

    getRAGParams() {
        // If RAG UI was not initialized (system disabled), return null
        if (!this.ragToggle || !this.enabled) {
            return null;
        }

        // Get selected sources
        const selectedOptions = Array.from(this.ragSourcesSelect.selectedOptions);
        const sourceIDs = selectedOptions.map(opt => opt.value).filter(v => v);

        if (sourceIDs.length === 0) {
            return null; // No sources selected
        }

        return {
            rag_enabled: true,
            rag_source_ids: sourceIDs,
            rag_top_k: parseInt(this.ragTopKSlider.value),
            rag_min_score: parseFloat(this.ragMinScoreSlider.value),
            rag_rerank: this.ragRerankCheckbox.checked
        };
    }

    isEnabled() {
        return this.enabled;
    }
}

// Create global managers
window.chatManager = new ChatManager();
window.ragManager = new RAGManager();

