#!/bin/bash
# AI Code Review для полного MR с chunking
# Вызывается из .gitlab-ci.yml для analyze_full_mr job

set -euo pipefail

# ============================================================================
# Параметры (передаются через environment variables)
# ============================================================================
# Обязательные:
# - PROXY_URL: URL Ollama-OpenAI прокси
# - PROXY_API_KEY: API ключ для прокси
# - FULL_MR_REVIEW_MODEL: Модель для детального review
# - CI_SERVER_URL, CI_PROJECT_ID, CI_MERGE_REQUEST_IID
# - CI_PIPELINE_ID, CI_PROJECT_PATH, CI_JOB_ID
# - GITLAB_API_TOKEN или CI_JOB_TOKEN

echo "🔍 AI Code Review: Full MR Analysis (Comprehensive)"
echo "📥 Fetching FULL MR changes..."

# ============================================================================
# 1. Получаем полный diff MR через GitLab API
# ============================================================================
# ⚠️ SECURITY: Never echo $API_TOKEN to logs!
API_TOKEN="${GITLAB_API_TOKEN:-$CI_JOB_TOKEN}"

DIFF_RAW=$(curl -s --header "PRIVATE-TOKEN: $API_TOKEN" \
  "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/changes")

if [ -z "$DIFF_RAW" ] || [ "$DIFF_RAW" = "null" ]; then
  echo "❌ Ошибка: Не удалось получить diff"
  exit 1
fi

