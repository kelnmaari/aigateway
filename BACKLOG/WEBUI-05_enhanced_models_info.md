# WEBUI-05: Enhanced Models Information

**Версия:** v1.4.4  
**Приоритет:** MEDIUM  
**Оценка:** 8-10 часов  
**Зависимости:** WEBUI-06 (опционально, для copy button)  
**Статус:** 📋 Planned

---

## 📋 Описание

Расширенная информация о моделях в админке с раскрывающимися параметрами (accordion). Показывать полную информацию о каждой модели: размер, семейство, формат, параметры и другие детали.

## 🎯 Цели

1. Предоставить детальную информацию о моделях для администраторов
2. Улучшить визуализацию параметров моделей
3. Упростить выбор моделей на основе характеристик
4. Показать реальные данные от Ollama API

## 📊 Scope

### Backend (API Endpoints)

#### 1. Enhanced Model Info Endpoint

**Новый endpoint:** `GET /api/admin/models/:name/details`

**Response Example:**

```json
{
  "name": "llama3.2:latest",
  "model": "llama3.2:latest",
  "modified_at": "2024-10-10T12:34:56Z",
  "size": 4661224960,
  "size_formatted": "4.34 GB",
  "digest": "sha256:a80c4f17acd5...",
  "details": {
    "parent_model": "",
    "format": "gguf",
    "family": "llama",
    "families": ["llama"],
    "parameter_size": "3.2B",
    "quantization_level": "Q4_0"
  },
  "parameters": {
    "num_ctx": 2048,
    "num_batch": 512,
    "num_gqa": 8,
    "num_gpu": 1,
    "num_thread": 8,
    "rope_frequency_base": 10000,
    "rope_frequency_scale": 1
  },
  "template": "{{ if .System }}System: {{ .System }}\n{{ end }}...",
  "system": "You are a helpful assistant.",
  "license": "MIT",
  "modelfile": "FROM llama3.2:latest\n...",
  "projector_info": null
}
```

#### 2. Backend Handler

**Файл:** `internal/api/handlers/admin.go`

```go
// GetModelDetails возвращает детальную информацию о модели
func (h *AdminHandlers) GetModelDetails(c *gin.Context) {
    modelName := c.Param("name")
    
    // Получить информацию от Ollama
    ollamaURL := h.config.Ollama.URL
    resp, err := http.Get(fmt.Sprintf("%s/api/show", ollamaURL), map[string]string{
        "name": modelName,
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to fetch model details",
            "details": err.Error(),
        })
        return
    }
    
    var modelInfo OllamaModelInfo
    if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to parse model details",
        })
        return
    }
    
    // Форматировать размер
    modelInfo.SizeFormatted = formatBytes(modelInfo.Size)
    
    c.JSON(http.StatusOK, modelInfo)
}

// formatBytes конвертирует байты в читаемый формат
func formatBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

#### 3. Data Models

**Файл:** `internal/models/ollama.go`

```go
// OllamaModelInfo детальная информация о модели
type OllamaModelInfo struct {
    Name           string              `json:"name"`
    Model          string              `json:"model"`
    ModifiedAt     time.Time           `json:"modified_at"`
    Size           int64               `json:"size"`
    SizeFormatted  string              `json:"size_formatted"`
    Digest         string              `json:"digest"`
    Details        ModelDetails        `json:"details"`
    Parameters     map[string]any      `json:"parameters,omitempty"`
    Template       string              `json:"template,omitempty"`
    System         string              `json:"system,omitempty"`
    License        string              `json:"license,omitempty"`
    Modelfile      string              `json:"modelfile,omitempty"`
    ProjectorInfo  *ProjectorInfo      `json:"projector_info,omitempty"`
}

// ModelDetails детали модели
type ModelDetails struct {
    ParentModel       string   `json:"parent_model"`
    Format            string   `json:"format"`
    Family            string   `json:"family"`
    Families          []string `json:"families"`
    ParameterSize     string   `json:"parameter_size"`
    QuantizationLevel string   `json:"quantization_level"`
}

// ProjectorInfo информация о projector (для multimodal моделей)
type ProjectorInfo struct {
    // Добавить поля при необходимости
}
```

#### 4. Router Integration

**Файл:** `internal/api/router/router.go`

```go
// В setupAdminRoutes добавить:
admin.GET("/models/:name/details", adminHandlers.GetModelDetails)
```

### Frontend (WebUI)

#### 1. Admin Models Page Enhancement

**Файл:** `web/admin.html`

Заменить текущий список моделей на accordion:

```html
<div class="card">
    <div class="card-header">
        <h5 class="mb-0">Models</h5>
    </div>
    <div class="card-body">
        <div id="modelsAccordion" class="accordion">
            <!-- Будет заполнено через JavaScript -->
        </div>
    </div>
</div>
```

#### 2. JavaScript Implementation

**Файл:** `web/js/admin.js`

```javascript
// Загрузка списка моделей
async function loadModels() {
    try {
        const response = await fetch('/api/admin/models', {
            headers: { 'Authorization': `Bearer ${getToken()}` }
        });
        const models = await response.json();
        renderModelsAccordion(models);
    } catch (error) {
        console.error('Failed to load models:', error);
        showError('Failed to load models');
    }
}

