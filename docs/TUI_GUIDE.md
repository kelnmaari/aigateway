# 🖥️ Terminal User Interface (TUI) Guide

Complete guide to using the Ollama-OpenAI Proxy Terminal User Interface.

## 📑 Table of Contents

- [Getting Started](#getting-started)
- [Navigation](#navigation)
- [Dashboard Screen](#1-dashboard-screen)
- [Requests Screen](#2-requests-screen)
- [Models Screen](#3-models-screen)
- [API Keys Screen](#4-api-keys-screen)
- [Configuration Screen](#5-configuration-screen)
- [Logs Screen](#6-logs-screen)
- [Control Screen](#7-control-screen)
- [Keyboard Shortcuts](#keyboard-shortcuts)
- [Tips & Tricks](#tips--tricks)

---

## Getting Started

### Launch TUI

```bash
# From project root
./bin/tui.exe

# Or with custom server URL
./bin/tui.exe --server-url=http://remote-server:8080
```

### First Time Setup

1. **Ensure the proxy server is running** in another terminal:

   ```bash
   ./bin/server.exe
   ```

2. **Launch TUI**:

   ```bash
   ./bin/tui.exe
   ```

3. **Navigate** using number keys `1-7`

4. **Refresh** data anytime with `r`

5. **Quit** with `q` or `Ctrl+C`

---

## Navigation

The TUI consists of 7 main screens:

| Key | Screen | Description |
|-----|--------|-------------|
| `1` | Dashboard | Overview, metrics, statistics |
| `2` | Requests | Request monitoring (placeholder) |
| `3` | Models | Available Ollama models |
| `4` | API Keys | API key management |
| `5` | Config | Server configuration viewer |
| `6` | Logs | Real-time log viewer |
| `7` | Control | Server status and control |

### Header

```
🦙 Ollama-OpenAI Proxy TUI
v1.0.0 | Обновлено: 12:34:56

[1: Dashboard] [2: Requests] [3: Models] [4: API Keys] [5: Config] [6: Logs] [7: Control]
                   └── Active screen is highlighted
```

### Footer

```
Прокрутка: ↑↓/j/k/PgUp/PgDn | Навигация: 1-7 | Обновить: r | Выход: q
```

---

## 1. Dashboard Screen

**Key:** `1`

Overview of the proxy server status and metrics.

### Sections

#### 📊 SERVER STATUS

- HTTP Server status (running/stopped)
- Server URL and uptime
- Ollama connection status
- Available models count

#### 📈 STATISTICS

- Total requests processed
- Success/Error counts
- Active requests currently in progress
- Average response duration

#### 🔑 API KEYS

- Total keys count
- Enabled keys count
- Admin keys count

#### 📊 PROMETHEUS METRICS

(If metrics are enabled)

- HTTP Requests total and in-flight
- Ollama Requests total and errors
- Average HTTP latency (ms)
- Average Ollama latency (ms)
- Total tokens used

### Example Output

```
📊 DASHBOARD

📊 СЕРВЕР
• Статус: ✅ Запущен
• URL: http://localhost:8080
• Uptime: 2h15m30s
• Ollama: ✅ Подключен (15 моделей)

📈 СТАТИСТИКА
• Всего запросов: 1,234
• Успешных: 1,180 (95.6%)
• Ошибок: 54 (4.4%)
• Активных: 3
• Средняя длительность: 1.2s

🔑 API KEYS
• Всего ключей: 5
• Активных: 4
• Админ ключей: 1

📊 PROMETHEUS METRICS
• HTTP Requests: 1234 (3 in-flight)
• Ollama Requests: 1100 (5 errors)
• Avg HTTP Latency: 45.32 ms
• Avg Ollama Latency: 2345.67 ms
• Total Tokens Used: 456789
```

---

## 2. Requests Screen

**Key:** `2`

Real-time request monitoring.

### Status

⚠️ **Currently a placeholder** - будет реализовано в будущих версиях.

### Planned Features

- Real-time request list
- Request details (method, endpoint, model, duration)
- Filtering by status, model, API key
- Search functionality
- Request body inspection

---

## 3. Models Screen

**Key:** `3`

View all available Ollama models.

### Information Displayed

For each model:

- **Name** - Full model name (e.g., `qwen2.5-coder:7b`)
- **Size** - Model size in GB
- **Family** - Model family (llama, qwen, etc.)
- **Format** - Quantization format (Q4_K_M, etc.)
- **Modified** - Last modification date

### Features

- Automatic refresh on screen switch
- Displays models from connected Ollama server
- Shows connection errors if Ollama is unavailable

### Example Output

```
🤖 ДОСТУПНЫЕ МОДЕЛИ

Подключено: http://localhost:11434
Всего моделей: 5

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 NAME                    SIZE      FAMILY     FORMAT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 llama3.1:latest         4.9 GB    llama      Q4_K_M
 qwen2.5-coder:7b        4.7 GB    qwen2      Q4_K_M
 qwen2.5-coder:30b       20 GB     qwen2      Q4_K_M
 nomic-embed-text        274 MB    nomic-bert F16
 mistral:latest          4.1 GB    mistral    Q4_0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Troubleshooting

**No models shown:**

1. Check Ollama is running: `ollama serve`
2. Verify URL in config: `configs/dev.yaml` → `ollama.url`
3. Test connection: `curl http://localhost:11434/api/tags`

---

## 4. API Keys Screen

**Key:** `4`

Manage API keys - view existing keys and create new ones.

### View API Keys

Shows a table of all API keys:

| Column | Description |
|--------|-------------|
| **Name** | Key descriptive name |
| **Key ID** | Unique key identifier (e.g., `ak_1234567890_abcdef`) |
| **Status** | Enabled/Disabled |
| **Permissions** | Allowed models (`*` for all) |
| **Rate Limit** | Requests/min, Requests/hour |
| **Usage** | Last used date |

#### Example

```
🔑 API KEYS MANAGEMENT

Всего ключей: 3 | Активных: 2

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 NAME              KEY ID             STATUS    MODELS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Admin Key         ak_1759404858...   ✅ Enabled  *
 Dev Key           ak_1759405123...   ✅ Enabled  qwen2.5-coder:7b
 Test Key          ak_1759405456...   ❌ Disabled *
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💡 Нажмите 'n' чтобы создать новый ключ
```

### Create New API Key

**Keyboard shortcut:** `n` (on API Keys screen)

#### Step-by-Step Process

1. **Press `n`** to open the create key form

2. **Fill in the form** (use `Tab` to move between fields):

   **Field 1: Name**

   ```
   NAME: █
   Enter a descriptive name (e.g., "Production App Key")
   ```

   **Field 2: Permissions (Models)**

   ```
   MODELS: █
   - Enter `*` for all models
   - Or specific models: `llama3.1,qwen2.5-coder:7b`
   ```

   **Field 3: Rate Limits** (optional)

   ```
   RATE LIMITS: █
   Format: requests_per_minute,requests_per_hour
   Example: 30,500
   Leave empty for default limits
   ```

3. **Submit** by pressing `Enter`

4. **View & Copy Key** - The plaintext key will be displayed:

   ```
   ✅ API KEY CREATED SUCCESSFULLY!

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🔑 YOUR NEW API KEY
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   Name: Production App Key
   Key ID: ak_1759406789_c620cc75

   ⚠️  IMPORTANT: Save this key securely!
   It will NOT be shown again!

   sk-1759406789-c620cc75-7a2b9d4e8f1c3a6b

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   💡 Press 'c' to copy to clipboard
   💡 Press any other key to close
   ```

5. **Copy to Clipboard** by pressing `c`

6. **Close** by pressing any other key

#### Form Controls

- `Tab` - Next field
- `Shift+Tab` - Previous field
- `Backspace` - Delete character
- `Enter` - Submit form
- `Esc` - Cancel and close form

### API Key Security

⚠️ **Important Security Notes:**

1. **Save keys immediately** - they are shown only once
2. **Store securely** - use environment variables or secrets manager
3. **Never commit keys** to version control
4. **Rotate regularly** - create new keys, delete old ones
5. **Use specific permissions** - avoid `*` in production
6. **Monitor usage** - check "Last Used" column

---

## 5. Configuration Screen

**Key:** `5`

View the complete server configuration.

### Sections Displayed

#### 🌐 SERVER

- Host and port
- Timeouts (read, write, idle)
- Max header bytes

#### 🦙 OLLAMA

- Ollama URL
- Connection timeout
- Retry attempts and delay
- Connection pool size
- Keep-alive setting

#### 🔐 AUTHENTICATION

- Enabled status
- Storage type (JSON/SQLite)
- Storage path
- Rate limiting enabled
- Default rate limits

#### 📝 LOGGING

- Log level (debug/info/warn/error)
- Log format (text/json)
- Output (stdout/file/both)
- File path
- Rotation settings (max size, backups, age, compress)

#### 🤖 MODELS

- Model mappings count
- Aliases count
- Hidden models count
- Cache enabled
- Cache TTL and refresh interval

#### 🔧 TOOLS (Function Calling)

- Force usage setting
- Default choice (auto/required/none)
- Fallback model

#### ⚡ PROMPT OPTIMIZER

- Enabled status
- Simplify system message
- Smart tool filtering
- Max tools per request
- Preserve instructions count

#### 📊 METRICS

- Enabled status
- Prometheus path

#### 🖥️ TUI

- Enabled status
- Refresh rate
- Theme

#### 🔬 DEVELOPMENT

- Hot reload
- Debug mode
- Profiling
- PProf
- Race detection

### Example Output

```
⚙️ КОНФИГУРАЦИЯ СЕРВЕРА

🌐 SERVER
• Host: 0.0.0.0
• Port: 8080
• Read Timeout: 30s
• Write Timeout: 180s
• Idle Timeout: 120s
• Max Header Bytes: 1048576

🦙 OLLAMA
• URL: http://localhost:11434
• Timeout: 180s
• Retry Attempts: 3
• Retry Delay: 2s
• Connection Pool: 100
• Keep Alive: true

🔐 AUTHENTICATION
• Enabled: true
• Storage Type: json
• Storage Path: data/api_keys.json
• Rate Limiting: true
  - Requests/Minute: 30
  - Requests/Hour: 500

... (continues with all sections)
```

### Scrolling

The configuration screen may be longer than your terminal height. Use:

- `↑`/`↓` or `j`/`k` - Scroll line by line
- `PgUp`/`PgDn` - Scroll 10 lines at a time
- `Home` - Jump to top

---

## 6. Logs Screen

**Key:** `6`

Real-time log viewer with color coding.

### Features

- **Last 100 lines** from `logs/proxy-dev.log`
- **Color coding** by log level:
  - 🔴 **ERROR** - Red
  - 🟡 **WARN** - Yellow
  - ⚪ **INFO** - Default
  - 🔵 **DEBUG** - Gray
- **Statistics** - Count by level (ERROR: N, WARN: M, etc.)
- **Auto-refresh** - Updates every second
- **Scrolling** - Navigate through logs

### Example Output

```
📋 ПРОСМОТР ЛОГОВ СЕРВЕРА

📊 Статистика: ERROR: 2 WARN: 5 INFO: 85 DEBUG: 8

📄 Последние 100 строк (из logs/proxy-dev.log):

time="2025-10-04 12:30:15" level=info msg="Server started" port=8080
time="2025-10-04 12:30:16" level=info msg="Connected to Ollama" url="http://localhost:11434"
time="2025-10-04 12:30:20" level=debug msg="API key validated" key_id="ak_123..."
time="2025-10-04 12:30:21" level=warning msg="Rate limit approaching" key_id="ak_456..."
time="2025-10-04 12:30:25" level=error msg="Ollama timeout" model="qwen2.5-coder:30b"
...

⬇️  Прокрутите вниз (↓/j/PgDn)

💡 Используйте прокрутку (↑↓/PgUp/PgDn) для просмотра всех логов
```

### Scrolling

- `↑`/`↓` or `j`/`k` - Scroll line by line
- `PgUp`/`PgDn` - Scroll 10 lines at a time
- `Home` - Jump to beginning

### Troubleshooting

**"Ошибка чтения логов":**

- Ensure server is running
- Check log file exists: `logs/proxy-dev.log`
- Verify file permissions

**Logs not updating:**

- Press `r` to manually refresh
- Check TUI refresh rate in config

---

## 7. Control Screen

**Key:** `7`

Server status overview and useful commands.

### Sections

#### 🌐 HTTP SERVER

- Status (running/unavailable)
- Server address
- Uptime
- Start time

#### 🦙 OLLAMA CONNECTION

- Status (connected/unavailable)
- Ollama URL
- Number of available models
- Model list (first 3 models shown)

#### 🖥️ TUI STATUS

- Status (always active when TUI is running)
- Refresh rate
- Last update time

#### 📊 STATISTICS

- Total requests
- Successful requests
- Error requests
- Active requests
- Average duration
- Active API keys count

#### 💡 USEFUL COMMANDS

Quick reference for common administration tasks:

- Server restart (systemd)
- Real-time log viewing (tail)
- Ollama connection test (curl)

#### 🎮 QUICK ACTIONS

- `[R]` Refresh statistics
- `[1-7]` Navigate to other screens
- `[Q]` Quit TUI

### Example Output

```
🎛️ УПРАВЛЕНИЕ СЕРВЕРОМ

🌐 HTTP SERVER
• Статус: ✅ Активен
• Адрес: http://localhost:8080
• Uptime: 2h15m30s
• Запущен: 2025-10-04 10:15:25

🦙 OLLAMA CONNECTION
• Статус: ✅ Подключен
• URL: http://localhost:11434
• Доступно моделей: 5
• Модели: llama3.1:latest, qwen2.5-coder:7b, mistral:latest
  ... и еще 2

🖥️ TUI STATUS
• Статус: ✅ Активен
• Refresh Rate: 1s
• Последнее обновление: 12:30:45

📊 СТАТИСТИКА
• Всего запросов: 1234
• Успешных: 1180
• Ошибок: 54
• Активных: 3
• Средняя длительность: 1.2s
• Активных ключей: 4

💡 ПОЛЕЗНЫЕ КОМАНДЫ
Перезапуск сервера (Linux):
  systemctl restart ollama-proxy
Просмотр логов в реальном времени:
  tail -f logs/proxy-dev.log
Проверка Ollama:
  curl http://localhost:11434/api/tags

🎮 БЫСТРЫЕ ДЕЙСТВИЯ
• [R] Обновить статистику
• [1-7] Перейти к другим экранам
• [Q] Выход из TUI
```

---

## Keyboard Shortcuts

### Global Shortcuts

(Available on all screens)

| Key | Action |
|-----|--------|
| `1` | Dashboard screen |
| `2` | Requests screen |
| `3` | Models screen |
| `4` | API Keys screen |
| `5` | Configuration screen |
| `6` | Logs screen |
| `7` | Control screen |
| `r` | Refresh data manually |
| `q` | Quit TUI |
| `Ctrl+C` | Quit TUI (alternative) |
| `Esc` | Clear error message |

### Scrolling Shortcuts

(Available on screens with scrollable content)

| Key | Action |
|-----|--------|
| `↑` or `k` | Scroll up (1 line) |
| `↓` or `j` | Scroll down (1 line) |
| `PgUp` | Scroll up (10 lines) |
| `PgDn` | Scroll down (10 lines) |
| `Home` | Jump to top |

### API Keys Screen Shortcuts

| Key | Action |
|-----|--------|
| `n` | Create new API key |
| `c` | Copy displayed key to clipboard |
| `Tab` | Next form field (in create form) |
| `Shift+Tab` | Previous form field (in create form) |
| `Enter` | Submit form |
| `Esc` | Cancel form / close key display |

---

## Tips & Tricks

### 1. Multiple TUI Instances

You can run multiple TUI instances simultaneously:

```bash
# Terminal 1: Monitor main metrics
./bin/tui.exe

# Terminal 2: Watch logs
./bin/tui.exe
# Then press '6' to go to Logs screen
```

### 2. Remote Server Monitoring

Monitor a remote proxy server:

```bash
./bin/tui.exe --server-url=http://10.0.0.5:8080
```

### 3. Quick Log Analysis

1. Open TUI → Press `6` (Logs)
2. Look at statistics at the top
3. Scroll to ERROR entries (shown in red)
4. Use `Home` to jump back to top

### 4. Efficient API Key Creation

Batch create keys by keeping TUI open:

1. Press `4` (API Keys)
2. Press `n` (New key)
3. Fill form → `Enter`
4. Press `c` to copy → Paste in secure location
5. Press any key to close → Repeat from step 2

### 5. Performance Monitoring

Watch performance metrics in real-time:

1. Press `1` (Dashboard)
2. Check "PROMETHEUS METRICS" section
3. Monitor:
   - HTTP latency (should be < 100ms)
   - Ollama latency (varies by model)
   - In-flight requests (watch for spikes)
   - Error counts (investigate if increasing)

### 6. Health Checks

Quick health check routine:

1. Press `7` (Control)
2. Verify:
   - ✅ HTTP Server: Active
   - ✅ Ollama Connection: Connected
   - Error count is low
   - No red error messages

### 7. Configuration Verification

Before changing config file:

1. Press `5` (Config)
2. Scroll through sections
3. Verify current values
4. Make changes to `configs/dev.yaml`
5. Restart server
6. Check Config screen again

### 8. Troubleshooting Workflow

When things go wrong:

1. **Check Control Screen** (`7`) - Is everything connected?
2. **Check Logs Screen** (`6`) - Any ERROR messages?
3. **Check Dashboard** (`1`) - Are requests failing?
4. **Check Models Screen** (`3`) - Is Ollama available?

### 9. Terminal Size Optimization

For best experience:

- **Minimum**: 80x24 characters
- **Recommended**: 120x40 characters
- **Optimal**: 160x50+ characters (for Config/Logs screens)

### 10. Auto-Refresh Rate

Adjust refresh rate in config:

```yaml
tui:
  refresh_rate: "1s"  # Faster updates (more CPU)
  # or
  refresh_rate: "5s"  # Slower updates (less CPU)
```

---

## Troubleshooting

### TUI Issues

#### "Connection refused"

**Problem**: TUI can't connect to proxy server.

**Solution**:

1. Ensure proxy server is running: `./bin/server.exe`
2. Check server URL: `http://localhost:8080/api/stats`
3. Verify firewall settings

#### "Failed to fetch statistics"

**Problem**: TUI shows error banner at top.

**Solution**:

1. Press `Esc` to clear error
2. Press `r` to manually refresh
3. Check server logs: `tail -f logs/proxy-dev.log`

#### TUI is blank/frozen

**Problem**: TUI doesn't render or is unresponsive.

**Solution**:

1. `Ctrl+C` to quit
2. Check terminal size: `echo $COLUMNS x $LINES`
3. Resize terminal to at least 80x24
4. Relaunch TUI

#### Scrolling doesn't work

**Problem**: Can't scroll long content.

**Solution**:

- Ensure you're on a screen with scrollable content (Config, Logs)
- Try different keys: `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn`
- Check if content is actually longer than screen

### API Key Creation Issues

#### Key not shown after creation

**Problem**: Created key but plaintext wasn't displayed.

**Solution**:

- Check for error message (red banner at top)
- Look in server logs for API key creation event
- **Cannot retrieve key** - it's hashed in storage
- Create a new key

#### Can't copy key to clipboard

**Problem**: `c` key doesn't copy.

**Solution**:

1. Manually select and copy text from terminal
2. Check clipboard library is installed (Linux: `xclip` or `xsel`)
3. On Windows, should work natively

### Display Issues

#### Colors not showing

**Problem**: All text is same color.

**Solution**:

- Use a terminal with 256-color support
- Try different terminal emulator (Windows Terminal, iTerm2, Alacritty)

#### Text is cut off

**Problem**: Content doesn't fit in terminal.

**Solution**:

- Resize terminal window (larger)
- Use scrolling: `↑`/`↓`, `PgUp`/`PgDn`
- Maximize terminal window

---

## FAQ

**Q: Can I run TUI without starting the server first?**

A: No, TUI requires the proxy server to be running. Start `./bin/server.exe` first.

**Q: Can I manage API keys only through TUI?**

A: No, you can also create keys via admin API endpoint. TUI is just a convenient interface.

**Q: Does TUI show real-time data?**

A: Yes, TUI polls the server every second (configurable via `tui.refresh_rate`).

**Q: Can I use TUI in a Docker container?**

A: Yes, but you need to attach to the container with a TTY: `docker exec -it container_name ./bin/tui`

**Q: Does TUI work over SSH?**

A: Yes! SSH supports full TUI functionality including colors and scrolling.

**Q: Can I customize TUI colors/theme?**

A: Not yet - theme customization is planned for future releases.

**Q: Does TUI log its own activities?**

A: No, TUI is read-only and doesn't perform actions that need logging (except API key creation).

**Q: What happens if server crashes while TUI is running?**

A: TUI will show "Connection refused" error. Restart the server, then press `r` in TUI to reconnect.

---

## Feedback & Support

Found a bug or have a feature request for TUI?

- **GitHub Issues**: <https://github.com/yourusername/ollama-openai-proxy/issues>
- **Discussions**: <https://github.com/yourusername/ollama-openai-proxy/discussions>

---

**Happy monitoring! 🖥️✨**
