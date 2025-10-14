// Chat UI Manager
class ChatManager {
    constructor() {
        this.currentConversationId = null;
        this.currentModel = null;
        this.messages = [];
        this.isStreaming = false;
        
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
            content: content
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
                
                // Save user message
                await api.createMessage(this.currentConversationId, 'user', content, this.currentModel);
                
                // Update conversations list
                if (window.app) {
                    await window.app.loadConversations();
                }
            } catch (error) {
                console.error('Failed to create conversation:', error);
                this.showError('Failed to save conversation');
            }
        } else {
            // Save user message
            try {
                await api.createMessage(this.currentConversationId, 'user', content, this.currentModel);
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
        } else {
            contentDiv.innerHTML = marked.parse(message.content);
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
}

// Create global chat manager
window.chatManager = new ChatManager();

