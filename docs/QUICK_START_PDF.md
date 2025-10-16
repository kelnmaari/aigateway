# 🚀 Quick Start: PDF Extraction

## Проблема решена! ✅

**До:** PDF работал только на Windows (требовал Poppler)  
**Сейчас:** PDF работает на **всех платформах** без внешних зависимостей!

---

## Что изменилось?

### 1. Добавлена Go-библиотека для PDF
```bash
go get github.com/ledongthuc/pdf
```

### 2. Автоматический выбор метода
```yaml
# configs/dev.yaml
extractors:
  pdf:
    method: "auto"  # ← Автоматически выбирает лучший метод
```

### 3. Три метода извлечения

| Метод | Описание | Когда использовать |
|-------|----------|-------------------|
| `auto` | Автоматический выбор | ✅ **По умолчанию** (рекомендуется) |
| `pdftotext` | Внешний Poppler | Если нужно высокое качество |
| `go-pdf` | Чистый Go | Docker, контейнеры |

---

## Как это работает?

```
Загрузка PDF → Проверка pdftotext → Извлечение текста
                        ↓
              Есть? → pdftotext (высокое качество)
              Нет?  → go-pdf (работает всегда)
```

**Результат:** PDF работает на любой платформе! 🎉

---

## Установка Poppler (опционально)

### ✅ У вас уже установлен (Windows)
```powershell
pdftotext -v
# pdftotext (poppler) 24.08.0
```

### 🐧 Linux (для улучшения качества)
```bash
# Ubuntu/Debian
sudo apt-get install poppler-utils

# CentOS/RHEL
sudo yum install poppler-utils

# Arch
sudo pacman -S poppler
```

### 🍎 macOS
```bash
brew install poppler
```

### 🐳 Docker
```dockerfile
# Dockerfile
FROM golang:1.25-alpine
RUN apk add --no-cache poppler-utils  # Опционально
COPY . /app
WORKDIR /app
CMD ["./server"]
```

---

## Тестирование

### 1. Без Poppler (работает везде)
```bash
# Убедитесь что pdftotext недоступен
pdftotext -v
# command not found

# Система автоматически использует go-pdf
./server
# ✅ PDF extraction работает!
```

### 2. С Poppler (высокое качество)
```bash
# Установите Poppler
pdftotext -v
# pdftotext (poppler) 24.08.0

# Система автоматически использует pdftotext
./server
# ✅ PDF extraction работает с высоким качеством!
```

---

## Примеры использования

### Загрузка PDF
```bash
curl -X POST http://localhost:8080/api/files/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf"
```

**Ответ:**
```json
{
  "id": "file_123",
  "filename": "document.pdf",
  "extraction_status": "completed",
  "metadata": {
    "extraction_method": "go-pdf",  // или "pdftotext"
    "page_count": 10,
    "word_count": 2500
  }
}
```

### Получение текста
```bash
curl http://localhost:8080/api/files/file_123/text \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Ответ:**
```json
{
  "text": "Извлеченный текст из PDF...",
  "word_count": 2500,
  "page_count": 10,
  "language": "ru"
}
```

---

## ⚙️ Конфигурация

### Рекомендуемая (по умолчанию)
```yaml
extractors:
  pdf:
    method: "auto"              # Автоматический выбор
    preserve_layout: false
    max_pages: 1000
```

### Только Go-библиотека (Docker)
```yaml
extractors:
  pdf:
    method: "go-pdf"            # Всегда используй Go
    max_pages: 1000
```

### Только Poppler (высокое качество)
```yaml
extractors:
  pdf:
    method: "pdftotext"         # Требует установки Poppler
    preserve_layout: true       # Сохранять форматирование
    max_pages: 1000
```

---

## 🎯 Результаты

✅ **Windows**: Работает (с или без Poppler)  
✅ **Linux**: Работает (с или без Poppler)  
✅ **macOS**: Работает (с или без Poppler)  
✅ **Docker**: Работает (без дополнительных зависимостей)  
✅ **Kubernetes**: Работает (без дополнительных зависимостей)  

---

## 📚 Дополнительная информация

Полная документация: [docs/PDF_EXTRACTION.md](PDF_EXTRACTION.md)

---

**Готово!** PDF extraction теперь работает кроссплатформенно! 🚀

