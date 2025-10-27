# Migration Guide: v2.0.0 → v2.1.0 (AIGateway Rebranding)

**Version:** 2.1.0  
**Release Date:** 2025-10-27  
**Migration Difficulty:** Easy (mostly configuration changes)  
**Downtime Required:** ~5-10 minutes

---

## 🎯 Overview

Version 2.1.0 is a **rebranding release**. The project has been renamed from **"Ollama-OpenAI Proxy"** to **"AIGateway Platform"** to better reflect its expanded capabilities.

### What Changed?
- ✅ Project name и repository
- ✅ Docker container names
- ✅ Environment variable prefixes (`PROXY_*` → `AIGATEWAY_*`)
- ✅ Database и log file names
- ✅ Go module path

### What DIDN'T Change?
- ✅ All functionality remains identical
- ✅ API endpoints unchanged
- ✅ Database schema unchanged
- ✅ Configuration structure unchanged
- ✅ WebUI functionality unchanged

**This is a pure rebranding release with ZERO functional changes.**

---

## 📋 Migration Steps

### Step 1: Backup Current Setup

```bash
# Backup database
cp data/proxy.db data/proxy.db.backup

# Backup configuration
cp configs/production.yaml configs/production.yaml.backup

# Backup docker-compose.yml (if using Docker)
cp docker-compose.yml docker-compose.yml.backup
```

### Step 2: Stop Running Services

```bash
# If using Docker Compose
docker-compose down

# If running as systemd service
sudo systemctl stop ollama-proxy

# If running manually
pkill -f "ollama-openai-proxy"
```

### Step 3: Update Docker Compose (если applicable)

Edit your `docker-compose.yml`:

```yaml
# OLD:
services:
  proxy:
    container_name: ollama-openai-proxy
    environment:
      - PROXY_SERVER_PORT=8080
      - PROXY_OLLAMA_URL=http://ollama:11434
      # ... more PROXY_* variables
    volumes:
      - proxy_data:/app/data
      - proxy_logs:/app/logs

# NEW:
services:
  aigateway:  # Changed service name
    container_name: aigateway  # Changed container name
    environment:
      - AIGATEWAY_SERVER_PORT=8080  # Changed variable prefix
      - AIGATEWAY_OLLAMA_URL=http://ollama:11434
      # ... all PROXY_* → AIGATEWAY_*
    volumes:
      - aigateway_data:/app/data  # Changed volume names
      - aigateway_logs:/app/logs
```

**Full list of environment variable changes:**

| Old Variable | New Variable |
|-------------|-------------|
| `PROXY_SERVER_PORT` | `AIGATEWAY_SERVER_PORT` |
| `PROXY_SERVER_HOST` | `AIGATEWAY_SERVER_HOST` |
| `PROXY_OLLAMA_URL` | `AIGATEWAY_OLLAMA_URL` |
| `PROXY_OLLAMA_TIMEOUT` | `AIGATEWAY_OLLAMA_TIMEOUT` |
| `PROXY_LOGGING_LEVEL` | `AIGATEWAY_LOGGING_LEVEL` |
| `PROXY_LOGGING_FORMAT` | `AIGATEWAY_LOGGING_FORMAT` |
| `PROXY_LOGGING_FILE` | `AIGATEWAY_LOGGING_FILE` |
| `PROXY_AUTH_ENABLED` | `AIGATEWAY_AUTH_ENABLED` |
| `PROXY_AUTH_ADMIN_KEY` | `AIGATEWAY_AUTH_ADMIN_KEY` |
| `PROXY_AUTH_JWT_SECRET` | `AIGATEWAY_AUTH_JWT_SECRET` |
| `PROXY_DATABASE_TYPE` | `AIGATEWAY_DATABASE_TYPE` |
| `PROXY_DATABASE_SQLITE_PATH` | `AIGATEWAY_DATABASE_SQLITE_PATH` |

... and all other `PROXY_*` variables follow the same pattern.

### Step 4: Update Configuration Files

If you use YAML configs with environment variable substitution:

```yaml
# configs/production.yaml
# Environment variables are now AIGATEWAY_* instead of PROXY_*
# The config structure itself is unchanged
```

### Step 5: Rename Database Files (Optional)

```bash
# Optional: rename database file для consistency
cd data/
mv proxy.db aigateway.db

# If renamed, update docker-compose.yml or config:
# AIGATEWAY_DATABASE_SQLITE_PATH=/app/data/aigateway.db
```

**Note:** This step is optional. The application works fine with the old filename.

### Step 6: Pull New Docker Image

