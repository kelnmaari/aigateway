-- Rollback: Revert is_shared column type from BOOLEAN to INTEGER
-- Step 1: Drop the boolean default
ALTER TABLE rag_data_sources ALTER COLUMN is_shared DROP DEFAULT;

-- Step 2: Change type back to INTEGER
ALTER TABLE rag_data_sources 
    ALTER COLUMN is_shared TYPE INTEGER 
    USING CASE WHEN is_shared = TRUE THEN 1 ELSE 0 END;

-- Step 3: Set integer default
ALTER TABLE rag_data_sources ALTER COLUMN is_shared SET DEFAULT 0;

