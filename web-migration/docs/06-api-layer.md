# API Layer Architecture

## Обзор

API слой в Svelte миграции будет построен на:
- TypeScript для полной типизации
- Fetch API с автоматическим refresh токена
- Svelte 5 Runes для реактивного состояния
- Модульная структура по доменам

## Структура файлов

```
lib/api/
├── client.ts          # Base HTTP client
├── types.ts           # Shared types
├── auth.ts            # Auth endpoints
├── conversations.ts   # Conversations API
├── messages.ts        # Messages API
├── tenants.ts         # Tenants API
├── api-keys.ts        # API Keys
├── rag.ts             # RAG sources
├── admin/
│   ├── users.ts       # Admin users
│   ├── audit.ts       # Audit logs
│   ├── invitations.ts # Invitations
│   ├── rbac.ts        # RBAC
│   └── registry.ts    # Model registry
└── index.ts           # Re-exports
```

## Base Client (client.ts)

```typescript
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

interface RequestOptions extends RequestInit {
  skipAuth?: boolean;
}

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

class ApiClient {
  private baseUrl = '';
  
  private getToken(): string | null {
    if (!browser) return null;
    return localStorage.getItem('access_token');
  }
  
  private getRefreshToken(): string | null {
    if (!browser) return null;
    return localStorage.getItem('refresh_token');
  }
  
  private setTokens(accessToken: string, refreshToken: string): void {
    if (!browser) return;
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
  }
  
  private clearTokens(): void {
    if (!browser) return;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
  }
  
  private async refreshAccessToken(): Promise<boolean> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) return false;
    
    try {
      const response = await fetch(`${this.baseUrl}/api/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      
      if (!response.ok) return false;
      
      const data = await response.json();
      this.setTokens(data.access_token, data.refresh_token || refreshToken);
      return true;
    } catch {
      return false;
    }
  }
  
  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { skipAuth = false, ...fetchOptions } = options;
    
    const headers = new Headers(fetchOptions.headers);
    headers.set('Content-Type', 'application/json');
    
    if (!skipAuth) {
      const token = this.getToken();
      if (token) {
        headers.set('Authorization', `Bearer ${token}`);
      }
    }
    
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...fetchOptions,
      headers,
    });
    
    // Handle 401 - try refresh token
    if (response.status === 401 && !skipAuth) {
      const refreshed = await this.refreshAccessToken();
      if (refreshed) {
        // Retry with new token
        headers.set('Authorization', `Bearer ${this.getToken()}`);
        const retryResponse = await fetch(`${this.baseUrl}${path}`, {
          ...fetchOptions,
          headers,
        });
        
        if (!retryResponse.ok) {
          const error = await retryResponse.json().catch(() => ({}));
          throw new ApiError(error.error || 'Request failed', retryResponse.status, error);
        }
        
        return retryResponse.json();
      }
      
      // Refresh failed - redirect to login
      this.clearTokens();
      if (browser) {
        localStorage.setItem('return_url', window.location.href);
        goto('/login');
      }
      throw new ApiError('Session expired', 401);
    }
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new ApiError(error.error || 'Request failed', response.status, error);
    }
    
    // Handle empty responses (204 No Content)
    if (response.status === 204) {
      return undefined as T;
    }
    
    return response.json();
  }
  
  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request(path, { ...options, method: 'GET' });
  }
  
  post<T>(path: string, data?: unknown, options?: RequestOptions): Promise<T> {
    return this.request(path, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }
  
  put<T>(path: string, data: unknown, options?: RequestOptions): Promise<T> {
    return this.request(path, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
  
  delete<T = void>(path: string, options?: RequestOptions): Promise<T> {
    return this.request(path, { ...options, method: 'DELETE' });
  }
}

export const api = new ApiClient();
export { ApiError };
```

## Types (types.ts)

```typescript
// Auth
export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  is_admin: boolean;
  created_at: string;
  last_login_at: string | null;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface AuthResponse {
  user: User;
  token: TokenPair;
}

export interface LoginRequest {
  username: string;
  password: string;
  remember_me?: boolean;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  display_name?: string;
  invitation_token?: string;
}

// Conversations
export interface Conversation {
  id: string;
  user_id: string;
  title: string;
  model: string;
  system_prompt?: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  model?: string;
  created_at: string;
  file_ids?: string[];
}

// Tenants
export interface Tenant {
  id: string;
  name: string;
  slug: string;
  description?: string;
  type: string;
  status: string;
  created_at: string;
}

export interface TenantMember {
  user_id: string;
  username: string;
  email: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  joined_at: string;
}

// API Keys
export interface APIKey {
  id: string;
  name: string;
  description?: string;
  key_prefix: string;
  rate_limit_rpm?: number;
  rate_limit_rph?: number;
  models: string[] | null;
  status: string;
  created_at: string;
  last_used_at: string | null;
}

export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  rate_limit_rpm?: number;
  rate_limit_rph?: number;
  models?: string[];
}

export interface CreateAPIKeyResponse {
  api_key: APIKey;
  key: string; // Full key shown only once
}

// RAG
export interface RAGSource {
  id: string;
  name: string;
  description?: string;
  source_type: 'api' | 'database' | 'file' | 'web';
  status: 'active' | 'inactive' | 'error' | 'syncing';
  config: RAGSourceConfig;
  chunks_count: number;
  tokens_count: number;
  is_shared: boolean;
  last_synced_at: string | null;
  created_at: string;
}

export interface RAGSourceConfig {
  // API config
  api_url?: string;
  api_method?: string;
  api_auth_type?: 'none' | 'bearer' | 'api_key' | 'basic';
  api_token?: string;
  
  // Database config
  db_host?: string;
  db_port?: number;
  db_name?: string;
  db_username?: string;
  db_password?: string;
  db_query?: string;
  
  // Web config
  web_url?: string;
  web_depth?: number;
  
  // Common
  chunk_size?: number;
  chunk_overlap?: number;
}

// Models
export interface Model {
  id: string;
  name: string;
  size?: number;
  modified_at?: string;
}

// System
export interface SystemInfo {
  version: string;
  git_commit: string;
  build_date: string;
  go_version: string;
  rag_enabled: boolean;
}

// Dashboard
export interface DashboardStats {
  conversations_count: number;
  tenants_count: number;
  api_keys_count: number;
  requests_30d: number;
}

// Pagination
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
}
```

## Domain APIs

### auth.ts

```typescript
import { api } from './client';
import type { AuthResponse, LoginRequest, RegisterRequest, User } from './types';

