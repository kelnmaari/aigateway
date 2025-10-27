// Main Application
class App {
    constructor() {
        this.user = null;
        this.models = [];
        this.conversations = [];
    }

    // Initialize application
    async init() {
        // Check authentication
        if (!this.isAuthenticated()) {
            window.location.href = '/web/login.html';
            return;
        }

        try {
            // Load user info
            await this.loadUser();
            
            // Load models
            await this.loadModels();
            
            // Load conversations
            await this.loadConversations();
            
            // Initialize chat manager
            chatManager.init();
            
            // Initialize RAG manager (v1.13.0+)
            if (window.ragManager) {
                await ragManager.init();
            }
            
            // Setup event listeners
            this.setupEventListeners();
            
            console.log('✅ Application initialized');
            
        } catch (error) {
            console.error('Failed to initialize app:', error);
            if (error.message.includes('401') || error.message.includes('unauthorized')) {
                this.logout();
            }
        }
    }

    // Check if user is authenticated
    isAuthenticated() {
        return !!localStorage.getItem('access_token');
    }

    // Load current user
    async loadUser() {
        try {
            this.user = await api.getCurrentUser();
            const userName = document.getElementById('user-name');
            if (userName) {
                userName.textContent = this.user.full_name || this.user.username;
            }
            
            // Show Admin link for admins in chat header
            if (this.user.is_admin) {
                const adminPlaceholder = document.getElementById('admin-link-placeholder');
                if (adminPlaceholder) {
                    adminPlaceholder.innerHTML = `
                        <a href="/admin.html" class="nav-link" title="Admin Panel">
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" stroke-width="2"/>
                                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" stroke-width="2"/>
                            </svg>
                            Admin
                        </a>
                    `;
                }
            }
        } catch (error) {
            console.error('Failed to load user:', error);
            throw error;
        }
    }

    // Load available models
    async loadModels() {
        try {
            this.models = await api.getModels();
            this.renderModels();
        } catch (error) {
            console.error('Failed to load models:', error);
            this.models = [];
        }
    }

    // Render models dropdown
    renderModels() {
        const modelSelect = document.getElementById('model-select');
        
        // Check if element exists (not all pages have model-select)
        if (!modelSelect) {
            return;
        }
        
        modelSelect.innerHTML = '';
        
        if (this.models.length === 0) {
            modelSelect.innerHTML = '<option value="">No models available</option>';
            return;
        }

        this.models.forEach(model => {
            const option = document.createElement('option');
            option.value = model.id;
            option.textContent = model.id;
            modelSelect.appendChild(option);
        });

        // Select first model by default
        if (this.models.length > 0 && !chatManager.currentModel) {
            chatManager.currentModel = this.models[0].id;
        }
    }

    // Load conversations
    async loadConversations() {
        try {
            const response = await api.getConversations();
            this.conversations = response.conversations || [];
            this.renderConversations();
        } catch (error) {
            console.error('Failed to load conversations:', error);
            this.conversations = [];
        }
    }

    // Render conversations list
    renderConversations() {
        const conversationsList = document.getElementById('conversations-list');
        conversationsList.innerHTML = '';

        if (this.conversations.length === 0) {
            conversationsList.innerHTML = '<p style="text-align: center; color: var(--text-secondary); padding: 20px;">No conversations yet</p>';
            return;
        }

        this.conversations.forEach(conv => {
            const item = this.createConversationItem(conv);
            conversationsList.appendChild(item);
        });
    }

    // Create conversation list item
    createConversationItem(conversation) {
        const item = document.createElement('div');
        item.className = 'conversation-item';
        if (conversation.id === chatManager.currentConversationId) {
            item.classList.add('active');
        }

        const title = document.createElement('div');
        title.className = 'conversation-title';
        title.textContent = conversation.title;

        const actions = document.createElement('div');
        actions.className = 'conversation-actions';

        // Delete button
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'btn btn-icon';
        deleteBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-width="2"/></svg>';
        deleteBtn.onclick = async (e) => {
            e.stopPropagation();
            if (confirm('Delete this conversation?')) {
                await this.deleteConversation(conversation.id);
            }
        };

        actions.appendChild(deleteBtn);

        item.appendChild(title);
        item.appendChild(actions);

        // Click to load conversation
        item.onclick = () => {
            document.querySelectorAll('.conversation-item').forEach(el => el.classList.remove('active'));
            item.classList.add('active');
            chatManager.loadConversation(conversation.id);
        };

        return item;
    }

    // Delete conversation
    async deleteConversation(id) {
        try {
            await api.deleteConversation(id);
            
            // If deleted current conversation, start new one
            if (id === chatManager.currentConversationId) {
                chatManager.newConversation();
            }
            
            // Reload conversations
            await this.loadConversations();
            
        } catch (error) {
            console.error('Failed to delete conversation:', error);
            alert('Failed to delete conversation');
        }
    }

    // Setup event listeners
    setupEventListeners() {
        // Chat form submit
        document.getElementById('chat-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            const content = document.getElementById('message-input').value.trim();
            if (content) {
                await chatManager.sendMessage(content);
            }
        });

        // New chat button
        document.getElementById('new-chat-btn').addEventListener('click', () => {
            chatManager.newConversation();
        });

        // Logout button
        document.getElementById('logout-btn').addEventListener('click', () => {
            this.logout();
        });

        // Sidebar toggle (mobile)
        const sidebarToggle = document.getElementById('sidebar-toggle');
        const sidebar = document.getElementById('sidebar');
        
        if (sidebarToggle) {
            sidebarToggle.addEventListener('click', () => {
                sidebar.classList.toggle('active');
            });
        }

        // Close sidebar when clicking outside (mobile)
        document.addEventListener('click', (e) => {
            if (window.innerWidth <= 768) {
                if (!sidebar.contains(e.target) && !sidebarToggle.contains(e.target)) {
                    sidebar.classList.remove('active');
                }
            }
        });
    }

    // Logout
    logout() {
        api.logout();
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new App();
    app.init();
});

