#!/bin/bash
# AI Code Review для последнего коммита
# Вызывается из .gitlab-ci.yml для analyze_latest_commit job

set -euo pipefail

# ============================================================================
# Параметры (передаются через environment variables)
# ============================================================================
# Обязательные:
# - PROXY_URL: URL Ollama-OpenAI прокси
# - PROXY_API_KEY: API ключ для прокси
# - COMMIT_REVIEW_MODEL: Модель для review
# - CI_SERVER_URL, CI_PROJECT_ID, CI_COMMIT_SHA, CI_COMMIT_SHORT_SHA
# - CI_MERGE_REQUEST_IID, CI_PIPELINE_ID, CI_PROJECT_PATH, CI_JOB_ID
# - GITLAB_API_TOKEN или CI_JOB_TOKEN

echo "🚀 AI Code Review: Latest Commit (Fast)"
echo "📥 Fetching latest commit changes..."

# ============================================================================
# 1. Получаем diff коммита через GitLab API
# ============================================================================
# ⚠️ SECURITY: Never echo $API_TOKEN to logs!
API_TOKEN="${GITLAB_API_TOKEN:-$CI_JOB_TOKEN}"

DIFF_RAW=$(curl -s --header "PRIVATE-TOKEN: $API_TOKEN" \
  "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/repository/commits/$CI_COMMIT_SHA/diff")

if [ -z "$DIFF_RAW" ] || [ "$DIFF_RAW" = "null" ]; then
  echo "❌ Ошибка: Не удалось получить diff"
  exit 1
fi

# Извлекаем diff текст (формат зависит от API endpoint)
# Для коммита: массив объектов diff
# Для MR: .changes[] с полями
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

echo "📊 Diff size: $(echo "$DIFF_TEXT" | wc -c) bytes"

# ============================================================================
# 2. Отправляем diff в AI для анализа
# ============================================================================
echo "🧠 Отправка в AI модель ($COMMIT_REVIEW_MODEL)..."

SYSTEM_PROMPT="Ты опытный Go разработчик проводящий CODE REVIEW ИЗМЕНЕНИЙ.

