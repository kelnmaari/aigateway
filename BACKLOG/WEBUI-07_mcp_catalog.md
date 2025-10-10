# WEBUI-07: MCP Servers Catalog

**Версия:** v1.4.5  
**Приоритет:** MEDIUM  
**Оценка:** 6-8 часов  
**Зависимости:** DB-01 (уже реализовано)  
**Статус:** 📋 Planned

---

## 📋 Описание

Создать каталог MCP (Model Context Protocol) серверов как справочную страницу. Администраторы могут добавлять/редактировать записи, пользователи просматривают каталог. Без функционала использования в чате - только информационный справочник.

## 🎯 Цели

1. Создать централизованный каталог доступных MCP серверов
2. Предоставить инструкции по установке и настройке
3. Упростить discovery новых MCP серверов для пользователей
4. Admin-friendly управление каталогом

## 🔍 Что такое MCP?

**Model Context Protocol** - протокол для расширения возможностей LLM моделей через внешние инструменты и источники данных.

**Примеры MCP серверов:**

- **filesystem** - доступ к файловой системе
- **postgres** - подключение к PostgreSQL БД
- **brave-search** - поиск через Brave API
- **github** - интеграция с GitHub
- **puppeteer** - веб-скрапинг и автоматизация

## 📊 Scope

### Database Schema

#### 1. MCP Servers Table

**Таблица:** `mcp_servers`

```sql
CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(100) NOT NULL,
    icon_url VARCHAR(500),
    repository_url VARCHAR(500),
    documentation_url VARCHAR(500),
    npm_package VARCHAR(255),
    installation_command TEXT,
    configuration_example TEXT,
    features JSONB,
    tags TEXT[],
    is_official BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT mcp_servers_name_check CHECK (name ~ '^[a-z0-9-]+$')
);

CREATE INDEX idx_mcp_servers_category ON mcp_servers(category);
CREATE INDEX idx_mcp_servers_tags ON mcp_servers USING GIN(tags);
CREATE INDEX idx_mcp_servers_active ON mcp_servers(is_active) WHERE is_active = true;
```

**Categories:**

- Development (filesystem, git, github)
- Data & Databases (postgres, sqlite, mongodb)
- Search & Web (brave-search, google-drive, web-scraper)
- Communication (slack, email, calendar)
- Cloud Services (aws, gcp, azure)
- AI & ML (embeddings, vector-db)
- Other

### Backend API

#### 1. Data Models

**Файл:** `internal/models/mcp.go`

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

// MCPServer представляет MCP сервер в каталоге
type MCPServer struct {
    ID                 uuid.UUID       `json:"id" db:"id"`
    Name               string          `json:"name" db:"name"`
    DisplayName        string          `json:"display_name" db:"display_name"`
    Description        string          `json:"description" db:"description"`
    Category           string          `json:"category" db:"category"`
    IconURL            *string         `json:"icon_url,omitempty" db:"icon_url"`
    RepositoryURL      *string         `json:"repository_url,omitempty" db:"repository_url"`
    DocumentationURL   *string         `json:"documentation_url,omitempty" db:"documentation_url"`
    NPMPackage         *string         `json:"npm_package,omitempty" db:"npm_package"`
    InstallationCmd    *string         `json:"installation_command,omitempty" db:"installation_command"`
    ConfigExample      *string         `json:"configuration_example,omitempty" db:"configuration_example"`
    Features           MCPFeatures     `json:"features,omitempty" db:"features"`
    Tags               []string        `json:"tags" db:"tags"`
    IsOfficial         bool            `json:"is_official" db:"is_official"`
    IsActive           bool            `json:"is_active" db:"is_active"`
    CreatedBy          *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
    CreatedAt          time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// MCPFeatures дополнительные возможности MCP сервера
type MCPFeatures struct {
    SupportsStreaming bool     `json:"supports_streaming"`
    RequiresAuth      bool     `json:"requires_auth"`
    Platform          []string `json:"platform"` // ["node", "python", "docker"]
}

// CreateMCPServerRequest запрос на создание MCP сервера
type CreateMCPServerRequest struct {
    Name              string      `json:"name" binding:"required,min=3,max=100"`
    DisplayName       string      `json:"display_name" binding:"required,min=3,max=255"`
    Description       string      `json:"description" binding:"required,min=10"`
    Category          string      `json:"category" binding:"required"`
    IconURL           *string     `json:"icon_url"`
    RepositoryURL     *string     `json:"repository_url"`
    DocumentationURL  *string     `json:"documentation_url"`
    NPMPackage        *string     `json:"npm_package"`
    InstallationCmd   *string     `json:"installation_command"`
    ConfigExample     *string     `json:"configuration_example"`
    Features          MCPFeatures `json:"features"`
    Tags              []string    `json:"tags"`
    IsOfficial        bool        `json:"is_official"`
}

// UpdateMCPServerRequest запрос на обновление MCP сервера
type UpdateMCPServerRequest struct {
    DisplayName      *string      `json:"display_name"`
    Description      *string      `json:"description"`
    Category         *string      `json:"category"`
    IconURL          *string      `json:"icon_url"`
    RepositoryURL    *string      `json:"repository_url"`
    DocumentationURL *string      `json:"documentation_url"`
    NPMPackage       *string      `json:"npm_package"`
    InstallationCmd  *string      `json:"installation_command"`
    ConfigExample    *string      `json:"configuration_example"`
    Features         *MCPFeatures `json:"features"`
    Tags             []string     `json:"tags"`
    IsOfficial       *bool        `json:"is_official"`
    IsActive         *bool        `json:"is_active"`
}
```

#### 2. Database Storage Layer

**Файл:** `internal/storage/mcp.go`

```go
package storage

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
    "your-project/internal/models"
)

