# WEBUI-06: Model Name Copy Button

**Версия:** v1.4.1  
**Приоритет:** MEDIUM  
**Оценка:** 1-2 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned

---

## 📋 Описание

Добавить кнопку копирования названия модели в буфер обмена во всех списках моделей в WebUI.

## 🎯 Цели

1. Упростить копирование названий моделей для использования в API
2. Улучшить UX при работе с длинными названиями моделей
3. Визуальный feedback при успешном копировании

## 📊 Scope

### Frontend изменения

#### 1. Model List Components

**Файлы для изменения:**

- `web/admin.html` - System → Models раздел
- `web/chat.html` - Model selector
- Любые другие списки моделей

**Требования:**

- Кнопка с иконкой копирования рядом с названием модели
- Иконка: `📋` или Font Awesome `fa-copy`
- Tooltip: "Copy model name"
- При hover: подсветка кнопки

#### 2. Clipboard Integration

```javascript
// Функция копирования в буфер обмена
async function copyModelName(modelName) {
    try {
        await navigator.clipboard.writeText(modelName);
        showCopySuccess();
    } catch (err) {
        // Fallback для старых браузеров
        fallbackCopyToClipboard(modelName);
    }
}

// Fallback метод
function fallbackCopyToClipboard(text) {
    const textArea = document.createElement("textarea");
    textArea.value = text;
    textArea.style.position = "fixed";
    textArea.style.left = "-999999px";
    document.body.appendChild(textArea);
    textArea.select();
    try {
        document.execCommand('copy');
        showCopySuccess();
    } catch (err) {
        showCopyError();
    }
    document.body.removeChild(textArea);
}
```

#### 3. Visual Feedback

**Success Toast/Tooltip:**

- Показывать на 2 секунды
- Текст: "✓ Copied: {modelName}"
- Зеленый цвет
- Позиция: рядом с кнопкой или top-right угол

**Error Toast:**

- Текст: "✗ Failed to copy"
- Красный цвет

**Варианты реализации:**

1. **Bootstrap Toast** (если уже используется)
2. **Custom CSS tooltip** (легковесный вариант)
3. **Temporary icon change** (✓ вместо 📋 на 2 сек)

### UI/UX Requirements

#### Layout Options

**Вариант 1: Inline Button**

```
[Model Name]  [📋]
llama3.2:latest  [📋]
```

**Вариант 2: Hover Button**

```
[Model Name]          (на hover появляется [📋])
```

**Вариант 3: Icon + Text Button** (для админки)

```
[Model Name]  [📋 Copy]
```

#### Accessibility

- ARIA label: `aria-label="Copy model name"`
- Keyboard support: Enter/Space для копирования
- Tab navigation через кнопки
- Screen reader friendly

## 🔧 Технические детали

### Browser Compatibility

**Clipboard API поддержка:**

- Chrome 66+
- Firefox 63+
- Safari 13.1+
- Edge 79+

**Fallback для:**

- IE 11 (document.execCommand)
- Старые версии браузеров

### Security Considerations

- Clipboard API требует HTTPS (или localhost)
- Нужно проверять permissions (если требуется)
- Graceful fallback для неподдерживаемых браузеров

## 📝 Implementation Plan

### Step 1: Create Copy Utility (15 мин)

Создать `web/js/utils/clipboard.js`:

