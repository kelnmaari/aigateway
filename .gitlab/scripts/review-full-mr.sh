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
# 2. File-based splitting: разбиваем diff по файлам (не по chunks!)
# ============================================================================
# Извлекаем список измененных файлов из MR
FILES_COUNT=$(echo "$DIFF_RAW" | jq -r '.changes | length')

if [ "$FILES_COUNT" -eq 0 ]; then
  echo "⚠️ Нет измененных файлов в MR"
  exit 0
fi

echo "📦 Обнаружено файлов в MR: $FILES_COUNT"
echo "🔍 Будем анализировать каждый файл отдельно"

# Флаг для определения file-based vs single-file review
WAS_SPLIT_BY_FILES=false

if [ "$FILES_COUNT" -gt 1 ]; then
  WAS_SPLIT_BY_FILES=true
  echo "📊 Разбиваем на $FILES_COUNT файлов для детального анализа"
  
  # Собираем ответы от всех файлов
  ALL_RESPONSES=""
  
  # Итерируемся по каждому файлу
  FILE_NUM=0
  while read -r FILE_DATA; do
    FILE_NUM=$((FILE_NUM + 1))
    
    FILE_PATH=$(echo "$FILE_DATA" | jq -r '.new_path // .old_path')
    FILE_DIFF=$(echo "$FILE_DATA" | jq -r '.diff // ""')
    FILE_STATUS=$(echo "$FILE_DATA" | jq -r 'if .new_file then "new" elif .deleted_file then "deleted" elif .renamed_file then "renamed" else "modified" end')
    
    # Пропускаем файлы без diff (например, бинарные или переименованные без изменений)
    if [ -z "$FILE_DIFF" ] || [ "$FILE_DIFF" = "null" ]; then
      echo "⏭️  Пропуск $FILE_PATH (нет diff)"
      continue
    fi
    
    FILE_DIFF_SIZE=$(echo "$FILE_DIFF" | wc -c)
    echo ""
    echo "📄 [$FILE_NUM/$FILES_COUNT] Анализ файла: $FILE_PATH"
    echo "   Status: $FILE_STATUS | Size: $FILE_DIFF_SIZE bytes"
    
    # Промпт для файла
    SYSTEM_PROMPT="Ты опытный Go разработчик проводящий CODE REVIEW ИЗМЕНЕНИЙ.

⚠️ КРИТИЧНО - CODE REVIEW ФАЙЛА $FILE_PATH:
- Это CODE REVIEW изменений в конкретном файле (что добавлено/удалено/изменено)
- НЕ описывай что делает код общими словами
- НЕ объясняй функциональность или алгоритмы
- ТОЛЬКО анализ изменений: строка → проблема → fix
- Анализируй ТОЛЬКО предоставленный diff этого файла

Задача: Найди проблемы в ИЗМЕНЕНИЯХ файла $FILE_PATH. Укажи конкретные строки.

Формат:
## Файл: $FILE_PATH

### Проблемы
- строка X - проблема → рекомендация

### Хорошо
- строка Y - что сделано правильно