// MCPStorage интерфейс для работы с MCP серверами
type MCPStorage interface {
    // CreateMCPServer создает новый MCP сервер
    CreateMCPServer(ctx context.Context, server *models.MCPServer) error
    
    // GetMCPServer получает MCP сервер по ID
    GetMCPServer(ctx context.Context, id uuid.UUID) (*models.MCPServer, error)
    
    // GetMCPServerByName получает MCP сервер по имени
    GetMCPServerByName(ctx context.Context, name string) (*models.MCPServer, error)
    
    // ListMCPServers возвращает список MCP серверов с фильтрацией
    ListMCPServers(ctx context.Context, filter MCPServerFilter) ([]*models.MCPServer, error)
    
    // UpdateMCPServer обновляет MCP сервер
    UpdateMCPServer(ctx context.Context, id uuid.UUID, updates *models.UpdateMCPServerRequest) error
    
    // DeleteMCPServer удаляет MCP сервер
    DeleteMCPServer(ctx context.Context, id uuid.UUID) error
    
    // GetCategories возвращает список доступных категорий
    GetCategories(ctx context.Context) ([]string, error)
    
    // GetTags возвращает список всех используемых тегов
    GetTags(ctx context.Context) ([]string, error)
}

// MCPServerFilter фильтр для поиска MCP серверов
type MCPServerFilter struct {
    Category   *string
    Tags       []string
    Search     *string  // поиск по name, display_name, description
    IsOfficial *bool
    IsActive   *bool
    Limit      int
    Offset     int
}
```

#### 3. API Handlers

**Файл:** `internal/api/handlers/mcp.go`

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// MCPHandlers обработчики для MCP каталога
type MCPHandlers struct {
    storage storage.MCPStorage
}

// ListMCPServers возвращает список MCP серверов (PUBLIC)
// GET /api/mcp/servers
func (h *MCPHandlers) ListMCPServers(c *gin.Context) {
    filter := storage.MCPServerFilter{
        Category:   getStringParam(c, "category"),
        Tags:       c.QueryArray("tags"),
        Search:     getStringParam(c, "search"),
        IsActive:   boolPtr(true), // только активные для пользователей
        Limit:      getIntParam(c, "limit", 50),
        Offset:     getIntParam(c, "offset", 0),
    }
    
    servers, err := h.storage.ListMCPServers(c.Request.Context(), filter)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch servers"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "servers": servers,
        "count":   len(servers),
    })
}

// GetMCPServer возвращает детали MCP сервера (PUBLIC)
// GET /api/mcp/servers/:id
func (h *MCPHandlers) GetMCPServer(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
        return
    }
    
    server, err := h.storage.GetMCPServer(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
        return
    }
    
    c.JSON(http.StatusOK, server)
}

// CreateMCPServer создает новый MCP сервер (ADMIN ONLY)
// POST /api/admin/mcp/servers
func (h *MCPHandlers) CreateMCPServer(c *gin.Context) {
    var req models.CreateMCPServerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    userID := c.GetString("user_id") // из auth middleware
    uid, _ := uuid.Parse(userID)
    
    server := &models.MCPServer{
        ID:               uuid.New(),
        Name:             req.Name,
        DisplayName:      req.DisplayName,
        Description:      req.Description,
        Category:         req.Category,
        IconURL:          req.IconURL,
        RepositoryURL:    req.RepositoryURL,
        DocumentationURL: req.DocumentationURL,
        NPMPackage:       req.NPMPackage,
        InstallationCmd:  req.InstallationCmd,
        ConfigExample:    req.ConfigExample,
        Features:         req.Features,
        Tags:             req.Tags,
        IsOfficial:       req.IsOfficial,
        IsActive:         true,
        CreatedBy:        &uid,
    }
    
    if err := h.storage.CreateMCPServer(c.Request.Context(), server); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create server"})
        return
    }
    
    c.JSON(http.StatusCreated, server)
}

// UpdateMCPServer обновляет MCP сервер (ADMIN ONLY)
// PATCH /api/admin/mcp/servers/:id
func (h *MCPHandlers) UpdateMCPServer(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
        return
    }
    
    var req models.UpdateMCPServerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := h.storage.UpdateMCPServer(c.Request.Context(), id, &req); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update server"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Server updated successfully"})
}

// DeleteMCPServer удаляет MCP сервер (ADMIN ONLY)
// DELETE /api/admin/mcp/servers/:id
func (h *MCPHandlers) DeleteMCPServer(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
        return
    }
    
    if err := h.storage.DeleteMCPServer(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete server"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Server deleted successfully"})
}

// GetCategories возвращает список категорий (PUBLIC)
// GET /api/mcp/categories
func (h *MCPHandlers) GetCategories(c *gin.Context) {
    categories, err := h.storage.GetCategories(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"categories": categories})
}
```

