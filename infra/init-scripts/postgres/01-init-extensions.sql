-- AIGateway PostgreSQL Initialization Script
-- This script runs automatically when the container is first created

-- Enable pgvector extension for RAG embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Enable uuid-ossp for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Confirm extensions are installed
SELECT extname, extversion FROM pg_extension WHERE extname IN ('vector', 'uuid-ossp');

