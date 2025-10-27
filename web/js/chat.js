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

        // Formatting toolbar handlers
        this.setupFormattingToolbar();
    }

    // Setup formatting toolbar for markdown
    setupFormattingToolbar() {
        const toolbar = document.getElementById('formatting-toolbar');
        if (!toolbar) return;

        // Show toolbar on text selection
        this.messageInput.addEventListener('mouseup', () => this.handleTextSelection());
        this.messageInput.addEventListener('keyup', () => this.handleTextSelection());

        // Keyboard shortcuts
        this.messageInput.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey)) {
                switch(e.key.toLowerCase()) {
                    case 'b':
                        e.preventDefault();
                        this.applyFormat('bold');
                        break;
                    case 'i':
                        e.preventDefault();
                        this.applyFormat('italic');
                        break;
                    case 'k':
                        e.preventDefault();
                        if (e.shiftKey) {
                            this.applyFormat('codeblock');
                        } else {
                            this.applyFormat('code');
                        }
                        break;
                    case 'l':
                        e.preventDefault();
                        this.applyFormat('link');
                        break;
                }
            }
        });

        // Toolbar button clicks
        toolbar.querySelectorAll('.toolbar-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const format = btn.dataset.format;
                this.applyFormat(format);
            });
        });

        // Hide toolbar on click outside
        document.addEventListener('mousedown', (e) => {
            if (!toolbar.contains(e.target) && e.target !== this.messageInput) {
                this.hideFormattingToolbar();
            }
        });
    }

    // Handle text selection in message input
    handleTextSelection() {
        const selection = window.getSelection();
        const selectedText = this.messageInput.value.substring(
            this.messageInput.selectionStart,
            this.messageInput.selectionEnd
        );

        const toolbar = document.getElementById('formatting-toolbar');
        
        if (selectedText.length > 0 && this.messageInput === document.activeElement) {
            // Position toolbar above selection
            const rect = this.messageInput.getBoundingClientRect();
            const selectionStart = this.messageInput.selectionStart;
            
            // Simple positioning - above textarea
            toolbar.style.left = `${rect.left + 20}px`;
            toolbar.style.top = `${rect.top - 50}px`;
            
            // Show toolbar with animation
            toolbar.style.display = 'flex';
            setTimeout(() => toolbar.classList.add('visible'), 10);
        } else {
            this.hideFormattingToolbar();
        }
    }

    // Hide formatting toolbar
    hideFormattingToolbar() {
        const toolbar = document.getElementById('formatting-toolbar');
        if (toolbar) {
            toolbar.classList.remove('visible');
            setTimeout(() => {
                if (!toolbar.classList.contains('visible')) {
                    toolbar.style.display = 'none';
                }
            }, 200);
        }
    }

    // Apply markdown formatting
    applyFormat(format) {
        const start = this.messageInput.selectionStart;
        const end = this.messageInput.selectionEnd;
        const selectedText = this.messageInput.value.substring(start, end);
        
        if (!selectedText && format !== 'ul' && format !== 'ol' && format !== 'codeblock') {
            return; // Need selection for inline formats
        }

        let before = '', after = '', replacement = '';
        
        switch(format) {
            case 'bold':
                before = '**';
                after = '**';
                replacement = `${before}${selectedText || 'bold text'}${after}`;
                break;
            case 'italic':
                before = '*';
                after = '*';
                replacement = `${before}${selectedText || 'italic text'}${after}`;
                break;
            case 'code':
                before = '`';
                after = '`';
                replacement = `${before}${selectedText || 'code'}${after}`;
                break;
            case 'codeblock':
                const lang = selectedText ? '' : 'javascript';
                replacement = `\n\`\`\`${lang}\n${selectedText || '// your code here'}\n\`\`\`\n`;
                break;
            case 'link':
                const url = prompt('Enter URL:', 'https://');
                if (url) {
                    replacement = `[${selectedText || 'link text'}](${url})`;
                }
                break;
            case 'ul':
                const ulLines = selectedText.split('\n').filter(l => l.trim());
                replacement = ulLines.length > 0 
                    ? ulLines.map(line => `- ${line}`).join('\n')
                    : '- List item';
                break;
            case 'ol':
                const olLines = selectedText.split('\n').filter(l => l.trim());
                replacement = olLines.length > 0
                    ? olLines.map((line, i) => `${i + 1}. ${line}`).join('\n')
                    : '1. List item';
                break;
        }

        if (replacement !== undefined) {
            const newValue = 
                this.messageInput.value.substring(0, start) +
                replacement +
                this.messageInput.value.substring(end);
            
            this.messageInput.value = newValue;
            
            // Restore focus and selection
            this.messageInput.focus();
            const newCursorPos = start + replacement.length;
            this.messageInput.setSelectionRange(newCursorPos, newCursorPos);
            
            // Trigger input event for auto-resize
            this.messageInput.dispatchEvent(new Event('input'));
            
            // Hide toolbar
            this.hideFormattingToolbar();
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
        
        // Parse markdown for all messages (user + assistant)
        contentDiv.innerHTML = marked.parse(message.content);
        
        // Show attached files if any (FILE-STORAGE-01: Phase 4)
        if (message.role === 'user' && message.file_ids && message.file_ids.length > 0) {
            this.renderMessageFiles(messageDiv, message.file_ids);
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

// ========================================
// Export/Import Functions (v2.0.0)
// ========================================

// Toggle export dropdown
document.getElementById('export-menu-btn')?.addEventListener('click', (e) => {
    e.stopPropagation();
    const dropdown = document.getElementById('export-dropdown');
    dropdown.style.display = dropdown.style.display === 'none' ? 'block' : 'none';
});

// Close dropdown when clicking outside
document.addEventListener('click', (e) => {
    const dropdown = document.getElementById('export-dropdown');
    const menuBtn = document.getElementById('export-menu-btn');
    if (dropdown && !dropdown.contains(e.target) && e.target !== menuBtn && !menuBtn.contains(e.target)) {
        dropdown.style.display = 'none';
    }
});

// Export as JSON
document.getElementById('export-json-btn')?.addEventListener('click', async () => {
    document.getElementById('export-dropdown').style.display = 'none';
    await exportConversation('json');
});

// Export as Markdown
document.getElementById('export-markdown-btn')?.addEventListener('click', async () => {
    document.getElementById('export-dropdown').style.display = 'none';
    await exportConversation('markdown');
});

// Export as Text
document.getElementById('export-text-btn')?.addEventListener('click', async () => {
    document.getElementById('export-dropdown').style.display = 'none';
    await exportConversation('text');
});

// Open import modal
document.getElementById('import-btn')?.addEventListener('click', () => {
    document.getElementById('export-dropdown').style.display = 'none';
    openImportModal();
});

// Import submit
document.getElementById('import-submit-btn')?.addEventListener('click', async () => {
    await importConversation();
});

// Export conversation to specified format
async function exportConversation(format) {
    const convId = window.chatManager.currentConversationId;
    
    if (!convId) {
        window.toast.error('No conversation selected');
        return;
    }
    
    try {
        // Make direct fetch to get blob response
        const response = await fetch(`${api.baseURL}/api/conversations/${convId}/export?format=${format}`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${api.accessToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error(`Export failed: ${response.statusText}`);
        }
        
        // Get response as blob
        const blob = await response.blob();
        
        // Generate filename
        const convTitle = window.chatManager.chatTitle?.textContent || 'conversation';
        const sanitizedTitle = convTitle.replace(/[^a-z0-9]/gi, '_').toLowerCase();
        
        let filename;
        switch(format) {
            case 'json':
                filename = `${sanitizedTitle}_${convId}.json`;
                break;
            case 'markdown':
                filename = `${sanitizedTitle}_${convId}.md`;
                break;
            case 'text':
                filename = `${sanitizedTitle}_${convId}.txt`;
                break;
        }
        
        // Trigger download
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        
        window.toast.success(`Conversation exported as ${format.toUpperCase()}`);
    } catch (error) {
        console.error('Export failed:', error);
        window.toast.error(`Failed to export conversation: ${error.message}`);
    }
}

// Open import modal
function openImportModal() {
    const modal = document.getElementById('import-modal');
    modal.classList.add('show');
    modal.style.display = 'flex';
    
    // Reset form
    document.getElementById('import-file-input').value = '';
    document.getElementById('import-preserve-timestamps').checked = false;
    document.getElementById('import-preserve-ids').checked = false;
    document.getElementById('import-error').style.display = 'none';
    document.getElementById('import-success').style.display = 'none';
}

// Close import modal
function closeImportModal() {
    const modal = document.getElementById('import-modal');
    modal.classList.remove('show');
    modal.style.display = 'none';
}

// Import conversation from JSON file
async function importConversation() {
    const fileInput = document.getElementById('import-file-input');
    const preserveTimestamps = document.getElementById('import-preserve-timestamps').checked;
    const preserveIds = document.getElementById('import-preserve-ids').checked;
    const errorDiv = document.getElementById('import-error');
    const successDiv = document.getElementById('import-success');
    
    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';
    
    if (!fileInput.files || fileInput.files.length === 0) {
        errorDiv.textContent = 'Please select a JSON file';
        errorDiv.style.display = 'block';
        return;
    }
    
    const file = fileInput.files[0];
    
    try {
        // Read file content
        const fileContent = await file.text();
        const conversationData = JSON.parse(fileContent);
        
        // Validate JSON structure
        if (!conversationData.id || !conversationData.messages) {
            throw new Error('Invalid conversation JSON format');
        }
        
        // Prepare import request
        const importRequest = {
            conversation: conversationData,
            options: {
                preserve_timestamps: preserveTimestamps,
                preserve_ids: preserveIds
            }
        };
        
        // Call import API
        const result = await api.request('/api/conversations/import', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(importRequest)
        });
        
        successDiv.textContent = `Conversation imported successfully! (${result.messages_imported} messages)`;
        successDiv.style.display = 'block';
        
        // Reload conversations list
        setTimeout(() => {
            closeImportModal();
            window.location.reload(); // Reload to show new conversation
        }, 1500);
        
    } catch (error) {
        console.error('Import failed:', error);
        errorDiv.textContent = `Import failed: ${error.message}`;
        errorDiv.style.display = 'block';
    }
}

// Make functions globally accessible
window.openImportModal = openImportModal;
window.closeImportModal = closeImportModal;



