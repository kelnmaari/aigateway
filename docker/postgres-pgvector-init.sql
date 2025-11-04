-- ═══════════════════════════════════════════════════════════════════════════
-- PostgreSQL + pgvector Initialization for RAG System
-- ═══════════════════════════════════════════════════════════════════════════
-- Version: 1.13.0+
-- Purpose: Initialize database with pgvector extension and RAG-specific schema
-- ═══════════════════════════════════════════════════════════════════════════

-- Set timezone to UTC
SET timezone = 'UTC';

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ═══════════════════════════════════════════════════════════════════════════
-- CRITICAL: Enable pgvector extension
-- ═══════════════════════════════════════════════════════════════════════════
CREATE EXTENSION IF NOT EXISTS vector;

-- Verify pgvector installation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
        RAISE EXCEPTION 'pgvector extension is not installed! Please use pgvector/pgvector Docker image.';
    END IF;
    
    RAISE NOTICE 'pgvector extension successfully enabled';
END $$;

-- ═══════════════════════════════════════════════════════════════════════════
-- Create schemas
-- ═══════════════════════════════════════════════════════════════════════════
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS rag;

-- ═══════════════════════════════════════════════════════════════════════════
-- Grant privileges
-- ═══════════════════════════════════════════════════════════════════════════
GRANT ALL PRIVILEGES ON SCHEMA public TO :POSTGRES_USER;
GRANT ALL PRIVILEGES ON SCHEMA rag TO :POSTGRES_USER;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO :POSTGRES_USER;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA rag TO :POSTGRES_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO :POSTGRES_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA rag TO :POSTGRES_USER;

-- ═══════════════════════════════════════════════════════════════════════════
-- RAG Schema: Vector Embeddings Storage
-- ═══════════════════════════════════════════════════════════════════════════

-- Document chunks with embeddings
CREATE TABLE IF NOT EXISTS rag.document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Document metadata
    document_id UUID NOT NULL,
    document_name TEXT NOT NULL,
    document_type TEXT,  -- pdf, docx, txt, csv, etc.
    
    -- Chunk metadata
    chunk_index INTEGER NOT NULL,
    chunk_text TEXT NOT NULL,
    chunk_tokens INTEGER,
    
    -- Embeddings (768 for nomic-embed-text, 1024 for mxbai-embed-large)
    embedding vector(1024),  -- Change dimension based on your model
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chunk_index_positive CHECK (chunk_index >= 0),
    CONSTRAINT chunk_tokens_positive CHECK (chunk_tokens > 0)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_document_chunks_document_id 
    ON rag.document_chunks(document_id);

CREATE INDEX IF NOT EXISTS idx_document_chunks_created_at 
    ON rag.document_chunks(created_at DESC);

-- ═══════════════════════════════════════════════════════════════════════════
-- CRITICAL: Vector similarity search indexes
-- ═══════════════════════════════════════════════════════════════════════════

-- HNSW index for fast approximate nearest neighbor search (recommended)
-- Best for: High-dimensional vectors, fast queries, memory-efficient
CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding_hnsw 
    ON rag.document_chunks 
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Alternative: IVFFlat index (uncomment if needed)
-- Best for: Large datasets, slightly slower but less memory
-- CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding_ivfflat 
--     ON rag.document_chunks 
--     USING ivfflat (embedding vector_cosine_ops)
--     WITH (lists = 100);

-- ═══════════════════════════════════════════════════════════════════════════
-- Documents metadata table
-- ═══════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS rag.documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Document info
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    size_bytes BIGINT,
    
    -- Storage info
    storage_path TEXT,
    s3_bucket TEXT,
    s3_key TEXT,
    
    -- Processing status
    status TEXT DEFAULT 'pending',  -- pending, processing, completed, failed
    error_message TEXT,
    
    -- Statistics
    total_chunks INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    
    -- User/tenant association
    user_id UUID,
    tenant_id UUID,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT status_valid CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_documents_user_id ON rag.documents(user_id);
CREATE INDEX IF NOT EXISTS idx_documents_tenant_id ON rag.documents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON rag.documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON rag.documents(created_at DESC);

-- ═══════════════════════════════════════════════════════════════════════════
-- RAG Collections (optional: organize documents into collections)
-- ═══════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS rag.collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    
    -- Owner
    user_id UUID,
    tenant_id UUID,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(name, user_id)
);

CREATE INDEX IF NOT EXISTS idx_collections_user_id ON rag.collections(user_id);
CREATE INDEX IF NOT EXISTS idx_collections_tenant_id ON rag.collections(tenant_id);

