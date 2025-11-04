-- ═══════════════════════════════════════════════════════════════════════════
-- PostgreSQL Performance Tuning for RAG Workloads
-- ═══════════════════════════════════════════════════════════════════════════
-- Version: 1.13.0+
-- Purpose: Optimize PostgreSQL for vector similarity search and RAG operations
-- ═══════════════════════════════════════════════════════════════════════════

-- ═══════════════════════════════════════════════════════════════════════════
-- Memory Configuration
-- ═══════════════════════════════════════════════════════════════════════════

-- shared_buffers: Cache for database blocks (default: 128MB)
-- Рекомендация: 25% от RAM сервера (но не более 8GB без дополнительной настройки)
-- Для RAG: Увеличиваем для кеширования векторных индексов
ALTER SYSTEM SET shared_buffers = '512MB';

-- effective_cache_size: Оценка доступной памяти для кеша ОС
-- Рекомендация: 50-75% от RAM сервера
-- Для RAG: Помогает планировщику выбирать оптимальные планы запросов
ALTER SYSTEM SET effective_cache_size = '2GB';

-- work_mem: Память для сортировок и хеш-таблиц в каждой операции
-- Рекомендация: (Total RAM * 0.25) / max_connections
-- Для RAG: Увеличиваем для сортировки векторных результатов
ALTER SYSTEM SET work_mem = '64MB';

-- maintenance_work_mem: Память для VACUUM, CREATE INDEX, ALTER TABLE
-- Рекомендация: min(RAM / 16, 2GB)
-- Для RAG: КРИТИЧНО для построения векторных индексов (HNSW)
ALTER SYSTEM SET maintenance_work_mem = '256MB';

-- ═══════════════════════════════════════════════════════════════════════════
-- Connection Pool Configuration
-- ═══════════════════════════════════════════════════════════════════════════

-- max_connections: Максимум одновременных подключений
-- Для RAG: Достаточно для API + worker threads
ALTER SYSTEM SET max_connections = '200';

-- ═══════════════════════════════════════════════════════════════════════════
-- WAL (Write-Ahead Log) Configuration
-- ═══════════════════════════════════════════════════════════════════════════

-- wal_buffers: Буферы для WAL (default: -1 = auto 3% of shared_buffers)
-- Для RAG: Оптимизируем для bulk insert операций (chunking, embeddings)
ALTER SYSTEM SET wal_buffers = '16MB';

-- checkpoint_completion_target: Распределение checkpoint I/O
-- Рекомендация: 0.9 для production (плавная запись)
ALTER SYSTEM SET checkpoint_completion_target = '0.9';

-- max_wal_size: Максимальный размер WAL между checkpoints
-- Для RAG: Увеличиваем для bulk loading документов
ALTER SYSTEM SET max_wal_size = '1GB';

-- min_wal_size: Минимальный размер WAL
ALTER SYSTEM SET min_wal_size = '80MB';

-- wal_compression: Сжатие WAL записей
-- Для RAG: Экономит место при вставке больших текстов
ALTER SYSTEM SET wal_compression = 'on';

-- ═══════════════════════════════════════════════════════════════════════════
-- Query Planner Configuration
-- ═══════════════════════════════════════════════════════════════════════════

-- random_page_cost: Относительная стоимость random I/O
-- Рекомендация: 1.1 для SSD, 4.0 для HDD (default)
-- Для RAG: SSD оптимизация для векторных индексов
ALTER SYSTEM SET random_page_cost = '1.1';

-- effective_io_concurrency: Параллельные I/O операции
-- Рекомендация: 200 для SSD, 2 для HDD
-- Для RAG: Ускоряет vector index scan
ALTER SYSTEM SET effective_io_concurrency = '200';

-- default_statistics_target: Размер статистики для планировщика
-- Рекомендация: 100-500 для критичных колонок
-- Для RAG: Помогает с selectivity estimation на векторных запросах
ALTER SYSTEM SET default_statistics_target = '100';