export const authApi = {
  login(credentials: LoginRequest): Promise<AuthResponse> {
    return api.post('/api/auth/login', credentials, { skipAuth: true });
  },
  
  register(data: RegisterRequest): Promise<AuthResponse> {
    return api.post('/api/auth/register', data, { skipAuth: true });
  },
  
  logout(): Promise<void> {
    return api.post('/api/auth/logout');
  },
  
  getCurrentUser(): Promise<User> {
    return api.get('/api/users/me');
  },
  
  checkInitStatus(): Promise<{ requires_bootstrap: boolean }> {
    return api.get('/api/system/init-status', { skipAuth: true });
  },
  
  bootstrap(data: {
    admin_token: string;
    username: string;
    email: string;
    password: string;
    display_name?: string;
  }): Promise<AuthResponse> {
    return api.post('/api/system/bootstrap', data, { skipAuth: true });
  },
  
  validateInvitation(token: string, email?: string): Promise<{
    valid: boolean;
    invitation?: { email?: string };
    error?: string;
  }> {
    const query = email ? `?email=${encodeURIComponent(email)}` : '';
    return api.get(`/api/invitations/${token}/validate${query}`, { skipAuth: true });
  },
};
```

### conversations.ts

```typescript
import { api } from './client';
import type { Conversation, Message } from './types';

export const conversationsApi = {
  list(): Promise<{ conversations: Conversation[] }> {
    return api.get('/api/conversations');
  },
  
  get(id: string): Promise<Conversation> {
    return api.get(`/api/conversations/${id}`);
  },
  
  create(data: { title?: string; model: string; system_prompt?: string }): Promise<Conversation> {
    return api.post('/api/conversations', data);
  },
  
  update(id: string, data: Partial<Conversation>): Promise<Conversation> {
    return api.put(`/api/conversations/${id}`, data);
  },
  
  delete(id: string): Promise<void> {
    return api.delete(`/api/conversations/${id}`);
  },
  
  getMessages(id: string): Promise<{ messages: Message[] }> {
    return api.get(`/api/conversations/${id}/messages`);
  },
  
  createMessage(
    conversationId: string,
    data: { role: string; content: string; model?: string; file_ids?: string[] }
  ): Promise<Message> {
    return api.post(`/api/conversations/${conversationId}/messages`, data);
  },
};
```

### tenants.ts

```typescript
import { api } from './client';
import type { Tenant, TenantMember } from './types';

