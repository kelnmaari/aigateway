-- Worker nodes for Agent Mode (remote inference workers)
CREATE TABLE IF NOT EXISTS worker_nodes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(255) NOT NULL UNIQUE,
    address           VARCHAR(512) NOT NULL,                    -- https://host:port
    api_key           VARCHAR(512) NOT NULL,                    -- encrypted shared secret
    status            VARCHAR(50)  NOT NULL DEFAULT 'pending',  -- pending | online | offline | draining
    node_type         VARCHAR(50)  NOT NULL DEFAULT 'gpu',      -- gpu | cpu | mixed
    gpu_info          JSONB        NOT NULL DEFAULT '[]'::jsonb, -- [{index,name,memory_total_mb,...}]
    cpu_info          JSONB        NOT NULL DEFAULT '{}'::jsonb,  -- {model,cores,threads}
    memory_info       JSONB        NOT NULL DEFAULT '{}'::jsonb,  -- {total_mb,used_mb,free_mb}
    models_running    JSONB        NOT NULL DEFAULT '[]'::jsonb, -- ["alias1","alias2"]
    max_running_models INT         NOT NULL DEFAULT 2,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_health_check TIMESTAMPTZ,
    last_error        TEXT
);

CREATE INDEX IF NOT EXISTS idx_worker_nodes_status ON worker_nodes(status);
CREATE INDEX IF NOT EXISTS idx_worker_nodes_node_type ON worker_nodes(node_type);