-- ═══════════════════════════════════════════════════════════════════════════
-- Autovacuum Configuration (критично для производительности!)
-- ═══════════════════════════════════════════════════════════════════════════

-- autovacuum: Включить автоочистку (всегда ON в production!)
ALTER SYSTEM SET autovacuum = 'on';

-- autovacuum_max_workers: Количество параллельных autovacuum workers
-- Для RAG: Увеличиваем из-за частых INSERT/UPDATE на chunks
ALTER SYSTEM SET autovacuum_max_workers = '4';

-- autovacuum_naptime: Интервал между запусками autovacuum
-- Для RAG: Чаще проверяем из-за активной записи
ALTER SYSTEM SET autovacuum_naptime = '30s';

-- Aggressive autovacuum для RAG таблиц (много INSERT/DELETE)
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = '0.1';  -- default: 0.2
ALTER SYSTEM SET autovacuum_analyze_scale_factor = '0.05'; -- default: 0.1

-- ═══════════════════════════════════════════════════════════════════════════
-- Parallel Query Configuration
-- ═══════════════════════════════════════════════════════════════════════════

-- max_parallel_workers_per_gather: Workers для параллельных запросов
-- Для RAG: Ускоряет vector similarity search на больших dataset
ALTER SYSTEM SET max_parallel_workers_per_gather = '4';

-- max_parallel_workers: Общее количество parallel workers
ALTER SYSTEM SET max_parallel_workers = '8';

-- max_worker_processes: Общее количество background workers
ALTER SYSTEM SET max_worker_processes = '16';

-- parallel_tuple_cost: Стоимость передачи tuple между workers
-- Для RAG: Снижаем для поощрения параллелизма на vector ops
ALTER SYSTEM SET parallel_tuple_cost = '0.05';

-- ═══════════════════════════════════════════════════════════════════════════
-- Logging Configuration (для debugging RAG queries)
-- ═══════════════════════════════════════════════════════════════════════════

-- log_min_duration_statement: Логировать медленные запросы
-- Для RAG: Отслеживаем медленные vector similarity searches
ALTER SYSTEM SET log_min_duration_statement = '1000';  -- 1 second

-- log_line_prefix: Формат логов
ALTER SYSTEM SET log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ';

-- log_checkpoints: Логировать checkpoints
ALTER SYSTEM SET log_checkpoints = 'on';

-- log_connections: Логировать подключения
ALTER SYSTEM SET log_connections = 'on';

-- log_disconnections: Логировать отключения
ALTER SYSTEM SET log_disconnections = 'on';

-- log_lock_waits: Логировать lock waits
-- Для RAG: Detect contention на векторных таблицах
ALTER SYSTEM SET log_lock_waits = 'on';

-- ═══════════════════════════════════════════════════════════════════════════
-- Extension-specific: pgvector optimization
-- ═══════════════════════════════════════════════════════════════════════════

-- Увеличиваем maintenance_work_mem для HNSW index build
-- HNSW требует много памяти при построении индекса
-- Рекомендация: min(RAM / 4, 2GB) для больших dataset
COMMENT ON EXTENSION vector IS 
'pgvector extension - stores embeddings in vector columns. 
HNSW index building requires high maintenance_work_mem (256MB+).
Index build time: O(N * M * ef_construction * log(N))
Query time: O(ef_search * log(N))';

-- ═══════════════════════════════════════════════════════════════════════════
-- Table-specific optimizations for RAG schema
-- ═══════════════════════════════════════════════════════════════════════════

-- document_chunks: Aggressive autovacuum (много INSERT/DELETE)
ALTER TABLE IF EXISTS rag.document_chunks SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_vacuum_cost_delay = 5,
    fillfactor = 90  -- Reserve 10% for HOT updates
);

-- processing_jobs: Частые UPDATE (status changes)
ALTER TABLE IF EXISTS rag.processing_jobs SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05,
    fillfactor = 80  -- Reserve 20% для HOT updates на status
);

-- documents: Средняя частота UPDATE
ALTER TABLE IF EXISTS rag.documents SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05,
    fillfactor = 90
);

