-- Fix is_shared column type from INTEGER to BOOLEAN in rag_data_sources
-- Step 1: Drop the default
ALTER TABLE rag_data_sources ALTER COLUMN is_shared DROP DEFAULT;

-- Step 2: Change the type (безопасная конвертация через текст)
ALTER TABLE rag_data_sources 
    ALTER COLUMN is_shared TYPE BOOLEAN 
    USING CASE 
        WHEN is_shared::text IN ('1', 't', 'true', 'TRUE', 'yes', 'on') THEN TRUE 
        ELSE FALSE 
    END;

-- Step 3: Set new default
ALTER TABLE rag_data_sources ALTER COLUMN is_shared SET DEFAULT FALSE;

