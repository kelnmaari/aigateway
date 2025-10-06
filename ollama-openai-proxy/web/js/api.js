// API Client for Ollama Proxy
class API {
    constructor() {
        this.baseURL = window.location.origin;
        this.accessToken = localStorage.getItem('access_token');
        this.refreshToken = localStorage.getItem('refresh_token');
    }

    // Get auth headers
    getAuthHeaders() {
        const headers = {
            'Content-Type': 'application/json'
        };
        
        if (this.accessToken) {
            headers['Authorization'] = `Bearer ${this.accessToken}`;
        }
        
        return headers;
    }

    // Refresh access token
    async refreshAccessToken() {
        try {
            const response = await fetch(`${this.baseURL}/api/auth/refresh`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    refresh_token: this.refreshToken
                })
            });

            if (!response.ok) {
                throw new Error('Token refresh failed');
            }

            const data = await response.json();
            this.accessToken = data.access_token;
            localStorage.setItem('access_token', data.access_token);
            
            return true;
        } catch (error) {
            console.error('Failed to refresh token:', error);
            this.logout();
            return false;
        }
    }

    // Make authenticated request with token refresh
    async request(url, options = {}) {
        try {
            const response = await fetch(url, {
                ...options,
                headers: {
                    ...this.getAuthHeaders(),
                    ...options.headers
                }
            });

            // If unauthorized, try refreshing token
            if (response.status === 401) {
                const refreshed = await this.refreshAccessToken();
                if (refreshed) {
                    // Retry request with new token
                    return fetch(url, {
                        ...options,
                        headers: {
                            ...this.getAuthHeaders(),
                            ...options.headers
                        }
                    });
                }
            }

            return response;
        } catch (error) {
            console.error('Request failed:', error);
            throw error;
        }
    }

    // ==================== Auth APIs ====================

    async getCurrentUser() {
        const response = await this.request(`${this.baseURL}/api/auth/me`);
        if (!response.ok) throw new Error('Failed to get current user');
        return response.json();
    }

    async logout() {
        try {
            await this.request(`${this.baseURL}/api/auth/logout`, {
                method: 'POST'
            });
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('user');
            window.location.href = '/web/login.html';
        }
    }

    // ==================== Models APIs ====================

    async getModels() {
        const response = await fetch(`${this.baseURL}/v1/models`, {
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('Failed to fetch models');
        const data = await response.json();
        return data.data || [];
    }

    // ==================== Chat APIs ====================

    async sendChatMessage(messages, model, stream = false) {
        const response = await this.request(`${this.baseURL}/v1/chat/completions`, {
            method: 'POST',
            body: JSON.stringify({
                model: model,
                messages: messages,
                stream: stream,
                temperature: 0.7,
                max_tokens: 2048
            })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Chat request failed');
        }

        return response;
    }

    // Stream chat response
    async *streamChatMessage(messages, model) {
        const response = await this.sendChatMessage(messages, model, true);
        
        if (!response.ok) {
            throw new Error('Stream request failed');
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        try {
            while (true) {
                const { done, value } = await reader.read();
                
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop(); // Keep incomplete line in buffer

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (!trimmed || trimmed === 'data: [DONE]') continue;
                    
                    if (trimmed.startsWith('data: ')) {
                        try {
                            const data = JSON.parse(trimmed.slice(6));
                            const content = data.choices?.[0]?.delta?.content;
                            if (content) {
                                yield content;
                            }
                        } catch (e) {
                            console.error('Failed to parse SSE:', e, trimmed);
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }

    // ==================== Conversation APIs ====================

    async getConversations() {
        const response = await this.request(`${this.baseURL}/api/conversations`);
        if (!response.ok) throw new Error('Failed to fetch conversations');
        return response.json();
    }

    async getConversation(id) {
        const response = await this.request(`${this.baseURL}/api/conversations/${id}`);
        if (!response.ok) throw new Error('Failed to fetch conversation');
        return response.json();
    }

    async createConversation(title, model) {
        const response = await this.request(`${this.baseURL}/api/conversations`, {
            method: 'POST',
            body: JSON.stringify({
                title: title || 'New Conversation',
                model: model,
                system_prompt: 'You are a helpful AI assistant.'
            })
        });
        
        if (!response.ok) throw new Error('Failed to create conversation');
        return response.json();
    }

    async updateConversation(id, updates) {
        const response = await this.request(`${this.baseURL}/api/conversations/${id}`, {
            method: 'PUT',
            body: JSON.stringify(updates)
        });
        
        if (!response.ok) throw new Error('Failed to update conversation');
        return response.json();
    }

    async deleteConversation(id) {
        const response = await this.request(`${this.baseURL}/api/conversations/${id}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) throw new Error('Failed to delete conversation');
        return response.ok;
    }

    // ==================== Message APIs ====================

    async getMessages(conversationId) {
        const response = await this.request(`${this.baseURL}/api/conversations/${conversationId}/messages`);
        if (!response.ok) throw new Error('Failed to fetch messages');
        return response.json();
    }

    async createMessage(conversationId, role, content, model) {
        const response = await this.request(`${this.baseURL}/api/conversations/${conversationId}/messages`, {
            method: 'POST',
            body: JSON.stringify({
                role: role,
                content: content,
                model: model
            })
        });
        
        if (!response.ok) throw new Error('Failed to create message');
        return response.json();
    }
}

// Create global API instance
window.api = new API();

