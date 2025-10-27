// API Client for AIGateway
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
            
            // CRITICAL: Update refresh token too (token rotation for security)
            if (data.refresh_token) {
                this.refreshToken = data.refresh_token;
                localStorage.setItem('refresh_token', data.refresh_token);
            }
            
            return true;
        } catch (error) {
            console.error('Failed to refresh token:', error);
            // Don't call logout() - just clear tokens and redirect
            // This avoids calling /api/auth/logout when token is already invalid
            this.clearSessionAndRedirect();
            return false;
        }
    }

    // Clear session without API call (used when tokens are already invalid)
    clearSessionAndRedirect() {
        console.log('Clearing session and redirecting to login');
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        localStorage.removeItem('user');
        window.location.href = '/login.html';
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

    // ==================== HTTP Shorthand Methods ====================

    async get(path) {
        const response = await this.request(`${this.baseURL}${path}`);
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || `GET ${path} failed`);
        }
        return response.json();
    }

    async post(path, data) {
        const response = await this.request(`${this.baseURL}${path}`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || `POST ${path} failed`);
        }
        return response.json();
    }

    async put(path, data) {
        const response = await this.request(`${this.baseURL}${path}`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || `PUT ${path} failed`);
        }
        return response.json();
    }

    async delete(path) {
        const response = await this.request(`${this.baseURL}${path}`, {
            method: 'DELETE'
        });
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || `DELETE ${path} failed`);
        }
        return response.ok;
    }

    // ==================== Auth APIs ====================

    async getCurrentUser() {
        const response = await this.request(`${this.baseURL}/api/users/me`);
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
            window.location.href = '/login.html';
        }
    }

    // ==================== Models APIs ====================

    async getModels() {
        // Use /api/models endpoint (public, no API key required)
        const response = await fetch(`${this.baseURL}/api/models`, {
            headers: this.getAuthHeaders()
        });
        
        if (!response.ok) throw new Error('Failed to fetch models');
        const data = await response.json();
        // /api/models returns {models: [...]} format
        return data.models || data.data || [];
    }

    // ==================== Chat APIs ====================

    async sendChatMessage(messages, modelParams, stream = false) {
        // Support both old (string) and new (object with params) formats
        const requestBody = {
            messages: messages,
            stream: stream
        };

        if (typeof modelParams === 'string') {
            // Old format: just model name
            requestBody.model = modelParams;
            requestBody.temperature = 0.7;
            requestBody.max_tokens = 2048;
        } else if (typeof modelParams === 'object') {
            // New format (v1.9.1+): model + parameters
            requestBody.model = modelParams.model;
            if (modelParams.temperature !== undefined) requestBody.temperature = modelParams.temperature;
            if (modelParams.top_p !== undefined) requestBody.top_p = modelParams.top_p;
            if (modelParams.max_tokens !== undefined) requestBody.max_tokens = modelParams.max_tokens;
            if (modelParams.options) requestBody.options = modelParams.options;
        }

        const response = await this.request(`${this.baseURL}/v1/chat/completions`, {
            method: 'POST',
            body: JSON.stringify(requestBody)
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Chat request failed');
        }

        return response;
    }

    // Stream chat response
    async *streamChatMessage(messages, modelParams) {
        const response = await this.sendChatMessage(messages, modelParams, true);
        
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

    async createMessage(conversationId, role, content, model, fileIds = []) {
        const body = {
            role: role,
            content: content,
            model: model
        };

        // Add file_ids if any (FILE-STORAGE-01: Phase 4, v1.10.0+)
        if (fileIds && fileIds.length > 0) {
            body.file_ids = fileIds;
        }

        const response = await this.request(`${this.baseURL}/api/conversations/${conversationId}/messages`, {
            method: 'POST',
            body: JSON.stringify(body)
        });
        
        if (!response.ok) throw new Error('Failed to create message');
        return response.json();
    }

    // ==================== User Profile APIs ====================

    async updateUserProfile(data) {
        const response = await this.request(`${this.baseURL}/api/users/me`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to update profile');
        }
        return response.json();
    }

    async changePassword(currentPassword, newPassword) {
        const response = await this.request(`${this.baseURL}/api/users/me/password`, {
            method: 'POST',
            body: JSON.stringify({
                current_password: currentPassword,
                new_password: newPassword
            })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to change password');
        }
        return response.json();
    }

    async deleteAccount() {
        const response = await this.request(`${this.baseURL}/api/users/me`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete account');
        }
        return true;
    }

    // ==================== Tenant APIs ====================

    async getUserTenants() {
        const response = await this.request(`${this.baseURL}/api/users/me/tenants`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch tenants');
        }
        return response.json();
    }

    async getTenant(tenantId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch tenant');
        }
        return response.json();
    }

    async createTenant(data) {
        const response = await this.request(`${this.baseURL}/api/tenants`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create tenant');
        }
        return response.json();
    }

    async updateTenant(tenantId, data) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to update tenant');
        }
        return response.json();
    }

    async deleteTenant(tenantId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete tenant');
        }
        return true;
    }

    async getTenantMembers(tenantId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/members`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch members');
        }
        return response.json();
    }

    async searchTenantUsers(tenantId, query) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/search-users?query=${encodeURIComponent(query)}`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'User not found');
        }
        return response.json();
    }

    async addTenantMember(tenantId, userInfo, role) {
        // userInfo can be { user_id, username, or email }
        const body = { role };
        if (userInfo.user_id) {
            body.user_id = userInfo.user_id;
        } else if (userInfo.username) {
            body.username = userInfo.username;
        } else if (userInfo.email) {
            body.email = userInfo.email;
        }

        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/members`, {
            method: 'POST',
            body: JSON.stringify(body)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to add member');
        }
        return response.json();
    }

    async updateTenantMember(tenantId, userId, role) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/members/${userId}`, {
            method: 'PUT',
            body: JSON.stringify({ role })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to update member');
        }
        return response.json();
    }

    async removeTenantMember(tenantId, userId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/members/${userId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to remove member');
        }
        return true;
    }

    // ==================== API Keys ====================

    async getPersonalAPIKeys() {
        const response = await this.request(`${this.baseURL}/api/users/me/api-keys`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch API keys');
        }
        return response.json();
    }

    async createPersonalAPIKey(data) {
        const response = await this.request(`${this.baseURL}/api/users/me/api-keys`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create API key');
        }
        return response.json();
    }

    async deletePersonalAPIKey(keyId) {
        const response = await this.request(`${this.baseURL}/api/users/me/api-keys/${keyId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete API key');
        }
        return true;
    }

    async getTenantAPIKeys(tenantId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/api-keys`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to fetch tenant API keys');
        }
        return response.json();
    }

    async createTenantAPIKey(tenantId, data) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/api-keys`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create tenant API key');
        }
        return response.json();
    }

    async deleteTenantAPIKey(tenantId, keyId) {
        const response = await this.request(`${this.baseURL}/api/tenants/${tenantId}/api-keys/${keyId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to delete tenant API key');
        }
        return true;
    }
    // ==================== RAG APIs (v1.13.0+) ====================

    async getRAGSources(filters = {}) {
        let query = '';
        if (Object.keys(filters).length > 0) {
            const params = new URLSearchParams();
            if (filters.source_type) params.append('source_type', filters.source_type);
            if (filters.status) params.append('status', filters.status);
            if (filters.limit) params.append('limit', filters.limit);
            if (filters.offset) params.append('offset', filters.offset);
            query = '?' + params.toString();
        }
        
        const response = await this.request(`${this.baseURL}/api/rag/sources${query}`);
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to fetch RAG sources');
        }
        return response.json();
    }

    async getRAGSource(sourceId) {
        const response = await this.request(`${this.baseURL}/api/rag/sources/${sourceId}`);
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to fetch RAG source');
        }
        return response.json();
    }

    async createRAGSource(data) {
        const response = await this.request(`${this.baseURL}/api/rag/sources`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to create RAG source');
        }
        return response.json();
    }

    async updateRAGSource(sourceId, data) {
        const response = await this.request(`${this.baseURL}/api/rag/sources/${sourceId}`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to update RAG source');
        }
        return response.json();
    }

    async deleteRAGSource(sourceId) {
        const response = await this.request(`${this.baseURL}/api/rag/sources/${sourceId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to delete RAG source');
        }
        return true;
    }

    async testRAGConnection(data) {
        const response = await this.request(`${this.baseURL}/api/rag/sources/test-connection`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Connection test failed');
        }
        return response.json();
    }

    async syncRAGSource(sourceId) {
        const response = await this.request(`${this.baseURL}/api/rag/sources/${sourceId}/sync`, {
            method: 'POST'
        });
        
        if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(error.error || 'Failed to sync RAG source');
        }
        return response.json();
    }

    // ==================== RBAC APIs (v1.11.5+) ====================

    async getRBACPermissions() {
        return this.get('/api/admin/rbac/permissions');
    }

    async getRBACRoles(includePermissions = true) {
        return this.get(`/api/admin/rbac/roles?include_permissions=${includePermissions}`);
    }

    async createRBACRole(data) {
        return this.post('/api/admin/rbac/roles', data);
    }

    async updateRBACRole(roleId, data) {
        return this.put(`/api/admin/rbac/roles/${roleId}`, data);
    }

    async deleteRBACRole(roleId) {
        return this.delete(`/api/admin/rbac/roles/${roleId}`);
    }

    async addRolePermission(roleId, permissionId) {
        return this.post(`/api/admin/rbac/roles/${roleId}/permissions`, { permission_id: permissionId });
    }

    async removeRolePermission(roleId, permissionId) {
        return this.delete(`/api/admin/rbac/roles/${roleId}/permissions/${permissionId}`);
    }

    async getUserRoles(userId, includeDetails = true) {
        return this.get(`/api/admin/rbac/users/${userId}/roles?include_details=${includeDetails}`);
    }

    async assignUserRole(userId, data) {
        return this.post(`/api/admin/rbac/users/${userId}/roles`, data);
    }

    async removeUserRole(userId, roleId, tenantId = null) {
        let url = `/api/admin/rbac/users/${userId}/roles/${roleId}`;
        if (tenantId) {
            url += `?tenant_id=${tenantId}`;
        }
        return this.delete(url);
    }

    async getAdminUsers() {
        return this.get('/api/admin/users');
    }

    async getAdminTenants() {
        return this.get('/api/admin/tenants');
    }

    // ==================== Audit APIs (v1.11.5+) ====================

    async getAuditStats() {
        return this.get('/api/admin/audit/stats');
    }

    async getAuditLogs(filters = {}) {
        const queryParams = new URLSearchParams(filters);
        return this.get(`/api/admin/audit?${queryParams}`);
    }

    async exportAuditLogs(filters = {}) {
        const queryParams = new URLSearchParams(filters);
        const response = await this.request(`${this.baseURL}/api/admin/audit/export?${queryParams}`, {
            method: 'GET'
        });
        return response; // Return raw response for file download
    }

    // ==================== Invitations APIs (AUTH-03, v2.2.0) ====================

    async createInvitation(data) {
        return this.post('/api/admin/invitations', data);
    }

    async listInvitations(queryString = '') {
        return this.get(`/api/admin/invitations${queryString ? '?' + queryString : ''}`);
    }

    async getInvitationStats() {
        return this.get('/api/admin/invitations/stats');
    }

    async getInvitationDetails(id) {
        return this.get(`/api/admin/invitations/${id}`);
    }

    async revokeInvitation(id, data = {}) {
        return this.delete(`/api/admin/invitations/${id}`, data);
    }

    async validateInvitation(token, email = null) {
        let url = `/api/invitations/${token}/validate`;
        if (email) {
            url += `?email=${encodeURIComponent(email)}`;
        }
        return this.get(url);
    }

    // ==================== System APIs ====================

    async getSystemInfo() {
        return this.get('/api/system/info');
    }

    // Check if RAG is enabled in system configuration
    async isRAGEnabled() {
        try {
            const info = await this.getSystemInfo();
            return info.rag_enabled === true;
        } catch (error) {
            console.error('Failed to check RAG status:', error);
            return false; // Default to disabled if can't check
        }
    }
}

// Create global API instance
window.api = new API();