Отвечай кратко на русском. Если проблем нет - напиши \"Нет критичных замечаний\"."
    
    REQUEST_JSON=$(jq -n \
      --arg model "$FULL_MR_REVIEW_MODEL" \
      --arg system "$SYSTEM_PROMPT" \
      --arg filepath "$FILE_PATH" \
      --arg status "$FILE_STATUS" \
      --arg user "CODE REVIEW файла: $FILE_PATH (status: $FILE_STATUS)

Найди проблемы в изменениях этого файла:\n\n$FILE_DIFF" \
      '{
        "model": $model,
        "messages": [
          {"role": "system", "content": $system},
          {"role": "user", "content": $user}
        ],
        "temperature": 0.1,
        "max_tokens": 10000,
        "stream": false
      }')
    
    # DEBUG: Выводим запрос
    echo "   🔍 DEBUG REQUEST:"
    echo "$REQUEST_JSON" | jq -C '.' | head -50
    echo ""
    
    # Отправляем запрос
    RESPONSE=$(curl -s -X POST "$PROXY_URL" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $PROXY_API_KEY" \
      -d "$REQUEST_JSON")
    
    # DEBUG: Выводим ответ
    echo "   🔍 DEBUG RESPONSE:"
    echo "$RESPONSE" | jq -C '.'
    echo ""
    
    # Проверяем ошибки
    ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error.message // empty')
    if [ -n "$ERROR_MSG" ]; then
      echo "   ⚠️ Ошибка анализа: $ERROR_MSG"
      FILE_RESPONSE="[Ошибка анализа $FILE_PATH: $ERROR_MSG]"
    else
      FILE_RESPONSE=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // "[Пустой ответ]"')
      
      # Проверяем что ответ не пустой
      FILE_RESPONSE_SIZE=$(echo "$FILE_RESPONSE" | wc -c)
      if [ "$FILE_RESPONSE_SIZE" -lt 50 ]; then
        echo "   ⚠️ Короткий ответ ($FILE_RESPONSE_SIZE bytes)"
        FILE_RESPONSE="$FILE_RESPONSE\n\n_⚠️ Warning: Модель вернула короткий ответ для $FILE_PATH._"
      else
        echo "   ✅ Получено $FILE_RESPONSE_SIZE bytes"
      fi
    fi
    
    # ============================================================================
    # 2.1. Создаем отдельный MD файл для этого файла
    # ============================================================================
    # Безопасное имя файла (замена / на -)
    SAFE_FILE_NAME=$(echo "$FILE_PATH" | sed 's/\//-/g' | sed 's/\./-/g')
    FILE_REPORT_FILE="ai-review-mr-${CI_MERGE_REQUEST_IID}-file-${SAFE_FILE_NAME}.md"
    FILE_REPORT_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    
    {
      printf "# 🔍 AI Code Review Report (MR !%s - %s)\n\n" "$CI_MERGE_REQUEST_IID" "$FILE_PATH"
      printf "**Pipeline ID:** %s\n" "$CI_PIPELINE_ID"
      printf "**Model:** %s\n" "$FULL_MR_REVIEW_MODEL"
      printf "**Date:** %s\n" "$FILE_REPORT_DATE"
      printf "**File:** %s\n" "$FILE_PATH"
      printf "**Status:** %s\n" "$FILE_STATUS"
      printf "**File Number:** %s из %s\n\n" "$FILE_NUM" "$FILES_COUNT"
      printf -- "---\n\n"
      printf "## 📊 Analysis Summary\n\n"
      printf "**File Path:** \`%s\`\n" "$FILE_PATH"
      printf "**File Status:** %s\n" "$FILE_STATUS"
      printf "**Diff Size:** %s bytes\n" "$FILE_DIFF_SIZE"
      printf "**Response Length:** %s characters\n" "$FILE_RESPONSE_SIZE"
      printf "**Analysis Type:** Full MR (Per-File)\n\n"
      printf -- "---\n\n"
      printf "## 🔍 AI Review\n\n"
      printf "%s\n\n" "$FILE_RESPONSE"
      printf -- "---\n\n"
      printf "*Generated by Ollama-OpenAI Proxy CI/CD Pipeline*\n"
    } > "$FILE_REPORT_FILE"
    
    echo "   📄 Создан файл: $FILE_REPORT_FILE"
    
    # ============================================================================
    # 2.2. Отправляем комментарий для этого файла в GitLab
    # ============================================================================
    FILE_ARTIFACT_URL="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/file/$FILE_REPORT_FILE"
    FILE_ARTIFACT_RAW="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/raw/$FILE_REPORT_FILE?inline=false"
    
    # Ограничиваем длину комментария (макс 2000 символов)
    if [ "$FILE_RESPONSE_SIZE" -gt 2000 ]; then
      echo "   📝 Ответ слишком длинный ($FILE_RESPONSE_SIZE chars), обрезаем для комментария"
      COMMENT_FILE_RESPONSE=$(echo "$FILE_RESPONSE" | head -c 2000)
      COMMENT_FILE_RESPONSE="$COMMENT_FILE_RESPONSE\n\n...\n\n_Полный отчет см. в файле ниже._"
    else
      COMMENT_FILE_RESPONSE="$FILE_RESPONSE"
    fi
    
    FILE_COMMENT_BODY=$(jq -n \
      --arg model "$FULL_MR_REVIEW_MODEL" \
      --arg response "$COMMENT_FILE_RESPONSE" \
      --arg pipeline "$CI_PIPELINE_ID" \
      --arg filepath "$FILE_PATH" \
      --arg filenum "$FILE_NUM" \
      --arg total "$FILES_COUNT" \
      --arg status "$FILE_STATUS" \
      --arg report_view "$FILE_ARTIFACT_URL" \
      --arg report_download "$FILE_ARTIFACT_RAW" \
      --arg mr "$CI_MERGE_REQUEST_IID" \
      '{body: ("📄 **File Review [" + $filenum + "/" + $total + "]: `" + $filepath + "`** (Status: " + $status + ")\n\n**Model:** " + $model + "\n\n---\n\n" + $response + "\n\n---\n\n📄 [Просмотр полного отчета](" + $report_view + ") | 💾 [Скачать](" + $report_download + ")\n\n*Pipeline ID: " + $pipeline + "*")}')
    
    echo "   💬 Отправка комментария в GitLab..."
    curl -s --request POST \
      --header "PRIVATE-TOKEN: $API_TOKEN" \
      --header "Content-Type: application/json" \
      --data "$FILE_COMMENT_BODY" \
      "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes" > /dev/null
    
    echo "   ✅ Комментарий отправлен"
    
    # Добавляем к итоговому результату
    ALL_RESPONSES="$ALL_RESPONSES\n\n## Файл: $FILE_PATH\n\n$FILE_RESPONSE"
    
    # Небольшая задержка между запросами
    sleep 3
  done < <(echo "$DIFF_RAW" | jq -c '.changes[]')
  
  echo ""
  echo "📦 Формируем итоговый отчет из всех файлов..."
  
  # Проверяем размер всех ответов
  ALL_RESPONSES_SIZE=$(echo "$ALL_RESPONSES" | wc -c)
  echo "📊 Размер всех частей: $ALL_RESPONSES_SIZE bytes"
  
  # Объединяем все отзывы по файлам в итоговый отчет
  echo "📦 Используем структурированное объединение всех файлов"
  AI_RESPONSE="# 🔍 AI Code Review (Полный анализ MR по файлам)\n\n"
  AI_RESPONSE="$AI_RESPONSE> MR был проанализирован по файлам для детального анализа.\n"
  AI_RESPONSE="$AI_RESPONSE> Всего файлов: $FILES_COUNT\n"
  AI_RESPONSE="$AI_RESPONSE> Ниже представлены все замечания по каждому файлу.\n\n"
  AI_RESPONSE="$AI_RESPONSE---\n\n"
  AI_RESPONSE="$AI_RESPONSE$ALL_RESPONSES\n\n"
  AI_RESPONSE="$AI_RESPONSE---\n\n"
  AI_RESPONSE="$AI_RESPONSE## 📋 Общие рекомендации\n\n"
  AI_RESPONSE="$AI_RESPONSE- Внимательно просмотрите ВСЕ файлы выше\n"
  AI_RESPONSE="$AI_RESPONSE- Приоритезируйте критичные замечания по Архитектуре и Безопасности\n"
  AI_RESPONSE="$AI_RESPONSE- Обратите внимание на производительность и тестирование\n"
  
  # ============================================================================
  # 2.3. Создаем итоговый комментарий с ссылками на все файлы
  # ============================================================================
  echo "💬 Формируем итоговый комментарий со ссылками на все файлы..."
  
  # Формируем список ссылок на все файлы
  FILES_LINKS=""
  while read -r FILE_PATH; do
    SAFE_FILE_NAME=$(echo "$FILE_PATH" | sed 's/\//-/g' | sed 's/\./-/g')
    FILE_REPORT_FILE="ai-review-mr-${CI_MERGE_REQUEST_IID}-file-${SAFE_FILE_NAME}.md"
    FILE_URL="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/file/$FILE_REPORT_FILE"
    FILE_RAW="$CI_SERVER_URL/$CI_PROJECT_PATH/-/jobs/$CI_JOB_ID/artifacts/raw/$FILE_REPORT_FILE?inline=false"
    FILES_LINKS="$FILES_LINKS\n- 📄 [\`$FILE_PATH\`]($FILE_URL) | [💾 Скачать]($FILE_RAW)"
  done < <(echo "$DIFF_RAW" | jq -r '.changes[] | .new_path // .old_path')
  
  # Формируем краткое резюме для итогового комментария
  SUMMARY_COMMENT="🔍 **Full MR Analysis - Итоговая сводка** (Model: $FULL_MR_REVIEW_MODEL)\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT**MR:** !$CI_MERGE_REQUEST_IID\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT**Diff Size:** $DIFF_SIZE bytes\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT**Files:** $FILES_COUNT файлов\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT---\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT## 📦 Детальные отчеты по файлам\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT$FILES_LINKS\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT---\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT## 📋 Краткие выводы\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT- ✅ Анализ завершен: все $FILES_COUNT файлов проверены\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT- 📄 Каждый файл доступен как отдельный MD файл (см. ссылки выше)\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT- 📊 Итоговый сводный отчет доступен ниже\n\n"
  SUMMARY_COMMENT="$SUMMARY_COMMENT*Pipeline ID: $CI_PIPELINE_ID*"
  
  # Отправляем итоговый комментарий
  SUMMARY_COMMENT_BODY=$(jq -n --arg body "$SUMMARY_COMMENT" '{body: $body}')
  
  echo "💬 Отправка итогового комментария с ссылками на все файлы..."
  curl -s --request POST \
    --header "PRIVATE-TOKEN: $API_TOKEN" \
    --header "Content-Type: application/json" \
    --data "$SUMMARY_COMMENT_BODY" \
    "$CI_SERVER_URL/api/v4/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes" > /dev/null
  
  echo "✅ Итоговый комментарий со ссылками отправлен"
  
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
  
  SYSTEM_PROMPT="Ты опытный Go разработчик проводящий ПОЛНЫЙ CODE REVIEW ИЗМЕНЕНИЙ MR.

⚠️ КРИТИЧНО - CODE REVIEW, НЕ ОПИСАНИЕ:
- Это CODE REVIEW изменений в MR diff (что добавлено/удалено/изменено)
- НЕ описывай функциональность кода или алгоритмы
- НЕ объясняй \"что делает этот код\"
- ТОЛЬКО анализ ИЗМЕНЕНИЙ: файл:строка → проблема → решение
- Анализируй ТОЛЬКО предоставленный diff

Задача: Проанализируй ВСЕ изменения детально. Укажи конкретные файлы и строки из diff.

Фокус анализа:
- Архитектура изменений (как новый код вписывается)
- Безопасность (уязвимости в новом/измененном коде)
- Производительность (проблемы в новых алгоритмах)
- Читаемость (качество новых/измененных функций)
- Go best practices (идиоматичность изменений)
- Тестирование (есть ли тесты для новой функциональности)

Формат ответа (Markdown):

## Архитектура
- файл:строка - как изменения влияют на архитектуру

## Безопасность
- файл:строка - уязвимости в новом/измененном коде

## Производительность
- файл:строка - проблемы производительности

## Замечания
- файл:строка - конкретная проблема → рекомендация

## Рекомендации
- как улучшить изменения

Отвечай на русском языке подробно."
  
  REQUEST_JSON=$(jq -n \
    --arg model "$FULL_MR_REVIEW_MODEL" \
    --arg system "$SYSTEM_PROMPT" \
    --arg user "CODE REVIEW задача: детально проанализируй ВСЕ изменения ниже.

НЕ описывай что делает код!
НЕ объясняй алгоритмы!
ТОЛЬКО анализ изменений (файл:строка → проблема → fix).

Full MR diff для анализа:\n\n$DIFF_TEXT" \
    '{
      "model": $model,
      "messages": [
        {"role": "system", "content": $system},
        {"role": "user", "content": $user}
      ],
      "temperature": 0.1,
      "max_tokens": 10000,
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
# 5. Добавляем комментарий в MR (ТОЛЬКО для single-file случая)
# ============================================================================
# Для file-based случая комментарии уже отправлены (каждый файл + итоговая сводка)
if [ "$WAS_SPLIT_BY_FILES" = "false" ]; then
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
else
  echo "ℹ️ Комментарии для file-based анализа уже отправлены (каждый файл + итоговая сводка)"
fi