#### 4. Router Configuration

**Файл:** `internal/api/router/router.go`

```go
// Public MCP endpoints
mcp := router.Group("/api/mcp")
{
    mcp.GET("/servers", mcpHandlers.ListMCPServers)
    mcp.GET("/servers/:id", mcpHandlers.GetMCPServer)
    mcp.GET("/categories", mcpHandlers.GetCategories)
}

// Admin MCP endpoints
adminMCP := router.Group("/api/admin/mcp")
adminMCP.Use(authMiddleware.RequireAdmin()) // только для админов
{
    adminMCP.POST("/servers", mcpHandlers.CreateMCPServer)
    adminMCP.PATCH("/servers/:id", mcpHandlers.UpdateMCPServer)
    adminMCP.DELETE("/servers/:id", mcpHandlers.DeleteMCPServer)
}
```

### Frontend (WebUI)

#### 1. Public MCP Catalog Page

**Файл:** `web/mcp-catalog.html`

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCP Servers Catalog</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link rel="stylesheet" href="/css/common.css">
</head>
<body>
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
        <div class="container-fluid">
            <a class="navbar-brand" href="/">Ollama Proxy</a>
            <div class="navbar-nav ms-auto">
                <a class="nav-link" href="/chat.html">Chat</a>
                <a class="nav-link" href="/dashboard.html">Dashboard</a>
                <a class="nav-link active" href="/mcp-catalog.html">MCP Catalog</a>
            </div>
        </div>
    </nav>

    <div class="container mt-4">
        <div class="row">
            <div class="col-12">
                <h2>MCP Servers Catalog</h2>
                <p class="text-muted">Browse available Model Context Protocol servers and integration tools</p>
            </div>
        </div>

        <!-- Filters -->
        <div class="row mb-4">
            <div class="col-md-4">
                <select id="categoryFilter" class="form-select">
                    <option value="">All Categories</option>
                </select>
            </div>
            <div class="col-md-4">
                <input type="text" id="searchInput" class="form-control" placeholder="Search servers...">
            </div>
            <div class="col-md-4">
                <div class="form-check">
                    <input class="form-check-input" type="checkbox" id="officialOnly">
                    <label class="form-check-label" for="officialOnly">
                        Official Only
                    </label>
                </div>
            </div>
        </div>

        <!-- Servers Grid -->
        <div id="serversGrid" class="row">
            <!-- Будет заполнено через JavaScript -->
        </div>
    </div>

    <!-- Server Details Modal -->
    <div class="modal fade" id="serverModal" tabindex="-1">
        <div class="modal-dialog modal-lg">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title" id="serverModalTitle"></h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                </div>
                <div class="modal-body" id="serverModalBody">
                    <!-- Детали сервера -->
                </div>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
    <script src="/js/mcp-catalog.js"></script>