-- ═══════════════════════════════════════════════════════════════════════════
-- Statistics Collection for Vector Columns
-- ═══════════════════════════════════════════════════════════════════════════

-- Увеличиваем statistics_target для vector колонок
-- Помогает планировщику с selectivity estimation
ALTER TABLE IF EXISTS rag.document_chunks 
    ALTER COLUMN embedding SET STATISTICS 500;

-- ═══════════════════════════════════════════════════════════════════════════
-- Apply configuration changes
-- ═══════════════════════════════════════════════════════════════════════════

-- Reload configuration (некоторые параметры требуют restart)
SELECT pg_reload_conf();

-- ═══════════════════════════════════════════════════════════════════════════
-- Verification Queries
-- ═══════════════════════════════════════════════════════════════════════════

-- Show current configuration
DO $$
DECLARE
    v_shared_buffers TEXT;
    v_work_mem TEXT;
    v_maintenance_work_mem TEXT;
    v_effective_cache_size TEXT;
BEGIN
    SELECT setting INTO v_shared_buffers 
        FROM pg_settings WHERE name = 'shared_buffers';
    SELECT setting INTO v_work_mem 
        FROM pg_settings WHERE name = 'work_mem';
    SELECT setting INTO v_maintenance_work_mem 
        FROM pg_settings WHERE name = 'maintenance_work_mem';
    SELECT setting INTO v_effective_cache_size 
        FROM pg_settings WHERE name = 'effective_cache_size';
    
    RAISE NOTICE '═══════════════════════════════════════════════════════════';
    RAISE NOTICE '✓ PostgreSQL Performance Tuning Applied';
    RAISE NOTICE '═══════════════════════════════════════════════════════════';
    RAISE NOTICE 'shared_buffers: %', v_shared_buffers;
    RAISE NOTICE 'work_mem: %', v_work_mem;
    RAISE NOTICE 'maintenance_work_mem: %', v_maintenance_work_mem;
    RAISE NOTICE 'effective_cache_size: %', v_effective_cache_size;
    RAISE NOTICE '═══════════════════════════════════════════════════════════';
    RAISE NOTICE 'NOTE: Some settings require PostgreSQL restart to take effect';
    RAISE NOTICE 'Run: docker-compose restart postgres-pgvector';
    RAISE NOTICE '═══════════════════════════════════════════════════════════';
END $$;

-- ═══════════════════════════════════════════════════════════════════════════
-- Performance Monitoring Queries (for reference)
-- ═══════════════════════════════════════════════════════════════════════════

-- Check index usage
CREATE OR REPLACE VIEW rag.index_usage AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'rag'
ORDER BY idx_scan DESC;

-- Check table sizes
CREATE OR REPLACE VIEW rag.table_sizes AS
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) AS indexes_size
FROM pg_tables
WHERE schemaname = 'rag'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Grant access to monitoring views
GRANT SELECT ON rag.index_usage TO PUBLIC;
GRANT SELECT ON rag.table_sizes TO PUBLIC;

-- ═══════════════════════════════════════════════════════════════════════════
-- Useful monitoring queries for RAG performance
-- ═══════════════════════════════════════════════════════════════════════════

COMMENT ON VIEW rag.index_usage IS 
'Monitor vector index usage. Low idx_scan means index not used.
For HNSW indexes, idx_tup_read shows number of vector comparisons.';

COMMENT ON VIEW rag.table_sizes IS 
'Monitor storage usage. Vector columns consume significant space.
Each vector(1024) dimension takes ~4KB per row.';

-- Example: Check slow queries on vector tables
CREATE OR REPLACE VIEW rag.slow_queries AS
SELECT
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
WHERE query LIKE '%rag.document_chunks%'
    OR query LIKE '%vector%'
    OR query LIKE '%<=>%'  -- cosine distance operator
ORDER BY mean_time DESC
LIMIT 20;

-- ═══════════════════════════════════════════════════════════════════════════
-- END OF TUNING SCRIPT
-- ═══════════════════════════════════════════════════════════════════════════