```bash
# Pull new image
docker pull yourusername/aigateway:2.1.0

# Or rebuild from source
docker-compose build
```

### Step 7: Update Systemd Service (if applicable)

If using systemd:

```bash
# Edit service file
sudo vi /etc/systemd/system/aigateway.service

# Update ExecStart path and environment variables
[Service]
ExecStart=/usr/local/bin/aigateway -config /etc/aigateway/production.yaml
Environment="AIGATEWAY_SERVER_PORT=8080"
# ... update all PROXY_* → AIGATEWAY_*

# Reload и restart
sudo systemctl daemon-reload
sudo systemctl start aigateway
sudo systemctl enable aigateway
```

### Step 8: Update Nginx/Reverse Proxy (if applicable)

If you have nginx config:

```nginx
# OLD:
upstream ollama_proxy {
    server localhost:8080;
}

# NEW (optional - can keep old name):
upstream aigateway {
    server localhost:8080;
}
```

**Note:** Upstream name is arbitrary - you can keep the old name if desired.

### Step 9: Start Services

```bash
# Using Docker Compose
docker-compose up -d

# Using systemd
sudo systemctl start aigateway

# Check logs
docker-compose logs -f aigateway
# or
sudo journalctl -u aigateway -f
```

### Step 10: Verify Migration

```bash
# Test API endpoint
curl http://localhost:8080/api/v1/models

# Check WebUI
open http://localhost:8080/login

# Verify environment
docker exec aigateway env | grep AIGATEWAY

# Check database
docker exec aigateway ls -lh /app/data/
```

---

## 🔧 Troubleshooting

### Issue: Service won't start after migration

**Solution:**
```bash
# Check logs
docker-compose logs aigateway

# Common issues:
# 1. Old container name conflict
docker rm -f ollama-openai-proxy

# 2. Volume mount issues
docker volume ls | grep proxy
docker volume ls | grep aigateway
```

### Issue: Environment variables not recognized

**Solution:**
```bash
# Verify all PROXY_* → AIGATEWAY_*
docker-compose config | grep AIGATEWAY

# If using .env file:
vi .env
# Update all PROXY_* variables
```

### Issue: Database file not found

**Solution:**
```bash
# Check database location
docker exec aigateway ls -lh /app/data/

# If you renamed proxy.db → aigateway.db,
# ensure config points to correct filename
```

### Issue: Existing API clients fail

**Solution:**
API endpoints are **unchanged**. Client code does NOT need updates.

```bash
# This still works:
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2","messages":[{"role":"user","content":"Hello"}]}'
```

---

## 🔄 Rollback Procedure

If you need to rollback to v2.0.0:

```bash
# Stop v2.1.0
docker-compose down

# Restore backups
cp configs/production.yaml.backup configs/production.yaml
cp docker-compose.yml.backup docker-compose.yml

# Pull old image
docker pull yourusername/ollama-openai-proxy:2.0.0

# Update docker-compose.yml to use old image
image: yourusername/ollama-openai-proxy:2.0.0

# Start old version
docker-compose up -d
```

---

## 📝 Post-Migration Checklist

- [ ] Service starts successfully
- [ ] WebUI accessible at http://localhost:8080/login
- [ ] API endpoints respond correctly
- [ ] Existing users can login
- [ ] Conversations history preserved
- [ ] API keys still work
- [ ] RAG data sources functional (if using RAG)
- [ ] Logs writing to correct location
- [ ] Database file accessible

---

## 🚀 Next Steps

After successful migration to v2.1.0, you're ready for:
- **v2.2.0** - Model Registry & vLLM Integration
- **v2.3.0** - HuggingFace Hub integration

See [Roadmap.MD](../Roadmap.MD) для details.

---

## 💡 Tips

### Update GitHub Workflows

If you have CI/CD pipelines:

```yaml
# .github/workflows/deploy.yml
# OLD:
docker build -t ollama-openai-proxy:latest .

# NEW:
docker build -t aigateway:latest .
```

### Update Documentation Links

Search your documentation for:
- `ollama-openai-proxy` → `aigateway`
- Old container names
- Old environment variable names

### Notify Team Members

If working in a team:
1. ✅ Share this migration guide
2. ✅ Update team documentation
3. ✅ Coordinate deployment timing
4. ✅ Plan for ~10 minutes downtime

---

## 📧 Support

If you encounter issues during migration:
- 📖 Check [docs/TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- 🐛 Open GitHub Issue with migration tag
- 💬 Ask in GitHub Discussions

---

**Migration completed? Welcome to AIGateway Platform! 🎉**