```javascript
// Clipboard utility
const ClipboardUtils = {
    async copy(text, options = {}) {
        try {
            await navigator.clipboard.writeText(text);
            if (options.onSuccess) options.onSuccess(text);
            return true;
        } catch (err) {
            console.warn('Clipboard API failed, using fallback:', err);
            return this.fallbackCopy(text, options);
        }
    },
    
    fallbackCopy(text, options = {}) {
        const textArea = document.createElement("textarea");
        textArea.value = text;
        textArea.style.cssText = "position:fixed;left:-9999px";
        document.body.appendChild(textArea);
        textArea.select();
        
        let success = false;
        try {
            success = document.execCommand('copy');
            if (success && options.onSuccess) {
                options.onSuccess(text);
            }
        } catch (err) {
            if (options.onError) options.onError(err);
        } finally {
            document.body.removeChild(textArea);
        }
        return success;
    },
    
    showToast(message, type = 'success') {
        // Создать временный toast
        const toast = document.createElement('div');
        toast.className = `copy-toast copy-toast-${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);
        
        setTimeout(() => toast.classList.add('show'), 10);
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => document.body.removeChild(toast), 300);
        }, 2000);
    }
};
```

### Step 2: Update Admin Models List (30 мин)

В `web/admin.html` и `web/js/admin.js`:

```javascript
function renderModelsList(models) {
    return models.map(model => `
        <div class="model-item">
            <span class="model-name">${model.name}</span>
            <button 
                class="btn btn-sm btn-outline-secondary copy-model-btn"
                onclick="copyModelName('${model.name}')"
                aria-label="Copy model name"
                title="Copy model name">
                <i class="fas fa-copy"></i>
            </button>
        </div>
    `).join('');
}

function copyModelName(modelName) {
    ClipboardUtils.copy(modelName, {
        onSuccess: () => {
            ClipboardUtils.showToast(`✓ Copied: ${modelName}`, 'success');
        },
        onError: (err) => {
            ClipboardUtils.showToast('✗ Failed to copy', 'error');
            console.error('Copy failed:', err);
        }
    });
}
```

### Step 3: Add CSS Styles (15 мин)

В соответствующий CSS файл:

```css
/* Copy button styles */
.copy-model-btn {
    padding: 2px 8px;
    margin-left: 8px;
    border: 1px solid #dee2e6;
    background: transparent;
    cursor: pointer;
    transition: all 0.2s;
}

.copy-model-btn:hover {
    background: #e9ecef;
    border-color: #adb5bd;
}

.copy-model-btn:active {
    transform: scale(0.95);
}

/* Toast styles */
.copy-toast {
    position: fixed;
    top: 20px;
    right: 20px;
    padding: 12px 20px;
    border-radius: 4px;
    font-size: 14px;
    opacity: 0;
    transform: translateY(-20px);
    transition: all 0.3s;
    z-index: 9999;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.copy-toast.show {
    opacity: 1;
    transform: translateY(0);
}

.copy-toast-success {
    background: #28a745;
    color: white;
}

.copy-toast-error {
    background: #dc3545;
    color: white;
}
```

### Step 4: Update Chat Model Selector (15 мин)

В `web/chat.html` - добавить копирование в model selector dropdown

### Step 5: Testing (15 мин)

- Тестирование на разных браузерах
- Проверка keyboard navigation
- Проверка fallback для старых браузеров

## ✅ Acceptance Criteria

- [ ] Кнопка копирования присутствует во всех списках моделей
- [ ] Clipboard API работает корректно
- [ ] Fallback работает в старых браузерах
- [ ] Visual feedback при успешном/неуспешном копировании
- [ ] Keyboard accessible (Tab + Enter/Space)
- [ ] ARIA labels для accessibility
- [ ] Работает на HTTPS и localhost
- [ ] CSS не конфликтует с существующими стилями

## 🧪 Testing Checklist

### Functional Testing

- [ ] Копирование в Chrome/Edge
- [ ] Копирование в Firefox
- [ ] Копирование в Safari
- [ ] Fallback в IE11 (если требуется)
- [ ] Toast появляется и исчезает
- [ ] Корректное отображение на мобильных

### Integration Testing

- [ ] Работает на всех страницах с моделями
- [ ] Не ломает существующий функционал
- [ ] CSS не конфликтует

### Accessibility Testing

- [ ] Keyboard navigation
- [ ] Screen reader compatibility
- [ ] ARIA labels корректны

## 📚 References

- [Clipboard API MDN](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API)
- [navigator.clipboard.writeText()](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText)
- [Bootstrap Toasts](https://getbootstrap.com/docs/5.0/components/toasts/)

## 🔄 Follow-up Tasks

- Рассмотреть добавление bulk copy (копирование всех названий)
- Добавить copy для других элементов (API keys с маскированием)
- Сохранить историю скопированных моделей (опционально)