⚠️ КРИТИЧНО - CODE REVIEW, НЕ ОПИСАНИЕ:
- Это CODE REVIEW изменений в diff (что добавлено/удалено/изменено)
- НЕ описывай что делает код (\"this script does X, Y, Z\")
- НЕ объясняй функциональность
- НЕ придумывай примеры из других проектов
- Анализируй ТОЛЬКО конкретные изменения в предоставленном diff

❌ ПЛОХОЙ ответ:
\"The code you provided is a bash script that uses the OpenAI API...\"
\"Here's what the script does: 1. It checks if... 2. For each chunk...\"

✅ ХОРОШИЙ ответ:
\"## Положительное
- .gitlab-ci.yml: правильно добавлен artifacts pattern
- review-full-mr.sh: корректная обработка ошибок в строке 108

## Замечания
- review-full-mr.sh:193 - отсутствует проверка на пустой PARTS_LINKS
- .gitlab-ci.yml:71 - слишком большой timeout (40m)\"

Задача: Дай ТОП-5 замечаний по ИЗМЕНЕНИЯМ. Укажи конкретные файлы и строки из diff. Фокусируйся на безопасность, производительность, читаемость, Go best practices.

Формат (Markdown):
## Положительное
- файл:строка - что сделано хорошо

## Замечания
- файл:строка - конкретная проблема

## Рекомендации
- как улучшить изменения

Максимум 3000 символов. Отвечай на русском языке кратко."

REQUEST_JSON=$(jq -n \
  --arg model "$COMMIT_REVIEW_MODEL" \
  --arg system "$SYSTEM_PROMPT" \
  --arg user "CODE REVIEW задача: найди проблемы в ИЗМЕНЕНИЯХ ниже.

НЕ описывай функциональность кода!
НЕ объясняй что делает скрипт!
ТОЛЬКО анализ изменений (что добавлено/удалено → проблемы → рекомендации).

Diff для анализа:\n\n$DIFF_TEXT" \
  '{
    "model": $model,
    "messages": [
      {"role": "system", "content": $system},
      {"role": "user", "content": $user}
    ],
    "temperature": 0.1,
    "max_tokens": 5000,
    "stream": false
  }')

# Отправляем в прокси
RESPONSE=$(curl -s -X POST "$PROXY_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $PROXY_API_KEY" \
  -d "$REQUEST_JSON")

# Проверяем ошибки
ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error.message // empty')
if [ -n "$ERROR_MSG" ]; then
  echo "❌ Ошибка от proxy: $ERROR_MSG"
  exit 1
fi

# Извлекаем ответ AI
AI_RESPONSE=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // empty')

if [ -z "$AI_RESPONSE" ]; then
  echo "❌ Ошибка: пустой ответ от AI"
  echo "Response: $RESPONSE"
  exit 1
fi

echo "✅ AI analysis completed"

# ============================================================================
# 3. Создаем MD файл с отчетом
# ============================================================================
REPORT_FILE="ai-review-commit-${CI_COMMIT_SHORT_SHA}.md"
DIFF_SIZE=$(echo "$DIFF_TEXT" | wc -c)
RESPONSE_SIZE=$(echo "$AI_RESPONSE" | wc -c)
REPORT_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

{
  printf "# 🤖 AI Code Review Report (Latest Commit)\n\n"
  printf "**Pipeline ID:** %s\n" "$CI_PIPELINE_ID"
  printf "**Model:** %s\n" "$COMMIT_REVIEW_MODEL"
  printf "**Date:** %s\n" "$REPORT_DATE"
  printf "**Commit:** %s\n\n" "$CI_COMMIT_SHORT_SHA"
  printf -- "---\n\n"
  printf "## 📊 Analysis Summary\n\n"
  printf "**Diff Size:** %s bytes\n" "$DIFF_SIZE"
  printf "**Response Length:** %s characters\n\n" "$RESPONSE_SIZE"
  printf -- "---\n\n"
  printf "## 🔍 AI Review\n\n"
  printf "%s\n\n" "$AI_RESPONSE"
  printf -- "---\n\n"
  printf "*Generated by Ollama-OpenAI Proxy CI/CD Pipeline*\n"
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
RESPONSE_LENGTH=$(echo "$AI_RESPONSE" | wc -c)
echo "📏 Response length: $RESPONSE_LENGTH bytes"

MAX_CHUNK_SIZE=50000

if [ "$RESPONSE_LENGTH" -gt "$MAX_CHUNK_SIZE" ]; then
  echo "⚠️ Response too long, splitting into multiple comments..."
  
  # Разрезаем на части по 50KB
  TEMP_FILE=$(mktemp)
  echo "$AI_RESPONSE" > "$TEMP_FILE"
  
  TOTAL_CHUNKS=$(( ($RESPONSE_LENGTH + $MAX_CHUNK_SIZE - 1) / $MAX_CHUNK_SIZE ))
  CHUNK_NUM=1
  
  while [ $CHUNK_NUM -le $TOTAL_CHUNKS ]; do
    CHUNK_START=$(( ($CHUNK_NUM - 1) * $MAX_CHUNK_SIZE ))
    CHUNK_TEXT=$(dd if="$TEMP_FILE" bs=1 skip=$CHUNK_START count=$MAX_CHUNK_SIZE 2>/dev/null || true)
    
    COMMENT_BODY=$(jq -n \
      --arg model "$COMMIT_REVIEW_MODEL" \
      --arg response "$CHUNK_TEXT" \
      --arg pipeline "$CI_PIPELINE_ID" \
      --arg part "$CHUNK_NUM" \
      --arg total "$TOTAL_CHUNKS" \
      --arg report "$FULL_REPORT_LINK" \
      '{body: ("🤖 **AI Code Review** [Part " + $part + "/" + $total + "] (Model: " + $model + ")\n\n" + $response + "\n\n---\n" + $report + " | Pipeline ID: " + $pipeline)}')
    
    echo "💬 Adding comment part $CHUNK_NUM/$TOTAL_CHUNKS..."
    curl -s --request POST \
      --header "PRIVATE-TOKEN: $API_TOKEN" \
      --header "Content-Type: application/json" \
      --data "$COMMENT_BODY" \
      "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes" > /dev/null
    
    CHUNK_NUM=$(($CHUNK_NUM + 1))
    sleep 1  # Небольшая задержка между комментариями
  done
  
  rm -f "$TEMP_FILE"
  echo "✅ All comment parts added successfully"
else
  # Ответ влезает в один комментарий
  COMMENT_BODY=$(jq -n \
    --arg model "$COMMIT_REVIEW_MODEL" \
    --arg response "$AI_RESPONSE" \
    --arg pipeline "$CI_PIPELINE_ID" \
    --arg report "$FULL_REPORT_LINK" \
    '{body: ("🤖 **AI Code Review** (Model: " + $model + ")\n\n---\n\n" + $response + "\n\n---\n" + $report + " | *Powered by Ollama-OpenAI Proxy | Review ID: " + $pipeline + "*")}')
  
  echo "💬 Adding comment to MR..."
  COMMENT_RESULT=$(curl -s --request POST \
    --header "PRIVATE-TOKEN: $API_TOKEN" \
    --header "Content-Type: application/json" \
    --data "$COMMENT_BODY" \
    "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes")
  
  COMMENT_ID=$(echo "$COMMENT_RESULT" | jq -r '.id // empty')
  if [ -n "$COMMENT_ID" ]; then
    echo "✅ Comment added successfully (ID: $COMMENT_ID)"
  else
    echo "⚠️ Failed to add comment"
    echo "$COMMENT_RESULT" | jq '.'
  fi
fi

echo "✅ Commit review completed successfully"