// Рендеринг accordion с моделями
function renderModelsAccordion(models) {
    const accordion = document.getElementById('modelsAccordion');
    accordion.innerHTML = models.map((model, index) => `
        <div class="accordion-item">
            <h2 class="accordion-header" id="heading-${index}">
                <button 
                    class="accordion-button ${index !== 0 ? 'collapsed' : ''}" 
                    type="button" 
                    data-bs-toggle="collapse" 
                    data-bs-target="#collapse-${index}"
                    onclick="loadModelDetails('${model.name}', ${index})">
                    <div class="d-flex justify-content-between align-items-center w-100">
                        <span>
                            <strong>${model.name}</strong>
                            <small class="text-muted ms-2">${formatBytes(model.size)}</small>
                        </span>
                        <button 
                            class="btn btn-sm btn-outline-secondary me-2"
                            onclick="event.stopPropagation(); copyModelName('${model.name}')"
                            title="Copy model name">
                            <i class="fas fa-copy"></i>
                        </button>
                    </div>
                </button>
            </h2>
            <div 
                id="collapse-${index}" 
                class="accordion-collapse collapse ${index === 0 ? 'show' : ''}"
                data-bs-parent="#modelsAccordion">
                <div class="accordion-body">
                    <div id="model-details-${index}" class="model-details-container">
                        <div class="text-center">
                            <div class="spinner-border spinner-border-sm" role="status">
                                <span class="visually-hidden">Loading...</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

// Загрузка деталей модели
async function loadModelDetails(modelName, index) {
    const container = document.getElementById(`model-details-${index}`);
    
    // Если уже загружено, не загружать повторно
    if (container.dataset.loaded === 'true') return;
    
    try {
        const response = await fetch(`/api/admin/models/${encodeURIComponent(modelName)}/details`, {
            headers: { 'Authorization': `Bearer ${getToken()}` }
        });
        const details = await response.json();
        
        container.innerHTML = renderModelDetails(details);
        container.dataset.loaded = 'true';
    } catch (error) {
        console.error('Failed to load model details:', error);
        container.innerHTML = '<div class="alert alert-danger">Failed to load model details</div>';
    }
}

// Рендеринг деталей модели
function renderModelDetails(details) {
    return `
        <div class="row">
            <div class="col-md-6">
                <h6>General Information</h6>
                <table class="table table-sm">
                    <tbody>
                        <tr>
                            <td><strong>Name:</strong></td>
                            <td>${details.name}</td>
                        </tr>
                        <tr>
                            <td><strong>Size:</strong></td>
                            <td>${details.size_formatted}</td>
                        </tr>
                        <tr>
                            <td><strong>Modified:</strong></td>
                            <td>${formatDate(details.modified_at)}</td>
                        </tr>
                        <tr>
                            <td><strong>Format:</strong></td>
                            <td>${details.details.format || 'N/A'}</td>
                        </tr>
                        <tr>
                            <td><strong>Family:</strong></td>
                            <td>${details.details.family || 'N/A'}</td>
                        </tr>
                        <tr>
                            <td><strong>Parameter Size:</strong></td>
                            <td>${details.details.parameter_size || 'N/A'}</td>
                        </tr>
                        <tr>
                            <td><strong>Quantization:</strong></td>
                            <td>${details.details.quantization_level || 'N/A'}</td>
                        </tr>
                    </tbody>
                </table>
            </div>
            
            <div class="col-md-6">
                <h6>Parameters</h6>
                <table class="table table-sm">
                    <tbody>
                        ${renderParameters(details.parameters)}
                    </tbody>
                </table>
            </div>
        </div>
        
        ${details.system ? `
        <div class="mt-3">
            <h6>System Prompt</h6>
            <pre class="bg-light p-2 rounded"><code>${escapeHtml(details.system)}</code></pre>
        </div>
        ` : ''}
        
        ${details.template ? `
        <div class="mt-3">
            <h6>Template</h6>
            <pre class="bg-light p-2 rounded"><code>${escapeHtml(details.template)}</code></pre>
        </div>
        ` : ''}
        
        ${details.modelfile ? `
        <div class="mt-3">
            <h6>Modelfile</h6>
            <button class="btn btn-sm btn-outline-primary mb-2" onclick="toggleModelfile(this)">
                <i class="fas fa-eye"></i> Show Modelfile
            </button>
            <pre class="bg-light p-2 rounded modelfile-content" style="display: none;"><code>${escapeHtml(details.modelfile)}</code></pre>
        </div>
        ` : ''}
        
        <div class="mt-3">
            <h6>Digest</h6>
            <small class="text-muted font-monospace">${details.digest}</small>
        </div>
    `;
}

// Рендеринг параметров
function renderParameters(params) {
    if (!params || Object.keys(params).length === 0) {
        return '<tr><td colspan="2">No parameters available</td></tr>';
    }
    
    return Object.entries(params).map(([key, value]) => `
        <tr>
            <td><strong>${key}:</strong></td>
            <td>${value}</td>
        </tr>
    `).join('');
}

// Toggle Modelfile visibility
function toggleModelfile(button) {
    const content = button.nextElementSibling;
    const icon = button.querySelector('i');
    
    if (content.style.display === 'none') {
        content.style.display = 'block';
        icon.className = 'fas fa-eye-slash';
        button.innerHTML = '<i class="fas fa-eye-slash"></i> Hide Modelfile';
    } else {
        content.style.display = 'none';
        icon.className = 'fas fa-eye';
        button.innerHTML = '<i class="fas fa-eye"></i> Show Modelfile';
    }
}

// Utility functions
function formatBytes(bytes) {
    if (!bytes) return 'N/A';
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) return '0 B';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
}

function formatDate(dateString) {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleString();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
```

#### 3. CSS Styles

**Файл:** `web/css/admin.css` (или inline в `web/admin.html`)

```css
/* Model details accordion */
.accordion-button {
    background-color: #f8f9fa;
}

.accordion-button:not(.collapsed) {
    background-color: #e9ecef;
    color: #000;
}

.model-details-container {
    min-height: 100px;
}

.model-details-container table {
    margin-bottom: 0;
}

.model-details-container pre {
    max-height: 300px;
    overflow-y: auto;
    font-size: 0.875rem;
}

.modelfile-content {
    max-height: 400px;
    overflow-y: auto;
}

/* Responsive adjustments */
@media (max-width: 768px) {
    .accordion-button .d-flex {
        flex-direction: column;
        align-items: flex-start !important;
    }
    
    .accordion-button .btn {
        margin-top: 8px;
    }
}
```

## 📝 Implementation Plan

### Phase 1: Backend (3-4 часа)

1. **Создать data models** (30 мин)
   - OllamaModelInfo struct
   - ModelDetails struct
   - ProjectorInfo struct

2. **Реализовать API endpoint** (1.5 часа)
   - GetModelDetails handler
   - Интеграция с Ollama API
   - Форматирование данных

3. **Добавить в router** (15 мин)
   - Зарегистрировать endpoint
   - Добавить auth middleware

4. **Тестирование backend** (1 час)
   - Unit tests для handler
   - Integration tests с mock Ollama

### Phase 2: Frontend (4-5 часов)

1. **HTML структура** (30 мин)
   - Создать accordion разметку
   - Добавить placeholder элементы

2. **JavaScript логика** (2 часа)
   - Функции загрузки моделей
   - Рендеринг accordion
   - Lazy loading деталей
   - Обработка ошибок

3. **CSS стилизация** (1 час)
   - Стили accordion
   - Responsive design
   - Анимации и transitions

4. **Интеграция с WEBUI-06** (30 мин)
   - Добавить copy buttons
   - Обеспечить совместимость

5. **Тестирование frontend** (1 час)
   - Тестирование на разных браузерах
   - Mobile responsiveness
   - Edge cases

### Phase 3: Integration & Polish (1 час)

1. **End-to-end тестирование** (30 мин)
2. **Bug fixes** (20 мин)
3. **Documentation** (10 мин)

## ✅ Acceptance Criteria

### Backend

- [ ] Endpoint `/api/admin/models/:name/details` возвращает полную информацию
- [ ] Размер форматируется в читаемый вид (GB, MB, etc.)
- [ ] Корректная обработка ошибок
- [ ] Unit tests покрывают >90%

### Frontend

- [ ] Accordion отображает все модели
- [ ] Детали загружаются по требованию (lazy loading)
- [ ] Показываются все параметры модели
- [ ] System prompt, template, modelfile отображаются корректно
- [ ] Copy button работает (если WEBUI-06 реализован)
- [ ] Responsive design на мобильных
- [ ] Обработка ошибок с user-friendly сообщениями

### UX

- [ ] Плавная анимация открытия/закрытия
- [ ] Loading indicators при загрузке
- [ ] Нет задержек UI при переключении
- [ ] Keyboard navigation работает

## 🧪 Testing Checklist

### Backend Testing

- [ ] Успешное получение деталей модели
- [ ] Обработка несуществующей модели (404)
- [ ] Обработка ошибки Ollama сервера
- [ ] Форматирование размеров корректно
- [ ] Auth middleware применяется

### Frontend Testing

- [ ] Accordion открывается/закрывается
- [ ] Lazy loading работает
- [ ] Детали отображаются корректно
- [ ] Error handling показывает сообщения
- [ ] Работает на Chrome, Firefox, Safari
- [ ] Mobile responsive

### Integration Testing

- [ ] End-to-end flow работает
- [ ] Нет race conditions при быстром клике
- [ ] Memory leaks отсутствуют

## 📚 References

- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md#show-model-information)
- [Bootstrap Accordion](https://getbootstrap.com/docs/5.0/components/accordion/)
- [Lazy Loading Patterns](https://web.dev/lazy-loading/)

## 🔄 Follow-up Tasks

- Добавить фильтрацию моделей (по семейству, размеру)
- Сортировка моделей (по имени, размеру, дате)
- Export деталей моделей в JSON/CSV
- Сравнение нескольких моделей side-by-side
- Model tags/labels management
