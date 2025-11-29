# State Management в Svelte 5

## Обзор

Svelte 5 использует систему Runes для реактивности:
- `$state` - реактивное состояние
- `$derived` - вычисляемые значения
- `$effect` - побочные эффекты
- `$props` - пропсы компонентов

## Структура stores

```
lib/stores/
├── auth.svelte.ts       # Аутентификация
├── models.svelte.ts     # Доступные модели
├── chat.svelte.ts       # Состояние чата
├── tenants.svelte.ts    # Организации
├── theme.svelte.ts      # Тема приложения
├── notifications.svelte.ts  # Уведомления
└── index.ts             # Re-exports
```

## Auth Store (auth.svelte.ts)

```typescript
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { authApi, type User, type TokenPair } from '$lib/api';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  loading: boolean;
}

function createAuthStore() {
  let user = $state<User | null>(null);
  let loading = $state(true);
  
  // Initialize from localStorage on mount
  function init() {
    if (!browser) return;
    
    const token = localStorage.getItem('access_token');
    const storedUser = localStorage.getItem('user');
    
    if (token && storedUser) {
      try {
        user = JSON.parse(storedUser);
      } catch {
        clearStorage();
      }
    }
    loading = false;
  }
  
  function clearStorage() {
    if (!browser) return;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
  }
  
  function setTokens(tokens: TokenPair) {
    if (!browser) return;
    localStorage.setItem('access_token', tokens.access_token);
    localStorage.setItem('refresh_token', tokens.refresh_token);
  }
  
  function setUser(newUser: User) {
    user = newUser;
    if (browser) {
      localStorage.setItem('user', JSON.stringify(newUser));
    }
  }
  
  async function login(username: string, password: string, rememberMe = false) {
    const response = await authApi.login({ username, password, remember_me: rememberMe });
    setTokens(response.token);
    setUser(response.user);
    return response.user;
  }
  
  async function register(data: { username: string; email: string; password: string; display_name?: string; invitation_token?: string }) {
    const response = await authApi.register(data);
    setTokens(response.token);
    setUser(response.user);
    return response.user;
  }
  
  async function logout() {
    try {
      await authApi.logout();
    } catch {
      // Ignore logout errors
    } finally {
      user = null;
      clearStorage();
      if (browser) {
        goto('/login');
      }
    }
  }
  
  async function refreshUser() {
    try {
      const freshUser = await authApi.getCurrentUser();
      setUser(freshUser);
      return freshUser;
    } catch {
      await logout();
      return null;
    }
  }
  
  // Derived values
  const isAuthenticated = $derived(!!user);
  const isAdmin = $derived(user?.is_admin ?? false);
  
  return {
    get user() { return user; },
    get loading() { return loading; },
    get isAuthenticated() { return isAuthenticated; },
    get isAdmin() { return isAdmin; },
    
    init,
    login,
    register,
    logout,
    refreshUser,
  };
}

export const auth = createAuthStore();
```

## Models Store (models.svelte.ts)

```typescript
import { browser } from '$app/environment';
import { modelsApi, type Model } from '$lib/api';

function createModelsStore() {
  let models = $state<Model[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let lastFetched = $state<number | null>(null);
  
  const CACHE_DURATION = 60_000; // 1 minute
  
  async function fetch(force = false) {
    // Check cache
    if (!force && lastFetched && Date.now() - lastFetched < CACHE_DURATION) {
      return models;
    }
    
    try {
      loading = true;
      error = null;
      const response = await modelsApi.list();
      models = response.models || [];
      lastFetched = Date.now();
      return models;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load models';
      throw e;
    } finally {
      loading = false;
    }
  }
  
  function find(id: string): Model | undefined {
    return models.find(m => m.id === id || m.name === id);
  }
  
  // Derived
  const hasModels = $derived(models.length > 0);
  const modelOptions = $derived(
    models.map(m => ({ value: m.id, label: m.name }))
  );
  
  return {
    get models() { return models; },
    get loading() { return loading; },
    get error() { return error; },
    get hasModels() { return hasModels; },
    get modelOptions() { return modelOptions; },
    
    fetch,
    find,
  };
}

export const modelsStore = createModelsStore();
```

## Chat Store (chat.svelte.ts)

