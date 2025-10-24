# AI Review - Быстрый старт

## 4 шага до первого AI Review

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

### Шаг 2: Создайте GitLab Access Token
```bash
# Project Access Token (рекомендуется):
GitLab Project → Settings → Access Tokens → Add new token

Name:   ai-review-token
Role:   Developer (минимум)
Scopes: ✓ api
        ✓ read_api
        ✓ write_repository

→ Create token
→ Copy token
```

### Шаг 3: Добавьте переменные в GitLab CI Variables
```bash
GitLab → Settings → CI/CD → Variables

# 1) API ключ для Ollama-OpenAI Proxy
Key:   OLLAMA_PROXY_API_KEY
Value: (ключ из WebUI proxy)
Flags: ✓ Mask variable

# 2) GitLab Access Token
Key:   GITLAB_API_TOKEN
Value: (токен из шага 2)
Flags: ✓ Mask variable
       ✓ Protect variable
```

### Шаг 4: Отредактируйте `.gitlab-ci.yml`
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

**401 Unauthorized (GitLab API)**: Создайте Project Access Token вместо CI_JOB_TOKEN  
**Connection refused**: Проверьте URL и firewall  
**401 Unauthorized (Proxy)**: Проверьте API ключ OLLAMA_PROXY_API_KEY в GitLab Variables  
**Model not found**: Загрузите модель: `ollama pull llama3.2`  
**Timeout**: Используйте меньшую модель или увеличьте timeout

---

Подробная документация: `docs/AI_REVIEW_SETUP.md`

