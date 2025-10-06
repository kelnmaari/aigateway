# WEBUI-03: Interactive Chat Interface

**Версия:** 1.0  
**Статус:** 📋 Запланирована  
**Приоритет:** HIGH  
**Оценка времени:** 15-20 часов  
**Зависимости:** AUTH-05, DB-01

---

## 📋 Описание

Полноценный ChatGPT-like интерфейс для взаимодействия с локальными Ollama моделями через WebUI. Vanilla JavaScript (без NPM зависимостей), современные Web APIs, real-time streaming responses.

---

## 🎯 Цели

1. **ChatGPT-like UI**: Знакомый интерфейс для пользователей
2. **Real-time Streaming**: Токен-по-токену вывод ответов
3. **Conversation Management**: Сохранение, загрузка, экспорт чатов
4. **Model Selection**: Переключение между моделями
5. **Rich Content**: Markdown, code highlighting, LaTeX
6. **Zero NPM Dependencies**: Только vanilla JS + CDN библиотеки

---

## 🎨 UI/UX Design

### Layout Structure

```
┌─────────────────────────────────────────────────────────┐
│  Header: [Logo] Ollama Chat [User Menu] [Theme Toggle] │
├──────────┬──────────────────────────────────────────────┤
│          │                                              │
│ Sidebar  │          Chat Area                          │
│          │                                              │
│ [+ New]  │  ┌────────────────────────────────────┐    │
│          │  │  System Message (if set)          │    │
│ Recent:  │  └────────────────────────────────────┘    │
│  • Chat1 │                                              │
│  • Chat2 │  ┌────────────────────────────────────┐    │
│  • Chat3 │  │ 👤 User message here...           │    │
│          │  └────────────────────────────────────┘    │
│ Saved:   │                                              │
│  ⭐ Fav1 │  ┌────────────────────────────────────┐    │
│  ⭐ Fav2 │  │ 🤖 Assistant response...          │    │
│          │  │    • Markdown rendering            │    │
│ Archive  │  │    • Code blocks with syntax       │    │
│          │  │    • Math equations                │    │
│          │  └────────────────────────────────────┘    │
│          │                                              │
│          │  ┌─────────────────────────────────────┐   │
│          │  │ 💬 Type your message...           │   │
│          │  │ [📎][🎤][🎨] [Model ▼] [Send ▶]  │   │
│          │  └─────────────────────────────────────┘   │
└──────────┴──────────────────────────────────────────────┘
```

---

## 🗄️ Data Models

### Conversation

```go
type Conversation struct {
    ID          string    `json:"id" db:"id"`
    UserID      string    `json:"user_id" db:"user_id"`
    TenantID    *string   `json:"tenant_id,omitempty" db:"tenant_id"`  // Optional: shared chat
    Title       string    `json:"title" db:"title"`                     // Auto-generated or custom
    Model       string    `json:"model" db:"model"`                     // e.g., "gpt-4"
    SystemMsg   string    `json:"system_msg,omitempty" db:"system_msg"` // Custom system prompt
    Settings    string    `json:"settings" db:"settings"`               // JSON: temperature, etc
    IsFavorite  bool      `json:"is_favorite" db:"is_favorite"`
    IsArchived  bool      `json:"is_archived" db:"is_archived"`
    TokensUsed  int       `json:"tokens_used" db:"tokens_used"`         // Total for conversation
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ConversationSettings (JSON в поле settings)
type ConversationSettings struct {
    Temperature float64 `json:"temperature"`
    TopP        float64 `json:"top_p"`
    TopK        int     `json:"top_k"`
    MaxTokens   int     `json:"max_tokens"`
    StopSeqs    []string `json:"stop_sequences"`
}
```

### Message

```go
type Message struct {
    ID              string    `json:"id" db:"id"`
    ConversationID  string    `json:"conversation_id" db:"conversation_id"`
    Role            string    `json:"role" db:"role"`               // user/assistant/system
    Content         string    `json:"content" db:"content"`
    TokensUsed      int       `json:"tokens_used" db:"tokens_used"` // For this message
    Model           string    `json:"model,omitempty" db:"model"`   // Which model generated (for assistant)
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
    
    // Optional metadata
    Metadata        string    `json:"metadata,omitempty" db:"metadata"` // JSON: attachments, etc
}
```