-- ═══════════════════════════════════════════════════════════════════════════
-- Collection-Document mapping (many-to-many)
-- ═══════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS rag.collection_documents (
    collection_id UUID NOT NULL REFERENCES rag.collections(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES rag.documents(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (collection_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_collection_documents_collection_id 
    ON rag.collection_documents(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_documents_document_id 
    ON rag.collection_documents(document_id);

-- ═══════════════════════════════════════════════════════════════════════════
-- RAG Job Queue (for async document processing)
-- ═══════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS rag.processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Job info
    document_id UUID REFERENCES rag.documents(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL,  -- 'extract', 'chunk', 'embed', 'index'
    
    -- Status
    status TEXT DEFAULT 'pending',  -- pending, running, completed, failed, cancelled
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Payload
    payload JSONB DEFAULT '{}',
    result JSONB,
    error_message TEXT,
    
    -- Timing
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    
    -- Lock mechanism for distributed processing
    locked_by TEXT,
    locked_until TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT status_valid CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT priority_range CHECK (priority >= 0 AND priority <= 10)
);

-- Indexes for job queue
CREATE INDEX IF NOT EXISTS idx_processing_jobs_status 
    ON rag.processing_jobs(status) WHERE status IN ('pending', 'running');
CREATE INDEX IF NOT EXISTS idx_processing_jobs_priority 
    ON rag.processing_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_processing_jobs_next_retry 
    ON rag.processing_jobs(next_retry_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_processing_jobs_locked_until 
    ON rag.processing_jobs(locked_until) WHERE locked_until IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════
-- Helper Functions
-- ═══════════════════════════════════════════════════════════════════════════

-- Update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION rag.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to all tables with updated_at
CREATE TRIGGER update_document_chunks_updated_at
    BEFORE UPDATE ON rag.document_chunks
    FOR EACH ROW EXECUTE FUNCTION rag.update_updated_at_column();

CREATE TRIGGER update_documents_updated_at
    BEFORE UPDATE ON rag.documents
    FOR EACH ROW EXECUTE FUNCTION rag.update_updated_at_column();

CREATE TRIGGER update_collections_updated_at
    BEFORE UPDATE ON rag.collections
    FOR EACH ROW EXECUTE FUNCTION rag.update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════════════
-- Similarity Search Function (example)
-- ═══════════════════════════════════════════════════════════════════════════
CREATE OR REPLACE FUNCTION rag.search_similar_chunks(
    query_embedding vector(1024),
    match_count INTEGER DEFAULT 5,
    similarity_threshold FLOAT DEFAULT 0.7
)
RETURNS TABLE (
    chunk_id UUID,
    document_id UUID,
    document_name TEXT,
    chunk_text TEXT,
    similarity FLOAT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        dc.id AS chunk_id,
        dc.document_id,
        dc.document_name,
        dc.chunk_text,
        1 - (dc.embedding <=> query_embedding) AS similarity,
        dc.metadata
    FROM rag.document_chunks dc
    WHERE 1 - (dc.embedding <=> query_embedding) > similarity_threshold
    ORDER BY dc.embedding <=> query_embedding
    LIMIT match_count;
END;
$$ LANGUAGE plpgsql;

-- ═══════════════════════════════════════════════════════════════════════════
-- Statistics View
-- ═══════════════════════════════════════════════════════════════════════════
CREATE OR REPLACE VIEW rag.statistics AS
SELECT
    (SELECT COUNT(*) FROM rag.documents) AS total_documents,
    (SELECT COUNT(*) FROM rag.documents WHERE status = 'completed') AS completed_documents,
    (SELECT COUNT(*) FROM rag.document_chunks) AS total_chunks,
    (SELECT SUM(size_bytes) FROM rag.documents) AS total_size_bytes,
    (SELECT COUNT(*) FROM rag.collections) AS total_collections,
    (SELECT COUNT(*) FROM rag.processing_jobs WHERE status = 'pending') AS pending_jobs,
    (SELECT COUNT(*) FROM rag.processing_jobs WHERE status = 'running') AS running_jobs;

-- Grant access to view
GRANT SELECT ON rag.statistics TO :POSTGRES_USER;

-- ═══════════════════════════════════════════════════════════════════════════
-- Success message
-- ═══════════════════════════════════════════════════════════════════════════
DO $$
BEGIN
    RAISE NOTICE '✓ PostgreSQL + pgvector initialized successfully';
    RAISE NOTICE '✓ RAG schema created with vector indexes';
    RAISE NOTICE '✓ Ready for RAG document processing';
END $$;

