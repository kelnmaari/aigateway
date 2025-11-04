-- Fix old RAG data source config to add connection_string

-- Update config for source 34d8d2ff-f743-44db-8417-03ccd2ed5709
UPDATE rag_data_sources
SET config = config 
    || '{"connection_string": "postgres://proxy_user:secure_password_change_me@192.168.1.101:32316/ollama_proxy?sslmode=disable"}'::jsonb
    || '{"database_type": "postgres"}'::jsonb
    || '{"host": "192.168.1.101"}'::jsonb
    || '{"port": 32316}'::jsonb
    || '{"database": "ollama_proxy"}'::jsonb
    || '{"username": "proxy_user"}'::jsonb
WHERE id = '34d8d2ff-f743-44db-8417-03ccd2ed5709';

-- Verify the result
SELECT 
    id, 
    name, 
    source_type,
    config->>'connection_string' as connection_string,
    config->>'query' as query,
    status
FROM rag_data_sources 
WHERE id = '34d8d2ff-f743-44db-8417-03ccd2ed5709';

