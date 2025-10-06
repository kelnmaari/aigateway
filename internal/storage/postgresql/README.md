# PostgreSQL Storage Driver

## 📊 Статус реализации

✅ **Завершено:**

- `postgresql.go` - Connection, Transactions, Migrations (полностью)
- `users.go` - Users CRUD (полностью)
- Зависимость `github.com/lib/pq v1.10.9` добавлена в go.mod

⏳ **Требует реализации:**

- `tenants.go` - Tenants & TenantMembers CRUD
- `apikeys.go` - API Keys CRUD
- `conversations.go` - Conversations & Messages CRUD
- `usage.go` - Usage Stats CRUD
- `transactions.go` - Transaction delegation

## 🔧 Как адаптировать SQLite → PostgreSQL

Все CRUD файлы можно адаптировать из `internal/storage/sqlite/` с помощью следующих замен:

### 1. Package и типы

```bash
# Заменить package name
sqlite -> postgresql

# Заменить типы
SQLiteDB -> PostgreSQLDB
sqliteTx -> postgresqlTx

# Заменить receiver
(s *SQLiteDB) -> (p *PostgreSQLDB)
(tx *sqliteTx) -> (tx *postgresqlTx)

# Заменить все s. на p.
s.db -> p.db
s.logger -> p.logger
s.config -> p.config
```

### 2. SQL Placeholder'ы

PostgreSQL использует **numbered placeholders**: `$1, $2, $3` вместо `?`

**SQLite:**

```sql
INSERT INTO users (id, username, email) VALUES (?, ?, ?)
```

**PostgreSQL:**

```sql
INSERT INTO users (id, username, email) VALUES ($1, $2, $3)
```

**Для динамических запросов:**

```go
// SQLite
paramIndex := 1
query += " AND status = ?"
args = append(args, status)

// PostgreSQL
paramIndex := 1
query += fmt.Sprintf(" AND status = $%d", paramIndex)
args = append(args, status)
paramIndex++
```

### 3. BOOLEAN значения

```bash
# SQLite
DEFAULT 0 -> DEFAULT FALSE
DEFAULT 1 -> DEFAULT TRUE

# В Go коде
0 -> FALSE
1 -> TRUE
```

### 4. Case-insensitive поиск

```bash
# SQLite
LIKE -> ILIKE

# Example
WHERE username LIKE ? -> WHERE username ILIKE $1
```

### 5. JSON поля

```bash
# SQLite
TEXT (для JSON) -> JSONB (native JSON в PostgreSQL)

# В схеме
preferences TEXT -> preferences JSONB
metadata TEXT -> metadata JSONB
```

## 🚀 Быстрая адаптация (пример)

### tenants.go

```bash
# 1. Скопировать файл
cp internal/storage/sqlite/tenants.go internal/storage/postgresql/tenants.go

# 2. Заменить package и типы
sed -i 's/package sqlite/package postgresql/g' internal/storage/postgresql/tenants.go
sed -i 's/SQLiteDB/PostgreSQLDB/g' internal/storage/postgresql/tenants.go
sed -i 's/(s \*/\(p */g' internal/storage/postgresql/tenants.go
sed -i 's/s\./p./g' internal/storage/postgresql/tenants.go

# 3. Заменить placeholders вручную (требует attention!)
# Открыть файл и заменить все ? на $1, $2, $3 последовательно
```

### Автоматизация замены placeholders

Можно использовать следующую Go функцию:

```go
func convertPlaceholders(query string) string {
    count := strings.Count(query, "?")
    result := query
    for i := 1; i <= count; i++ {
        result = strings.Replace(result, "?", fmt.Sprintf("$%d", i), 1)
    }
    return result
}
```

## 📝 Пример адаптированного метода

**SQLite version:**

```go
func (s *SQLiteDB) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
    query := `INSERT INTO tenants (id, name, slug) VALUES (?, ?, ?)`
    _, err = s.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Slug)
    return err
}
```

**PostgreSQL version:**

```go
func (p *PostgreSQLDB) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
    query := `INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`
    _, err = p.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Slug)
    return err
}
```

## ✅ Чек-лист для каждого файла

- [ ] Package name: `package postgresql`
- [ ] Тип: `*PostgreSQLDB` вместо `*SQLiteDB`
- [ ] Receiver: `p` вместо `s`
- [ ] Placeholders: `$1, $2, $3` вместо `?`
- [ ] BOOLEAN: `TRUE/FALSE` вместо `1/0`
- [ ] LIKE: `ILIKE` для case-insensitive
- [ ] Динамические placeholders: `fmt.Sprintf("$%d", index)`

## 🎯 Приоритет реализации

1. **transactions.go** - копия из sqlite/transactions.go с заменой типов
2. **apikeys.go** - критично для auth
3. **tenants.go** - нужно для multi-tenancy
4. **conversations.go** - для chat UI
5. **usage.go** - для статистики

## 🧪 Тестирование

После создания файлов:

```bash
# Компиляция
go build ./internal/storage/postgresql/...

# Unit tests
go test ./internal/storage/postgresql/...

# Integration test с реальным PostgreSQL
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=test postgres:16
go test -tags=integration ./internal/storage/postgresql/...
```

## 📚 Ссылки

- [PostgreSQL lib/pq documentation](https://pkg.go.dev/github.com/lib/pq)
- [PostgreSQL placeholder syntax](https://www.postgresql.org/docs/current/sql-prepare.html)
- [SQLite vs PostgreSQL differences](https://www.sqlite.org/different.html)