</body>
</html>
```

#### 2. JavaScript для каталога

**Файл:** `web/js/mcp-catalog.js`

```javascript
let allServers = [];
let categories = [];

// Load catalog on page load
document.addEventListener('DOMContentLoaded', async () => {
    await loadCategories();
    await loadServers();
    setupFilters();
});

// Load categories
async function loadCategories() {
    try {
        const response = await fetch('/api/mcp/categories');
        const data = await response.json();
        categories = data.categories || [];
        
        const select = document.getElementById('categoryFilter');
        categories.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat;
            option.textContent = cat;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

// Load MCP servers
async function loadServers() {
    try {
        const response = await fetch('/api/mcp/servers?limit=100');
        const data = await response.json();
        allServers = data.servers || [];
        renderServers(allServers);
    } catch (error) {
        console.error('Failed to load servers:', error);
        showError('Failed to load MCP servers');
    }
}

// Render servers grid
function renderServers(servers) {
    const grid = document.getElementById('serversGrid');
    
    if (servers.length === 0) {
        grid.innerHTML = '<div class="col-12"><p class="text-center text-muted">No servers found</p></div>';
        return;
    }
    
    grid.innerHTML = servers.map(server => `
        <div class="col-md-6 col-lg-4 mb-4">
            <div class="card h-100 server-card" onclick="showServerDetails('${server.id}')">
                <div class="card-body">
                    <div class="d-flex align-items-start mb-2">
                        ${server.icon_url ? 
                            `<img src="${server.icon_url}" class="server-icon me-2" alt="${server.display_name}">` :
                            '<i class="fas fa-server fa-2x me-2 text-primary"></i>'
                        }
                        <div class="flex-grow-1">
                            <h5 class="card-title mb-1">
                                ${server.display_name}
                                ${server.is_official ? '<span class="badge bg-primary ms-1">Official</span>' : ''}
                            </h5>
                            <p class="text-muted small mb-0">${server.category}</p>
                        </div>
                    </div>
                    <p class="card-text">${truncate(server.description, 120)}</p>
                    <div class="mt-2">
                        ${server.tags.map(tag => `<span class="badge bg-secondary me-1">${tag}</span>`).join('')}
                    </div>
                </div>
                <div class="card-footer bg-transparent">
                    <small class="text-muted">
                        ${server.npm_package ? `<i class="fab fa-npm"></i> ${server.npm_package}` : ''}
                    </small>
                </div>
            </div>
        </div>
    `).join('');
}

// Show server details in modal
async function showServerDetails(serverId) {
    try {
        const response = await fetch(`/api/mcp/servers/${serverId}`);
        const server = await response.json();
        
        document.getElementById('serverModalTitle').textContent = server.display_name;
        document.getElementById('serverModalBody').innerHTML = `
            <div class="mb-3">
                <p>${server.description}</p>
            </div>
            
            ${server.repository_url ? `
            <div class="mb-3">
                <strong>Repository:</strong> 
                <a href="${server.repository_url}" target="_blank">${server.repository_url}</a>
            </div>
            ` : ''}
            
            ${server.documentation_url ? `
            <div class="mb-3">
                <strong>Documentation:</strong>
                <a href="${server.documentation_url}" target="_blank">${server.documentation_url}</a>
            </div>
            ` : ''}
            
            ${server.installation_command ? `
            <div class="mb-3">
                <strong>Installation:</strong>
                <pre class="bg-light p-2 rounded"><code>${server.installation_command}</code></pre>
                <button class="btn btn-sm btn-outline-primary" onclick="copyToClipboard('${server.installation_command}')">
                    <i class="fas fa-copy"></i> Copy
                </button>
            </div>
            ` : ''}
            
            ${server.configuration_example ? `
            <div class="mb-3">
                <strong>Configuration Example:</strong>
                <pre class="bg-light p-2 rounded"><code>${server.configuration_example}</code></pre>
            </div>
            ` : ''}
            
            <div class="mb-3">
                <strong>Features:</strong>
                <ul>
                    ${server.features.supports_streaming ? '<li>Supports Streaming</li>' : ''}
                    ${server.features.requires_auth ? '<li>Requires Authentication</li>' : ''}
                    ${server.features.platform ? `<li>Platform: ${server.features.platform.join(', ')}</li>` : ''}
                </ul>
            </div>
        `;
        
        new bootstrap.Modal(document.getElementById('serverModal')).show();
    } catch (error) {
        console.error('Failed to load server details:', error);
    }
}

// Setup filters
function setupFilters() {
    const categoryFilter = document.getElementById('categoryFilter');
    const searchInput = document.getElementById('searchInput');
    const officialOnly = document.getElementById('officialOnly');
    
    const applyFilters = () => {
        let filtered = allServers;
        
        // Category filter
        const category = categoryFilter.value;
        if (category) {
            filtered = filtered.filter(s => s.category === category);
        }
        
        // Search filter
        const search = searchInput.value.toLowerCase();
        if (search) {
            filtered = filtered.filter(s => 
                s.display_name.toLowerCase().includes(search) ||
                s.description.toLowerCase().includes(search) ||
                s.tags.some(tag => tag.toLowerCase().includes(search))
            );
        }
        
        // Official filter
        if (officialOnly.checked) {
            filtered = filtered.filter(s => s.is_official);
        }
        
        renderServers(filtered);
    };
    
    categoryFilter.addEventListener('change', applyFilters);
    searchInput.addEventListener('input', applyFilters);
    officialOnly.addEventListener('change', applyFilters);
}

// Utility functions
function truncate(text, length) {
    if (text.length <= length) return text;
    return text.substring(0, length) + '...';
}

async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('Copied to clipboard!', 'success');
    } catch (err) {
        console.error('Failed to copy:', err);
    }
}