```typescript
import { browser } from '$app/environment';
import { conversationsApi, type Conversation, type Message } from '$lib/api';
import { streamChat, type ChatMessage, type ChatParams } from '$lib/api/chat/stream';

interface ChatState {
  conversations: Conversation[];
  currentConversation: Conversation | null;
  messages: Message[];
  streaming: boolean;
  streamContent: string;
}

function createChatStore() {
  // Conversations
  let conversations = $state<Conversation[]>([]);
  let conversationsLoading = $state(false);
  
  // Current conversation
  let currentConversation = $state<Conversation | null>(null);
  let messages = $state<Message[]>([]);
  let messagesLoading = $state(false);
  
  // Streaming
  let streaming = $state(false);
  let streamContent = $state('');
  let abortController = $state<AbortController | null>(null);
  
  // Chat parameters
  let params = $state<ChatParams>({
    model: '',
    temperature: 0.7,
    top_p: 0.9,
    max_tokens: 4096,
    rag_enabled: false,
    rag_source_ids: [],
    rag_top_k: 5,
    rag_min_score: 0.3,
  });
  
  async function loadConversations() {
    try {
      conversationsLoading = true;
      const data = await conversationsApi.list();
      conversations = data.conversations || [];
    } finally {
      conversationsLoading = false;
    }
  }
  
  async function selectConversation(id: string | null) {
    if (!id) {
      currentConversation = null;
      messages = [];
      return;
    }
    
    try {
      messagesLoading = true;
      const [conv, messagesData] = await Promise.all([
        conversationsApi.get(id),
        conversationsApi.getMessages(id),
      ]);
      currentConversation = conv;
      messages = messagesData.messages || [];
      
      // Update model from conversation
      if (conv.model) {
        params.model = conv.model;
      }
    } finally {
      messagesLoading = false;
    }
  }
  
  async function createConversation(title?: string) {
    const conv = await conversationsApi.create({
      title: title || 'New Chat',
      model: params.model,
    });
    conversations = [conv, ...conversations];
    await selectConversation(conv.id);
    return conv;
  }
  
  async function deleteConversation(id: string) {
    await conversationsApi.delete(id);
    conversations = conversations.filter(c => c.id !== id);
    
    if (currentConversation?.id === id) {
      currentConversation = null;
      messages = [];
    }
  }
  
  async function sendMessage(content: string, fileIds?: string[]) {
    if (!content.trim() || streaming) return;
    
    let conv = currentConversation;
    
    // Create conversation if none selected
    if (!conv) {
      conv = await createConversation(content.slice(0, 50));
    }
    
    // Add user message
    const userMessage: Message = {
      id: `temp-${Date.now()}`,
      conversation_id: conv.id,
      role: 'user',
      content,
      created_at: new Date().toISOString(),
      file_ids: fileIds,
    };
    messages = [...messages, userMessage];
    
    // Save user message
    const savedUserMessage = await conversationsApi.createMessage(conv.id, {
      role: 'user',
      content,
      file_ids: fileIds,
    });
    
    // Update temp message with real ID
    messages = messages.map(m => 
      m.id === userMessage.id ? savedUserMessage : m
    );
    
    // Prepare chat messages
    const chatMessages: ChatMessage[] = messages.map(m => ({
      role: m.role,
      content: m.content,
    }));
    
    // Start streaming
    streaming = true;
    streamContent = '';
    
    try {
      const stream = streamChat(chatMessages, params);
      
      for await (const chunk of stream) {
        streamContent += chunk;
      }
      
      // Save assistant message
      const assistantMessage = await conversationsApi.createMessage(conv.id, {
        role: 'assistant',
        content: streamContent,
        model: params.model,
      });
      
      messages = [...messages, assistantMessage];
      streamContent = '';
      
      // Update conversation title if first exchange
      if (messages.length === 2 && conv.title === 'New Chat') {
        const newTitle = content.slice(0, 50) + (content.length > 50 ? '...' : '');
        await conversationsApi.update(conv.id, { title: newTitle });
        conversations = conversations.map(c => 
          c.id === conv!.id ? { ...c, title: newTitle } : c
        );
        if (currentConversation?.id === conv.id) {
          currentConversation = { ...currentConversation, title: newTitle };
        }
      }
    } catch (e) {
      // Add error message
      const errorMessage: Message = {
        id: `error-${Date.now()}`,
        conversation_id: conv.id,
        role: 'assistant',
        content: `Error: ${e instanceof Error ? e.message : 'Failed to generate response'}`,
        created_at: new Date().toISOString(),
      };
      messages = [...messages, errorMessage];
    } finally {
      streaming = false;
      streamContent = '';
    }
  }
  
  function stopStream() {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
    streaming = false;
  }
  
  function updateParams(updates: Partial<ChatParams>) {
    params = { ...params, ...updates };
  }
  
  function clearChat() {
    currentConversation = null;
    messages = [];
    streamContent = '';
  }
  
  // Derived
  const hasConversations = $derived(conversations.length > 0);
  const sortedConversations = $derived(
    [...conversations].sort((a, b) => 
      new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
  );
  
  return {
    // State
    get conversations() { return conversations; },
    get sortedConversations() { return sortedConversations; },
    get conversationsLoading() { return conversationsLoading; },
    get currentConversation() { return currentConversation; },
    get messages() { return messages; },
    get messagesLoading() { return messagesLoading; },
    get streaming() { return streaming; },
    get streamContent() { return streamContent; },
    get params() { return params; },
    get hasConversations() { return hasConversations; },
    
    // Actions
    loadConversations,
    selectConversation,
    createConversation,
    deleteConversation,
    sendMessage,
    stopStream,
    updateParams,
    clearChat,
  };
}

export const chat = createChatStore();
```