# Извлекаем diff текст (до 100KB для полного анализа)
DIFF_TEXT=$(echo "$DIFF_RAW" | jq -r '
  if type == "array" then
    .[] | "File: \(.new_path // .old_path)\nDiff:\n\(.diff // "No diff")\n---\n"
  else
    .changes[]? | "File: \(.new_path // .old_path)\nStatus: \(.new_file // false | if . then "new" else "modified" end)\nDiff:\n\(.diff // "No diff")\n---\n"
  end' | head -c 100000 || true)

if [ -z "$DIFF_TEXT" ]; then
  echo "⚠️ Diff пустой, пропускаем review"
  exit 0
fi

DIFF_SIZE=$(echo "$DIFF_TEXT" | wc -c)
echo "📊 Diff size: $DIFF_SIZE bytes"

# ============================================================================
# 2. Chunking: разбиваем большой diff на части
# ============================================================================
CHUNK_SIZE=20000  # ~20KB per chunk (примерно 5000 токенов)

if [ "$DIFF_SIZE" -gt "$CHUNK_SIZE" ]; then
  echo "📦 Diff слишком большой, разбиваем на части (chunk size: $CHUNK_SIZE bytes)"
  
  # Сохраняем diff во временный файл
  TEMP_DIFF=$(mktemp)
  echo "$DIFF_TEXT" > "$TEMP_DIFF"
  
  # Подсчитываем количество частей
  CHUNKS_COUNT=$(( ($DIFF_SIZE + $CHUNK_SIZE - 1) / $CHUNK_SIZE ))
  echo "📊 Создано частей: $CHUNKS_COUNT"
  
  # Собираем ответы от всех частей
  ALL_RESPONSES=""
  
  for CHUNK_NUM in $(seq 1 $CHUNKS_COUNT); do
    CHUNK_START=$(( ($CHUNK_NUM - 1) * $CHUNK_SIZE ))
    CHUNK_TEXT=$(dd if="$TEMP_DIFF" bs=1 skip=$CHUNK_START count=$CHUNK_SIZE 2>/dev/null || true)
    
    echo "🧠 Анализ части $CHUNK_NUM/$CHUNKS_COUNT (Model: $FULL_MR_REVIEW_MODEL)..."
    
    # Промпт для части
    SYSTEM_PROMPT="Ты опытный Go разработчик проводящий code review.

КРИТИЧНО: Это ЧАСТЬ $CHUNK_NUM из $CHUNKS_COUNT одного большого MR diff. Анализируй ТОЛЬКО предоставленный код ниже. НЕ придумывай примеры. НЕ показывай код из других проектов.

Задача: Дай краткие замечания по этой части diff. Укажи конкретные файлы и проблемы из предоставленного кода.

Формат: список замечаний (файл → проблема → рекомендация). Отвечай кратко на русском."
    
    REQUEST_JSON=$(jq -n \
      --arg model "$FULL_MR_REVIEW_MODEL" \
      --arg system "$SYSTEM_PROMPT" \
      --arg user "Go проект: часть $CHUNK_NUM/$CHUNKS_COUNT diff для анализа (НЕ придумывай примеры!):\n\n$CHUNK_TEXT" \
      '{
        "model": $model,
        "messages": [
          {"role": "system", "content": $system},
          {"role": "user", "content": $user}
        ],
        "temperature": 0.1,
        "max_tokens": 3000,
        "stream": false
      }')
    
    # Отправляем запрос
    RESPONSE=$(curl -s -X POST "$PROXY_URL" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $PROXY_API_KEY" \
      -d "$REQUEST_JSON")
    
    # Проверяем ошибки
    ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error.message // empty')
    if [ -n "$ERROR_MSG" ]; then
      echo "⚠️ Ошибка в части $CHUNK_NUM: $ERROR_MSG"
      CHUNK_RESPONSE="[Часть $CHUNK_NUM: Ошибка анализа - $ERROR_MSG]"
    else
      CHUNK_RESPONSE=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // "[Пустой ответ]"')
      
      # Проверяем что ответ не пустой
      CHUNK_SIZE_BYTES=$(echo "$CHUNK_RESPONSE" | wc -c)
      if [ "$CHUNK_SIZE_BYTES" -lt 50 ]; then
        echo "⚠️ ВНИМАНИЕ: Часть $CHUNK_NUM вернула очень короткий ответ ($CHUNK_SIZE_BYTES bytes)"
        echo "⚠️ Модель $FULL_MR_REVIEW_MODEL возможно не подходит для chunked анализа"
        echo "⚠️ Рекомендация: попробуйте qwen2.5-coder:7b или deepseek-coder:6.7b"
        CHUNK_RESPONSE="$CHUNK_RESPONSE\n\n_⚠️ Warning: Модель вернула короткий ответ. Возможно модель $FULL_MR_REVIEW_MODEL не справляется с chunked анализом._"
      else
        echo "✅ Часть $CHUNK_NUM: получено $CHUNK_SIZE_BYTES bytes"
      fi
    fi
    
    # Добавляем к итоговому результату
    ALL_RESPONSES="$ALL_RESPONSES\n\n## Часть $CHUNK_NUM/$CHUNKS_COUNT\n\n$CHUNK_RESPONSE"
    
    # Небольшая задержка между запросами
    sleep 2
  done
  
  # Удаляем временный файл
  rm -f "$TEMP_DIFF"
  
  echo "📦 Формируем итоговый отчет из всех частей..."
  
  # Проверяем размер всех ответов
  ALL_RESPONSES_SIZE=$(echo "$ALL_RESPONSES" | wc -c)
  echo "📊 Размер всех частей: $ALL_RESPONSES_SIZE bytes"
  
  # AI merge отключен из-за ненадежности gpt-oss-tuned:latest
  # Модель часто возвращает только последнюю часть, игнорируя остальные
  # Используем простое объединение - это надежнее
  echo "📦 Используем структурированное объединение всех частей"
  AI_RESPONSE="# 🔍 AI Code Review (Полный анализ MR)\n\n"
  AI_RESPONSE="$AI_RESPONSE> Diff был разбит на $CHUNKS_COUNT частей для детального анализа.\n"
  AI_RESPONSE="$AI_RESPONSE> Ниже представлены все замечания по каждой части.\n\n"
  AI_RESPONSE="$AI_RESPONSE---\n\n"
  AI_RESPONSE="$AI_RESPONSE$ALL_RESPONSES\n\n"
  AI_RESPONSE="$AI_RESPONSE---\n\n"
  AI_RESPONSE="$AI_RESPONSE## 📋 Общие рекомендации\n\n"
  AI_RESPONSE="$AI_RESPONSE- Внимательно просмотрите ВСЕ части выше\n"
  AI_RESPONSE="$AI_RESPONSE- Приоритезируйте критичные замечания по Архитектуре и Безопасности\n"
  AI_RESPONSE="$AI_RESPONSE- Обратите внимание на производительность и тестирование\n"
  
  # OPTIONAL: AI merge можно включить для более мощных моделей
  # Раскомментируй следующий блок если используешь qwen2.5-coder:32b или claude
  # if [ "$ALL_RESPONSES_SIZE" -lt 30000 ] && [ "$FULL_MR_REVIEW_MODEL" != "gpt-oss-tuned:latest" ]; then
  #   echo "🔗 Пробуем AI merge для улучшения отчета..."
  #   MERGE_PROMPT="Ты опытный Go разработчик. Объедини $CHUNKS_COUNT частей code review в ЕДИНЫЙ отчет. Сгруппируй по категориям. НЕ теряй информацию."
  #   MERGE_REQUEST=$(jq -n --arg model "$FULL_MR_REVIEW_MODEL" --arg system "$MERGE_PROMPT" --arg user "Объедини:\n\n$ALL_RESPONSES" '{model: $model, messages: [{role: "system", content: $system}, {role: "user", content: $user}], temperature: 0.1, max_tokens: 6000}')
  #   MERGE_RESPONSE=$(curl -s -X POST "$PROXY_URL" -H "Content-Type: application/json" -H "Authorization: Bearer $PROXY_API_KEY" -d "$MERGE_REQUEST")
  #   MERGED_CONTENT=$(echo "$MERGE_RESPONSE" | jq -r '.choices[0].message.content // empty')
  #   if [ -n "$MERGED_CONTENT" ] && [ "$MERGED_CONTENT" != "null" ]; then
  #     echo "✅ AI merge успешен"
  #     AI_RESPONSE="$MERGED_CONTENT"
  #   fi
  # fi
  
  PROMPT_TOKENS=$(($CHUNKS_COUNT * 5000))  # Все части
  COMPLETION_TOKENS=$(($CHUNKS_COUNT * 2000))  # Все ответы
  
else
  # Маленький diff - отправляем целиком
  echo "🧠 Отправка в AI модель ($FULL_MR_REVIEW_MODEL) для детального анализа..."
  
  SYSTEM_PROMPT="Ты опытный Go разработчик проводящий ПОЛНЫЙ code review всего MR.

КРИТИЧНО: Анализируй ТОЛЬКО предоставленный diff ниже. НЕ придумывай примеры. НЕ показывай код из других проектов. Если diff пустой - так и скажи.

Задача: Проанализируй ВСЕ изменения детально и дай исчерпывающий отчет. Укажи конкретные файлы и строки из diff. Фокусируйся на:
- Архитектура и дизайн решений
- Безопасность (уязвимости, API keys, SQL injection)
- Производительность (алгоритмы, памяти, goroutines)
- Читаемость и поддерживаемость
- Go best practices и idiomatic code
- Тестирование (coverage, edge cases)

Формат ответа (Markdown):
## Архитектура
## Безопасность
## Производительность
## Замечания (файл:строка → проблема → fix)
## Рекомендации

Отвечай на русском языке подробно."
  
  REQUEST_JSON=$(jq -n \
    --arg model "$FULL_MR_REVIEW_MODEL" \
    --arg system "$SYSTEM_PROMPT" \
    --arg user "Go проект: Full MR diff для детального анализа (НЕ придумывай примеры!):\n\n$DIFF_TEXT" \
    '{
      "model": $model,
      "messages": [
        {"role": "system", "content": $system},
        {"role": "user", "content": $user}
      ],
      "temperature": 0.1,
      "max_tokens": 4000,
      "stream": false
    }')
  
  # Отправляем в прокси
  RESPONSE=$(curl -s -X POST "$PROXY_URL" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $PROXY_API_KEY" \
    -d "$REQUEST_JSON")
  
  # Проверяем ошибки (для не-chunked)
  ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error.message // empty')
  if [ -n "$ERROR_MSG" ]; then
    echo "❌ Ошибка от proxy: $ERROR_MSG"
    
    # Создаем fallback для timeout/connection errors (не фейлим job)
    if echo "$ERROR_MSG" | grep -qi "timeout\|deadline\|exceeded\|connection\|refused"; then
      echo "⚠️ Timeout или connection error - создаем fallback report"
      AI_RESPONSE="AI Review Failed - Connection Timeout"
      AI_RESPONSE="$AI_RESPONSE\n\nError - $ERROR_MSG"
      AI_RESPONSE="$AI_RESPONSE\n\nModel - $FULL_MR_REVIEW_MODEL"
      AI_RESPONSE="$AI_RESPONSE\nDiff size - $(echo "$DIFF_TEXT" | wc -c) bytes"
      AI_RESPONSE="$AI_RESPONSE\n\nPossible causes:"
      AI_RESPONSE="$AI_RESPONSE\n- Ollama server is down or unreachable at localhost:11434"
      AI_RESPONSE="$AI_RESPONSE\n- Request is too large for available resources"
      AI_RESPONSE="$AI_RESPONSE\n- Model is not loaded or crashed"
      AI_RESPONSE="$AI_RESPONSE\n- Network timeout or firewall blocking"
      AI_RESPONSE="$AI_RESPONSE\n\nRecommendations:"
      AI_RESPONSE="$AI_RESPONSE\n- Check Ollama server status on proxy host"
      AI_RESPONSE="$AI_RESPONSE\n- Verify model is loaded with: ollama list"
      AI_RESPONSE="$AI_RESPONSE\n- Check proxy logs at: journalctl -u aigateway-proxy"
      AI_RESPONSE="$AI_RESPONSE\n- Try smaller diff or manual review trigger"
      AI_RESPONSE="$AI_RESPONSE\n- Consider using lighter model like qwen2.5-coder:7b"
      PROMPT_TOKENS=0
      COMPLETION_TOKENS=0
    else
      # Другие ошибки - fail job
      exit 1
    fi
  else
    # Извлекаем ответ AI (успешный случай)
    AI_RESPONSE=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // empty')
    
    # Debug: показываем usage stats
    PROMPT_TOKENS=$(echo "$RESPONSE" | jq -r '.usage.prompt_tokens // 0')
    COMPLETION_TOKENS=$(echo "$RESPONSE" | jq -r '.usage.completion_tokens // 0')
    echo "📊 Tokens: prompt=$PROMPT_TOKENS, completion=$COMPLETION_TOKENS"
  fi
fi  # Конец chunking if

# Проверка на пустой ответ (баг некоторых моделей)
if [ -z "$AI_RESPONSE" ] || [ "$AI_RESPONSE" = "null" ]; then
  echo "❌ Ошибка: Модель вернула пустой content"
  echo "⚠️  Это может быть баг модели $FULL_MR_REVIEW_MODEL"
  echo "💡 Рекомендация: Попробуйте другую модель (qwen2.5-coder:7b, deepseek-coder:6.7b)"
  echo ""
  echo "Debug Response:"
  echo "$RESPONSE" | jq '.'
  
  # Создаем fallback отчет (простой текст без спецсимволов)
  AI_RESPONSE="AI Review Failed - Model returned empty content"
  AI_RESPONSE="$AI_RESPONSE\n\nModel used - $FULL_MR_REVIEW_MODEL"
  AI_RESPONSE="$AI_RESPONSE\nPrompt tokens - $PROMPT_TOKENS"
  AI_RESPONSE="$AI_RESPONSE\nCompletion tokens - $COMPLETION_TOKENS"
  AI_RESPONSE="$AI_RESPONSE\n\nRecommendations:"
  AI_RESPONSE="$AI_RESPONSE\n- Try qwen2.5-coder:7b model (recommended for Go)"
  AI_RESPONSE="$AI_RESPONSE\n- Try deepseek-coder:6.7b model (fast alternative)"
  AI_RESPONSE="$AI_RESPONSE\n- Try codellama:7b model (universal)"
  AI_RESPONSE="$AI_RESPONSE\n\nCheck if model is loaded with: ollama list"
  AI_RESPONSE="$AI_RESPONSE\n\nDiff size - $(echo "$DIFF_TEXT" | wc -c) bytes"
  AI_RESPONSE="$AI_RESPONSE\nFiles changed - $(echo "$DIFF_RAW" | jq -r '.changes | length // 0')"
fi

echo "✅ AI analysis completed"

# ============================================================================
# 3. Создаем MD файл с отчетом
# ============================================================================
REPORT_FILE="ai-review-mr-${CI_MERGE_REQUEST_IID}-full.md"
DIFF_SIZE=$(echo "$DIFF_TEXT" | wc -c)
RESPONSE_SIZE=$(echo "$AI_RESPONSE" | wc -c)
REPORT_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

{
  printf "# 🔍 AI Code Review Report (Full MR)\n\n"
  printf "**Pipeline ID:** %s\n" "$CI_PIPELINE_ID"
  printf "**Model:** %s\n" "$FULL_MR_REVIEW_MODEL"
  printf "**Date:** %s\n" "$REPORT_DATE"
  printf "**MR:** !%s\n\n" "$CI_MERGE_REQUEST_IID"
  printf -- "---\n\n"
  printf "## 📊 Analysis Summary\n\n"
  printf "**Diff Size:** %s bytes\n" "$DIFF_SIZE"
  printf "**Response Length:** %s characters\n"  "$RESPONSE_SIZE"
  printf "**Analysis Type:** Full MR (Comprehensive)\n\n"
  printf -- "---\n\n"
  printf "## 🔍 Detailed AI Review\n\n"
  printf "%s\n\n" "$AI_RESPONSE"
  printf -- "---\n\n"
  printf "*Generated by Ollama-OpenAI Proxy CI/CD Pipeline (Full MR Analysis)*\n"
} > "$REPORT_FILE"

echo "📄 Report file created: $REPORT_FILE"

# ============================================================================
# 4. Формируем ссылки на артефакт
# ============================================================================
echo "🔗 Generating artifact link..."
ARTIFACT_FILE_URL="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/file/$REPORT_FILE"
ARTIFACT_RAW_URL="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/raw/$REPORT_FILE?inline=false"

echo "✅ Artifact will be available at:"
echo "   📄 View: $ARTIFACT_FILE_URL"
echo "   💾 Download: $ARTIFACT_RAW_URL"

FULL_REPORT_LINK="[📄 Просмотр]($ARTIFACT_FILE_URL) | [💾 Скачать]($ARTIFACT_RAW_URL)"

# ============================================================================
# 5. Добавляем комментарий в MR
# ============================================================================
# Формируем комментарий (ограничиваем длину для читаемости)
RESPONSE_LENGTH=$(echo "$AI_RESPONSE" | wc -c)
if [ "$RESPONSE_LENGTH" -gt 3000 ]; then
  echo "📝 Ответ слишком длинный ($RESPONSE_LENGTH chars), обрезаем для комментария"
  COMMENT_RESPONSE=$(echo "$AI_RESPONSE" | head -c 3000)
  COMMENT_RESPONSE="$COMMENT_RESPONSE\n\n...\n\n_Полный отчет слишком длинный для комментария. См. файл ниже._"
else
  COMMENT_RESPONSE="$AI_RESPONSE"
fi

COMMENT_BODY=$(jq -n \
  --arg model "$FULL_MR_REVIEW_MODEL" \
  --arg response "$COMMENT_RESPONSE" \
  --arg pipeline "$CI_PIPELINE_ID" \
  --arg report "$FULL_REPORT_LINK" \
  --arg mr "$CI_MERGE_REQUEST_IID" \
  '{body: ("🔍 **Full MR Analysis** (Model: " + $model + ")\n\n**MR:** !" + $mr + "\n\n---\n\n" + $response + "\n\n---\n\n" + $report + "\n\n*Pipeline ID: " + $pipeline + "*")}')

echo "💬 Adding full MR review comment..."
curl -s --request POST \
  --header "PRIVATE-TOKEN: $API_TOKEN" \
  --header "Content-Type: application/json" \
  --data "$COMMENT_BODY" \
  "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes" > /dev/null

echo "✅ Full MR review posted successfully"

