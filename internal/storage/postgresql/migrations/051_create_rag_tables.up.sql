
-- ========================================
-- RAG Data Sources Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_data_sources (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    
    -- Основная информация
    name TEXT NOT NULL,
    description TEXT,
    source_type TEXT NOT NULL CHECK (source_type IN ('file', 'api', 'database', 'web')),
    
    -- Конфигурация (JSON)
    config JSONB NOT NULL DEFAULT '{}',
    
    -- Credentials (зашифрованные AES-256)
    credentials_encrypted TEXT,
    
    -- Статус и метрики
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error', 'syncing')),
    last_sync_at TIMESTAMP,
    last_sync_status TEXT CHECK (last_sync_status IN ('success', 'failed', 'partial')),
    last_error TEXT,
    sync_frequency TEXT,  -- "6h", "daily", "weekly"
    
    -- Настройки индексации
    indexing_config JSONB NOT NULL DEFAULT '{}',
    
    -- Статистика
    total_chunks INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    last_chunk_count INTEGER,
    
    -- Метаданные
    tags JSONB, --   array
    is_shared BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_sources_user ON rag_data_sources(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_tenant ON rag_data_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_type ON rag_data_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_rag_sources_status ON rag_data_sources(status);

-- ========================================
-- RAG Documents Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_documents (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    
    -- Файл информация
    filename TEXT,
    mime_type TEXT,
    size_bytes INTEGER,
    
    -- Хранилище
    storage_backend TEXT,  -- 'local', 's3'
    storage_path TEXT NOT NULL,
    storage_bucket TEXT,  -- для S3
    
    -- Обработка
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_error TEXT,
    
    -- Метаданные документа
    metadata JSONB DEFAULT '{}',
    
    -- Статистика
    total_chunks INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_docs_source ON rag_documents(source_id);
CREATE INDEX IF NOT EXISTS idx_rag_docs_status ON rag_documents(status);

-- ========================================
-- RAG Chunks Table (v1.13.1)
-- Note: Embeddings будут добавлены в v1.13.3 после pgvector setup
-- ========================================
CREATE TABLE IF NOT EXISTS rag_chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    source_id TEXT NOT NULL,
    
    -- Chunk контент
    chunk_text TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,  -- позиция в документе
    chunk_tokens INTEGER NOT NULL,
    
    -- Метаданные чанка
    metadata JSONB DEFAULT '{}',  -- page_number, headers, context, etc.
    
    -- Для overlap detection
    start_offset INTEGER,
    end_offset INTEGER,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (document_id) REFERENCES rag_documents(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_document ON rag_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_source ON rag_chunks(source_id);

-- ========================================
-- RAG Jobs Queue (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_type TEXT NOT NULL,  -- 'file_upload', 'api_sync', 'db_query', 'web_scrape'
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    
    -- Данные
    payload TEXT NOT NULL,  -- JSON
    result JSONB,            
    
    -- Приоритет и повторы
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Ошибки
    error TEXT,
    
    -- Для visibility timeout
    locked_until TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rag_jobs_status ON rag_jobs(status, priority DESC, created_at);
CREATE INDEX IF NOT EXISTS idx_rag_jobs_type ON rag_jobs(job_type);

-- ========================================
-- RAG Query Logs (v1.13.1 - аналитика)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_query_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT,
    conversation_id TEXT,
    
    -- Запрос
    query_text TEXT NOT NULL,
    
    -- Использованные источники (JSON array of IDs)
    source_ids TEXT,
    
    -- Результаты поиска
    chunks_retrieved INTEGER,
    chunks_used INTEGER,
    
    -- Метрики
    search_time_ms INTEGER,
    total_tokens_used INTEGER,
    
    -- Результат
    response_quality_score REAL,  -- опционально, от пользователя
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_rag_query_logs_user ON rag_query_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_query_logs_created ON rag_query_logs(created_at DESC);
    