## Theme Store (theme.svelte.ts)

```typescript
import { browser } from '$app/environment';
import { persisted } from 'svelte-persisted-store';

type Theme = 'light' | 'dark' | 'system';

function createThemeStore() {
  let theme = $state<Theme>('dark');
  let resolvedTheme = $state<'light' | 'dark'>('dark');
  
  function init() {
    if (!browser) return;
    
    const stored = localStorage.getItem('theme') as Theme | null;
    theme = stored || 'dark';
    updateResolvedTheme();
    applyTheme();
    
    // Listen for system theme changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', () => {
      if (theme === 'system') {
        updateResolvedTheme();
        applyTheme();
      }
    });
  }
  
  function updateResolvedTheme() {
    if (theme === 'system') {
      resolvedTheme = window.matchMedia('(prefers-color-scheme: dark)').matches 
        ? 'dark' 
        : 'light';
    } else {
      resolvedTheme = theme;
    }
  }
  
  function applyTheme() {
    if (!browser) return;
    
    document.documentElement.classList.remove('light', 'dark');
    document.documentElement.classList.add(resolvedTheme);
  }
  
  function setTheme(newTheme: Theme) {
    theme = newTheme;
    if (browser) {
      localStorage.setItem('theme', newTheme);
    }
    updateResolvedTheme();
    applyTheme();
  }
  
  function toggle() {
    setTheme(resolvedTheme === 'dark' ? 'light' : 'dark');
  }
  
  const isDark = $derived(resolvedTheme === 'dark');
  
  return {
    get theme() { return theme; },
    get resolvedTheme() { return resolvedTheme; },
    get isDark() { return isDark; },
    
    init,
    setTheme,
    toggle,
  };
}

export const themeStore = createThemeStore();
```

## Notifications Store (notifications.svelte.ts)

```typescript
import { toast } from 'svelte-sonner';

export const notifications = {
  success(message: string, description?: string) {
    toast.success(message, { description });
  },
  
  error(message: string, description?: string) {
    toast.error(message, { description });
  },
  
  info(message: string, description?: string) {
    toast.info(message, { description });
  },
  
  warning(message: string, description?: string) {
    toast.warning(message, { description });
  },
  
  promise<T>(
    promise: Promise<T>,
    messages: {
      loading: string;
      success: string;
      error: string;
    }
  ): Promise<T> {
    return toast.promise(promise, messages);
  },
  
  dismiss(id?: string | number) {
    toast.dismiss(id);
  },
};
```

## Persisted Store Helper

```typescript
// lib/stores/persisted.svelte.ts
import { browser } from '$app/environment';

export function createPersistedState<T>(
  key: string,
  initialValue: T
): { value: T; reset: () => void } {
  let value = $state(initialValue);
  
  // Load from localStorage
  if (browser) {
    const stored = localStorage.getItem(key);
    if (stored) {
      try {
        value = JSON.parse(stored);
      } catch {
        // Use initial value
      }
    }
  }
  
  // Save to localStorage on change
  $effect(() => {
    if (browser) {
      localStorage.setItem(key, JSON.stringify(value));
    }
  });
  
  function reset() {
    value = initialValue;
    if (browser) {
      localStorage.removeItem(key);
    }
  }
  
  return {
    get value() { return value; },
    set value(v: T) { value = v; },
    reset,
  };
}
```

## Использование в компонентах

```svelte
<script>
  import { auth } from '$lib/stores/auth.svelte';
  import { chat } from '$lib/stores/chat.svelte';
  import { notifications } from '$lib/stores/notifications.svelte';
  
  let message = $state('');
  
  async function handleSend() {
    if (!message.trim()) return;
    
    try {
      await chat.sendMessage(message);
      message = '';
    } catch (e) {
      notifications.error('Failed to send message');
    }
  }
</script>

{#if auth.isAuthenticated}
  <div class="chat">
    {#each chat.messages as msg}
      <MessageBubble {msg} />
    {/each}
    
    {#if chat.streaming}
      <MessageBubble 
        msg={{ role: 'assistant', content: chat.streamContent }} 
        isStreaming 
      />
    {/if}
    
    <form on:submit|preventDefault={handleSend}>
      <input bind:value={message} placeholder="Type a message..." />
      <button type="submit" disabled={chat.streaming}>Send</button>
    </form>
  </div>
{/if}
```

