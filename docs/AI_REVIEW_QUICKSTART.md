# AI Review - Быстрый старт

## 3 шага до первого AI Review

### Шаг 1: Создайте API ключ
```bash
# Откройте WebUI вашего Ollama-OpenAI Proxy
http://your-proxy-host:8080

# Создайте ключ:
# → API Keys → Create
# → Name: gitlab-ai-review
# → Models: llama3.2:latest
# → Copy key
```

### Шаг 2: Добавьте в GitLab CI Variables
```bash
GitLab → Settings → CI/CD → Variables → Add Variable

Key:   OLLAMA_PROXY_API_KEY
Value: (вставьте скопированный ключ)
Flags: ✓ Mask variable
```

### Шаг 3: Отредактируйте `.gitlab-ci.yml`
```yaml
variables:
  # Замените на URL вашего proxy
  LLM__HTTP_CLIENT__API_URL: "http://YOUR_PROXY_HOST:8080/v1"
  
  # Замените на вашу модель
  LLM__META__MODEL: "llama3.2:latest"
```

---

## Запуск

1. Создайте Merge Request
2. Pipelines → ▶️ Play на `ai-review:full`
3. Дождитесь комментариев AI

---

## Проверка доступности proxy

```bash
# Должен вернуть список моделей
curl http://YOUR_PROXY_HOST:8080/v1/models \
  -H "Authorization: Bearer $OLLAMA_PROXY_API_KEY"
```

---

## Рекомендуемые модели

| Модель | Использование |
|--------|---------------|
| `qwen2.5-coder:1.5b` | Тестирование, быстрые проверки |
| `qwen2.5-coder:7b` | Production CI/CD |
| `deepseek-coder:33b` | Критичный код, final review |

---

## Troubleshooting

**Connection refused**: Проверьте URL и firewall  
**401 Unauthorized**: Проверьте API ключ в GitLab Variables  
**Model not found**: Загрузите модель: `ollama pull llama3.2`  
**Timeout**: Используйте меньшую модель или увеличьте timeout

---

Подробная документация: `docs/AI_REVIEW_SETUP.md`