---

## 🚀 Core Features

### 1. Chat Interface

```javascript
// static/js/chat/chat-app.js
class ChatApp {
    constructor() {
        this.state = {
            currentConversation: null,
            conversations: [],
            messages: [],
            selectedModel: 'gpt-4',
            isStreaming: false,
            user: null
        };
        
        this.init();
    }
    
    async init() {
        await this.loadUser();
        await this.loadConversations();
        this.render();
        this.attachEventListeners();
        this.setupWebSocket();  // For real-time updates
    }
    
    async sendMessage(text) {
        if (!text.trim() || this.state.isStreaming) return;
        
        // Add user message to UI immediately
        this.addMessage({
            role: 'user',
            content: text,
            timestamp: Date.now()
        });
        
        // Clear input
        this.clearInput();
        
        // Start streaming
        await this.streamAssistantResponse(text);
    }
    
    async streamAssistantResponse(userMessage) {
        this.setState({ isStreaming: true });
        
        const response = await fetch('/api/chat/stream', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.getAuthToken()}`
            },
            body: JSON.stringify({
                conversation_id: this.state.currentConversation?.id,
                message: userMessage,
                model: this.state.selectedModel,
                stream: true
            })
        });
        
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        
        let assistantMessage = '';
        let messageElement = this.createStreamingMessageElement();
        
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            
            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');
            
            for (const line of lines) {
                if (!line.startsWith('data: ')) continue;
                
                const data = line.slice(6);
                if (data === '[DONE]') continue;
                
                try {
                    const json = JSON.parse(data);
                    const content = json.choices[0]?.delta?.content || '';
                    
                    assistantMessage += content;
                    this.updateStreamingMessage(messageElement, assistantMessage);
                } catch (e) {
                    console.error('Parse error:', e);
                }
            }
        }
        
        // Finalize message
        this.finalizeStreamingMessage(messageElement, assistantMessage);
        this.setState({ isStreaming: false });
    }
    
    updateStreamingMessage(element, content) {
        // Render markdown in real-time
        const html = this.markdownToHTML(content);
        element.querySelector('.message-content').innerHTML = html;
        
        // Syntax highlighting for code blocks
        element.querySelectorAll('pre code').forEach(block => {
            hljs.highlightElement(block);
        });
        
        // Auto-scroll to bottom
        this.scrollToBottom();
    }
}
```

### 2. Markdown & Code Highlighting

```javascript
// static/js/chat/markdown-renderer.js
class MarkdownRenderer {
    constructor() {
        // Используем marked.js (single file, ~20KB)
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
    
    render(text) {
        // Обработка LaTeX формул (если нужно)
        text = this.processLatex(text);
        
        // Markdown → HTML
        return marked.parse(text);
    }
    
    processLatex(text) {
        // Inline: $formula$
        text = text.replace(/\$([^\$]+)\$/g, (match, formula) => {
            return `<span class="math-inline">${this.renderMath(formula)}</span>`;
        });
        
        // Block: $$formula$$
        text = text.replace(/\$\$([^\$]+)\$\$/g, (match, formula) => {
            return `<div class="math-block">${this.renderMath(formula)}</div>`;
        });
        
        return text;
    }
    
    renderMath(formula) {
        // Можем использовать KaTeX (single file, ~200KB)
        try {
            return katex.renderToString(formula, {
                throwOnError: false,
                displayMode: false
            });
        } catch (e) {
            return formula;  // Fallback to plain text
        }
    }
}
```

### 3. Conversation Management

```javascript
// static/js/chat/conversation-manager.js
class ConversationManager {
    async createConversation(title, model) {
        const response = await fetch('/api/conversations', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.getAuthToken()}`
            },
            body: JSON.stringify({ title, model })
        });
        
        return await response.json();
    }
    
    async loadConversations() {
        const response = await fetch('/api/conversations', {
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
        
        return await response.json();
    }
    
    async loadMessages(conversationId) {
        const response = await fetch(`/api/conversations/${conversationId}/messages`, {
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
        
        return await response.json();
    }
    
    async deleteConversation(conversationId) {
        await fetch(`/api/conversations/${conversationId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
    }
    
    async toggleFavorite(conversationId) {
        await fetch(`/api/conversations/${conversationId}/favorite`, {
            method: 'PATCH',
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
    }
    
    async exportConversation(conversationId, format = 'markdown') {
        const response = await fetch(`/api/conversations/${conversationId}/export?format=${format}`, {
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
        
        const blob = await response.blob();
        this.downloadFile(blob, `conversation-${conversationId}.${format}`);
    }
}
```

### 4. Model Selector

```html
<!-- В chat UI -->
<div class="model-selector">
    <label>Model:</label>
    <select id="model-select" class="model-dropdown">
        <option value="gpt-4">GPT-4 (llama3.1:70b)</option>
        <option value="gpt-3.5-turbo">GPT-3.5 Turbo (llama3.1:8b)</option>
        <option value="text-embedding-ada-002">Ada Embeddings</option>
    </select>
    <button class="model-info-btn" title="Model Info">ℹ️</button>
</div>
```

```javascript
// Model info modal
class ModelInfoModal {
    show(modelName) {
        // Fetch model info
        fetch(`/v1/models/${modelName}`)
            .then(r => r.json())
            .then(data => {
                this.render({
                    name: data.id,
                    size: this.formatSize(data.size),
                    parameters: data.parameter_size,
                    family: data.family,
                    quantization: data.quantization_level
                });
            });
    }
}
```

### 5. Advanced Settings Panel

```html
<div class="settings-panel">
    <h3>Generation Settings</h3>
    
    <div class="setting">
        <label>Temperature: <span id="temp-value">0.7</span></label>
        <input type="range" id="temperature" min="0" max="2" step="0.1" value="0.7">
        <small>Lower = more focused, Higher = more creative</small>
    </div>
    
    <div class="setting">
        <label>Top P: <span id="topp-value">0.9</span></label>
        <input type="range" id="top_p" min="0" max="1" step="0.05" value="0.9">
    </div>
    
    <div class="setting">
        <label>Max Tokens: <span id="maxtokens-value">2048</span></label>
        <input type="range" id="max_tokens" min="256" max="8192" step="256" value="2048">
    </div>
    
    <div class="setting">
        <label>System Prompt:</label>
        <textarea id="system-prompt" rows="3" placeholder="You are a helpful assistant..."></textarea>
    </div>
    
    <button class="apply-settings-btn">Apply</button>
    <button class="reset-settings-btn">Reset to Defaults</button>
</div>
```

---

## 🎨 Styling (CSS)

```css
/* static/css/chat.css */

/* Modern chat layout */
.chat-layout {
    display: grid;
    grid-template-columns: 280px 1fr;
    height: 100vh;
    background: var(--bg-primary);
}

/* Sidebar */
.sidebar {
    background: var(--bg-secondary);
    border-right: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    overflow-y: auto;
}

.new-chat-btn {
    margin: 1rem;
    padding: 0.75rem;
    background: var(--accent-color);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 1rem;
    cursor: pointer;
    transition: background 0.2s;
}

.conversation-item {
    padding: 0.75rem 1rem;
    margin: 0.25rem 0.5rem;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.2s;
}

.conversation-item:hover {
    background: var(--hover-bg);
}

.conversation-item.active {
    background: var(--accent-color);
    color: white;
}

/* Chat area */
.chat-area {
    display: flex;
    flex-direction: column;
    height: 100vh;
}

.messages-container {
    flex: 1;
    overflow-y: auto;
    padding: 2rem;
}

.message {
    margin-bottom: 1.5rem;
    animation: fadeIn 0.3s;
}

.message.user {
    display: flex;
    justify-content: flex-end;
}

.message.assistant {
    display: flex;
    justify-content: flex-start;
}

.message-bubble {
    max-width: 70%;
    padding: 1rem 1.25rem;
    border-radius: 16px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.message.user .message-bubble {
    background: var(--user-message-bg);
    color: white;
}

.message.assistant .message-bubble {
    background: var(--assistant-message-bg);
    color: var(--text-primary);
}

/* Code blocks */
.message-bubble pre {
    background: var(--code-bg);
    padding: 1rem;
    border-radius: 8px;
    overflow-x: auto;
    margin: 0.5rem 0;
}

.message-bubble code {
    font-family: 'Fira Code', 'Consolas', monospace;
    font-size: 0.9rem;
}

/* Input area */
.input-area {
    padding: 1.5rem;
    border-top: 1px solid var(--border-color);
    background: var(--bg-secondary);
}

.input-wrapper {
    display: flex;
    gap: 0.5rem;
    align-items: flex-end;
}

.message-input {
    flex: 1;
    padding: 0.75rem 1rem;
    border: 2px solid var(--border-color);
    border-radius: 12px;
    font-size: 1rem;
    resize: vertical;
    min-height: 50px;
    max-height: 200px;
}

.send-btn {
    padding: 0.75rem 1.5rem;
    background: var(--accent-color);
    color: white;
    border: none;
    border-radius: 12px;
    cursor: pointer;
    transition: background 0.2s;
}

.send-btn:disabled {
    background: var(--disabled-bg);
    cursor: not-allowed;
}

/* Streaming indicator */
.streaming-indicator {
    display: inline-block;
    margin-left: 0.5rem;
}

.streaming-indicator span {
    animation: blink 1.4s infinite;
}

.streaming-indicator span:nth-child(2) {
    animation-delay: 0.2s;
}

.streaming-indicator span:nth-child(3) {
    animation-delay: 0.4s;
}

@keyframes blink {
    0%, 60%, 100% { opacity: 0; }
    30% { opacity: 1; }
}

@keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
}
```

---

## 📦 External Libraries (CDN)

```html
<!-- В HTML головном файле -->

<!-- Markdown rendering -->
<script src="https://cdn.jsdelivr.net/npm/marked@11.0.0/marked.min.js"></script>

<!-- Syntax highlighting -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.9.0/build/styles/github-dark.min.css">
<script src="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.9.0/build/highlight.min.js"></script>

<!-- Math rendering (optional) -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">
<script src="https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.js"></script>

<!-- Icons (optional) -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.0/font/bootstrap-icons.css">
```

**Важно**: Все библиотеки также можно скачать и поместить в `static/vendor/` для offline работы.

---

## 🔌 Backend API Endpoints

### Chat Streaming

```
POST /api/chat/stream
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "conversation_id": "conv_123",  // optional, создаст новый если NULL
  "message": "Привет, расскажи про Go",
  "model": "gpt-4",
  "stream": true,
  "settings": {
    "temperature": 0.7,
    "max_tokens": 2048
  }
}

Response: text/event-stream
data: {"choices":[{"delta":{"content":"При"}}]}
data: {"choices":[{"delta":{"content":"вет"}}]}
...
data: [DONE]
```

### Conversations CRUD

```
GET    /api/conversations              # List all user's conversations
POST   /api/conversations              # Create new conversation
GET    /api/conversations/:id          # Get conversation details
PUT    /api/conversations/:id          # Update conversation
DELETE /api/conversations/:id          # Delete conversation
PATCH  /api/conversations/:id/favorite # Toggle favorite
PATCH  /api/conversations/:id/archive  # Archive/unarchive
GET    /api/conversations/:id/messages # Get all messages
GET    /api/conversations/:id/export   # Export (markdown/json/txt)
```

---

## 🧪 Тестирование

### Unit Tests (JavaScript)

- Message rendering
- Markdown conversion
- Streaming parser
- State management

### E2E Tests

- Create new conversation
- Send message → receive streaming response
- Switch models
- Export conversation
- Delete conversation

---

## 🚀 Deployment

### Embed в Go binary

```go
//go:embed static/chat/*
var chatAssets embed.FS

func ServeChatUI(c *gin.Context) {
    // Serve index.html
    data, _ := chatAssets.ReadFile("static/chat/index.html")
    c.Data(200, "text/html", data)
}
```

---

## 🎯 Success Criteria

- ✅ ChatGPT-like интерфейс реализован
- ✅ Real-time streaming работает плавно
- ✅ Markdown + code highlighting работает
- ✅ Conversations сохраняются в БД
- ✅ Model selector работает
- ✅ Export в markdown/json/txt работает
- ✅ Responsive design (mobile-friendly)
- ✅ Zero NPM dependencies
- ✅ Single binary deployment

---

**Автор:** AI Assistant  
**Дата создания:** 2025-10-05  
**Последнее обновление:** 2025-10-05
