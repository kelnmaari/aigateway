# AI Review - Действия требуются

## ✅ Что уже сделано

1. ✅ Исправлен bugfix `completion_tokens` в proxy
2. ✅ Создана документация по настройке
3. ✅ Модель в `.gitlab-ci.yml` изменена на `deepseek-coder:33b`
4. ✅ Тесты подтверждают исправления

## ⚠️ Что нужно сделать СЕЙЧАС

### 1. Перезапустите AIGateway Platform с новым билдом

```bash
# Остановите старый процесс
pkill -f server.exe

# Запустите новый билд
cd G:\golang\my_projects
.\bin\server.exe -config configs/dev.yaml

# Или если используете systemd:
sudo systemctl restart ollama-proxy
```

**Зачем:** Новый билд содержит исправление `completion_tokens` для AI-review совместимости.

### 2. Загрузите модель deepseek-coder:33b в Ollama

```bash
# Проверьте установленные модели
ollama list

# Если deepseek-coder:33b отсутствует:
ollama pull deepseek-coder:33b

# Проверка:
ollama list | grep deepseek
```

**Зачем:** `devstral-tuned:latest` не работает с AI-review (возвращает markdown вместо JSON).

### 3. Создайте GitLab Access Token

```
GitLab Project → Settings → Access Tokens → Add new token

Name:   ai-review-token
Role:   Developer
Scopes: ✓ api
        ✓ read_api
        ✓ write_repository

→ Create token
→ Copy token (показывается один раз!)
```

### 4. Добавьте токен в GitLab CI Variables

```
GitLab → Settings → CI/CD → Variables → Add Variable

# Токен 1: Для proxy
Key:   OLLAMA_PROXY_API_KEY
Value: (ваш API ключ из WebUI proxy)
Flags: ✓ Mask variable

# Токен 2: Для GitLab API
Key:   GITLAB_API_TOKEN
Value: (токен из шага 3)
Flags: ✓ Mask variable
       ✓ Protect variable
```

### 5. Проверьте что proxy работает

```bash
curl http://176.53.180.178:8085/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-coder:33b",
    "messages": [{"role": "user", "content": "Hi"}]
  }' | jq '.usage'
```

**Ожидаемый ответ:**
```json
{
  "prompt_tokens": 5,
  "completion_tokens": 3,  // ← ДОЛЖЕН присутствовать!
  "total_tokens": 8
}
```

❌ **Если `completion_tokens` отсутствует** - proxy не обновлен, вернитесь к шагу 1.

### 6. Запустите AI review снова

```bash
# Для больших MR (17+ файлов) используйте summary:
ai-review run-summary

# Для маленьких MR (< 10 файлов) можно context:
ai-review run-context
```

---

## 🔍 Диагностика проблем

### Проблема: 401 Unauthorized (GitLab API)

**Проверка:**
```bash
curl -H "PRIVATE-TOKEN: $GITLAB_API_TOKEN" \
  https://gitlab.alexue4.dev/api/v4/projects/146/merge_requests/1
```

**Если ошибка:** Токен неверный или отсутствует → вернитесь к шагу 3-4.

### Проблема: 401 Unauthorized (Proxy)

**Проверка:**
```bash
curl http://176.53.180.178:8085/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Если ошибка:** API ключ неверный → создайте новый в WebUI.

### Проблема: Model not found (deepseek-coder:33b)

**Проверка:**
```bash
ollama list | grep deepseek
```

**Если пусто:** Модель не загружена → выполните шаг 2.

### Проблема: 503 Service Unavailable

**Причина:** Ollama перегружен или недоступен.

**Решение:**
1. Проверьте что Ollama запущен: `ollama serve`
2. Уменьшите количество файлов в MR
3. Используйте меньшую модель: `deepseek-coder:6.7b`

### Проблема: JSON parse error

**Симптом:**
```
ERROR | LLM_JSON_PARSER | No valid JSON found in output
```

**Причина:** Модель возвращает markdown wrapper.

**Решение:**
1. Убедитесь что используется `deepseek-coder:33b` (проверьте `.gitlab-ci.yml`)
2. Если проблема сохраняется → используйте `ai-review run-summary`

---

## 📊 Ожидаемый результат

После выполнения всех шагов, AI review должен:

1. ✅ Успешно подключиться к GitLab API
2. ✅ Получить данные MR (файлы, discussions)
3. ✅ Отправить запрос в Ollama через proxy
4. ✅ Получить валидный JSON ответ
5. ✅ Распарсить JSON без ошибок
6. ✅ Создать комментарии в MR

**Пример успешного лога:**
```
2025-10-22 XX:XX:XX | INFO | GITLAB_VCS_CLIENT | Fetched inline discussions for project_id=146
2025-10-22 XX:XX:XX | INFO | REVIEW_POLICY_SERVICE | Proceeding with 17 files after policy filter
2025-10-22 XX:XX:XX | INFO | OPENAI_V1_HTTP_CLIENT | POST http://176.53.180.178:8085/v1/chat/completions - Status 200
2025-10-22 XX:XX:XX | INFO | INLINE_COMMENT_SERVICE | Successfully parsed 5 inline comments from LLM
2025-10-22 XX:XX:XX | INFO | GITLAB_VCS_CLIENT | Created 5 inline comments in MR
AI review completed successfully!
```

---

## 📚 Документация

- **Bugfix details**: `docs/BUGFIX_AI_REVIEW_COMPATIBILITY.md`
- **Full setup guide**: `docs/AI_REVIEW_SETUP.md`
- **Quick start**: `docs/AI_REVIEW_QUICKSTART.md`
- **Custom prompts**: `docs/AI_REVIEW_GO_PROMPTS.md`

---

## 🚀 После настройки

Когда все работает, можете:

1. **Настроить автоматический review** для маленьких MR
2. **Кастомизировать промпты** под ваш проект
3. **Добавить фильтры** для важных файлов
4. **Создать разные jobs** для разных размеров MR

См. Best Practices в `docs/AI_REVIEW_SETUP.md`.

---

**Version:** 1.0  
**Last Updated:** 2025-10-22  
**Status:** Action Required


