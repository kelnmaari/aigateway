# PDF Text Extraction

Система поддерживает **кроссплатформенное извлечение текста из PDF** файлов с автоматическим выбором метода.

## 🔧 Поддерживаемые Методы

### 1. **Auto (Рекомендуется)** ✅
```yaml
extractors:
  pdf:
    method: "auto"
```

Автоматически выбирает лучший доступный метод:
- Если установлен `pdftotext` → использует его (наилучшее качество)
- Если `pdftotext` недоступен → использует чистую Go-библиотеку

**Преимущества:**
- ✅ Работает на всех платформах без дополнительной настройки
- ✅ Автоматический fallback
- ✅ Не требует внешних зависимостей

---

### 2. **PDFToText (Высокое качество)**
```yaml
extractors:
  pdf:
    method: "pdftotext"
```

Использует внешнюю утилиту `pdftotext` из пакета Poppler.

**Установка:**

#### Windows
```powershell
# Скачайте Poppler для Windows:
https://github.com/oschwartz10612/poppler-windows/releases/

# Добавьте в PATH:
C:\Program Files\poppler-24.08.0\Library\bin
```

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get install poppler-utils
```

#### Linux (CentOS/RHEL)
```bash
sudo yum install poppler-utils
```

#### macOS
```bash
brew install poppler
```

**Преимущества:**
- ✅ Наилучшее качество извлечения текста
- ✅ Поддержка сложных PDF
- ✅ Извлечение метаданных (автор, название)

**Недостатки:**
- ❌ Требует установки внешнего пакета
- ❌ Не работает в контейнерах без дополнительной настройки

---

### 3. **Go-PDF (Pure Go, всегда работает)**
```yaml
extractors:
  pdf:
    method: "go-pdf"
```

Использует чистую Go-библиотеку `github.com/ledongthuc/pdf`.

**Преимущества:**
- ✅ Не требует внешних зависимостей
- ✅ Работает на всех платформах
- ✅ Работает в Docker контейнерах
- ✅ Простое развертывание

**Недостатки:**
- ⚠️ Может хуже работать со сложными PDF
- ⚠️ Ограниченное извлечение метаданных

---

## 🐳 Docker/Kubernetes

Для production-среды рекомендуется использовать `method: "auto"` или `method: "go-pdf"`.

Если нужен `pdftotext` в Docker:

```dockerfile
# Dockerfile
FROM golang:1.25-alpine

# Установка Poppler
RUN apk add --no-cache poppler-utils

COPY . /app
WORKDIR /app
RUN go build -o server cmd/server/main.go

CMD ["./server"]
```

---

## 📊 Сравнение методов

| Метод       | Качество | Скорость | Внешние зависимости | Docker | Рекомендация |
|-------------|----------|----------|---------------------|--------|--------------|
| **auto**    | 🟢 Высокое | 🟢 Быстро | ❌ Нет | ✅ Да | ✅ **Лучший выбор** |
| pdftotext   | 🟢 Отличное | 🟢 Очень быстро | ✅ Да (Poppler) | ⚠️ Требует установки | Для сложных PDF |
| go-pdf      | 🟡 Хорошее | 🟡 Средне | ❌ Нет | ✅ Да | Для контейнеров |

---

## 🔍 Как это работает

### Метод "auto" (по умолчанию)

```
1. Пользователь загружает PDF
   ↓
2. Система проверяет доступность pdftotext
   ↓
3a. Если pdftotext найден:
    → Извлекает текст с помощью pdftotext (высокое качество)
   
3b. Если pdftotext НЕ найден:
    → Использует Go-библиотеку (работает всегда)
   
   ↓
4. Возвращает извлеченный текст
```

### Fallback механизм

Если выбран метод `pdftotext`, но утилита недоступна:
- Система **автоматически** переключится на `go-pdf`
- Извлечение текста продолжится **без ошибок**
- В логах будет предупреждение о fallback

---

## 🧪 Тестирование

```bash
# Проверка доступности pdftotext
pdftotext -v

# Если установлен:
pdftotext (poppler) 24.08.0

# Если НЕ установлен:
command not found: pdftotext
```

Система автоматически определит доступность и выберет правильный метод.

---

## ⚙️ Конфигурация

```yaml
extractors:
  pdf:
    method: "auto"              # "auto", "pdftotext", "go-pdf"
    preserve_layout: false      # Сохранять layout (только pdftotext)
    extract_images: false       # Извлечение изображений (future)
    ocr_enabled: false          # OCR для отсканированных PDF (future)
    max_pages: 1000            # Максимум страниц для извлечения
  
  timeout: "30s"               # Таймаут для всех extractors
  parallel_workers: 4          # Параллельная обработка
```

---

## 🚀 Рекомендации

### Development (Windows/macOS/Linux)
```yaml
method: "auto"  # Автоматический выбор
```
- Установите Poppler для лучшего качества (опционально)
- Без Poppler система будет использовать Go-библиотеку

### Production (Docker/Kubernetes)
```yaml
method: "auto"  # или "go-pdf"
```
- Не требует установки внешних зависимостей
- Работает из коробки в любой среде
- Упрощает CI/CD

### High-Quality Extraction
```yaml
method: "pdftotext"
```
- Установите Poppler в системе/контейнере
- Лучшее качество для сложных PDF с таблицами, колонками

---

## 📝 Примеры использования

### Загрузка PDF через API

```bash
# Загрузка PDF файла
curl -X POST http://localhost:8080/api/files/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf"

# Ответ:
{
  "id": "file_123",
  "filename": "document.pdf",
  "extraction_status": "completed",  # или "pending", "failed"
  "metadata": {
    "extraction_method": "pdftotext",  # или "go-pdf"
    "page_count": 5,
    "word_count": 1234
  }
}
```

### Получение извлеченного текста

```bash
curl http://localhost:8080/api/files/file_123/text \
  -H "Authorization: Bearer YOUR_TOKEN"

# Ответ:
{
  "text": "Extracted PDF content...",
  "word_count": 1234,
  "page_count": 5,
  "language": "ru"
}
```

---

## ❓ FAQ

**Q: Нужно ли устанавливать Poppler?**  
A: Нет! Система работает без внешних зависимостей благодаря методу `auto`.

**Q: Какой метод использовать в Docker?**  
A: `auto` или `go-pdf` - они работают без дополнительной настройки.

**Q: Как улучшить качество извлечения?**  
A: Установите Poppler и используйте `method: "pdftotext"` или `method: "auto"`.

**Q: Что если pdftotext установлен, но система его не видит?**  
A: Добавьте путь к pdftotext в переменную окружения `PATH`.

**Q: Поддерживаются ли отсканированные PDF (OCR)?**  
A: В текущей версии нет. OCR планируется в версии 1.10.2 (IMAGE-01).

---

## 🔗 Связанные возможности

- **FILE-STORAGE-01**: Базовая система хранения файлов
- **IMAGE-01** (v1.10.2): OCR для отсканированных PDF и изображений
- **RAG-04** (v1.13.0): Поиск по содержимому документов

---

**Вопросы?** Откройте issue в репозитории или обратитесь в поддержку.