function showToast(message, type = 'info') {
    // Реализовать toast notification
    alert(message); // Temporary
}

function showError(message) {
    const grid = document.getElementById('serversGrid');
    grid.innerHTML = `<div class="col-12"><div class="alert alert-danger">${message}</div></div>`;
}
```

#### 3. Admin MCP Management

**Файл:** `web/admin-mcp.html` (новая страница в админке)

Форма CRUD для управления MCP серверами (аналогично API keys management).

## 📝 Implementation Plan

### Phase 1: Database & Backend (3-4 часа)

1. **Database migration** (30 мин)
   - Создать таблицу mcp_servers
   - Добавить индексы
   - Seed с примерами

2. **Data models** (30 мин)
   - MCPServer struct
   - Request/Response models

3. **Storage layer** (1 час)
   - Реализовать MCPStorage interface
   - CRUD операции

4. **API handlers** (1-1.5 часа)
   - Public endpoints
   - Admin endpoints
   - Validation

### Phase 2: Frontend Public (2-3 часа)

1. **Catalog page** (1 час)
   - HTML структура
   - Grid layout

2. **JavaScript** (1 час)
   - Load & render servers
   - Filters
   - Modal details

3. **CSS styling** (30 мин)

### Phase 3: Admin Interface (1-1.5 часа)

1. **Admin CRUD page** (1 час)
   - Форма создания/редактирования
   - Список с управлением

## ✅ Acceptance Criteria

- [ ] Database schema создана с индексами
- [ ] API endpoints работают (public + admin)
- [ ] Public каталог отображает серверы
- [ ] Фильтрация по категории, поиск, official only работает
- [ ] Modal показывает детали сервера
- [ ] Admin может создавать/редактировать/удалять записи
- [ ] Копирование installation команд работает
- [ ] Responsive design на мобильных

## 🧪 Testing

- [ ] Backend unit tests
- [ ] API integration tests
- [ ] Frontend тестирование на разных браузерах
- [ ] Mobile responsiveness
- [ ] Admin CRUD операции

## 📚 References

- [Model Context Protocol Specification](https://modelcontextprotocol.io)
- [MCP Servers List](https://github.com/modelcontextprotocol/servers)

## 🔄 Future Enhancements

- Рейтинг серверов (звезды, отзывы)
- Community contributions
- Version tracking
- Compatibility matrix (какие модели поддерживают)
- Auto-install для популярных серверов
