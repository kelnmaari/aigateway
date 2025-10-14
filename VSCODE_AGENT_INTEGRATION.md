# VS Code Web + AI Agent Integration

> **Идея:** Интеграция Ollama-OpenAI Proxy с web-based VS Code для создания агентского режима разработки

## 📋 Оглавление

- [Обзор решения](#обзор-решения)
- [Готовые платформы](#готовые-платформы)
- [Архитектура extension](#архитектура-extension)
- [Агентский режим](#агентский-режим)
- [План реализации](#план-реализации)
- [Ресурсы и ссылки](#ресурсы-и-ссылки)

---

## 🎯 Обзор решения

### Цель

Создать web-based VS Code с AI агентом, использующим наш Ollama-OpenAI Proxy для:

- Автоматического написания кода
- Рефакторинга и оптимизации
- Генерации тестов
- Debugging assistance
- Multi-file operations

### Преимущества нашего подхода

✅ **Privacy-first**: Код не покидает инфраструктуру  
✅ **Cost-effective**: Нулевые затраты на API (локальный Ollama)  
✅ **Offline-capable**: Работает без интернета  
✅ **Full control**: Полный контроль над моделями и данными  

---

## 🖥️ Готовые платформы

### 1. **code-server** ⭐ (Рекомендуется)

**Описание:** Полноценный VS Code в браузере от Coder.com

**Установка:**

```bash
# Docker (самый простой способ)
docker run -it -p 8080:8080 \
  -v "$HOME/.config:/home/coder/.config" \
  -v "$PWD:/home/coder/project" \
  -e PASSWORD="your-password" \
  codercom/code-server:latest

# Или через curl
curl -fsSL https://code-server.dev/install.sh | sh
code-server --bind-addr 0.0.0.0:8080
```

**Плюсы:**

- ✅ 100% совместимость с VS Code extensions
- ✅ Полный доступ к файловой системе
- ✅ Встроенный терминал
- ✅ Git integration
- ✅ Extension marketplace доступен

**Где взять:**

- GitHub: <https://github.com/coder/code-server>
- Docs: <https://coder.com/docs/code-server>

---

### 2. **Eclipse Theia**

**Описание:** Open-source Cloud & Desktop IDE platform

**Установка:**

```bash
# Docker
docker run -it -p 3000:3000 \
  -v "$(pwd):/home/project:cached" \
  theiaide/theia:latest

# Или build from source
git clone https://github.com/eclipse-theia/theia
cd theia
yarn && yarn browser build
```

**Плюсы:**

- ✅ Более гибкая архитектура для кастомизации
- ✅ Plugin API схож с VS Code
- ✅ Можно полностью переделать под себя

**Где взять:**

- GitHub: <https://github.com/eclipse-theia/theia>
- Docs: <https://theia-ide.org/docs/>

---

### 3. **vscode.dev** (Official Microsoft)

**Описание:** Официальный VS Code в браузере

**URL:** <https://vscode.dev>

**Ограничения:**

- ⚠️ Только File System Access API (ограниченный FS)
- ⚠️ Не все extensions работают
- ⚠️ Нет прямого доступа к терминалу

**Когда использовать:**

- Для просмотра/редактирования файлов
- Для легких задач без агентского режима

---

## 🔌 Архитектура Extension

### Структура проекта extension

```
ollama-agent-extension/
├── package.json              # Extension manifest
├── tsconfig.json            # TypeScript config
├── src/
│   ├── extension.ts         # Entry point
│   ├── providers/
│   │   ├── AIProvider.ts    # API integration с Ollama Proxy
│   │   ├── ContextProvider.ts  # Сбор контекста workspace
│   │   └── AgentExecutor.ts    # Агентская логика
│   ├── webview/
│   │   ├── ChatPanel.tsx    # React UI для чата
│   │   └── AgentUI.tsx      # UI агентского режима
│   └── utils/
│       ├── fileOperations.ts
│       └── terminalHelper.ts
├── media/                   # Icons, CSS
└── README.md
```

### package.json (Extension Manifest)

```json
{
  "name": "ollama-agent",
  "displayName": "Ollama AI Agent",
  "description": "AI coding agent powered by local Ollama models",
  "version": "0.1.0",
  "engines": {
    "vscode": "^1.85.0"
  },
  "categories": ["AI", "Programming Languages", "Other"],
  "activationEvents": [
    "onCommand:ollama.chat",
    "onCommand:ollama.agentMode",
    "onLanguage:*"
  ],
  "main": "./dist/extension.js",
  "contributes": {
    "commands": [
      {
        "command": "ollama.chat",
        "title": "Open Ollama Chat",
        "category": "Ollama"
      },
      {
        "command": "ollama.agentMode",
        "title": "Start Agent Mode",
        "category": "Ollama"
      },
      {
        "command": "ollama.refactor",
        "title": "AI Refactor",
        "category": "Ollama"
      }
    ],
    "configuration": {
      "title": "Ollama Agent",
      "properties": {
        "ollama.proxyUrl": {
          "type": "string",
          "default": "http://localhost:8080",
          "description": "Ollama-OpenAI Proxy URL"
        },
        "ollama.apiKey": {
          "type": "string",
          "description": "API Key for authentication"
        },
        "ollama.defaultModel": {
          "type": "string",
          "default": "qwen2.5-coder-tuned:7b",
          "description": "Default model for code generation"
        },
        "ollama.temperature": {
          "type": "number",
          "default": 0.2,
          "description": "Model temperature (0.0-2.0)"
        },
        "ollama.contextWindow": {
          "type": "number",
          "default": 8192,
          "description": "Context window size"
        }
      }
    }
  },
  "scripts": {
    "vscode:prepublish": "npm run compile",
    "compile": "tsc -p ./",
    "watch": "tsc -watch -p ./",
    "package": "vsce package"
  },
  "devDependencies": {
    "@types/node": "^20.x",
    "@types/vscode": "^1.85.0",
    "typescript": "^5.3.0",
    "@vscode/vsce": "^2.22.0"
  },
  "dependencies": {
    "openai": "^4.20.1",
    "axios": "^1.6.0"
  }
}
```

### src/extension.ts (Entry Point)

```typescript
import * as vscode from 'vscode';
import { AIProvider } from './providers/AIProvider';
import { AgentExecutor } from './providers/AgentExecutor';
import { ChatPanel } from './webview/ChatPanel';

export function activate(context: vscode.ExtensionContext) {
    console.log('Ollama Agent extension activated');

    // Получаем конфигурацию
    const config = vscode.workspace.getConfiguration('ollama');
    const proxyUrl = config.get<string>('proxyUrl') || 'http://localhost:8080';
    const apiKey = config.get<string>('apiKey') || '';

    // Инициализируем AI Provider
    const aiProvider = new AIProvider(proxyUrl, apiKey);

    // Command: Open Chat
    const chatCommand = vscode.commands.registerCommand('ollama.chat', () => {
        ChatPanel.createOrShow(context.extensionUri, aiProvider);
    });

    // Command: Agent Mode
    const agentCommand = vscode.commands.registerCommand('ollama.agentMode', async () => {
        const task = await vscode.window.showInputBox({
            prompt: 'Describe what you want to build or modify',
            placeHolder: 'e.g., Create a REST API for user authentication'
        });

        if (!task) return;

        const agent = new AgentExecutor(aiProvider);
        
        await vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: 'AI Agent working...',
            cancellable: true
        }, async (progress, token) => {
            await agent.executeTask(task, progress, token);
        });
    });

    // Inline Completions Provider
    const completionProvider = vscode.languages.registerInlineCompletionItemProvider(
        { pattern: '**' },
        {
            async provideInlineCompletionItems(document, position, context, token) {
                const textBeforeCursor = document.getText(
                    new vscode.Range(
                        new vscode.Position(Math.max(0, position.line - 10), 0),
                        position
                    )
                );

                const completion = await aiProvider.complete(textBeforeCursor);
                
                return {
                    items: [{
                        insertText: completion,
                        range: new vscode.Range(position, position)
                    }]
                };
            }
        }
    );

    context.subscriptions.push(chatCommand, agentCommand, completionProvider);
}

export function deactivate() {}
```

### src/providers/AIProvider.ts (API Integration)

```typescript
import axios, { AxiosInstance } from 'axios';

export interface ChatMessage {
    role: 'system' | 'user' | 'assistant';
    content: string;
}

export interface ChatResponse {
    content: string;
    model: string;
    usage?: {
        prompt_tokens: number;
        completion_tokens: number;
        total_tokens: number;
    };
}

export class AIProvider {
    private client: AxiosInstance;
    private model: string;

    constructor(
        private proxyUrl: string,
        private apiKey: string
    ) {
        this.client = axios.create({
            baseURL: proxyUrl,
            headers: {
                'Authorization': `Bearer ${apiKey}`,
                'Content-Type': 'application/json'
            },
            timeout: 60000
        });

        const config = vscode.workspace.getConfiguration('ollama');
        this.model = config.get<string>('defaultModel') || 'qwen2.5-coder-tuned:7b';
    }

    /**
     * Chat completion (non-streaming)
     */
    async chat(messages: ChatMessage[]): Promise<ChatResponse> {
        const config = vscode.workspace.getConfiguration('ollama');
        
        const response = await this.client.post('/v1/chat/completions', {
            model: this.model,
            messages: messages,
            temperature: config.get<number>('temperature') || 0.2,
            max_tokens: 4096,
            stream: false,
            options: {
                num_ctx: config.get<number>('contextWindow') || 8192
            }
        });

        return {
            content: response.data.choices[0].message.content,
            model: response.data.model,
            usage: response.data.usage
        };
    }

    /**
     * Streaming chat completion
     */
    async *streamChat(messages: ChatMessage[]): AsyncGenerator<string> {
        const config = vscode.workspace.getConfiguration('ollama');
        
        const response = await this.client.post('/v1/chat/completions', {
            model: this.model,
            messages: messages,
            temperature: config.get<number>('temperature') || 0.2,
            stream: true,
            options: {
                num_ctx: config.get<number>('contextWindow') || 8192
            }
        }, {
            responseType: 'stream'
        });

        for await (const chunk of response.data) {
            const lines = chunk.toString().split('\n').filter((line: string) => line.trim());
            
            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.slice(6);
                    if (data === '[DONE]') return;
                    
                    try {
                        const parsed = JSON.parse(data);
                        const content = parsed.choices[0]?.delta?.content;
                        if (content) yield content;
                    } catch (e) {
                        console.error('Failed to parse chunk:', e);
                    }
                }
            }
        }
    }

    /**
     * Code completion (inline suggestions)
     */
    async complete(prefix: string): Promise<string> {
        const response = await this.chat([
            {
                role: 'system',
                content: 'You are a code completion assistant. Complete the code based on the prefix. Return ONLY the completion, no explanations.'
            },
            {
                role: 'user',
                content: `Complete this code:\n\n${prefix}`
            }
        ]);

        return response.content.trim();
    }

    /**
     * Get available models
     */
    async getModels(): Promise<string[]> {
        const response = await this.client.get('/v1/models');
        return response.data.data.map((model: any) => model.id);
    }
}
```

---

## 🤖 Агентский режим

### src/providers/AgentExecutor.ts

```typescript
import * as vscode from 'vscode';
import { AIProvider, ChatMessage } from './AIProvider';
import { ContextProvider } from './ContextProvider';

interface AgentAction {
    type: 'create_file' | 'edit_file' | 'delete_file' | 'run_command' | 'search_replace';
    path?: string;
    content?: string;
    command?: string;
    search?: string;
    replace?: string;
    reasoning: string;
}

interface AgentPlan {
    goal: string;
    steps: AgentAction[];
    reasoning: string;
}

export class AgentExecutor {
    private contextProvider: ContextProvider;

    constructor(private aiProvider: AIProvider) {
        this.contextProvider = new ContextProvider();
    }

    async executeTask(
        task: string,
        progress: vscode.Progress<{ message?: string; increment?: number }>,
        token: vscode.CancellationToken
    ): Promise<void> {
        // Получаем контекст workspace
        const context = await this.contextProvider.getWorkspaceContext();
        
        // Формируем system prompt для агента
        const systemPrompt = this.buildAgentSystemPrompt();
        
        let conversationHistory: ChatMessage[] = [
            { role: 'system', content: systemPrompt },
            { role: 'user', content: `Task: ${task}\n\nWorkspace Context:\n${JSON.stringify(context, null, 2)}` }
        ];

        let maxIterations = 10;
        let iteration = 0;

        while (iteration < maxIterations && !token.isCancellationRequested) {
            iteration++;
            progress.report({ 
                message: `Iteration ${iteration}/${maxIterations}`,
                increment: 10
            });

            // Получаем план от AI
            const planResponse = await this.aiProvider.chat(conversationHistory);
            
            let plan: AgentPlan;
            try {
                plan = JSON.parse(planResponse.content);
            } catch (e) {
                vscode.window.showErrorMessage('Failed to parse agent plan');
                break;
            }

            // Проверяем завершение
            if (plan.steps.length === 0 || plan.goal === 'completed') {
                vscode.window.showInformationMessage('Task completed!');
                break;
            }

            // Выполняем действия
            const results: string[] = [];
            for (const action of plan.steps) {
                const result = await this.executeAction(action, progress);
                results.push(result);
            }

            // Добавляем результаты в историю
            conversationHistory.push(
                { role: 'assistant', content: planResponse.content },
                { role: 'user', content: `Execution results:\n${results.join('\n\n')}` }
            );
        }

        if (iteration >= maxIterations) {
            vscode.window.showWarningMessage('Max iterations reached. Task may be incomplete.');
        }
    }

    private async executeAction(
        action: AgentAction,
        progress: vscode.Progress<{ message?: string }>
    ): Promise<string> {
        progress.report({ message: action.reasoning });

        switch (action.type) {
            case 'create_file':
                return await this.createFile(action.path!, action.content!);
            
            case 'edit_file':
                return await this.editFile(action.path!, action.content!);
            
            case 'delete_file':
                return await this.deleteFile(action.path!);
            
            case 'run_command':
                return await this.runTerminalCommand(action.command!);
            
            case 'search_replace':
                return await this.searchReplace(action.path!, action.search!, action.replace!);
            
            default:
                return `Unknown action type: ${action.type}`;
        }
    }

    private async createFile(path: string, content: string): Promise<string> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) return 'No workspace folder';

        const uri = vscode.Uri.joinPath(workspaceFolder.uri, path);
        const encoder = new TextEncoder();
        
        await vscode.workspace.fs.writeFile(uri, encoder.encode(content));
        
        return `Created file: ${path}`;
    }

    private async editFile(path: string, newContent: string): Promise<string> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) return 'No workspace folder';

        const uri = vscode.Uri.joinPath(workspaceFolder.uri, path);
        const encoder = new TextEncoder();
        
        await vscode.workspace.fs.writeFile(uri, encoder.encode(newContent));
        
        return `Edited file: ${path}`;
    }

    private async deleteFile(path: string): Promise<string> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) return 'No workspace folder';

        const uri = vscode.Uri.joinPath(workspaceFolder.uri, path);
        await vscode.workspace.fs.delete(uri);
        
        return `Deleted file: ${path}`;
    }

    private async runTerminalCommand(command: string): Promise<string> {
        const terminal = vscode.window.createTerminal('Ollama Agent');
        terminal.show();
        terminal.sendText(command);
        
        return `Executed command: ${command}`;
    }

    private async searchReplace(path: string, search: string, replace: string): Promise<string> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) return 'No workspace folder';

        const uri = vscode.Uri.joinPath(workspaceFolder.uri, path);
        const document = await vscode.workspace.openTextDocument(uri);
        
        const edit = new vscode.WorkspaceEdit();
        const text = document.getText();
        const newText = text.replace(new RegExp(search, 'g'), replace);
        
        edit.replace(uri, new vscode.Range(0, 0, document.lineCount, 0), newText);
        await vscode.workspace.applyEdit(edit);
        
        return `Replaced "${search}" with "${replace}" in ${path}`;
    }

    private buildAgentSystemPrompt(): string {
        return `You are an AI coding agent that helps developers by autonomously writing, editing, and managing code.

Your capabilities:
- Create, edit, and delete files
- Run terminal commands
- Search and replace in files
- Read workspace structure and diagnostics

Response format (JSON):
{
  "goal": "completed" | "in_progress",
  "reasoning": "explanation of current step",
  "steps": [
    {
      "type": "create_file" | "edit_file" | "delete_file" | "run_command" | "search_replace",
      "path": "relative/path/to/file",
      "content": "file content (for create/edit)",
      "command": "terminal command (for run_command)",
      "search": "regex pattern (for search_replace)",
      "replace": "replacement text (for search_replace)",
      "reasoning": "why this step"
    }
  ]
}

Rules:
1. ALWAYS respond with valid JSON
2. Break complex tasks into small steps (max 3 steps per iteration)
3. Verify results before marking as completed
4. Use TypeScript/JavaScript best practices
5. Follow existing code style
6. Set goal="completed" when task is done

Example:
{
  "goal": "in_progress",
  "reasoning": "Creating user authentication API",
  "steps": [
    {
      "type": "create_file",
      "path": "src/auth/user.ts",
      "content": "export interface User { id: string; email: string; }",
      "reasoning": "Define User type"
    }
  ]
}`;
    }
}
```

### src/providers/ContextProvider.ts

```typescript
import * as vscode from 'vscode';

export interface WorkspaceContext {
    files: string[];
    openFiles: { path: string; content: string }[];
    diagnostics: { file: string; errors: string[] }[];
    gitStatus?: {
        branch: string;
        modified: string[];
        untracked: string[];
    };
}

export class ContextProvider {
    async getWorkspaceContext(): Promise<WorkspaceContext> {
        const context: WorkspaceContext = {
            files: [],
            openFiles: [],
            diagnostics: []
        };

        // Получаем список файлов
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (workspaceFolder) {
            const files = await vscode.workspace.findFiles('**/*', '**/node_modules/**');
            context.files = files.map(f => vscode.workspace.asRelativePath(f));
        }

        // Получаем открытые файлы
        for (const doc of vscode.workspace.textDocuments) {
            if (doc.uri.scheme === 'file') {
                context.openFiles.push({
                    path: vscode.workspace.asRelativePath(doc.uri),
                    content: doc.getText()
                });
            }
        }

        // Получаем diagnostics (ошибки, warnings)
        const diagnostics = vscode.languages.getDiagnostics();
        for (const [uri, diags] of diagnostics) {
            if (diags.length > 0) {
                context.diagnostics.push({
                    file: vscode.workspace.asRelativePath(uri),
                    errors: diags.map(d => `${d.severity}: ${d.message}`)
                });
            }
        }

        // TODO: Git status (requires git extension API)
        
        return context;
    }
}
```

---

## 🚀 План реализации

### Этап 1: Базовая интеграция (1-2 недели)

**Задачи:**

- [ ] Настроить code-server в Docker
- [ ] Создать структуру extension (package.json, tsconfig.json)
- [ ] Реализовать AIProvider для интеграции с Ollama Proxy
- [ ] Добавить базовый chat panel (webview)
- [ ] Протестировать completions

**Результат:** Работающий chat и code completions через наш proxy

---

### Этап 2: Agent Mode MVP (2-4 недели)

**Задачи:**

- [ ] Реализовать AgentExecutor с базовыми действиями
- [ ] Добавить ContextProvider для сбора workspace info
- [ ] Создать system prompt для агентской логики
- [ ] Реализовать file operations (create, edit, delete)
- [ ] Добавить terminal integration
- [ ] Тестирование на простых задачах

**Результат:** Агент, способный выполнять простые задачи (создание файлов, базовый рефакторинг)

---

### Этап 3: Advanced Features (4-8 недель)

**Задачи:**

- [ ] Multi-file refactoring
- [ ] Test generation и execution
- [ ] Git integration (commits, branches)
- [ ] Debugging assistance (breakpoints, variable inspection)
- [ ] Codebase indexing + RAG (vector DB)
- [ ] Error recovery и retry logic

**Результат:** Полнофункциональный AI agent

---

## 📚 Ресурсы и ссылки

### Документация VS Code Extension API

- **Official Docs**: <https://code.visualstudio.com/api>
- **Extension Guides**: <https://code.visualstudio.com/api/extension-guides/overview>
- **API Reference**: <https://code.visualstudio.com/api/references/vscode-api>
- **Sample Extensions**: <https://github.com/microsoft/vscode-extension-samples>

### Готовые AI Extension для изучения

#### 1. **Continue.dev** ⭐⭐⭐

- GitHub: <https://github.com/continuedev/continue>
- Особенности: OpenAI-compatible API support, agent mode, context-aware
- **Можно форкнуть и адаптировать!**

#### 2. **Cody by Sourcegraph**

- GitHub: <https://github.com/sourcegraph/cody>
- Особенности: Codebase indexing, multi-repo support

#### 3. **GitHub Copilot** (Closed Source, для reference)

- Docs: <https://docs.github.com/en/copilot>
- Useful для понимания UX patterns

### Платформы для Web IDE

#### code-server

- GitHub: <https://github.com/coder/code-server>
- Docs: <https://coder.com/docs/code-server/latest>
- Docker Hub: <https://hub.docker.com/r/codercom/code-server>

#### Eclipse Theia

- Website: <https://theia-ide.org/>
- GitHub: <https://github.com/eclipse-theia/theia>
- Architecture: <https://theia-ide.org/docs/architecture/>

### Инструменты разработки

#### VS Code Extension Tools

```bash
# Генератор шаблона extension
npm install -g yo generator-code
yo code

# Packaging и publishing
npm install -g @vscode/vsce
vsce package
vsce publish
```

#### Testing Tools

- **vscode-test**: <https://github.com/microsoft/vscode-test>
- **@vscode/test-electron**: для E2E тестов

### OpenAI-compatible API клиенты

#### JavaScript/TypeScript

```bash
npm install openai  # Official SDK (работает с любым OpenAI-compatible API)
npm install axios   # Для прямых HTTP запросов
```

#### Примеры использования

```typescript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',  // Наш Ollama Proxy
  apiKey: 'your-api-key'
});

const completion = await client.chat.completions.create({
  model: 'qwen2.5-coder-tuned:7b',
  messages: [{ role: 'user', content: 'Hello!' }]
});
```

---

## 🔧 Практические шаги для старта

### Шаг 1: Запустить code-server

```bash
# Создать docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  code-server:
    image: codercom/code-server:latest
    ports:
      - "8080:8080"
    volumes:
      - ./workspace:/home/coder/workspace
      - ./extensions:/home/coder/.local/share/code-server/extensions
    environment:
      - PASSWORD=your-secure-password
    restart: unless-stopped
EOF

# Запустить
docker-compose up -d

# Открыть в браузере: http://localhost:8080
```

### Шаг 2: Создать extension

```bash
# Установить генератор
npm install -g yo generator-code

# Создать новый extension
yo code

# Выбрать:
# - New Extension (TypeScript)
# - Extension name: ollama-agent
# - Identifier: ollama-agent
# - Description: AI coding agent with Ollama
# - Git: yes
# - Package manager: npm
```

### Шаг 3: Настроить базовую интеграцию

```bash
cd ollama-agent

# Установить зависимости
npm install openai axios

# Создать структуру
mkdir -p src/providers src/webview src/utils

# Скопировать код из этого документа в соответствующие файлы
```

### Шаг 4: Локальное тестирование

```bash
# Открыть в VS Code
code .

# Нажать F5 для запуска в debug mode
# Откроется новое окно VS Code с установленным extension

# Протестировать команды:
# Ctrl+Shift+P → "Ollama: Open Chat"
# Ctrl+Shift+P → "Ollama: Start Agent Mode"
```

### Шаг 5: Package для code-server

```bash
# Собрать .vsix файл
npm run vscode:prepublish
vsce package

# Установить в code-server
# 1. Открыть code-server в браузере
# 2. Extensions → Install from VSIX
# 3. Загрузить ollama-agent-0.1.0.vsix
```

---

## ⚙️ Конфигурация интеграции с Ollama Proxy

### settings.json для VS Code / code-server

```json
{
  "ollama.proxyUrl": "http://localhost:8080",
  "ollama.apiKey": "your-jwt-token",
  "ollama.defaultModel": "qwen2.5-coder-tuned:7b",
  "ollama.temperature": 0.2,
  "ollama.contextWindow": 8192,
  "ollama.enableCompletions": true,
  "ollama.enableAgentMode": true
}
```

### Переменные окружения

```bash
# .env для локальной разработки
OLLAMA_PROXY_URL=http://localhost:8080
OLLAMA_API_KEY=your-api-key
OLLAMA_DEFAULT_MODEL=qwen2.5-coder-tuned:7b
```

---

## 🎯 Ключевые особенности нашего подхода

### 1. Privacy-First Architecture

- ✅ Весь код и данные остаются локально
- ✅ Нет отправки данных на внешние серверы
- ✅ Compliance с корпоративными политиками

### 2. Cost-Free Operations

- ✅ Неограниченные API requests
- ✅ Нет затрат на токены
- ✅ Масштабируется на ваших мощностях

### 3. Customizable Models

- ✅ Полный контроль над моделями (температура, контекст, промпты)
- ✅ Fine-tuning под специфику проекта
- ✅ A/B тестирование разных моделей

### 4. Integration Ready

- ✅ OpenAI-compatible API → легкая интеграция
- ✅ JWT authentication → безопасность из коробки
- ✅ Rate limiting → защита от перегрузок

---

## 📝 Чеклист для реализации

### MVP (минимально жизнеспособный продукт)

- [ ] code-server запущен и доступен
- [ ] Extension scaffold создан
- [ ] AIProvider интегрирован с Ollama Proxy
- [ ] Chat panel работает
- [ ] Code completions работают
- [ ] Базовый agent mode (create/edit files)

### Production-Ready

- [ ] Multi-step agent reasoning
- [ ] Error handling и retry logic
- [ ] Context управление (суммаризация при переполнении)
- [ ] Git integration
- [ ] Test generation
- [ ] Performance monitoring
- [ ] Документация для пользователей
- [ ] CI/CD для автоматической сборки

---

## 🚨 Важные замечания

### Безопасность

1. **API Key Storage**: Используй VS Code Secret Storage API вместо plaintext
2. **Command Execution**: Всегда спрашивай подтверждение перед выполнением terminal команд
3. **File Operations**: Показывай diff перед применением изменений

### Performance

1. **Context Size**: Ограничивай размер контекста, чтобы не превысить лимиты модели
2. **Caching**: Кешируй результаты для повторяющихся запросов
3. **Streaming**: Используй streaming для больших ответов

### UX

1. **Progress Indication**: Всегда показывай прогресс для долгих операций
2. **Cancellation**: Позволяй отменять агентские задачи
3. **Transparency**: Показывай, что делает агент (reasoning, actions)

---

## 📞 Контакты и поддержка

**Если есть вопросы или нужна помощь:**

1. Изучи примеры из Continue.dev (максимально похожая архитектура)
2. Проверь VS Code Extension Samples
3. Задай вопрос в issues нашего проекта

**Полезные community:**

- VS Code Extensions Discord
- Ollama Discord
- r/vscode на Reddit

---

## 🎉 Заключение

**Реализуемо**: ✅ Да  
**Сложность**: 🟡 Средняя  
**Время на MVP**: 2-4 недели  
**Время на Production**: 2-3 месяца  

**Самый быстрый путь:**

1. Форкнуть Continue.dev
2. Заменить API endpoint на наш Ollama Proxy
3. Кастомизировать под свои нужды
4. Упаковать в .vsix и установить в code-server

**Результат:**

- Полноценный AI coding assistant в браузере
- 100% privacy и контроль
- $0 затрат на API
- Работает offline

Удачи в реализации! 🚀
