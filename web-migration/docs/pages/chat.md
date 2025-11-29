# Миграция Chat Page

## Приоритет: 🔴 Критический (Phase 1)

Chat - основной функционал приложения. Наиболее сложная страница.

## Текущие файлы

- `web/chat.html` - HTML разметка
- `web/js/chat.js` - Основная логика чата
- `web/js/model-panel.js` - Панель настроек модели
- `web/js/context-manager.js` - Управление контекстом
- `web/js/vlm.js` - Vision Language Models (работа с изображениями)
- `web/css/model-panel.css` - Стили панели модели

## Функционал для миграции

### 1. Sidebar (Conversations List)

**Текущий функционал:**
- Список conversations с поиском
- Создание новой беседы
- Удаление беседы
- Сворачивание sidebar

**Svelte компонент:** `ChatSidebar.svelte`

```svelte
<script>
  import { chat } from '$lib/stores/chat.svelte';
  import { Search, Plus, Trash2, ChevronLeft } from 'lucide-svelte';
  import * as ScrollArea from '$lib/components/ui/scroll-area';
  
  let searchQuery = $state('');
  let collapsed = $state(false);
  
  const filteredConversations = $derived(
    chat.sortedConversations.filter(c => 
      c.title.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );
</script>
```

### 2. Message List

**Текущий функционал:**
- Отображение user/assistant сообщений
- Markdown рендеринг (marked.js)
- Syntax highlighting (highlight.js)
- Copy to clipboard
- Streaming отображение

**Svelte компонент:** `MessageList.svelte`, `MessageBubble.svelte`

```svelte
<script>
  import { marked } from 'marked';
  import hljs from 'highlight.js';
  import { Copy, Check } from 'lucide-svelte';
  
  let { message, isStreaming = false } = $props();
  let copied = $state(false);
  
  const renderedContent = $derived(() => {
    const renderer = new marked.Renderer();
    renderer.code = (code, lang) => {
      const highlighted = lang && hljs.getLanguage(lang)
        ? hljs.highlight(code, { language: lang }).value
        : hljs.highlightAuto(code).value;
      return `<pre><code class="hljs ${lang}">${highlighted}</code></pre>`;
    };
    return marked(message.content, { renderer });
  });
</script>
```

### 3. Model Control Panel

**Текущий функционал:**
- Выбор модели
- Temperature slider (0-2)
- Top P slider (0-1)
- Max Tokens input
- Context Window indicator
- Collapsible parameters section

**Svelte компонент:** `ModelPanel.svelte`

```svelte
<script>
  import { chat } from '$lib/stores/chat.svelte';
  import { modelsStore } from '$lib/stores/models.svelte';
  import * as Select from '$lib/components/ui/select';
  import * as Slider from '$lib/components/ui/slider';
  import { Input } from '$lib/components/ui/input';
  import { ChevronDown, ChevronUp } from 'lucide-svelte';
  
  let expanded = $state(false);
</script>

<div class="model-panel">
  <div class="model-selector">
    <Select.Root 
      value={chat.params.model}
      onValueChange={(v) => chat.updateParams({ model: v })}
    >
      <Select.Trigger>
        <Select.Value placeholder="Select model" />
      </Select.Trigger>
      <Select.Content>
        {#each modelsStore.models as model}
          <Select.Item value={model.id}>{model.name}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
  </div>
  
  <button onclick={() => expanded = !expanded} class="expand-btn">
    {expanded ? 'Hide' : 'Show'} Parameters
    {#if expanded}<ChevronUp />{:else}<ChevronDown />{/if}
  </button>
  
  {#if expanded}
    <div class="parameters" transition:slide>
      <div class="param">
        <label>Temperature: {chat.params.temperature}</label>
        <Slider.Root
          value={[chat.params.temperature]}
          onValueChange={([v]) => chat.updateParams({ temperature: v })}
          min={0}
          max={2}
          step={0.1}
        />
      </div>
      <!-- ... other params -->
    </div>
  {/if}
</div>
```

### 4. RAG Integration

**Текущий функционал:**
- Toggle RAG on/off
- Data source selection (multi-select)
- Top K chunks
- Min similarity score
- Reranking toggle

**Svelte компонент:** `RAGPanel.svelte`

```svelte
<script>
  import { chat } from '$lib/stores/chat.svelte';
  import { ragSourcesStore } from '$lib/stores/rag-sources.svelte';
  import * as Checkbox from '$lib/components/ui/checkbox';
  import { Switch } from '$lib/components/ui/switch';
  
  $effect(() => {
    if (chat.params.rag_enabled) {
      ragSourcesStore.fetch();
    }
  });
</script>

<div class="rag-panel">
  <div class="rag-toggle">
    <Switch 
      checked={chat.params.rag_enabled}
      onCheckedChange={(v) => chat.updateParams({ rag_enabled: v })}
    />
    <span>Enable RAG</span>
  </div>
  
  {#if chat.params.rag_enabled}
    <div class="rag-sources">
      {#each ragSourcesStore.sources as source}
        <Checkbox.Root
          checked={chat.params.rag_source_ids?.includes(source.id)}
          onCheckedChange={(checked) => {
            const ids = chat.params.rag_source_ids || [];
            chat.updateParams({
              rag_source_ids: checked 
                ? [...ids, source.id]
                : ids.filter(id => id !== source.id)
            });
          }}
        >
          <Checkbox.Indicator />
          {source.name}
        </Checkbox.Root>
      {/each}
    </div>
  {/if}
</div>
```