export const tenantsApi = {
  listUserTenants(): Promise<{ tenants: Tenant[] }> {
    return api.get('/api/users/me/tenants');
  },
  
  get(id: string): Promise<Tenant> {
    return api.get(`/api/tenants/${id}`);
  },
  
  create(data: { name: string; slug?: string; description?: string }): Promise<Tenant> {
    return api.post('/api/tenants', data);
  },
  
  update(id: string, data: Partial<Tenant>): Promise<Tenant> {
    return api.put(`/api/tenants/${id}`, data);
  },
  
  delete(id: string): Promise<void> {
    return api.delete(`/api/tenants/${id}`);
  },
  
  getMembers(id: string): Promise<{ members: TenantMember[] }> {
    return api.get(`/api/tenants/${id}/members`);
  },
  
  searchUsers(id: string, query: string): Promise<{ users: { id: string; username: string; email: string }[] }> {
    return api.get(`/api/tenants/${id}/search-users?query=${encodeURIComponent(query)}`);
  },
  
  addMember(tenantId: string, data: { user_id?: string; username?: string; email?: string; role: string }): Promise<TenantMember> {
    return api.post(`/api/tenants/${tenantId}/members`, data);
  },
  
  updateMember(tenantId: string, userId: string, role: string): Promise<TenantMember> {
    return api.put(`/api/tenants/${tenantId}/members/${userId}`, { role });
  },
  
  removeMember(tenantId: string, userId: string): Promise<void> {
    return api.delete(`/api/tenants/${tenantId}/members/${userId}`);
  },
};
```

## Streaming API (chat/stream.ts)

```typescript
import { browser } from '$app/environment';

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface ChatParams {
  model: string;
  temperature?: number;
  top_p?: number;
  max_tokens?: number;
  rag_enabled?: boolean;
  rag_source_ids?: string[];
  rag_top_k?: number;
  rag_min_score?: number;
}

export async function* streamChat(
  messages: ChatMessage[],
  params: ChatParams
): AsyncGenerator<string, void, unknown> {
  if (!browser) return;
  
  const token = localStorage.getItem('access_token');
  
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` }),
    },
    body: JSON.stringify({
      ...params,
      messages,
      stream: true,
    }),
  });
  
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Stream failed' }));
    throw new Error(error.error || 'Stream request failed');
  }
  
  const reader = response.body?.getReader();
  if (!reader) throw new Error('No response body');
  
  const decoder = new TextDecoder();
  let buffer = '';
  
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';
      
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed === 'data: [DONE]') continue;
        
        if (trimmed.startsWith('data: ')) {
          try {
            const data = JSON.parse(trimmed.slice(6));
            const content = data.choices?.[0]?.delta?.content;
            if (content) yield content;
          } catch {
            // Skip malformed JSON
          }
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}
```

## Re-exports (index.ts)

```typescript
export { api, ApiError } from './client';
export { authApi } from './auth';
export { conversationsApi } from './conversations';
export { tenantsApi } from './tenants';
export { apiKeysApi } from './api-keys';
export { ragApi } from './rag';
export { modelsApi } from './models';
export { systemApi } from './system';

// Admin
export { adminUsersApi } from './admin/users';
export { auditApi } from './admin/audit';
export { invitationsApi } from './admin/invitations';
export { rbacApi } from './admin/rbac';
export { registryApi } from './admin/registry';

// Types
export type * from './types';
```

## Использование в компонентах

```svelte
<script>
  import { authApi, conversationsApi, ApiError } from '$lib/api';
  import { toast } from 'svelte-sonner';
  
  let conversations = $state([]);
  let loading = $state(true);
  let error = $state(null);
  
  async function loadConversations() {
    try {
      loading = true;
      const data = await conversationsApi.list();
      conversations = data.conversations;
    } catch (e) {
      if (e instanceof ApiError) {
        error = e.message;
        toast.error(e.message);
      }
    } finally {
      loading = false;
    }
  }
  
  $effect(() => {
    loadConversations();
  });
</script>

{#if loading}
  <Skeleton />
{:else if error}
  <Alert variant="destructive">{error}</Alert>
{:else}
  {#each conversations as conv}
    <ConversationCard {conv} />
  {/each}
{/if}
```