### 5. Input Area

**Текущий функционал:**
- Textarea с автоматическим resize
- Send button
- Stop streaming button
- File attachment (VLM)
- Image attachment (VLM)
- Markdown toolbar

**Svelte компонент:** `ChatInput.svelte`

```svelte
<script>
  import { chat } from '$lib/stores/chat.svelte';
  import { Button } from '$lib/components/ui/button';
  import { Textarea } from '$lib/components/ui/textarea';
  import { Send, Square, Paperclip, Image } from 'lucide-svelte';
  
  let message = $state('');
  let textarea: HTMLTextAreaElement;
  
  function autoResize() {
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
    }
  }
  
  async function handleSend() {
    if (!message.trim() || chat.streaming) return;
    const content = message;
    message = '';
    await chat.sendMessage(content);
  }
  
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }
</script>

<div class="chat-input">
  <div class="input-toolbar">
    <Button variant="ghost" size="icon">
      <Paperclip class="w-4 h-4" />
    </Button>
    <Button variant="ghost" size="icon">
      <Image class="w-4 h-4" />
    </Button>
  </div>
  
  <Textarea
    bind:ref={textarea}
    bind:value={message}
    oninput={autoResize}
    onkeydown={handleKeyDown}
    placeholder="Type a message..."
    rows={1}
  />
  
  {#if chat.streaming}
    <Button variant="destructive" onclick={() => chat.stopStream()}>
      <Square class="w-4 h-4" />
    </Button>
  {:else}
    <Button onclick={handleSend} disabled={!message.trim()}>
      <Send class="w-4 h-4" />
    </Button>
  {/if}
</div>
```

### 6. VLM (Vision Language Models)

**Текущий функционал:**
- Drag & drop изображений
- Preview изображений в чате
- Base64 encoding для API
- Multi-image support

**Svelte компонент:** `ImageAttachment.svelte`

```svelte
<script>
  let { onAttach } = $props();
  let isDragging = $state(false);
  let previews = $state<{ file: File; dataUrl: string }[]>([]);
  
  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    
    const files = Array.from(e.dataTransfer?.files || [])
      .filter(f => f.type.startsWith('image/'));
    
    for (const file of files) {
      const dataUrl = await readFileAsDataURL(file);
      previews = [...previews, { file, dataUrl }];
    }
    
    onAttach(previews.map(p => p.dataUrl));
  }
</script>
```

### 7. Export/Import Conversations

**Текущий функционал:**
- Export conversation to JSON
- Import conversation from JSON
- Download как файл

**Svelte компонент:** Часть `ChatSidebar.svelte`

```svelte
<script>
  import { Download, Upload } from 'lucide-svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  
  function exportConversation(conv) {
    const data = {
      title: conv.title,
      model: conv.model,
      messages: chat.messages,
      exportedAt: new Date().toISOString()
    };
    
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${conv.title.replace(/\s+/g, '_')}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }
</script>
```

## Структура файлов

```
routes/(app)/chat/
├── +page.svelte          # Main chat page
├── +page.ts              # Data loading
└── +layout.svelte        # Chat-specific layout (no main padding)

lib/components/chat/
├── ChatSidebar.svelte
├── ConversationItem.svelte
├── MessageList.svelte
├── MessageBubble.svelte
├── ModelPanel.svelte
├── RAGPanel.svelte
├── ChatInput.svelte
├── ImageAttachment.svelte
└── MarkdownToolbar.svelte
```

## Зависимости

```json
{
  "dependencies": {
    "marked": "^15.0.0",
    "highlight.js": "^11.10.0"
  }
}
```

## API Endpoints используемые

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/conversations` | GET | Список conversations |
| `/api/conversations` | POST | Создать conversation |
| `/api/conversations/:id` | GET | Получить conversation |
| `/api/conversations/:id` | PUT | Обновить conversation |
| `/api/conversations/:id` | DELETE | Удалить conversation |
| `/api/conversations/:id/messages` | GET | Сообщения conversation |
| `/api/conversations/:id/messages` | POST | Добавить сообщение |
| `/v1/chat/completions` | POST | OpenAI-compatible chat (streaming) |
| `/api/models` | GET | Список моделей |
| `/api/rag/sources` | GET | Список RAG источников |

## Особенности миграции

1. **Streaming** - реализовать через async generator
2. **Markdown rendering** - marked + highlight.js (без изменений)
3. **Real-time updates** - Svelte 5 reactivity вместо DOM manipulation
4. **VLM support** - drag & drop через native events
5. **Keyboard shortcuts** - Ctrl+Enter для отправки, Esc для отмены

## Тестирование

- [ ] Создание новой беседы
- [ ] Отправка сообщений
- [ ] Streaming response
- [ ] Stop streaming
- [ ] Markdown rendering с code blocks
- [ ] Copy code button
- [ ] Model selection
- [ ] Parameter adjustment
- [ ] RAG toggle и source selection
- [ ] File/image attachment
- [ ] Export/Import conversation
- [ ] Sidebar collapse
- [ ] Mobile responsiveness

